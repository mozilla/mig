// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"mig"
	"mig/pgp"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/howeyc/fsnotify"
	"github.com/streadway/amqp"
)

// build version
var version string

// the list of active agents is shared globally
// TODO: make this a database thing to work with scheduler clusters
var activeAgentsList []string

// main initializes the mongodb connection, the directory watchers and the
// AMQP ctx.MQ.Chan. It also launches the goroutines.
func main() {
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)

	// command line options
	var config = flag.String("c", "/etc/mig/scheduler.cfg", "Load configuration from file")
	flag.Parse()

	// The context initialization takes care of parsing the configuration,
	// and creating connections to database, message broker, syslog, ...
	fmt.Fprintf(os.Stderr, "Initializing Scheduler context...")
	ctx, err := Init(*config)
	if err != nil {
		fmt.Printf("\nFATAL: %v\n", err)
		os.Exit(9)
	}
	fmt.Fprintf(os.Stderr, "OK\n")

	// Goroutine that handles events, such as logs and panics,
	// and decides what to do with them
	go func() {
		for event := range ctx.Channels.Log {
			stop, err := mig.ProcessLog(ctx.Logging, event)
			if err != nil {
				panic("Unable to process logs")
			}
			// if ProcessLog says we should stop now, feed the Terminate chan
			if stop {
				ctx.Channels.Terminate <- errors.New(event.Desc)
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "mig.ProcessLog() routine started"}

	// Watch the data directories for new files
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	go watchDirectories(watcher, ctx)
	// TODO: looks like the watchers are lost after a while. the (ugly) loop
	// below reinits the watchers every 137 seconds to prevent that from happening
	go func() {
		for {
			err = initWatchers(watcher, ctx)
			if err != nil {
				panic(err)
			}
			time.Sleep(137 * time.Second)
		}
	}()

	// Goroutine that loads actions dropped into ctx.Directories.Action.New
	go func() {
		for actionPath := range ctx.Channels.NewAction {
			ctx.OpID = mig.GenID()
			err := processNewAction(actionPath, ctx)
			// if something fails in the action processing, move it to the invalid folder
			if err != nil {
				// move action to INVALID folder and log
				dest := fmt.Sprintf("%s/%d.json", ctx.Directories.Action.Invalid, time.Now().UTC().UnixNano())
				os.Rename(actionPath, dest)
				reason := fmt.Sprintf("%v. '%s' moved to '%s'", err, actionPath, dest)
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: reason}.Warning()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "processNewAction() routine started"}

	// Goroutine that loads and sends commands dropped in ready state
	// it uses a select and a timeout to load a batch of commands instead of
	// sending them one by one
	go func() {
		ctx.OpID = mig.GenID()
		readyCmd := make(map[float64]mig.Command)
		ctr := 0
		for {
			select {
			case cmd := <-ctx.Channels.CommandReady:
				ctr++
				readyCmd[cmd.ID] = cmd
			case <-time.After(1 * time.Second):
				if ctr > 0 {
					var cmds []mig.Command
					for id, cmd := range readyCmd {
						cmds = append(cmds, cmd)
						delete(readyCmd, id)
					}
					err := sendCommands(cmds, ctx)
					if err != nil {
						ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
					}
				}
				// reinit
				ctx.OpID = mig.GenID()
				ctr = 0
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "sendCommands() routine started"}

	// Goroutine that loads commands from the ctx.Directories.Command.Returned and marks
	// them as finished or cancelled
	go func() {
		ctx.OpID = mig.GenID()
		returnedCmd := make(map[uint64]string)
		var ctr uint64 = 0
		for {
			select {
			case cmdFile := <-ctx.Channels.CommandReturned:
				ctr++
				returnedCmd[ctr] = cmdFile
			case <-time.After(1 * time.Second):
				if ctr > 0 {
					var cmdFiles []string
					for id, cmdFile := range returnedCmd {
						cmdFiles = append(cmdFiles, cmdFile)
						delete(returnedCmd, id)
					}
					err := returnCommands(cmdFiles, ctx)
					if err != nil {
						ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
					}
				}
				// reinit
				ctx.OpID = mig.GenID()
				ctr = 0
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "terminateCommand() routine started"}

	// Goroutine that updates an action when a command is done
	go func() {
		ctx.OpID = mig.GenID()
		doneCmd := make(map[float64]mig.Command)
		ctr := 0
		for {
			select {
			case cmd := <-ctx.Channels.CommandDone:
				ctr++
				doneCmd[cmd.ID] = cmd
			case <-time.After(1 * time.Second):
				if ctr > 0 {
					var cmds []mig.Command
					for id, cmd := range doneCmd {
						cmds = append(cmds, cmd)
						delete(doneCmd, id)
					}
					err := updateAction(cmds, ctx)
					if err != nil {
						ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
					}
				}
				// reinit
				ctx.OpID = mig.GenID()
				ctr = 0
			}
		}

	}()
	ctx.Channels.Log <- mig.Log{Desc: "updateAction() routine started"}

	// Init is completed, restart queues of active agents
	err = pickUpAliveAgents(ctx)
	if err != nil {
		panic(err)
	}

	// start a listening channel to receive heartbeats from agents
	activeAgentsChan, err := startActiveAgentsChannel(ctx)
	if err != nil {
		panic(err)
	}

	// launch the routine that handles registrations
	go func() {
		for msg := range activeAgentsChan {
			ctx.OpID = mig.GenID()
			err := getHeartbeats(msg, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "Agent KeepAlive routine started"}

	// launch the routine that regularly walks through the local directories
	go func() {
		collectorSleeper, err := time.ParseDuration(ctx.Collector.Freq)
		if err != nil {
			panic(err)
		}
		for {
			ctx.OpID = mig.GenID()
			err := spoolInspection(ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
			}
			time.Sleep(collectorSleeper)
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "spoolInspection() routine started"}

	// launch the routine that handles multi agents on same queue
	go func() {
		for queueLoc := range ctx.Channels.DetectDupAgents {
			ctx.OpID = mig.GenID()
			err = inspectMultiAgents(queueLoc, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "inspectMultiAgents() routine started"}

	// won't exit until this chan received something
	exitReason := <-ctx.Channels.Terminate
	fmt.Fprintf(os.Stderr, "Scheduler is shutting down. Reason: %s", exitReason)
	Destroy(ctx)
}

// initWatchers initializes the watcher flags for all the monitored directories
func initWatchers(watcher *fsnotify.Watcher, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initWatchers() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initWatchers()"}.Debug()
	}()

	err = watcher.WatchFlags(ctx.Directories.Action.New, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Action.New)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Action.New)}.Debug()

	err = watcher.WatchFlags(ctx.Directories.Command.InFlight, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Command.InFlight)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Command.InFlight)}.Debug()

	err = watcher.WatchFlags(ctx.Directories.Command.Returned, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Command.Returned)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Command.Returned)}.Debug()

	err = watcher.WatchFlags(ctx.Directories.Action.Done, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Action.Done)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Action.Done)}.Debug()

	return
}

// watchDirectories calls specific function when a file appears in a watched directory
func watchDirectories(watcher *fsnotify.Watcher, ctx Context) {
	for {
		select {
		case ev := <-watcher.Event:
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watchDirectories(): %s", ev.String())}.Debug()

			// New file detected, but the file size might still be zero, because inotify wakes up before
			// the file is fully written. If that's the case, wait a little and hope that's enough to finish writing
			if fileHasSizeZero(ev.Name) {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("file '%s' is size zero. waiting for write to finish.", ev.Name)}.Debug()
				time.Sleep(200 * time.Millisecond)
			}

			// Use the prefix of the filename to send it to the appropriate channel
			if strings.HasPrefix(ev.Name, ctx.Directories.Action.New) {
				ctx.Channels.NewAction <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.InFlight) {
				ctx.Channels.UpdateCommand <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.Returned) {
				ctx.Channels.CommandReturned <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Action.Done) {
				ctx.Channels.ActionDone <- ev.Name
			}
		case err := <-watcher.Error:
			// in case of error, raise an emergency
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watchDirectories(): %v", err)}.Emerg()
		}
	}
}

// processNewAction is called when a new action is available. It pulls
// the action from the directory, parse it, retrieve a list of targets from
// the backend database, and create individual command for each target.
func processNewAction(actionPath string, ctx Context) (err error) {
	var action mig.Action
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("processNewAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: "leaving processNewAction()"}.Debug()
	}()
	// load the action file
	action, err = mig.ActionFromFile(actionPath)
	if err != nil {
		panic(err)
	}
	action.StartTime = time.Now()
	// generate an action id
	if action.ID < 1 {
		action.ID = mig.GenID()
	}
	desc := fmt.Sprintf("new action received: Name='%s' Target='%s' ValidFrom='%s' ExpireAfter='%s'",
		action.Name, action.Target, action.ValidFrom, action.ExpireAfter)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: desc}
	// TODO: replace with action.Validate(), to include signature verification
	if time.Now().Before(action.ValidFrom) {
		// queue new action
		desc := fmt.Sprintf("action '%s' is not ready for scheduling", action.Name)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: desc}.Debug()
		return
	}
	if time.Now().After(action.ExpireAfter) {
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: fmt.Sprintf("action '%s' is expired. invalidating.", action.Name)}
		err = invalidAction(ctx, action, actionPath)
		if err != nil {
			panic(err)
		}
		return
	}
	// find target agents for the action
	agents, err := ctx.DB.ActiveAgentsByTarget(action.Target)
	if err != nil {
		panic(err)
	}
	action.Counters.Sent = len(agents)
	if action.Counters.Sent == 0 {
		err = fmt.Errorf("No agents found for target '%s'. invalidating action.", action.Target)
		err = invalidAction(ctx, action, actionPath)
		if err != nil {
			panic(err)
		}
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: fmt.Sprintf("Found %d target agents", action.Counters.Sent)}

	action.Status = "preparing"
	inserted, err := ctx.DB.InsertOrUpdateAction(action)
	if err != nil {
		panic(err)
	}
	if inserted {
		// action was inserted, and not updated, so we need to insert
		// the signatures as well
		astr, err := action.String()
		if err != nil {
			panic(err)
		}
		for _, sig := range action.PGPSignatures {
			pubring, err := getPubring(ctx)
			if err != nil {
				panic(err)
			}
			fp, err := pgp.GetFingerprintFromSignature(astr, sig, pubring)
			if err != nil {
				panic(err)
			}
			inv, err := ctx.DB.InvestigatorByFingerprint(fp)
			if err != nil {
				panic(err)
			}
			err = ctx.DB.InsertSignature(action.ID, inv.ID, sig)
			if err != nil {
				panic(err)
			}
		}
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: "Action written to database"}.Debug()

	// create an array of empty results to serve as default for all commands
	emptyResults := make([]mig.ModuleResult, len(action.Operations))
	created := 0
	for _, agent := range agents {
		err := createCommand(ctx, action, agent, emptyResults)
		if err != nil {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: "Failed to create commmand on agent" + agent.Name}.Err()
			continue
		}
		created++
	}

	if created == 0 {
		// no command created found
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: "No command created. Invalidating action."}.Err()
		err = invalidAction(ctx, action, actionPath)
		if err != nil {
			panic(err)
		}
		return nil
	}
	// move action to flying state
	err = flyAction(ctx, action, actionPath)
	if err != nil {
		panic(err)
	}
	return
}

