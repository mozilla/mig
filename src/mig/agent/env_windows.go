// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
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
	for {
		lineBytes, _, err := reader.ReadLine()
		// don't loop more than 2000 times
		if err != nil || iter > 2000 {
			goto exit
		}
		line := fmt.Sprintf("%s", lineBytes)
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
	return `C:\Program Files\mig\`
}
