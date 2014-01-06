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
)

// Fly moves an action file to the InFlight directory and
// write it to database
func flyAction(ctx Context, ea mig.ExtendedAction, origin string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("flyAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{ActionID: ea.Action.ID, Desc: "leaving flyAction()"}.Debug()
	}()

	// move action to inflight dir
	jsonEA, err := json.Marshal(ea)
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%d.json", ctx.Directories.Action.InFlight, ea.Action.ID)
	err = ioutil.WriteFile(dest, jsonEA, 0640)
	if err != nil {
		panic(err)
	}

	// remove the action from its origin
	os.Remove(origin)
	if err != nil {
		panic(err)
	}

	// The extended action is stored in database
	err = ctx.DB.Col.Action.Insert(ea)
	if err != nil {
		panic(err)
	}

	desc := fmt.Sprintf("Fly(): Action '%s' is in flight", ea.Action.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: ea.Action.ID, Desc: desc}

	return
}

// safeWrite performs a two steps write:
// 1) a temp file is written
// 2) the temp file is moved into the target folder
// this prevents the dir watcher from waking up before the file is fully written
func safeWrite(ctx Context, destination string, data []byte)(err error){
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