func createCommand(ctx Context, action mig.Action, agent mig.Agent, emptyResults []mig.ModuleResult) (err error) {
	cmdid := mig.GenID()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createCommand() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, CommandID: cmdid, Desc: "leaving createCommand()"}.Debug()
	}()
	var cmd mig.Command
	cmd.Status = "sent"
	cmd.Action = action
	cmd.Agent = agent
	cmd.ID = cmdid
	cmd.StartTime = time.Now().UTC()
	cmd.Results = emptyResults
	ctx.Channels.CommandReady <- cmd
	desc := fmt.Sprintf("created command for action '%s' on agent '%s'", action.Name, agent.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, CommandID: cmdid, Desc: desc}
	return
}

// sendCommand is called when a command file is created in ctx.Directories.Command.Ready
// it read the command, sends it to the agent via AMQP, and update the DB
func sendCommands(cmds []mig.Command, ctx Context) (err error) {
	aid := cmds[0].ID
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendCommand() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: aid, Desc: "leaving sendCommand()"}.Debug()
	}()
	// store all the commands into the database at once
	insertCount, err := ctx.DB.InsertCommands(cmds)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: aid, Desc: fmt.Sprintf("%d commands inserted into database", insertCount)}

	for _, cmd := range cmds {
		data, err := json.Marshal(cmd)
		if err != nil {
			panic(err)
		}
		expire := cmd.Action.ExpireAfter.Sub(cmd.Action.ValidFrom)
		// build amqp message for sending
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "text/plain",
			Expiration:   fmt.Sprintf("%d", int64(expire/time.Millisecond)),
			Body:         []byte(data),
		}
		agtQueue := fmt.Sprintf("mig.agt.%s", cmd.Agent.QueueLoc)
		// send
		go func() {
			err = ctx.MQ.Chan.Publish("mig", agtQueue, true, false, msg)
			if err != nil {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "Failed to publish command to agent queue"}.Err()
			} else {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "command sent to agent queue"}.Debug()
			}
		}()
		// write command to InFlight directory
		dest := fmt.Sprintf("%s/%.0f-%.0f.json", ctx.Directories.Command.InFlight, cmd.Action.ID, cmd.ID)
		err = safeWrite(ctx, dest, data)
		if err != nil {
			panic(err)
		}
	}
	return
}

