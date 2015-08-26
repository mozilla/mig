// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package netstat /* import "mig.ninja/mig/modules/netstat" */

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// HasSeenMac on linux looks for a matching mac address in /proc/net/arp
// and returns its MAC and IP address if found
func HasSeenMac(val string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasSeenMac(): %v", e)
		}
	}()
	found = false
	fd, err := os.Open("/proc/net/arp")
	defer fd.Close()
	if err != nil {
		panic(err)
	}
	// /proc/net/arp has a static format:
	// IP address       HW type     Flags       HW address            Mask     Device
	// we split the string on fields, and compare field #4 with our search regex
	re, err := regexp.Compile(val)
	if err != nil {
		panic(err)
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
			var el element
			el.RemoteAddr = fields[0]
			el.RemoteMACAddr = fields[3]
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
	var ip net.IP
	var ipnet *net.IPNet

	// if val contains a /, treat it as a cidr
	if strings.IndexAny(val, "/") > 0 {
		ip, ipnet, err = net.ParseCIDR(val)
		if err != nil {
			panic(err)
		}
	} else {
		ip = net.ParseIP(val)
		if ip == nil {
			panic("Invalid IP")
		}
	}
	// test if we have an ipv4
	if ip.To4() != nil {
		found, elements, err = hasIP4Connected(ip, ipnet)
		if err != nil {
			panic(err)
		}
		// or an ipv6
	} else {
		found, elements, err = hasIP6Connected(ip, ipnet)
		if err != nil {
			panic(err)
		}
	}
	return
}

// hasIP4Connected parse the list of remote addresses in /proc/net/{tcp,udp} and returns addresses
// that are contained within the ipnet submitted. It always uses CIDR inclusion, even when only
// searching for a single IP (but assuming a /32 bitmask).
// Remote addresses exposed in /proc are in hexadecimal notation, and converted into byte slices
// to use in ipnet.Contains()
func hasIP4Connected(ip net.IP, ipnet *net.IPNet) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("hasIP4Connected(): %v", e)
		}
	}()
	var connfiles = [...]string{`/proc/net/tcp`, `/proc/net/udp`}
	// if the ipnet is nil, assume that its a full 32bits mask
	if ipnet == nil {
		ipnet = new(net.IPNet)
		ipnet.IP = ip
		ipnet.Mask = net.CIDRMask(net.IPv4len*8, net.IPv4len*8)
	}
	for _, f := range connfiles {
		fd, err := os.Open(f)
		defer fd.Close()
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(fd)
		scanner.Scan() // skip the header
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			fields := strings.Fields(scanner.Text())
			if len(fields) < 4 {
				panic("/proc doesn't respect the expected format")
			}
			remote := strings.Split(fields[2], ":")
			if len(remote) != 2 {
				panic("remote isn't in the form <ip>:<port>")
			}
			remoteIP := hexToIP4(remote[0])
			if remoteIP == nil {
				panic("failed to convert remote IP")
			}
			// if we've got a match, store the element
			if ipnet.Contains(remoteIP) {
				var el element
				el.RemoteAddr = remoteIP.String()
				remotePort, err := strconv.ParseUint(remote[1], 16, 16)
				if err != nil {
					panic("failed to convert remote port")
				}
				el.RemotePort = float64(remotePort)
				local := strings.Split(fields[1], ":")
				if len(local) != 2 {
					panic("local isn't in the form <ip>:<port>")
				}
				localAddr := hexToIP4(local[0])
				if localAddr == nil {
					panic("failed to convert local ip")
				}
				el.LocalAddr = localAddr.String()
				localPort, err := strconv.ParseUint(local[1], 16, 16)
				if err != nil {
					panic("failed to convert local port")
				}
				el.LocalPort = float64(localPort)
				elements = append(elements, el)
				found = true
			}
			stats.Examined++
		}
	}
	return
}

// hexToIP4 converts the hexadecimal representation of an IP address as found in
// /proc/net/tcp, into a net.IP byte slice as defined in the net package
func hexToIP4(hexIP string) net.IP {
	ip := make(net.IP, net.IPv4len)
	ipPos := 3
	pos := 0
	for {
		if ipPos < 0 {
			break
		}
		currentByte, err := strconv.ParseUint(string(hexIP[pos:pos+2]), 16, 8)
		if err != nil {
			return nil
		}
		ip[ipPos] = uint8(currentByte)
		ipPos--
		pos += 2
	}
	return ip
}

