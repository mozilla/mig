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

	//err = evaluateInFlightAction(ctx)
	//if err != nil {
	//	panic(err)
	//}
	//err = evaluateInFlightCommand(ctx)
	//if err != nil {
	//	panic(err)
	//}

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