// recvAgentResults listens on the AMQP channel for command results from agents
// each iteration processes one command received from one agent. The json.Body
// in the message is extracted and written into ctx.Directories.Command.Done
func recvAgentResults(msg amqp.Delivery, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("recvAgentResults() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving recvAgentResults()"}.Debug()
	}()

	// write to disk Returned directory
	dest := fmt.Sprintf("%s/%.0f", ctx.Directories.Command.Returned, mig.GenID())
	err = safeWrite(ctx, dest, msg.Body)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("Received result from '%s'", msg.RoutingKey)}.Debug()

	return
}

// returnCommands is called when commands have returned
// it stores the result of a command and mark it as completed/failed and then
// send a message to the Action completion routine to update the action status
func returnCommands(cmdFiles []string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("terminateCommand() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving terminateCommand()"}.Debug()
	}()
	for _, cmdFile := range cmdFiles {
		// load and parse the command. If this fail, skip it and continue.
		cmd, err := mig.CmdFromFile(cmdFile)
		if err != nil {
			panic(err)
		}
		//FIXME: backward compatibility hack, must remove after Dec. 1st 2014
		if cmd.Status == "done" {
			cmd.Status = mig.StatusSuccess
		}
		cmd.FinishTime = time.Now().UTC()
		// update command in database
		go func() {
			err = ctx.DB.FinishCommand(cmd)
			if err != nil {
				desc := fmt.Sprintf("failed to finish command in database: '%v'", err)
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: desc}.Err()
			} else {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "command updated in database"}.Debug()
			}
		}()
		// remove command from inflight dir
		inflightPath := fmt.Sprintf("%s/%.0f-%.0f.json", ctx.Directories.Command.InFlight, cmd.Action.ID, cmd.ID)
		os.Remove(inflightPath)

		// remove command from Returned dir
		os.Remove(cmdFile)

		// pass the command over to the Command Done channel
		ctx.Channels.CommandDone <- cmd
	}
	return
}

