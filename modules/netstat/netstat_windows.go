// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor(s):
// - Julien Vehent jvehent@mozilla.com [:ulfr]
// - Rob Murtha robmurtha@gmail.com [:robmurtha]

package netstat /* import "mig.ninja/mig/modules/netstat" */

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
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
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasIPConnected(): %v", e)
		}
	}()
	found = false
	var ipnet *net.IPNet

	// if val contains a /, treat it as a cidr
	if strings.IndexAny(val, "/") > 0 {
		_, ipnet, err = net.ParseCIDR(val)
		if err != nil {
			panic(err)
		}
		// otherwise assume it's a single address (v4 /32 or v6 /128)
	} else {
		ip := net.ParseIP(val)
		if ip == nil {
			panic("Invalid IP")
		}
		ipnet = new(net.IPNet)
		ipnet.IP = ip
		// See if it's an IPv4 address, set our mask
		if ipnet.IP.To4() != nil {
			ipnet.Mask = net.CIDRMask(net.IPv4len*8, net.IPv4len*8)
		} else {
			ipnet.Mask = net.CIDRMask(net.IPv6len*8, net.IPv6len*8)
		}
	}

	var out []byte
	if ipnet.IP.To4() != nil {
		tcp, err := exec.Command("netstat", "-na", "-p", "TCP").Output()
		if err != nil {
			panic(err)
		}
		udp, err := exec.Command("netstat", "-na", "-p", "UDP").Output()
		if err != nil {
			panic(err)
		}
		out = append(tcp, udp...)
	} else {
		tcp, err := exec.Command("netstat", "-na", "-p", "TCPv6").Output()
		if err != nil {
			panic(err)
		}
		udp, err := exec.Command("netstat", "-na", "-p", "UDPv6").Output()
		if err != nil {
			panic(err)
		}
		out = append(tcp, udp...)
	}

	net := new(NetstatWinOutput)
	err = net.UnmarshalText(out)
	if err != nil {
		panic(err)
	}
	elements = net.HasIPConnected(ipnet)
	found = len(elements) > 0
	return
}

func HasListeningPort(port string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasListeningPort(): %v", e)
		}
	}()
	portInt, err := strconv.Atoi(port)
	if err != nil {
		panic(err)
	}

	out, err := exec.Command("netstat", "-na").Output()
	if err != nil {
		panic(err)
	}
	net := new(NetstatWinOutput)
	err = net.UnmarshalText(out)
	if err != nil {
		panic(err)
	}

	elements = net.HasListeningPort(portInt)
	found = len(elements) > 0
	return
}

func HasSeenIP(val string) (found bool, elements []element, err error) {
	// XXX Currently not implemented for windows.
	err = fmt.Errorf("HasSeenIP(): operation is not implemented on windows")
	return
}
