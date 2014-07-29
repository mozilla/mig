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

	"bitbucket.org/kardianos/service"
)

// daemonize() adds logic to deal with several invocation scenario.
//
// If the parent of the process is 1, we assume the process is already attached
// to an init daemon, of which there is two kind: systemd/upstart, and system-v
// If it's system-v, we need to fork, or system-v will block. Otherwise, we don't
//
// If the parent process is not 1, then we try to install the mig-agent
// service, start the service, and exit. Or we just fork, and exit.
func daemonize(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("daemonize() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving daemonize()"}.Debug()
	}()
	if os.Getppid() == 1 {
		if ctx.Agent.Env.Init == "upstart" || ctx.Agent.Env.Init == "systemd" {
			// if not controlled by system-v, we tell the agent
			// to not respawn itself. upstart/systemd will do it
			ctx.Agent.Respawn = false
			return
		}
		// that's system-v, fork a new agent in foreground mode
		cmd := exec.Command(ctx.Agent.BinPath, "-f")
		err = cmd.Start()
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to spawn new agent from '%s': '%v'", ctx.Agent.BinPath, err)}.Err()
		}
		os.Exit(0)
	} else {
		// install the service, start it, and exit
		if MUSTINSTALLSERVICE {
			svc, err := service.NewService("mig-agent", "MIG Agent", "Mozilla InvestiGator Agent")
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Service initialization failed: '%v'", err)}.Err()
				return ctx, err
			}
			// if already running, stop it
			// _ = svc.Stop()
			err = svc.Remove()
			if err != nil {
				// fail but continue, the service may not exist yet
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Service removal failed: '%v'", err)}.Err()
			}
			err = svc.Install()
			if err != nil {
				// if installation fails, do not continue
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Service installation failed: '%v'", err)}.Err()
				return ctx, err
			}
			err = svc.Start()
			if err != nil {
				// if starting fails, do not continue either
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Service startup failed: '%v'", err)}.Err()
				return ctx, err
			}
		} else {
			// we are not in foreground mode, and we don't want a service installation
			// so just fork in foreground mode, and exit the current process
			cmd := exec.Command(ctx.Agent.BinPath, "-f")
			err = cmd.Start()
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to spawn new agent from '%s': '%v'", ctx.Agent.BinPath, err)}.Err()
				return ctx, err
			}
		}
		os.Exit(0)
	}
	return
}
