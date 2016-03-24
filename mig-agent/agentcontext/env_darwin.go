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
	out, err := exec.Command("hostname", "-f").Output()
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
	sysv, kerv, err := getSysProf()
	if err != nil {
		panic(err)
	}
	ctx.OSIdent = sysv + " " + kerv
	ctx.Init = "launchd"
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

func GetRunDir() string {
	return "/Library/Preferences/mig/"
}
