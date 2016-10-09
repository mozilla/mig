// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/modules"
)

func startPersist(ctx *Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startPersist() -> %v", e)
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "initializing any persistent modules"}.Debug()

	for k, v := range modules.Available {
		if _, ok := v.NewRun().(modules.PersistRunner); ok {
			err = startPersistModule(ctx, k)
			if err != nil {
				panic(err)
			}
		}
	}
	return
}

func startPersistModule(ctx *Context, name string) (err error) {
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("starting persistent module %v", name)}.Info()
	go managePersistModule(ctx, name)
	return
}

func managePersistModule(ctx *Context, name string) {
	var (
		cmd        *exec.Cmd
		isRunning  bool
		pipeout    io.WriteCloser
		pipein     io.ReadCloser
		err        error
		failDelay  bool
		killModule bool
		inChan     chan modules.Message
		lastPing   time.Time
	)

	logfunc := func(f string, a ...interface{}) {
		buf := fmt.Sprintf(f, a...)
		buf = fmt.Sprintf("[%v] %v", name, buf)
		ctx.Channels.Log <- mig.Log{Desc: buf}.Info()
	}

	pingtick := time.Tick(time.Second * 10)

	for {
		if failDelay {
			time.Sleep(time.Second * 10)
			failDelay = false
		}

		if !isRunning {
			logfunc("starting module")
			lastPing = time.Now()
			cmd = exec.Command(ctx.Agent.BinPath, "-P", strings.ToLower(name))
			pipeout, err = cmd.StdinPipe()
			if err != nil {
				logfunc("error creating stdin pipe, %v", err)
				failDelay = true
				continue
			}
			pipein, err = cmd.StdoutPipe()
			if err != nil {
				logfunc("error creating stdout pipe, %v", err)
				failDelay = true
				continue
			}
			err = cmd.Start()
			if err != nil {
				logfunc("error starting module, %v", err)
				failDelay = true
				continue
			}
			inChan = make(chan modules.Message, 0)

			go func() {
				for {
					msg, err := modules.ReadInput(pipein)
					if err != nil {
						logfunc("%v", err)
						close(inChan)
						break
					}
					inChan <- msg
				}
			}()

			isRunning = true
		}
		select {
		case msg, ok := <-inChan:
			if !ok {
				err = cmd.Wait()
				logfunc("module is down, %v", err)
				isRunning = false
				failDelay = true
				break
			}
			switch msg.Class {
			case modules.MsgClassPing:
				lastPing = time.Now()
			case modules.MsgClassLog:
				var lp modules.LogParams
				buf, err := json.Marshal(msg.Parameters)
				if err != nil {
					logfunc("%v", err)
					break
				}
				err = json.Unmarshal(buf, &lp)
				if err != nil {
					logfunc("%v", err)
					break
				}
				logfunc("(module log) %v", lp.Message)
			default:
				logfunc("unknown message class")
				killModule = true
				break
			}
		case _ = <-pingtick:
			// If we haven't received a reply in the past 3 cycles we will
			// kill the module
			if time.Now().Sub(lastPing) >= time.Duration(30*time.Second) {
				logfunc("no ping response from module, killing")
				killModule = true
				break
			}

			pm, err := modules.MakeMessage("ping", nil, false)
			if err != nil {
				// Failure here should not occur but does not
				// mean the module is down
				logfunc("failed to create ping, %v", err)
				break
			}
			err = modules.WriteOutput(pm, pipeout)
			if err != nil {
				logfunc("ping failed, %v", err)
				isRunning = false
				failDelay = true
				break
			}
		}

		if killModule {
			logfunc("killing module")
			err = cmd.Process.Kill()
			if err != nil {
				logfunc("failed to kill module, %v", err)
				// If this happens we are in a bad state, return from here
				// as we cannot recover
				return
			}
			_ = cmd.Wait()
			isRunning = false
			failDelay = true
			killModule = false
		}
	}
}
