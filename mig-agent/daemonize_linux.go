// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"os"
	"os/exec"
)

// daemonize() adds logic to deal with several invocation scenario.
//
// If the parent of the process is 1, we assume the process is already attached
// to an init daemon, of which there is two kind: systemd/upstart, and system-v
// If it's system-v, we need to fork, or system-v will block. Otherwise, we don't
//
// If the parent process is not 1, then we try to install the mig-agent
// service, start the service, and exit. Or we just fork, and exit.
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
		if ctx.Agent.Respawn {
			// install a cron job that acts as a watchdog,
			err = installCron(ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
			}
		}
		// Reset the value of err here; since we continue if the cron installation fails,
		// we don't want subsequent bare returns in this function returning the cron
		// installation error.
		err = nil
		// We are not upgrading, and parent is init. We must decide how to handle
		// respawns based on the type of init system: upstart and systemd will
		// take care of respawning agents automatically. sysvinit won't.
		if ctx.Agent.Env.Init == "upstart" || ctx.Agent.Env.Init == "systemd" {
			ctx.Channels.Log <- mig.Log{Desc: "Running as a service."}.Debug()
			ctx.Agent.Respawn = false
			return
		}
		// this is sysvinit: exec a new agent and exit the current one
		cmd := exec.Command(ctx.Agent.BinPath, "-f")
		err = cmd.Start()
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to spawn new agent from '%s': '%v'", ctx.Agent.BinPath, err)}.Err()
		}
		ctx.Channels.Log <- mig.Log{Desc: "Started new foreground agent. Exit."}.Debug()
		os.Exit(0)
	} else {
		ctx.Channels.Log <- mig.Log{Desc: "Parent is not init."}.Debug()
		// if this agent has been launched by a user, check whether service installation is requested
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
	var job = []byte(`# mig agent monitoring cronjob
PATH="/usr/local/sbin:/usr/sbin:/sbin:/usr/local/bin:/usr/bin:/bin"
SHELL=/bin/bash
MAILTO=""
*/30 * * * * root /sbin/mig-agent -q=pid 2>&1 1>/dev/null || /sbin/mig-agent
`)
	err = ioutil.WriteFile("/etc/cron.d/mig-agent", job, 0644)
	if err != nil {
		panic("Failed to create cron job")
	}
	return
}