// hasIP6Connected parse the list of remote addresses in /proc/net/{tcp,udp}6 and returns addresses
// that are contained within the ipnet submitted. It always uses CIDR inclusion, even when only
// searching for a single IP (but assuming a /128 bitmask).
// Remote addresses exposed in /proc are in hexadecimal notation, and converted into byte slices
// to use in ipnet.Contains()
func hasIP6Connected(ip net.IP, ipnet *net.IPNet) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("hasIP6Connected(): %v", e)
		}
	}()
	var connfiles = [...]string{`/proc/net/tcp6`, `/proc/net/udp6`}
	// if the ipnet is nil, assume that its a full 128bits mask
	if ipnet == nil {
		ipnet = new(net.IPNet)
		ipnet.IP = ip
		ipnet.Mask = net.CIDRMask(net.IPv6len*8, net.IPv6len*8)
	}
	for _, f := range connfiles {
		fd, err := os.Open(f)
		defer fd.Close()
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(fd)
		scanner.Scan() // skip the header
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			fields := strings.Fields(scanner.Text())
			if len(fields) < 4 {
				panic("/proc doesn't respect the expected format")
			}
			remote := strings.Split(fields[2], ":")
			if len(remote) != 2 {
				panic("remote isn't in the form <ip>:<port>")
			}
			remoteIP := hexToIP6(remote[0])
			if remoteIP == nil {
				panic("failed to convert remote IP")
			}
			// if we've got a match, store the element
			if ipnet.Contains(remoteIP) {
				var el element
				el.RemoteAddr = remoteIP.String()
				remotePort, err := strconv.ParseUint(remote[1], 16, 16)
				if err != nil {
					panic("failed to convert remote port")
				}
				el.RemotePort = float64(remotePort)
				local := strings.Split(fields[1], ":")
				if len(local) != 2 {
					panic("local isn't in the form <ip>:<port>")
				}
				localAddr := hexToIP6(local[0])
				if localAddr == nil {
					panic("failed to convert local ip")
				}
				el.LocalAddr = localAddr.String()
				localPort, err := strconv.ParseUint(local[1], 16, 16)
				if err != nil {
					panic("failed to convert local port")
				}
				el.LocalPort = float64(localPort)
				elements = append(elements, el)
				found = true
			}
			stats.Examined++
		}
	}
	return
}

// hexToIP6 converts the hexadecimal representation of an IP address as found in
// /proc/net/tcp6, into a net.IP byte slice as defined in the net package
func hexToIP6(hexIP string) net.IP {
	ip := make(net.IP, net.IPv6len)
	ipPos := 15
	pos := 0
	for {
		if ipPos < 0 {
			break
		}
		currentByte, err := strconv.ParseUint(string(hexIP[pos:pos+2]), 16, 8)
		if err != nil {
			return nil
		}
		ip[ipPos] = uint8(currentByte)
		ipPos--
		pos += 2
	}
	return ip
}

func HasListeningPort(val string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasListeningPort(): %v", e)
		}
	}()
	portInt, err := strconv.Atoi(val)
	if err != nil {
		panic(err)
	}
	port := fmt.Sprintf("%X", portInt)
	// port must be exactly 4 characters long. prepend 0s if needed
	for {
		if len(port) < 4 {
			port = "0" + port
		} else {
			break
		}
	}
	var connfiles = [...]string{`/proc/net/tcp`, `/proc/net/udp`, `/proc/net/tcp6`, `/proc/net/udp6`}
	for _, f := range connfiles {
		fd, err := os.Open(f)
		defer fd.Close()
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(fd)
		scanner.Scan() // skip the header
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			fields := strings.Fields(scanner.Text())
			if len(fields) < 4 {
				panic("/proc doesn't respect the expected format")
			}
			local := strings.Split(fields[1], ":")
			lPort := local[1]
			// if we've got a match, store the element
			if lPort == port {
				var el element
				var localAddr net.IP
				switch len(local[0]) {
				case 8:
					localAddr = hexToIP4(local[0])
				case 32:
					localAddr = hexToIP6(local[0])
				default:
					panic("invalid length for local address")
				}
				if localAddr == nil {
					panic("failed to convert local ip")
				}
				el.LocalAddr = localAddr.String()
				el.LocalPort = float64(portInt)
				elements = append(elements, el)
				found = true
			}
			stats.Examined++
		}
	}
	return
}
