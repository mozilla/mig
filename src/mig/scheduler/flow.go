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
Portions created by the Initial Developer are Copyright (C) 2014
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
	"mig"
	"os"
	"time"
)

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
	dest := fmt.Sprintf("%s/%d.json", ctx.Directories.Action.InFlight, a.ID)
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
	err = ctx.DB.InsertOrUpdateAction(a)
	if err != nil {
		panic(err)
	}
	desc := fmt.Sprintf("flyAction(): Action '%s' is in flight", a.Name)
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
	dest := fmt.Sprintf("%s/%d.json", ctx.Directories.Action.Done, a.ID)
	err = safeWrite(ctx, dest, jsonA)
	if err != nil {
		panic(err)
	}
	// remove the action from its origin
	origin := fmt.Sprintf("%s/%d.json", ctx.Directories.Action.InFlight, a.ID)
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
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("safeWrite() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving safeWrite()"}.Debug()
	}()
	// write the file temp dir
	tmp := fmt.Sprintf("%s/%d", ctx.Directories.Tmp, mig.GenID())
	err = ioutil.WriteFile(tmp, data, 0640)
	if err != nil {
		panic(err)
	}
	// move to destination
	err = os.Rename(tmp, destination)
	if err != nil {
		panic(err)
	}
	return
}
