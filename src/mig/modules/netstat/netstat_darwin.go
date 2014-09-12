// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package netstat

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// HasSeenMac on darwin looks at the output of `arp -a` for a matching mac address
// and returns its MAC and IP address if found
func HasSeenMac(val string) (found bool, macaddr, addr string, err error) {
	found = false
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		return found, macaddr, addr, err
	}
	// arp -a has a static format:
	// <Hostname> (<IP Address>) at <Mac Address> on <Interface> [ifscope <type>]
	// fedbox (172.21.0.3) at 8c:70:5a:c8:be:50 on en1 ifscope [ethernet]
	re, err := regexp.Compile(val)
	if err != nil {
		return found, macaddr, addr, err
	}
	buf := bytes.NewReader(out)
	reader := bufio.NewReader(buf)
	for {
		lineBytes, _, err := reader.ReadLine()
		line := fmt.Sprintf("%s", lineBytes)
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if re.MatchString(fields[3]) {
			found = true
			// remove heading and trailing parenthesis
			if len(fields[1]) > 2 {
				addr = fields[1][1 : len(fields[1])-1]
			}
			macaddr = fields[3]
			return found, macaddr, addr, err
		}
	}
	return
}

func HasIPConnected(val string) (r result, err error) {
	err = fmt.Errorf("HasIPConnected() is not implemented on %s", runtime.GOOS)
	return
}

func HasListeningPort(val string) (r result, err error) {
	err = fmt.Errorf("HasListeningPort() is not implemented on %s", runtime.GOOS)
	return
}
