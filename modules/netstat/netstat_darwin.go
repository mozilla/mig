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
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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
			break
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
		stats.Examined++
	}
	return
}

var endpoint_re = regexp.MustCompile("^(.*?)(%[a-z0-9]+)?\\.(\\*|[0-9]+)$")

func parseEndpointString(str string) (ip net.IP, port int, err error) {
	// Note that 'netstat' will sometimes truncate a long IPv6 address, in
	// which case this function may return an incorrect address or (if the
	// result ends with ':') nil.

	matches := endpoint_re.FindStringSubmatch(str)
	if matches != nil {
		if matches[1] == "*" {
			return nil, -1, nil
		}
		ip = net.ParseIP(matches[1])
		if matches[3] == "*" {
			port = -1
		} else {
			p, err := strconv.Atoi(matches[3])
			if err != nil {
				return nil, -1, err
			}
			port = int(p)
		}
		return
	}
	return nil, -1, nil
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

	var fam string
	if ipnet.IP.To4() != nil {
		fam = "inet"
	} else {
		fam = "inet6"
	}

	out, err := exec.Command("netstat", "-naW", "-f", fam).Output()
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
		if len(fields) <= 4 {
			continue
		}
		localIP, localPort, err := parseEndpointString(fields[3])
		if err != nil {
			break
		}
		remoteIP, remotePort, err := parseEndpointString(fields[4])
		if err != nil {
			break
		}
		if remoteIP != nil && ipnet.Contains(remoteIP) {
			var el element
			el.RemoteAddr = remoteIP.String()
			if remotePort != -1 {
				el.RemotePort = float64(remotePort)
			}
			if localIP != nil {
				el.LocalAddr = localIP.String()
			}
			if localPort != -1 {
				el.LocalPort = float64(localPort)
			}
			elements = append(elements, el)
			found = true
		}
		stats.Examined++
	}
	return
}

func HasListeningPort(port string) (found bool, elements []element, err error) {
	err = fmt.Errorf("HasListeningPort() is not implemented on %s", runtime.GOOS)
	return
}

func HasSeenIP(val string) (found bool, elements []element, err error) {
	// XXX Currently not implemented for darwin.
	err = fmt.Errorf("HasSeenIP(): operation is not implemented on darwin")
	return
}
