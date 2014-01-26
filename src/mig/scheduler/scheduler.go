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
	"github.com/howeyc/fsnotify"
	"github.com/streadway/amqp"
	"labix.org/v2/mgo/bson"
	"mig"
	"os"
	"strings"
	"time"
)

// the list of active agents is shared globally
// TODO: wrap this around mutexes for safety
var activeAgentsList []string

// main initializes the mongodb connection, the directory watchers and the
// AMQP ctx.MQ.Chan. It also launches the goroutines.
func main() {
	// command line options
	var config = flag.String("c", "/etc/mig/mig.cfg", "Load configuration from file")
	flag.Parse()

	// The context initialization takes care of parsing the configuration,
	// and creating connections to database, message broker, syslog, ...
	fmt.Fprintf(os.Stderr, "Initializing Scheduler context\n")
	ctx, err := Init(*config)
	if err != nil {
		panic(err)
	}

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
				reason := fmt.Sprintf("%v. %s moved to %s", err, actionPath, dest)
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

	// Init is completed, restart queues of agents that have recent
	// keepalives in the database
	err = pickUpAliveAgents(ctx)
	if err != nil {
		panic(err)
	}

	// start a listening channel to receive keepalives from agents
	keepaliveChan, err := startKeepAliveChannel(ctx)
	if err != nil {
		panic(err)
	}

	// launch the routine that handles registrations
	go func() {
		for msg := range keepaliveChan {
			ctx.OpID = mig.GenID()
			err := getKeepAlives(msg, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "Agent KeepAlive routine started"}

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
	return
}

// processNewAction is called when a new action is available. It pulls
// the action from the directory, parse it, retrieve a list of targets from
// the backend database, and create individual command for each target.
func processNewAction(actionPath string, ctx Context) (err error) {
	var ea mig.ExtendedAction
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("processNewAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: ea.Action.ID, Desc: "leaving processNewAction()"}.Debug()
	}()

	// load the action file
	ea, err = mig.ActionFromFile(actionPath)
	if err != nil {
		panic(err)
	}

	// generate an action id
	ea.Action.ID = mig.GenID()

	desc := fmt.Sprintf("new action received: Name='%s' Target='%s' Order='%s' ScheduledDate='%s' ExpirationDate='%s'",
			ea.Action.Name, ea.Action.Target, ea.Action.Order, ea.Action.ScheduledDate, ea.Action.ExpirationDate)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: ea.Action.ID, Desc: desc}

	// expand the action in one command per agent
	ea.CommandIDs, err = prepareCommands(ea.Action, ctx)
	if err != nil {
		panic(err)
	}

	// move action to Fly-ing state
	err = flyAction(ctx, ea, actionPath)
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

	// query the database for alive agent, that have sent keepalive
	// messages in the last AGTIMEOUT period
	targets := []mig.KeepAlive{}
	period, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	since := time.Now().Add(-period)

	// Mongo query that looks for a list of targets. The query uses a OR to
	// select on the OS type, the queueloc and the name. It also only retrieve
	// agents that have sent an heartbeat in the last `since` period
	iter := ctx.DB.Col.Reg.Find(bson.M{"$or": []bson.M{
							bson.M{"os": action.Target},
							bson.M{"queueloc": action.Target},
							bson.M{"name": action.Target},
						},
					"heartbeatts": bson.M{"$gte": since},
				}).Iter()
	err = iter.All(&targets)
	if err != nil {
		panic(err)
	}
	if len(targets) == 0 {
		panic("0 targets found in database")
	}

	// loop over the list of targets and create a command for each
	for _, target := range targets {
		cmdid, err := createCommand(ctx, action, target)
		if err != nil {
			panic(err)
		}
		cmdIDs = append(cmdIDs, cmdid)
	}
	return
}

func createCommand(ctx Context, action mig.Action, target mig.KeepAlive) (cmdid uint64, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createCommand() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: action.ID, CommandID: cmdid, Desc: "leaving createCommand()"}.Debug()
	}()

	cmdid = mig.GenID()

	cmd := mig.Command{
		AgentName: target.Name,
		AgentQueueLoc: target.QueueLoc,
		Status: "prepared",
		Action:	action,
		ID: cmdid,
		StartTime: time.Now().UTC(),
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		panic(err)
	}

	dest := fmt.Sprintf("%s/%d-%d.json", ctx.Directories.Command.Ready, action.ID, cmdid)
	err = safeWrite(ctx, dest, data)
	if err != nil {
		panic(err)
	}

	desc := fmt.Sprintf("created command for action '%s' on target '%s'", action.Name, target.Name)
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

	agtQueue := fmt.Sprintf("mig.agt.%s", cmd.AgentQueueLoc)

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
	err = ctx.DB.Col.Cmd.Insert(cmd)
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

// terminateCommand is called when a command result is dropped into ctx.Directories.Command.Done
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
	cmd.Status = "completed"

	// remove command from inflight dir
	inflightPath := fmt.Sprintf("%s/%d-%d.json", ctx.Directories.Command.InFlight, cmd.Action.ID, cmd.ID)
	os.Remove(inflightPath)

	// update command in database
	err = ctx.DB.Col.Cmd.Update(bson.M{"id": cmd.ID}, cmd)
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
	var eas []mig.ExtendedAction
	actionCursor := ctx.DB.Col.Action.Find(bson.M{"action.id": cmd.Action.ID}).Iter()
	err = actionCursor.All(&eas)
	if err != nil {
		panic(err)
	}
	if len(eas) > 1 {
		err = fmt.Errorf("found multiple actions with same ID")
		panic(err)
	}

	// there is only one entry in the slice, so take the first entry from
	ea := eas[0]
	switch cmd.Status {
	case "completed":
		ea.CmdCompleted++
	case "cancelled":
		ea.CmdCancelled++
	case "timedout":
		ea.CmdTimedOut++
	default:
		err = fmt.Errorf("unknown command status: %s", cmd.Status)
		panic(err)
	}

	desc := fmt.Sprintf("updating action '%s': completion=%d/%d, cancelled=%d, timeout=%d",
			ea.Action.Name, ea.CmdCompleted, len(ea.CommandIDs), ea.CmdCancelled, ea.CmdTimedOut)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: ea.Action.ID, CommandID: cmd.ID, Desc: desc}

	// Has the action completed?
	finished := ea.CmdCompleted + ea.CmdCancelled + ea.CmdTimedOut
	if finished == len(ea.CommandIDs) {
		// update status and timestamps
		ea.Status = "completed"
		ea.FinishTime = time.Now().UTC()
		duration := ea.FinishTime.Sub(ea.StartTime)

		// log
		desc = fmt.Sprintf("action has completed in %s", duration.String())
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: ea.Action.ID, CommandID: cmd.ID, Desc: desc}

		// TODO write action to Done directory
		// landAction()

		// delete Action from ctx.Directories.Action.InFlight
		actFile := fmt.Sprintf("%d.json", ea.Action.ID)
		os.Rename(ctx.Directories.Action.InFlight+"/"+actFile, ctx.Directories.Action.Done+"/"+actFile)
	}

	// store updated action in database
	ea.LastUpdateTime = time.Now().UTC()
	err = ctx.DB.Col.Action.Update(bson.M{"action.id": ea.Action.ID}, ea)
	if err != nil {
		panic(err)
	}

	return
}
