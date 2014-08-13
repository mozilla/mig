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

	err = loadNewActions(ctx)
	if err != nil {
		panic(err)
	}
	err = loadCommandsReady(ctx)
	if err != nil {
		panic(err)
	}
	err = loadReturnedCommands(ctx)
	if err != nil {
		panic(err)
	}
	err = loadCommandsDone(ctx)
	if err != nil {
		panic(err)
	}

	err = evaluateInFlightCommands(ctx)
	if err != nil {
		panic(err)
	}

	err = cleanDir(ctx, ctx.Directories.Action.Done)
	if err != nil {
		panic(err)
	}
	err = cleanDir(ctx, ctx.Directories.Command.Done)
	if err != nil {
		panic(err)
	}
	err = cleanDir(ctx, ctx.Directories.Action.Invalid)
	if err != nil {
		panic(err)
	}

	return
}

// loadNewActions walks through the new actions directories and load the actions
// that are passed their scheduled date. It also delete expired actions.
func loadNewActions(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadNewActions() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving loadNewActions()"}.Debug()
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
		a, err := mig.ActionFromFile(filename)
		if err != nil {
			panic(err)
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

// loadCommandsReady walks through the commands ready directory and load the
// commands that are passed their scheduled date. It also delete expired commands.
func loadCommandsReady(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadCommandsReady() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving loadCommandsReady()"}.Debug()
	}()
	dir, err := os.Open(ctx.Directories.Command.Ready)
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
		filename := ctx.Directories.Command.Ready + "/" + DirEntry.Name()
		cmd, err := mig.CmdFromFile(filename)
		if err != nil {
			panic(err)
		}
		if time.Now().After(cmd.Action.ExpireAfter) {
			// delete expired
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("removing expired command '%s'", cmd.Action.Name)}
			os.Remove(filename)
		} else if time.Now().After(cmd.Action.ValidFrom) {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("launching command '%s'", cmd.Action.Name)}
			ctx.Channels.CommandReady <- filename
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
		cmd, err := mig.CmdFromFile(filename)
		if err != nil {
			panic(err)
		}
		// queue it
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("loading returned command '%s'", cmd.Action.Name)}
		ctx.Channels.CommandReturned <- filename
	}
	dir.Close()
	return
}

// loadCommandsDone walks through the returned commands directory and loads
// the commands into the scheduler
func loadCommandsDone(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadCommandsDone() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving loadCommandsDone()"}.Debug()
	}()
	dir, err := os.Open(ctx.Directories.Command.Done)
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
		filename := ctx.Directories.Command.Done + "/" + DirEntry.Name()
		cmd, err := mig.CmdFromFile(filename)
		if err != nil {
			panic(err)
		}
		// queue it
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: fmt.Sprintf("loading returned command '%s'", cmd.Action.Name)}
		ctx.Channels.CommandDone <- filename
	}
	dir.Close()
	return
}

// evaluateInFlightCommand loads commands in the inflight directory
// and terminate the expired ones
func evaluateInFlightCommands(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("evaluateInFlightCommands() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving evaluateInFlightCommands()"}.Debug()
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
		cmd, err := mig.CmdFromFile(filename)
		if err != nil {
			panic(err)
		}

		if time.Now().After(cmd.Action.ExpireAfter) {
			desc := fmt.Sprintf("expiring action '%s' on agent '%s'", cmd.Action.Name, cmd.Agent.Name)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, CommandID: cmd.ID, ActionID: cmd.Action.ID, Desc: desc}
			// expired command must be terminated
			cmd.Status = "cancelled"
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
