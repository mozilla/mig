// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package agentcontext

import (
	"bufio"
	"bytes"
	"fmt"
	"mig.ninja/mig"
	"os/exec"
	"strings"
)

func findHostname(orig_ctx AgentContext) (ctx AgentContext, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findHostname() -> %v", e)
		}
		logChan <- mig.Log{Desc: "leaving findHostname()"}.Debug()
	}()

	// get the hostname
	out, err := exec.Command("hostname").Output()
	if err != nil {
		panic(err)
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
	hostname, domain, osname, osversion, err := getSysInfo()
	if err != nil {
		panic(err)
	}
	ctx.Hostname = hostname + "." + domain
	ctx.OSIdent = osname + " " + osversion
	ctx.Init = "windows"
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
	out, err := exec.Command("cmd","/C", "wmic.exe os get Caption, Version /format:list & wmic.exe computersystem get Name, domain /format:list").Output()
	if err != nil {
		panic(err)
	}
	buf := bytes.NewReader(out)
	reader := bufio.NewReader(buf)
	iter := 0
	for {
		lineBytes, _, err := reader.ReadLine()
		// don't loop more than 200 times
		if err != nil || iter > 200 {
			goto exit
		}
		line := fmt.Sprintf("%s", lineBytes)
		if strings.Contains(line, "Caption=") {
			out := strings.SplitN(line, "=", 2)
			if len(out) == 2 {
				osname = out[1]
			}
			osname = cleanString(osname)
		} else if strings.Contains(line, "Version=") {
			out := strings.SplitN(line, "=", 2)
			if len(out) == 2 {
				osversion = out[1]
			}
			osversion = cleanString(osversion)
		} else if strings.Contains(line, "Domain=") {
			out := strings.SplitN(line, "=", 2)
			if len(out) == 2 {
				domain = out[1]
			}
			domain = cleanString(domain)
		} else if strings.Contains(line, "Name=") {
			out := strings.SplitN(line, "=", 2)
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

func GetRunDir() string {
	return `C:\Program Files\mig\`
}
