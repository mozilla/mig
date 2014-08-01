/* Mozilla InvestiGator Agent

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

package main

import (
	"fmt"
	"mig"
	"os"
	"os/exec"
	"time"
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
		ctx.Channels.Log <- mig.Log{Desc: "Parent is init."}.Debug()
		// if this agent has been launched as part of an upgrade, its parent will be
		// detached to init, but no service would be launched, so we launch one
		if upgrading && MUSTINSTALLSERVICE {
			ctx.Channels.Log <- mig.Log{Desc: "Agent is an upgrade. Deploying service."}.Debug()
			time.Sleep(3 * time.Second)
			ctx, err = serviceDeploy(ctx)
			if err != nil {
				panic(err)
			}
			// mig-agent service has been launched, exit this process
			ctx.Channels.Log <- mig.Log{Desc: "Service deployed. Exit."}.Debug()
			os.Exit(0)
		}
		// We are not upgrading, and parent is init. We must decide how to handle
		// respawns based on the type of init system: upstart and systemd will
		// take care of respawning agents automatically. sysvinit won't.
		if ctx.Agent.Env.Init == "upstart" || ctx.Agent.Env.Init == "systemd" {
			ctx.Channels.Log <- mig.Log{Desc: "Running as a service."}.Debug()
			ctx.Agent.Respawn = false
			return
		}
		// init is sysvinit, fork and exit the current process
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
