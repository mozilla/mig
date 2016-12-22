// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Rob Murtha robmurtha@gmail.com [:robmurtha]
package netstat

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var errSkip = errors.New("skip")

// NetstatWinOutput supports functions for parsing and responding to queries based on the output of the windows netstat -na command.
type NetstatWinOutput struct {
	Entries []netstatWinEntry
}

// HasIPConnected searches the parsed Entries for matches based on ipnet.
// This function also increments stats.Examined for each entry that is processed.
func (n *NetstatWinOutput) HasIPConnected(ipnet *net.IPNet) []element {
	elements := []element{}
	for _, e := range n.Entries {
		if e.RemoteIP != nil && ipnet.Contains(e.RemoteIP) {
			elements = append(elements, e.Element())
		}
		stats.Examined++
	}
	return elements
}

// HasListeningPort returns elements with matching host listeners for the given port.
func (n *NetstatWinOutput) HasListeningPort(port int) []element {
	elements := []element{}
	for _, e := range n.Entries {
		if e.LocalPort == port {
			elements = append(elements, e.Element())
		}
		stats.Examined++
	}
	return elements
}

// UnmarshalText parses the output of windows netstat -na containing IPv4 and IPv6 addresses into the NetstatWinOutput struct.
//  Sample windows netstat -na output:
//   Active Connections
//   Proto  Local Address          Foreign Address        State
//   TCP    0.0.0.0:135            0.0.0.0:0              LISTENING
//   TCP    0.0.0.0:445            0.0.0.0:0              LISTENING
//   UDP    [fe80::1c5a:2c75:6ce3:9f0%3]:1900  *:*
//   UDP    [fe80::1c5a:2c75:6ce3:9f0%3]:54109  *:*
func (n *NetstatWinOutput) UnmarshalText(text []byte) error {
	n.Entries = []netstatWinEntry{}
	buf := bytes.NewReader(text)
	reader := bufio.NewReader(buf)
	for {
		lineBytes, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		entry := netstatWinEntry{}
		err = entry.UnmarshalText(lineBytes)
		if err != nil {
			if err != errSkip {
				return err
			}
			continue
		}
		n.Entries = append(n.Entries, entry)
	}
	return nil
}

// getIPNet returns a *net.IPNet for a CIDR or an address and sets masks for CIDRs.
// An Invalid IP error is returned for invalid addresses, or errors from net.ParseCIDR for invalid CIDRs.
func (n *NetstatWinOutput) getIPNet(val string) (*net.IPNet, error) {
	// if val contains a /, treat it as a cidr
	if strings.IndexAny(val, "/") > 0 {
		_, ipnet, err := net.ParseCIDR(val)
		if err != nil {
			return nil, err
		}
		return ipnet, nil
	}

	ip := net.ParseIP(val)
	if ip == nil {
		return nil, errors.New("Invalid IP")
	}
	ipnet := new(net.IPNet)
	ipnet.IP = ip

	if ipnet.IP.To4() != nil {
		ipnet.Mask = net.CIDRMask(net.IPv4len*8, net.IPv4len*8)
	} else {
		ipnet.Mask = net.CIDRMask(net.IPv6len*8, net.IPv6len*8)
	}

	return ipnet, nil
}

type netstatWinEntry struct {
	Proto       string
	LocalIP     net.IP
	LocalPort   int
	RemoteIP    net.IP
	RemotePort  int
	RemoteValue string
	State       string
}

func (n *netstatWinEntry) UnmarshalText(text []byte) (err error) {
	fields := strings.Fields(string(text))
	if len(fields) < 3 {
		return errSkip
	}
	if len(fields) == 4 {
		n.State = string(fields[2])
	}
	n.Proto = fields[0]
	if n.LocalIP, n.LocalPort, err = n.parseEndpointString(fields[1]); err != nil {
		return errSkip
	}

	// don't return err if local is ok
	var xerr error
	if n.RemoteIP, n.RemotePort, xerr = n.parseEndpointString(fields[2]); xerr != nil {
		n.RemoteValue = fields[2]
	}
	return
}

// Element returns the entry as an element.
func (n *netstatWinEntry) Element() element {
	var el element
	el.RemoteAddr = n.RemoteIP.String()
	if n.RemotePort != -1 {
		el.RemotePort = float64(n.RemotePort)
	}
	if n.LocalIP != nil {
		el.LocalAddr = n.LocalIP.String()
	}
	if n.LocalPort != -1 {
		el.LocalPort = float64(n.LocalPort)
	}
	return el
}

var endpoint_win_re = regexp.MustCompile("^(.*?)(%[a-z0-9]+)?:(\\*|[0-9]+)$")

func (n *netstatWinEntry) parseEndpointString(str string) (ip net.IP, port int, err error) {
	port = -1
	matches := endpoint_win_re.FindStringSubmatch(str)
	if matches != nil {
		if matches[1] == "*" {
			return
		}
		ip = net.ParseIP(matches[1])
		if matches[3] != "*" {
			if p, err := strconv.Atoi(matches[3]); err == nil {
				port = int(p)
			}
		}
	}
	return
}
