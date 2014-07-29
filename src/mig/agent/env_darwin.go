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
	"bytes"
	"fmt"
	"mig"
	"os/exec"
	"runtime"
	"strings"
)

func findHostname(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findHostname() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving findHostname()"}.Debug()
	}()

	// get the hostname
	out, err := exec.Command("hostname", "-f").Output()
	if err != nil {
		panic(err)
	}
	// remove trailing newline
	ctx.Agent.Hostname = fmt.Sprintf("%s", out[0:len(out)-1])
	return
}

func findOSInfo(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findOSInfo() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving findOSInfo()"}.Debug()
	}()
	ctx.Agent.OS = runtime.GOOS
	ctx.Agent.Env.Arch = runtime.GOARCH
	sysv, kerv, err := getSysProf()
	if err != nil {
		panic(err)
	}
	ctx.Agent.Env.Ident = sysv + " " + kerv
	ctx.Agent.Env.Init = "launchd"
	return
}

// getVersion reads the MacOS system profile
func getSysProf() (sysv, kerv string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getSysProf() -> %v", e)
		}
	}()
	// get data from the system_profiler
	// don't attempt to use their weird xml format, better parse plain text
	out, err := exec.Command("system_profiler", "SPSoftwareDataType").Output()
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(out)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			goto exit
		}
		if len(line) < 10 {
			continue
		}
		// chomp
		if line[len(line)-1] == '\n' {
			line = line[0 : len(line)-1]
		}
		if strings.Contains(line, "System Version") {
			out := strings.SplitN(line, ":", 2)
			if len(out) == 2 {
				sysv = out[1]
			}
			sysv = cleanString(sysv)
		} else if strings.Contains(line, "Kernel Version") {
			out := strings.SplitN(line, ":", 2)
			if len(out) == 2 {
				kerv = out[1]
			}
			kerv = cleanString(kerv)
		}
	}
exit:
	return
}

func getRunDir() string {
	return "/Library/Preferences/mig/"
}
