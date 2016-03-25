// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package netstat /* import "mig.ninja/mig/modules/netstat" */

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// HasSeenMac on windows looks at the output of `arp -a` for a matching mac address
// and returns its MAC and IP address if found
func HasSeenMac(val string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasSeenMac() -> %v", e)
		}
	}()
	found = false
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		panic(err)
	}
	// arp -a has a static format:
	// <IP Address>) <Mac Address> <Type>
	// fedbox (172.21.0.3) at 8c:70:5a:c8:be:50 on en1 ifscope [ethernet]
	re, err := regexp.Compile(val)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewReader(out)
	reader := bufio.NewReader(buf)
	for {
		lineBytes, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		line := fmt.Sprintf("%s", lineBytes)
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// match against a second variable with '-' characters converted to ':'
		// because windows likes to display mac address as 8c-70-5a-c8-be-50
		convertedMac := strings.Replace(fields[1], "-", ":", 5)
		if re.MatchString(fields[1]) || re.MatchString(convertedMac) {
			found = true
			var el element
			el.RemoteAddr = fields[0]
			el.RemoteMACAddr = convertedMac
			elements = append(elements, el)
		}
		stats.Examined++
	}
	return
}

func HasIPConnected(val string) (found bool, elements []element, err error) {
	err = fmt.Errorf("HasIPConnected() is not implemented on %s", runtime.GOOS)
	return
}

func HasListeningPort(port string) (found bool, elements []element, err error) {
	err = fmt.Errorf("HasListeningPort() is not implemented on %s", runtime.GOOS)
	return
}

func HasSeenIP(val string) (found bool, elements []element, err error) {
	// XXX Currently not implemented for windows.
	err = fmt.Errorf("HasSeenIP(): operation is not implemented on windows")
	return
}
