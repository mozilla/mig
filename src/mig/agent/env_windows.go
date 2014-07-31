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
	"bufio"
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
	out, err := exec.Command("hostname").Output()
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
	hostname, domain, osname, osversion, err := getSysInfo()
	if err != nil {
		panic(err)
	}
	ctx.Agent.Hostname = hostname + "." + domain
	ctx.Agent.Env.Ident = osname + " " + osversion
	ctx.Agent.Env.Init = "windows"
	return
}

// getSysInfo call systeminfo and returns the OS version
func getSysInfo() (hostname, domain, osname, osversion string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getSysInfo() -> %v", e)
		}
	}()
	// get data from the systeminfo
	out, err := exec.Command("systeminfo").Output()
	if err != nil {
		panic(err)
	}
	buf := bytes.NewReader(out)
	reader := bufio.NewReader(buf)
	iter := 0
	fmt.Println("parsing systeminfo output")
	for {
		lineBytes, _, err := reader.ReadLine()
		// don't loop more than 2000 times
		if err != nil || iter > 2000 {
			goto exit
		}
		line := fmt.Sprintf("%s", lineBytes)
		fmt.Println(line)
		if strings.Contains(line, "OS Name:") {
			out := strings.SplitN(line, ":", 2)
			if len(out) == 2 {
				osname = out[1]
			}
			osname = cleanString(osname)
		} else if strings.Contains(line, "OS Version:") {
			out := strings.SplitN(line, ":", 2)
			if len(out) == 2 {
				osversion = out[1]
			}
			osversion = cleanString(osversion)
		} else if strings.Contains(line, "Domain:") {
			out := strings.SplitN(line, ":", 2)
			if len(out) == 2 {
				domain = out[1]
			}
			domain = cleanString(domain)
		} else if strings.Contains(line, "Host Name:") {
			out := strings.SplitN(line, ":", 2)
			if len(out) == 2 {
				hostname = out[1]
			}
			hostname = cleanString(hostname)
		}
		iter++
	}
exit:
	return
}

func getRunDir() string {
	return `C:\Windows\`
}
