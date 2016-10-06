// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/modules"
)

// runModule is a generic module launcher that takes an operation and calls
// the correct MIG module. Depending on the type of module being called a couple
// different things can happen here.
//
// If the module is not a persistent module, runModule will look after executing
// the module directly by calling executeModule, which will then execute mig-agent
// with the correct command line flags. Finally it will then gather the result and
// send it to result channel for the operation.
//
// If the module is persistent, runModule will interface with the persistent
// module management system to communicate with the module, as it will already
// be running and we do not need to execute it.
func runModule(ctx *Context, op moduleOp) (err error) {
	var result moduleResult
	result.id = op.id
	result.position = op.position

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("runModule() -> %v", e)
			result.err = err
			result.status = mig.StatusFailed
		}
		// upon exit, remove the op from the running Ops
		delete(runningOps, op.id)
		// whatever happens, always send the results
		op.resultChan <- result
		ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "leaving runModule()"}.Debug()
	}()

	ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: fmt.Sprintf("calling module %q", op.mode)}.Debug()
	mod, ok := modules.Available[op.mode]
	if !ok {
		err = fmt.Errorf("module %q is not available", op.mode)
		panic(err)
	}
	if mod.NewRun().IsPersistent() {
	} else {
		// Execute the module once
		result, err = executeModule(ctx, op)
		if err != nil {
			panic(err)
		}
	}
	return
}

func executeModule(ctx *Context, op moduleOp) (result moduleResult, err error) {
	result.id = op.id
	result.position = op.position
	defer func() {
		if e := recover(); e != nil {
			// if running the module failed, store the error in the module result
			// and sets the status to failed before passing the results along
			err = fmt.Errorf("runModule() -> %v", e)
			result.err = err
			result.status = mig.StatusFailed
		}
		ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "leaving executeModule()"}.Debug()
	}()

	// waiter is a channel that receives a message when the timeout expires
	waiter := make(chan error, 1)
	var out bytes.Buffer

	// calculate the max exec time by taking the smallest duration between the expiration date
	// sent with the command, and the default MODULETIMEOUT value from the agent configuration
	execTimeOut := MODULETIMEOUT
	if op.expireafter.Before(time.Now().Add(MODULETIMEOUT)) {
		execTimeOut = op.expireafter.Sub(time.Now())
	}

	// Build parameters message
	modParams, err := modules.MakeMessage(modules.MsgClassParameters, op.params, op.isCompressed)
	if err != nil {
		panic(err)
	}

	// build the command line and execute
	cmd := exec.Command(ctx.Agent.BinPath, "-m", strings.ToLower(op.mode))
	stdinpipe, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	// Spawn a goroutine to write the parameter data to stdin of the module
	// if required. Doing this in a goroutine ensures the timeout logic
	// later in this function will fire if for some reason the module does
	// not drain the pipe, and the agent ends up blocking on Write().
	go func() {
		left := len(modParams)
		for left > 0 {
			nb, err := stdinpipe.Write(modParams)
			if err != nil {
				stdinpipe.Close()
				return
			}
			left -= nb
			modParams = modParams[nb:]
		}
		stdinpipe.Close()
	}()

	// launch the waiter in a separate goroutine
	go func() {
		waiter <- cmd.Wait()
	}()

	select {
	// Timeout case: command has reached timeout, kill it
	case <-time.After(execTimeOut):
		ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command timed out. Killing it."}.Err()

		// update the command status and send the response back
		result.status = mig.StatusTimeout

		// kill the command
		err := cmd.Process.Kill()
		if err != nil {
			panic(err)
		}
		<-waiter // allow goroutine to exit

	// Normal exit case: command has run successfully
	case err := <-waiter:
		if err != nil {
			ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command failed."}.Err()
			panic(err)

		} else {
			ctx.Channels.Log <- mig.Log{OpID: op.id, Desc: "command done."}
			err = json.Unmarshal(out.Bytes(), &result.output)
			if err != nil {
				panic(err)
			}
			// mark command status as successfully completed
			result.status = mig.StatusSuccess
		}
	}
	return
}
