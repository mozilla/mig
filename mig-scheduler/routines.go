// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"mig.ninja/mig"
)

func startRoutines(ctx Context) {
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
			sendFunction := func(rc map[float64]mig.Command) map[float64]mig.Command {
				var cmdlist []mig.Command
				for _, migcmd := range rc {
					cmdlist = append(cmdlist, migcmd)
				}
				err := sendCommands(cmdlist, ctx)
				if err != nil {
					ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
				}
				return make(map[float64]mig.Command)
			}
			select {
			case cmd := <-ctx.Channels.CommandReady:
				ctr++
				readyCmd[cmd.ID] = cmd
				if ctr >= 1024 {
					readyCmd = sendFunction(readyCmd)
					ctr = 0
				}
			case <-time.After(1 * time.Second):
				if ctr > 0 {
					readyCmd = sendFunction(readyCmd)
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
			returnFunc := func(rc map[uint64]string) map[uint64]string {
				var cmdlist []string
				for _, migcmd := range rc {
					cmdlist = append(cmdlist, migcmd)
				}
				err := returnCommands(cmdlist, ctx)
				if err != nil {
					ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
				}
				return make(map[uint64]string)
			}
			select {
			case cmdFile := <-ctx.Channels.CommandReturned:
				ctr++
				returnedCmd[ctr] = cmdFile
				if ctr >= 1024 {
					returnedCmd = returnFunc(returnedCmd)
					ctr = 0
				}
			case <-time.After(1 * time.Second):
				if ctr > 0 {
					returnedCmd = returnFunc(returnedCmd)
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
			updateFunc := func(dc map[float64]mig.Command) map[float64]mig.Command {
				var cmdlist []mig.Command
				for _, migcmd := range dc {
					cmdlist = append(cmdlist, migcmd)
				}
				err := updateAction(cmdlist, ctx)
				if err != nil {
					ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
				}
				return make(map[float64]mig.Command)
			}
			select {
			case cmd := <-ctx.Channels.CommandDone:
				ctr++
				doneCmd[cmd.ID] = cmd
				if ctr >= 1024 {
					doneCmd = updateFunc(doneCmd)
					ctr = 0
				}
			case <-time.After(1 * time.Second):
				if ctr > 0 {
					doneCmd = updateFunc(doneCmd)
				}
				// reinit
				ctx.OpID = mig.GenID()
				ctr = 0
			}
		}

	}()
	ctx.Channels.Log <- mig.Log{Desc: "updateAction() routine started"}

	// start a listening channel to receive heartbeats from agents
	heartbeatsChan, err := startHeartbeatsListener(ctx)
	if err != nil {
		panic(err)
	}
	go func() {
		for msg := range heartbeatsChan {
			ctx.OpID = mig.GenID()
			err := getHeartbeats(msg, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("heartbeat routine failed with error '%v'", err)}.Err()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "agents heartbeats listener routine started"}

	// start a listening channel to results from agents
	agtResultsChan, err := startResultsListener(ctx)
	if err != nil {
		panic(err)
	}
	go func() {
		for delivery := range agtResultsChan {
			ctx.OpID = mig.GenID()
			// validate the size of the data received, and make sure its first and
			// last bytes are valid json enclosures. if not, discard the message.
			if len(delivery.Body) < 10 || delivery.Body[0] != '{' || delivery.Body[len(delivery.Body)-1] != '}' {
				ctx.Channels.Log <- mig.Log{
					OpID: ctx.OpID,
					Desc: fmt.Sprintf("discarding invalid message received in results channel"),
				}.Err()
				continue
			}
			// write to disk in Returned directory, discard and continue on failure
			dest := fmt.Sprintf("%s/%.0f", ctx.Directories.Command.Returned, ctx.OpID)
			err = safeWrite(ctx, dest, delivery.Body)
			if err != nil {
				ctx.Channels.Log <- mig.Log{
					OpID: ctx.OpID,
					Desc: fmt.Sprintf("failed to write agent results to disk: %v", err),
				}.Err()
				continue
			}
			// publish an event in the command results queue
			err = sendEvent(mig.Ev_Q_Cmd_Res, delivery.Body, ctx)
			if err != nil {
				panic(err)
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "agents results listener routine started"}

	// launch the routine that regularly walks through the local directories
	go func() {
		collectorSleeper, err := time.ParseDuration(ctx.Collector.Freq)
		if err != nil {
			panic(err)
		}
		for {
			ctx.OpID = mig.GenID()
			err := collector(ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("collector routined failed with error '%v'", err)}.Err()
			}
			time.Sleep(collectorSleeper)
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "collector routine started"}

	// launch the routine that periodically runs jobs
	go func() {
		periodicSleeper, err := time.ParseDuration(ctx.Periodic.Freq)
		if err != nil {
			panic(err)
		}
		for {
			ctx.OpID = mig.GenID()
			err := periodic(ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("period routine failed with error '%v'", err)}.Err()
			}
			time.Sleep(periodicSleeper)
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "periodic routine started"}

	// launch the routine that cleans up unused amqp queues
	go func() {
		sleeper, err := time.ParseDuration(ctx.Periodic.QueuesCleanupFreq)
		if err != nil {
			panic(err)
		}
		for {
			ctx.OpID = mig.GenID()
			err = QueuesCleanup(ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("queues cleanup routine failed with error '%v'", err)}.Err()
			}
			time.Sleep(sleeper)
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "queue cleanup routine started"}

	// launch the routine that handles multi agents on same queue
	go func() {
		for queueLoc := range ctx.Channels.DetectDupAgents {
			ctx.OpID = mig.GenID()
			err = killDupAgents(queueLoc, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "killDupAgents() routine started"}

	// launch the routine that heartbeats the relays and terminates if connection is lost
	go func() {
		hostname, _ := os.Hostname()
		hbmsg := fmt.Sprintf("host='%s' pid='%d'", hostname, os.Getpid())
		for {
			ctx.OpID = mig.GenID()
			err = sendEvent(mig.Ev_Q_Sched_Hb, []byte(hbmsg+time.Now().UTC().String()), ctx)
			if err != nil {
				err = fmt.Errorf("relay heartbeating failed with error '%v'", err)
				ctx.Channels.Terminate <- err
			}
			time.Sleep(60 * time.Second)
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "relay heartbeating routine started"}

	// block here until a terminate message is received
	exitReason := <-ctx.Channels.Terminate
	time.Sleep(time.Second)
	fmt.Fprintf(os.Stderr, "Scheduler is shutting down. Reason: %s\n", exitReason)
	Destroy(ctx)
	return
}
