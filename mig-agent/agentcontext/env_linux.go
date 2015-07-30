// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package agentcontext

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"os"
	"os/exec"
	"strings"
)

func findHostname(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findHostname() -> %v", e)
		}
	}()

	// get the hostname
	out, err := exec.Command("hostname", "--fqdn").Output()
	if err != nil {
		// --fqdn can fail sometimes. when that happens, use Go's builtin
		// hostname lookup (reads /proc/sys/kernel/hostname)
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		ctx.Hostname = hostname
		return ctx, err
	}
	// remove trailing newline
	ctx.Hostname = fmt.Sprintf("%s", out[0:len(out)-1])
	return
}

func findOSInfo(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findOSInfo() -> %v", e)
		}
		logChan <- mig.Log{Desc: "leaving findOSInfo()"}.Debug()
	}()
	ctx.OSIdent, err = getLSBRelease()
	if err != nil {
		logChan <- mig.Log{Desc: fmt.Sprintf("getLSBRelease() failed: %v", err)}.Info()
		ctx.OSIdent, err = getIssue()
		if err != nil {
			logChan <- mig.Log{Desc: fmt.Sprintf("getIssue() failed: %v", err)}.Info()
		}
	}
	logChan <- mig.Log{Desc: fmt.Sprintf("Ident is %s", ctx.OSIdent)}.Debug()

	ctx.Init, err = getInit()
	if err != nil {
		panic(err)
	}
	logChan <- mig.Log{Desc: fmt.Sprintf("Init is %s", ctx.Init)}.Debug()

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
		initname = "sysvinit"
	} else if strings.Contains(init, "systemd") {
		initname = "systemd"
	} else if strings.Contains(init, "init") {
		initname = "upstart"
	} else {
		// failed to detect init system, falling back to sysvinit
		initname = "sysvinit-fallback"
	}
	return
}

func GetRunDir() string {
	return "/var/lib/mig/"
}
