// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"os"
	"os/exec"

	"mig.ninja/mig"
	"mig.ninja/mig/service"
)

// On Windows, processes aren't forked by the init, so when the service is
// started, it will need to fork itself, with the child in foreground mode,
// and exit. We also need to cover service installation prior to the fork.
func daemonize(orig_ctx Context, upgrading bool) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("daemonize() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving daemonize()"}.Debug()
	}()

	if !service.IsInteractive() {
		ctx.Channels.Log <- mig.Log{Desc: "Parent process is PID 1"}.Debug()
		// We are being launched by the Windows SC manager; the Run function in the service
		// package is utilized here to properly handle signalling between the new agent
		// process and the SC manager.
		svc, err := service.NewService("mig-agent", "MIG Agent", "Mozilla InvestiGator Agent")
		if err != nil {
			panic(err)
		}
		ctx.Channels.Log <- mig.Log{Desc: "Running as a service."}.Debug()
		// Create a couple functions here we will use to process start and stop signals
		//
		// XXX the ostop function should be adjusted to cleanly shut the agent down.
		ostart := func() error {
			return nil
		}
		ostop := func() error {
			ctx.Channels.Terminate <- "shutdown requested by windows"
			return nil
		}
		// We don't want the agent to respawn itself if it's being supervised
		ctx.Agent.Respawn = false
		// Handle interactions with SC manager in a go-routine and then proceed on with
		// initialization
		go svc.Run(ostart, ostop)
	} else {
		// install the service
		if MUSTINSTALLSERVICE {
			ctx, err = serviceDeploy(ctx)
			if err != nil {
				panic(err)
			}
			ctx.Channels.Log <- mig.Log{Desc: "Service deployed. Exit."}.Debug()
		} else {
			// we are not in foreground mode, and we don't want a service installation
			// so just fork in foreground mode, and exit the current process
			cmd := exec.Command(ctx.Agent.BinPath, "-f")
			err = cmd.Start()
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to spawn new agent from '%s': '%v'", ctx.Agent.BinPath, err)}.Err()
				return ctx, err
			}
			ctx.Channels.Log <- mig.Log{Desc: "Started new foreground agent. Exit."}.Debug()
		}
		os.Exit(0)
	}
	return
}

func installCron(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("installCron() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving installCron()"}.Debug()
	}()
	panic("mig-agent doesn't have a cronjob for windows.")
	return
}
