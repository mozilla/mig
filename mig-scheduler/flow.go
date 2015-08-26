// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// The functions in this file control the flow of actions and commands through
// the scheduler.
//
//             {~~~~~~~~~~~ MIG SCHEDULER DIRECTORIES ~~~~~~~~~~~~~~~}
//                                                         +-------+    +-------------+
//                                                         |Action |    |             |
//                                                      +->|  Done |    |-------------|
//                                                      |  |       |    |             |
//                                      +-------+       |  +-------+    |             |
// New Action                           |Action |       |               |             |
//     +                            +-->| In    |+-landAction()         |             |
//     |                            |   | Flight|       |               |             |
//     |           +-------+        |   +-------+       +-------------->|             |
//     +---------->|Action |        |                                   |             |
//                 |  New  +-flyAction()                                | Database    |
//                 |       |        |                                   |             |
//                 +-+-+++++        +---------------------------------->|             |
//       Invalidate()| ||||                                             |             |
//                   | |||| create                                      |             |
// +-------+         | |||| one or many                                 |             |
// |Action |<--------+ |||| commands                                    |             |
// |Invalid|           ||||                                             |             |
// |       |           ||||                                             |             |
// +-------+           vvvv                                             |             |
//                 +-------+        +---------------------------------->|             |
//                 |Command|        |                                   |             |
//                 |   New +-flyCommand()                               |             |
//                 |       |        |                                   |             |
//                 +-------+        |   +-------+       +-------------->|             |
//                                  |   |Command|       |               |             |
//                                  +-->|   New +-landCommand()         |             |
//                                      |       |       |               |             |
//                                      +-------+       |  +-------+    |             |
//                                                      |  |Command|    |             |
//                                                      +->|   New |    |             |
//                                                         |       |    |             |
//                                                         +-------+    |             |
//                                                                      +-------------+

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"os"
	"time"
)

// setupAction takes an initialized action and writes it in the new action spool
func setupAction(ctx Context, a mig.Action) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("setupAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "leaving setupAction()"}.Debug()
	}()
	// move action to inflight dir
	jsonA, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%.0f.json", ctx.Directories.Action.New, a.ID)
	err = safeWrite(ctx, dest, jsonA)
	if err != nil {
		panic(err)
	}
	desc := fmt.Sprintf("setupAction(): Action '%s' has been setup", a.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: desc}.Debug()
	return
}

// flyAction moves an action file to the InFlight directory and
// write it to database
func flyAction(ctx Context, a mig.Action, origin string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("flyAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "leaving flyAction()"}.Debug()
	}()
	// move action to inflight dir
	jsonA, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%.0f.json", ctx.Directories.Action.InFlight, a.ID)
	err = safeWrite(ctx, dest, jsonA)
	if err != nil {
		panic(err)
	}
	// remove the action from its origin
	os.Remove(origin)
	if err != nil {
		panic(err)
	}
	a.Status = "inflight"
	err = ctx.DB.UpdateActionStatus(a)
	if err != nil {
		panic(err)
	}
	desc := fmt.Sprintf("flyAction(): Action '%s' is in flight", a.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: desc}.Debug()
	return
}

// invalidAction marks actions that have failed to run
func invalidAction(ctx Context, a mig.Action, origin string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("invalidAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "leaving invalidAction()"}.Debug()
	}()
	// move action to invalid dir
	jsonA, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%.0f.json", ctx.Directories.Action.Invalid, a.ID)
	err = safeWrite(ctx, dest, jsonA)
	if err != nil {
		panic(err)
	}
	// remove the action from its origin
	os.Remove(origin)
	if err != nil {
		panic(err)
	}
	a.Status = "invalid"
	a.LastUpdateTime = time.Now().UTC()
	a.FinishTime = time.Now().UTC()
	a.Counters.Sent = 0
	err = ctx.DB.UpdateAction(a)
	if err != nil {
		panic(err)
	}
	desc := fmt.Sprintf("invalidAction(): Action '%s' has been marked as invalid.", a.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: desc}.Debug()
	return
}

// landAction moves an action file to the Done directory and
// updates it in database
func landAction(ctx Context, a mig.Action) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("landAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "leaving landAction()"}.Debug()
	}()
	// update status and timestamps
	a.Status = "done"
	a.FinishTime = time.Now().UTC()
	duration := a.FinishTime.Sub(a.StartTime)
	// log
	desc := fmt.Sprintf("action has completed in %s", duration.String())
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: desc}
	// move action to done dir
	jsonA, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%.0f.json", ctx.Directories.Action.Done, a.ID)
	err = safeWrite(ctx, dest, jsonA)
	if err != nil {
		panic(err)
	}
	// remove the action from its origin
	origin := fmt.Sprintf("%s/%.0f.json", ctx.Directories.Action.InFlight, a.ID)
	os.Remove(origin)
	if err != nil {
		panic(err)
	}
	err = ctx.DB.FinishAction(a)
	if err != nil {
		panic(err)
	}
	desc = fmt.Sprintf("landAction(): Action '%s' has landed", a.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: a.ID, Desc: desc}.Debug()
	return
}

// safeWrite performs a two steps write:
// 1) a temp file is written
// 2) the temp file is moved into the target folder
// this prevents the dir watcher from waking up before the file is fully written
func safeWrite(ctx Context, destination string, data []byte) (err error) {
	if len(data) == 0 {
		return fmt.Errorf("data slice is empty. file not written")
	}
	// write the file temp dir
	tmp := fmt.Sprintf("%s/%.0f", ctx.Directories.Tmp, mig.GenID())
	err = ioutil.WriteFile(tmp, data, 0640)
	if err != nil {
		return fmt.Errorf("safeWrite: %v", err)
	}
	// move to destination
	err = os.Rename(tmp, destination)
	if err != nil {
		return fmt.Errorf("safeWrite: %v", err)
	}
	return
}

// waitForFileOrDelete checks that a file has a non-zero size several time in a loop
// waiting 200 milliseconds every time. If after several attempts, the file still has
// size zero, it is removed and an error is returned.
func waitForFileOrDelete(filepath string, tries int) error {
	for i := 0; i < tries; i++ {
		fi, err := os.Stat(filepath)
		if err != nil {
			return fmt.Errorf("stat failed. error: %v", err)
		}
		if fi.Size() != 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	os.Remove(filepath)
	return fmt.Errorf("file reached timeout with a zero size and has been deleted")
}