// updateAction is called with an array of commands that have finished
// Each action that needs updating is processed in a way that reduce IOs
func updateAction(cmds []mig.Command, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("updateAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving updateAction()"}.Debug()
	}()

	// there may be multiple actions to update, since commands can be mixed,
	// so we keep a map of actions
	actions := make(map[float64]mig.Action)

	for _, cmd := range cmds {
		var a mig.Action
		// retrieve the action from the DB if we don't already have it mapped
		a, ok := actions[cmd.Action.ID]
		if !ok {
			a, err = ctx.DB.ActionMetaByID(cmd.Action.ID)
			if err != nil {
				panic(err)
			}
		}
		a.LastUpdateTime = time.Now().UTC()

		// store action in the map
		actions[a.ID] = a

		// slightly unrelated to updating the action:
		// in case the action is about upgrading agents, do some magical stuff
		// to continue the upgrade protocol
		if cmd.Status == mig.StatusSuccess && len(a.Operations) > 0 {
			if a.Operations[0].Module == "upgrade" {
				go markUpgradedAgents(cmd, ctx)
			}
		}
	}
	for _, a := range actions {
		a.Counters, err = ctx.DB.GetActionCounters(a.ID)
		if err != nil {
			panic(err)
		}
		// Has the action completed?
		if a.Counters.Done == a.Counters.Sent {
			err = landAction(ctx, a)
			if err != nil {
				panic(err)
			}
			// delete Action from ctx.Directories.Action.InFlight
			actFile := fmt.Sprintf("%.0f.json", a.ID)
			os.Rename(ctx.Directories.Action.InFlight+"/"+actFile, ctx.Directories.Action.Done+"/"+actFile)
		} else {
			// store updated action in database
			err = ctx.DB.UpdateRunningAction(a)
			if err != nil {
				panic(err)
			}
			desc := fmt.Sprintf("updated action '%s': progress=%d/%d, success=%d, cancelled=%d, expired=%d, failed=%d, timeout=%d, duration=%s",
				a.Name, a.Counters.Done, a.Counters.Sent, a.Counters.Success, a.Counters.Cancelled, a.Counters.Expired,
				a.Counters.Failed, a.Counters.TimeOut, a.LastUpdateTime.Sub(a.StartTime).String())
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: desc}
		}
	}
	return
}

func fileHasSizeZero(filepath string) bool {
	fd, _ := os.Open(filepath)
	defer fd.Close()
	fi, _ := fd.Stat()
	if fi.Size() == 0 {
		return true
	} else {
		return false
	}
}
