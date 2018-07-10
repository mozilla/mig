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
	"mig.ninja/mig/service"
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
	var kernhosterr bool
	kernhostname, err := os.Hostname()
	if err == nil {
		if strings.ContainsAny(kernhostname, ".") {
			ctx.Hostname = kernhostname
			return
		}
	} else {
		kernhostname = "localhost"
		kernhosterr = true
	}
	fqdnhostbuf, err := exec.Command("hostname", "--fqdn").Output()
	if err != nil {
		ctx.Hostname = kernhostname
		err = nil
		return
	}
	fqdnhost := string(fqdnhostbuf)
	fqdnhost = fqdnhost[0 : len(fqdnhost)-1]
	if kernhosterr {
		ctx.Hostname = fqdnhost
		return
	}
	hcomp := strings.Split(fqdnhost, ".")
	if kernhostname == hcomp[0] {
		ctx.Hostname = fqdnhost
		return
	}
	ctx.Hostname = kernhostname
	return
}

// findOSInfo gathers information about the Linux distribution if possible, and
// determines the init type of the system.
func findOSInfo(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findOSInfo() -> %v", e)
		}
		logChan <- mig.Log{Desc: "leaving findOSInfo()"}.Debug()
	}()
	ctx.OSIdent, err = getLSBRelease()
	if err == nil {
		logChan <- mig.Log{Desc: "using lsb release for distribution ident"}.Debug()
		goto haveident
	}
	logChan <- mig.Log{Desc: fmt.Sprintf("getLSBRelease() failed: %v", err)}.Debug()
	ctx.OSIdent, err = getIssue()

	// Here we check that we read more than '\S'.
	// See https://access.redhat.com/solutions/1138953
	if err == nil && len(ctx.OSIdent) > 3 {
		logChan <- mig.Log{Desc: "using /etc/issue for distribution ident"}.Debug()
		goto haveident
	}
	logChan <- mig.Log{Desc: fmt.Sprintf("getIssue() failed: %v", err)}.Debug()

	ctx.OSIdent, err = getOSRelease()
	if err == nil {
		logChan <- mig.Log{Desc: "using /etc/os-release for distribution ident"}.Debug()
		goto haveident
	}
	log <- mig.Log{Desc: fmt.Sprintf("getOSRelease() failed: %v", err)}.Debug()

	logChan <- mig.Log{Desc: "warning, no valid linux os identification could be found"}.Info()
haveident:
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

// getOSRelease reads /etc/os-release to retrieve the agent's ident from the
// first line.
func getOSRelease() (string, error) {
	contents, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("getOSRelease() -> %v", err)
	}
	index := bytes.IndexByte(contents, byte('\n'))
	if index < 0 {
		return "", fmt.Errorf("getOSRelease() -> OS release name not found")
	}
	return string(contents[0:index]), nil
}

// getInit parses /proc/1/cmdline to find out which init system is used
func getInit() (initname string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getInit() -> %v", e)
		}
	}()
	itype, err := service.GetFlavor()
	if err != nil {
		panic(err)
	}
	switch itype {
	case service.InitSystemV:
		return "sysvinit", nil
	case service.InitSystemd:
		return "systemd", nil
	case service.InitUpstart:
		return "upstart", nil
	default:
		return "sysvinit-fallback", nil
	}
}
