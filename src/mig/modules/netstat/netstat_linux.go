// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package netstat

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
)

// HasSeenMac on linux looks for a matching mac address in /proc/net/arp
// and returns its MAC and IP address if found
func HasSeenMac(val string) (found bool, macaddr, addr string, err error) {
	found = false
	fd, err := os.Open("/proc/net/arp")
	if err != nil {
		return found, macaddr, addr, err
	}
	// /proc/net/arp has a static format:
	// IP address       HW type     Flags       HW address            Mask     Device
	// we split the string on fields, and compare field #4 with our search regex
	re, err := regexp.Compile(val)
	if err != nil {
		return found, macaddr, addr, err
	}
	scanner := bufio.NewScanner(fd)
	scanner.Scan() // skip the header
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		if re.MatchString(fields[3]) {
			found = true
			addr = fields[0]
			macaddr = fields[3]
			return found, macaddr, addr, err
		}
	}
	fd.Close()
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
