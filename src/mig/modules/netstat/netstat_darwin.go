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
	// <Hostname> (<IP Address>) at <Mac Address> on <Interface> [ifscope <type>]
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
			panic(err)
		}
		line := fmt.Sprintf("%s", lineBytes)
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if re.MatchString(fields[3]) {
			found = true
			var el element
			// remove heading and trailing parenthesis
			if len(fields[1]) > 2 {
				el.RemoteAddr = fields[1][1 : len(fields[1])-1]
			}
			el.RemoteMACAddr = fields[3]
			elements = append(elements, el)
		}
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
