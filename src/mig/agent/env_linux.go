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
	"io/ioutil"
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
	out, err := exec.Command("hostname", "--fqdn").Output()
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
	ctx.Agent.Env.Ident, err = getLSBRelease()
	if err != nil {
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("getLSBRelease() failed: %v", err)}.Info()
		ctx.Agent.Env.Ident, err = getIssue()
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("getIssue() failed: %v", err)}.Info()
		}
	}
	ctx.Agent.Env.Init, err = getInit()
	if err != nil {
		panic(err)
	}
	return
}

// getLSBRelease reads the linux identity from lsb_release -a
func getLSBRelease() (desc string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getLSBRelease() -> %v", e)
		}
	}()
	path, err := exec.LookPath("lsb_release")
	if err != nil {
		return "", fmt.Errorf("lsb_release is not present")
	}
	out, err := exec.Command(path, "-i", "-r", "-c", "-s").Output()
	if err != nil {
		panic(err)
	}
	desc = fmt.Sprintf("%s", out[0:len(out)-1])
	desc = cleanString(desc)
	if err != nil {
		panic(err)
	}
	return
}

// getIssue parses /etc/issue and returns the first line
func getIssue() (initname string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getIssue() -> %v", e)
		}
	}()
	issue, err := ioutil.ReadFile("/etc/issue")
	if err != nil {
		panic(err)
	}
	loc := bytes.IndexAny(issue, "\n")
	if loc < 2 {
		return "", fmt.Errorf("issue string not found")
	}
	initname = fmt.Sprintf("%s", issue[0:loc])
	return
}

// getInit parses /proc/1/cmdline to find out which init system is used
func getInit() (initname string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getInit() -> %v", e)
		}
	}()
	initCmd, err := ioutil.ReadFile("/proc/1/cmdline")
	if err != nil {
		panic(err)
	}
	init := fmt.Sprintf("%s", initCmd)
	if strings.Contains(init, "init [") {
		initname = "upstart"
	} else if strings.Contains(init, "systemd") {
		initname = "systemd"
	} else if strings.Contains(init, "init") {
		initname = "sysvinit"
	} else {
		// failed to detect init system, falling back to sysvinit
		initname = "sysvinit-fallback"
	}
	return
}

func getRunDir() string {
	return "/var/run/mig/"
}
