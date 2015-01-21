// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"mig"
	"os"
	"time"
)

// spoolInspection walks through the local directories and performs the following
// 1. load actions and commandsthat are sitting in the directories and waiting for processing
// 2. evaluate actions and commands that are inflight (todo)
// 3. remove finished and invalid actions and commands once the DeleteAfter period is passed
func spoolInspection(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("spoolInspection() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving spoolInspection()"}.Debug()
	}()
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "initiating spool inspection"}.Debug()

	err = loadNewActionsFromDB(ctx)
	if err != nil {
		panic(err)
	}
	err = loadNewActionsFromSpool(ctx)
	if err != nil {
		panic(err)
	}
	err = loadReturnedCommands(ctx)
	if err != nil {
		panic(err)
	}
	err = expireCommands(ctx)
	if err != nil {
		panic(err)
	}
	err = cleanDir(ctx, ctx.Directories.Action.Done)
	if err != nil {
		panic(err)
	}
	err = cleanDir(ctx, ctx.Directories.Action.Invalid)
	if err != nil {
		panic(err)
	}
	err = markOfflineAgents(ctx)
	if err != nil {
		panic(err)
	}
	err = markIdleAgents(ctx)
	if err != nil {
		panic(err)
	}

	return
}

// loadNewActionsFromDB retrieves action that are ready to run from the database
// and writes them into the spool for scheduling
func loadNewActionsFromDB(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadNewActionsFromDB() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving loadNewActionsFromDB()"}.Debug()
	}()
	actions, err := ctx.DB.SetupRunnableActions()
	if err != nil {
		panic(err)
	}
	for _, a := range actions {
		err = setupAction(ctx, a)
		if err != nil {
			panic(err)
		}
	}
	return
}

// loadNewActionsFromSpool walks through the new actions spool and loads the actions
// that are passed their scheduled date. It also deletes expired actions.
func loadNewActionsFromSpool(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadNewActionsFromSpool() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving loadNewActionsFromSpool()"}.Debug()
	}()
	dir, err := os.Open(ctx.Directories.Action.New)
	dirContent, err := dir.Readdir(-1)
	if err != nil {
		panic(err)
	}
	// loop over the content of the directory
	for _, DirEntry := range dirContent {
		if !DirEntry.Mode().IsRegular() {
			// ignore non file
			continue
		}
		filename := ctx.Directories.Action.New + "/" + DirEntry.Name()
		err := waitForFileOrDelete(filename, 3)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("error while reading '%s': %v", filename, err)}.Err()
			continue
		}
		a, err := mig.ActionFromFile(filename)
		if err != nil {
			// failing to load this file, log and skip it
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("failed to load new action file %s", filename)}.Err()
			continue
		}
		if time.Now().After(a.ExpireAfter) {
			// delete expired
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: fmt.Sprintf("removing expired action '%s'", a.Name)}
			os.Remove(filename)
		} else if time.Now().After(a.ValidFrom) {
			// queue it
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: fmt.Sprintf("scheduling action '%s'", a.Name)}
			ctx.Channels.NewAction <- filename
		}
	}
	dir.Close()
	return
}

// loadReturnedCommands walks through the returned commands directory and loads
// the commands into the scheduler
func loadReturnedCommands(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadReturnedCommands() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving loadReturnedCommands()"}.Debug()
	}()
	dir, err := os.Open(ctx.Directories.Command.Returned)
	dirContent, err := dir.Readdir(-1)
	if err != nil {
		panic(err)
	}
	// loop over the content of the directory
	for _, DirEntry := range dirContent {
		if !DirEntry.Mode().IsRegular() {
			// ignore non file
			continue
		}
		filename := ctx.Directories.Command.Returned + "/" + DirEntry.Name()
		err := waitForFileOrDelete(filename, 3)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("error while reading '%s': %v", filename, err)}.Err()
			continue
		}
		cmd, err := mig.CmdFromFile(filename)
		if err != nil {
			// failing to load this file, log and skip it
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("failed to load returned command file %s", filename)}.Err()
			continue
		}
		// queue it
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("loading returned command '%s'", cmd.Action.Name)}
		ctx.Channels.CommandReturned <- filename
	}
	dir.Close()
	return
}

// expireCommands loads commands in the inflight directory
// and terminate the expired ones
func expireCommands(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("expireCommands() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving expireCommands()"}.Debug()
	}()
	dir, err := os.Open(ctx.Directories.Command.InFlight)
	dirContent, err := dir.Readdir(-1)
	if err != nil {
		panic(err)
	}
	// loop over the content of the directory
	for _, DirEntry := range dirContent {
		if !DirEntry.Mode().IsRegular() {
			// ignore non file
			continue
		}
		filename := ctx.Directories.Command.InFlight + "/" + DirEntry.Name()
		err := waitForFileOrDelete(filename, 3)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("error while reading '%s': %v", filename, err)}.Err()
			continue
		}
		cmd, err := mig.CmdFromFile(filename)
		if err != nil {
			// failing to load this file, log and skip it
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("failed to inflight command file %s", filename)}.Err()
			continue
		}

		if time.Now().After(cmd.Action.ExpireAfter) {
			desc := fmt.Sprintf("expiring command '%s' on agent '%s'", cmd.Action.Name, cmd.Agent.Name)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: desc}
			cmd.Status = "expired"
			cmd.FinishTime = time.Now().UTC()
			// write it into the returned command directory
			data, err := json.Marshal(cmd)
			if err != nil {
				panic(err)
			}
			dest := fmt.Sprintf("%s/%.0f-%.0f.json", ctx.Directories.Command.Returned, cmd.Action.ID, cmd.ID)
			err = safeWrite(ctx, dest, data)
			if err != nil {
				panic(err)
			}
			//ctx.Directories.Command.Returned
			os.Remove(filename)
		}
	}
	dir.Close()
	return
}

// cleanDir walks through a directory and delete the files that
// are older than the configured DeleteAfter parameter
func cleanDir(ctx Context, targetDir string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("cleanDir() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving cleanDir()"}.Debug()
	}()
	deletionPoint, err := time.ParseDuration(ctx.Collector.DeleteAfter)
	dir, err := os.Open(targetDir)
	dirContent, err := dir.Readdir(-1)
	if err != nil {
		panic(err)
	}
	// loop over the content of the directory
	for _, DirEntry := range dirContent {
		if !DirEntry.Mode().IsRegular() {
			// ignore non file
			continue
		}
		// if the DeleteAfter value is after the time of last modification,
		// the file is due for deletion
		if time.Now().Add(-deletionPoint).After(DirEntry.ModTime()) {
			filename := targetDir + "/" + DirEntry.Name()
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("removing '%s'", filename)}
			os.Remove(filename)
		}
	}
	dir.Close()
	return
}

// markOfflineAgents updates the status of idle agents that passed the agent timeout to "offline"
func markOfflineAgents(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("markOfflineAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving markOfflineAgents()"}.Debug()
	}()
	timeOutPeriod, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	pointInTime := time.Now().Add(-timeOutPeriod)
	err = ctx.DB.MarkOfflineAgents(pointInTime)
	if err != nil {
		panic(err)
	}
	return
}

// markIdleAgents updates the status of agents that stopped sending heartbeats
func markIdleAgents(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("markIdleAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving markIdleAgents()"}.Debug()
	}()
	hbFreq, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
	if err != nil {
		panic(err)
	}
	pointInTime := time.Now().Add(-hbFreq * 5)
	err = ctx.DB.MarkIdleAgents(pointInTime)
	if err != nil {
		panic(err)
	}
	return
}
