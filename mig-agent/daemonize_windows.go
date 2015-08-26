// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"mig.ninja/mig"
	"os"
	"os/exec"
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

	if os.Getppid() == 1 {
		ctx.Channels.Log <- mig.Log{Desc: "Parent process is PID 1"}.Debug()
		// if this agent has been launched as part of an upgrade, its parent will be
		// detached to init, but no service would be launched, so we launch one
		if upgrading && MUSTINSTALLSERVICE {
			ctx.Channels.Log <- mig.Log{Desc: "Agent is upgrading. Deploying service."}.Debug()
			ctx, err = serviceDeploy(ctx)
			if err != nil {
				panic(err)
			}
			// mig-agent service has been launched, exit this process
			ctx.Channels.Log <- mig.Log{Desc: "Service deployed. Exit."}.Debug()
			os.Exit(0)
		}
		// fork a new agent and detach. new agent will reattach to init
		cmd := exec.Command(ctx.Agent.BinPath, "-f")
		err = cmd.Start()
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to spawn new agent from '%s': '%v'", ctx.Agent.BinPath, err)}.Err()
		}
		ctx.Channels.Log <- mig.Log{Desc: "Started new foreground agent. Exit."}.Debug()
		os.Exit(0)
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
