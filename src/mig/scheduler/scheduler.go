/* Mozilla InvestiGator Scheduler

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2013
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

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
		panic(err)
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
	err = initWatchers(watcher, ctx)
	if err != nil {
		panic(err)
	}

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

	// Goroutine that loads and sends commands dropped into ctx.Directories.Command.Ready
	go func() {
		for cmdPath := range ctx.Channels.CommandReady {
			ctx.OpID = mig.GenID()
			err := sendCommand(cmdPath, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "sendCommand() routine started"}

	// Goroutine that loads commands from the ctx.Directories.Command.Returned and marks
	// them as finished or cancelled
	go func() {
		for cmdPath := range ctx.Channels.CommandReturned {
			ctx.OpID = mig.GenID()
			err := terminateCommand(cmdPath, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "terminateCommand() routine started"}

	// Goroutine that updates an action when a command is done
	go func() {
		for cmdPath := range ctx.Channels.CommandDone {
			ctx.OpID = mig.GenID()
			err = updateAction(cmdPath, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
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

	err = watcher.WatchFlags(ctx.Directories.Command.Ready, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Command.Ready)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Command.Ready)}.Debug()

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

	err = watcher.WatchFlags(ctx.Directories.Command.Done, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Command.Done)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Command.Done)}.Debug()

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
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.Ready) {
				ctx.Channels.CommandReady <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.InFlight) {
				ctx.Channels.UpdateCommand <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.Returned) {
				ctx.Channels.CommandReturned <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.Done) {
				ctx.Channels.CommandDone <- ev.Name
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
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: fmt.Sprintf("removing expired action '%s'", action.Name)}
		// delete expired action
		os.Remove(actionPath)
		return
	}
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
			// TODO: opening the keyring in a loop is really ugly. rewind!
			k, err := os.Open(ctx.PGP.PubRing)
			if err != nil {
				panic(err)
			}
			defer k.Close()
			fp, err := pgp.GetFingerprintFromSignature(astr, sig, k)
			if err != nil {
				panic(err)
			}
			iid, err := ctx.DB.InvestigatorByFingerprint(fp)
			if err != nil {
				panic(err)
			}
			err = ctx.DB.InsertSignature(action.ID, iid, sig)
			if err != nil {
				panic(err)
			}
		}
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: "Action written to database"}.Debug()

	// expand the action in one command per agent
	action.CommandIDs, err = prepareCommands(action, ctx)
	if err != nil {
		panic(err)
	}
	// move action to flying state
	action.Counters.Sent = len(action.CommandIDs)
	err = flyAction(ctx, action, actionPath)
	if err != nil {
		panic(err)
	}
	return
}

// prepareCommands transforms an action into one or many commands
// it retrieves a list of target agents from the database, creates one
// command for each target agent, and stores the command into ctx.Directories.Command.Ready.
// An array of command IDs is returned
func prepareCommands(action mig.Action, ctx Context) (cmdIDs []uint64, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("prepareCommands() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, Desc: "leaving prepareCommands()"}.Debug()
	}()
	timeOutPeriod, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	pointInTime := time.Now().Add(-timeOutPeriod)
	agents, err := ctx.DB.ActiveAgentsByTarget(action.Target, pointInTime)
	if err != nil {
		panic(err)
	}
	if len(agents) == 0 {
		err = fmt.Errorf("No agents in database match the target: '%s'", action.Target)
		panic(err)
	}
	// loop over the list of targets and create one command for each agent queue
	var targetedAgents []string
	for _, agent := range agents {
		skip := false
		for _, q := range targetedAgents {
			if q == agent.QueueLoc {
				// if already done this agent, skip it
				skip = true
			}
		}
		if skip {
			continue
		}
		cmdid, err := createCommand(ctx, action, agent)
		if err != nil {
			panic(err)
		}
		cmdIDs = append(cmdIDs, cmdid)
		targetedAgents = append(targetedAgents, agent.QueueLoc)
	}
	return
}

func createCommand(ctx Context, action mig.Action, agent mig.Agent) (cmdid uint64, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createCommand() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, CommandID: cmdid, Desc: "leaving createCommand()"}.Debug()
	}()
	cmdid = mig.GenID()
	cmd := mig.Command{
		Status:    "prepared",
		Action:    action,
		ID:        cmdid,
		StartTime: time.Now().UTC(),
	}
	cmd.Agent.Name = agent.Name
	cmd.Agent.QueueLoc = agent.QueueLoc
	data, err := json.Marshal(cmd)
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%d-%d.json", ctx.Directories.Command.Ready, action.ID, cmdid)
	err = safeWrite(ctx, dest, data)
	if err != nil {
		panic(err)
	}
	err = ctx.DB.InsertCommand(cmd, agent)
	if err != nil {
		panic(err)
	}
	desc := fmt.Sprintf("created command for action '%s' on agent '%s'", action.Name, agent.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, CommandID: cmdid, Desc: desc}
	return
}

// sendCommand is called when a command file is created in ctx.Directories.Command.Ready
// it read the command, sends it to the agent via AMQP, and update the DB
func sendCommand(cmdPath string, ctx Context) (err error) {
	var cmd mig.Command
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendCommand() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, Desc: "leaving sendCommand()"}.Debug()
	}()
	// load and parse the command. If this fail, skip it and continue.
	cmd, err = mig.CmdFromFile(cmdPath)
	if err != nil {
		panic(err)
	}
	cmd.Status = "sent"
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "command received for sending"}.Debug()
	data, err := json.Marshal(cmd)
	if err != nil {
		panic(err)
	}
	// build amqp message for sending
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/plain",
		Body:         []byte(data),
	}
	agtQueue := fmt.Sprintf("mig.agt.%s", cmd.Agent.QueueLoc)
	// send
	err = ctx.MQ.Chan.Publish("mig", agtQueue, true, false, msg)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "command sent to agent queue"}.Debug()
	// write command to InFlight directory
	dest := fmt.Sprintf("%s/%d-%d.json", ctx.Directories.Command.InFlight, cmd.Action.ID, cmd.ID)
	err = safeWrite(ctx, dest, data)
	if err != nil {
		panic(err)
	}
	// remove original command file
	err = os.Remove(cmdPath)
	if err != nil {
		panic(err)
	}
	// store command in database
	err = ctx.DB.UpdateSentCommand(cmd)
	if err != nil {
		panic(err)
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
	dest := fmt.Sprintf("%s/%d", ctx.Directories.Command.Returned, mig.GenID())
	err = safeWrite(ctx, dest, msg.Body)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("Received result from '%s'", msg.RoutingKey)}.Debug()

	return
}

// terminateCommand is called when a command result is dropped into ctx.Directories.Command.Returned
// it stores the result of a command and mark it as completed/failed and then
// send a message to the Action completion routine to update the action status
func terminateCommand(cmdPath string, ctx Context) (err error) {
	var cmd mig.Command
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("terminateCommand() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "leaving terminateCommand()"}.Debug()
	}()

	// load and parse the command. If this fail, skip it and continue.
	cmd, err = mig.CmdFromFile(cmdPath)
	if err != nil {
		panic(err)
	}
	cmd.FinishTime = time.Now().UTC()
	// update command in database
	cmd.Status = "done"
	err = ctx.DB.FinishCommand(cmd)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "command updated in database"}.Debug()
	// write to disk to Command Done directory
	data, err := json.Marshal(cmd)
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%d-%d.json", ctx.Directories.Command.Done, cmd.Action.ID, cmd.ID)
	err = safeWrite(ctx, dest, data)
	if err != nil {
		panic(err)
	}
	// remove command from inflight dir
	inflightPath := fmt.Sprintf("%s/%d-%d.json", ctx.Directories.Command.InFlight, cmd.Action.ID, cmd.ID)
	os.Remove(inflightPath)
	// remove command from Returned dir
	os.Remove(cmdPath)
	return
}

// updateAction is called when a command has finished and the parent action
// must be updated. It retrieves an action from the database, loops over the
// commands, and if all commands have finished, marks the action as finished.
func updateAction(cmdPath string, ctx Context) (err error) {
	var cmd mig.Command
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("updateAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "leaving updateAction()"}.Debug()
	}()

	// load the command file
	cmd, err = mig.CmdFromFile(cmdPath)
	if err != nil {
		panic(err)
	}

	// use the action ID from the command file to get the action from the database
	a, err := ctx.DB.ActionByID(cmd.Action.ID)
	if err != nil {
		panic(err)
	}

	// there is only one entry in the slice, so take the first entry from
	switch cmd.Status {
	case "done":
		a.Counters.Done++
	case "cancelled":
		a.Counters.Cancelled++
	case "failed":
		a.Counters.Failed++
	case "timeout":
		a.Counters.TimeOut++
	default:
		err = fmt.Errorf("unknown command status: %s", cmd.Status)
		panic(err)
	}
	// regardless of returned status, increase completion counter
	a.Counters.Returned++
	a.LastUpdateTime = time.Now().UTC()

	desc := fmt.Sprintf("updating action '%s': completion=%d/%d, done=%d, cancelled=%d, failed=%d, timeout=%d. duration=%s",
		a.Name, a.Counters.Returned, a.Counters.Sent, a.Counters.Done,
		a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut, a.LastUpdateTime.Sub(a.StartTime).String())
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, CommandID: cmd.ID, Desc: desc}

	// Has the action completed?
	if a.Counters.Returned == a.Counters.Sent {
		err = landAction(ctx, a)
		if err != nil {
			panic(err)
		}
		// delete Action from ctx.Directories.Action.InFlight
		actFile := fmt.Sprintf("%d.json", a.ID)
		os.Rename(ctx.Directories.Action.InFlight+"/"+actFile, ctx.Directories.Action.Done+"/"+actFile)
	} else {
		// store updated action in database
		err = ctx.DB.UpdateAction(a)
		if err != nil {
			panic(err)
		}
	}

	// in case the action is related to upgrading agents, do stuff
	if cmd.Status == "done" {
		// this can fail for many reason, do not panic on err return
		err = markUpgradedAgents(cmd, ctx)
		if err != nil {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, CommandID: cmd.ID, Desc: fmt.Sprintf("%v", err)}.Err()
		}
	}

	// remove the command from the spool
	os.Remove(cmdPath)

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
