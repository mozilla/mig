// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/streadway/amqp"
	"mig.ninja/mig"
	"mig.ninja/mig/modules"
	"mig.ninja/mig/pgp"
)

func main() {
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus * 2)

	// command line options
	var config = flag.String("c", "/etc/mig/scheduler.cfg", "Load configuration from file")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	// The context initialization takes care of parsing the configuration,
	// and creating connections to database, message broker, syslog, ...
	fmt.Fprintf(os.Stderr, "Initializing Scheduler context...")
	ctx, err := Init(*config)
	if err != nil {
		fmt.Printf("\nFATAL: %v\n", err)
		os.Exit(9)
	}
	fmt.Fprintf(os.Stderr, "OK\n")

	// startroutines is where all the work is done.
	// it blocks until a Terminate message is received
	startRoutines(ctx)
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
	emptyResults := make([]modules.Result, len(action.Operations))
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

func createCommand(ctx Context, action mig.Action, agent mig.Agent, emptyResults []modules.Result) (err error) {
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
	return
}

// sendCommand is called when a command file is created in ctx.Directories.Command.Ready
// it read the command, sends it to the agent via AMQP, and update the DB
func sendCommands(cmds []mig.Command, ctx Context) (err error) {
	aid := cmds[0].Action.ID
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendCommands() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: aid, Desc: "leaving sendCommands()"}.Debug()
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
		// write command to InFlight directory
		dest := fmt.Sprintf("%s/%.0f-%.0f.json", ctx.Directories.Command.InFlight, cmd.Action.ID, cmd.ID)
		err = safeWrite(ctx, dest, data)
		if err != nil {
			panic(err)
		}
		// send amqp message with an expiration timer
		expire := cmd.Action.ExpireAfter.Sub(cmd.Action.ValidFrom)
		msg := amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			ContentType:  "text/plain",
			Expiration:   fmt.Sprintf("%d", int64(expire/time.Millisecond)),
			Body:         []byte(data),
		}
		agtQueue := fmt.Sprintf("mig.agt.%s", cmd.Agent.QueueLoc)
		go func() {
			err = ctx.MQ.Chan.Publish(mig.Mq_Ex_ToAgents, agtQueue, true, false, msg)
			if err != nil {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "publishing failed to queue" + agtQueue}.Err()
			} else {
				desc := fmt.Sprintf("published to queue %s", agtQueue)
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: desc}
			}
		}()
	}
	return
}

// returnCommands is called when commands have returned
// it stores the result of a command and mark it as completed/failed and then
// send a message to the Action completion routine to update the action status
func returnCommands(cmdFiles []string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("returnCommands() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving returnCommands()"}.Debug()
	}()
	for _, cmdFile := range cmdFiles {
		// load and parse the command. If this fail, skip it and continue.
		cmd, err := mig.CmdFromFile(cmdFile)
		if err != nil {
			// if CmdFromFile fails, rename the cmdFile and skip.
			desc := fmt.Sprintf("Command in %s failed, renaming to %s.fail", cmdFile, cmdFile)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
			os.Rename(cmdFile, cmdFile+".fail")
			continue
		}
		cmd.FinishTime = time.Now().UTC()
		// update command in database
		go func() {
			err = ctx.DB.FinishCommand(cmd)
			if err != nil {
				desc := fmt.Sprintf("command results insertion in database failed with error: %v", err)
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: desc}.Err()
			} else {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "command updated in database"}.Debug()
			}

			// pass the command over to the Command Done channel
			ctx.Channels.CommandDone <- cmd
		}()
		// remove command from inflight dir
		inflightPath := fmt.Sprintf("%s/%.0f-%.0f.json", ctx.Directories.Command.InFlight, cmd.Action.ID, cmd.ID)
		os.Remove(inflightPath)

		// remove command from Returned dir
		os.Remove(cmdFile)
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
