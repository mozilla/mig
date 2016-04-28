// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package netstat /* import "mig.ninja/mig/modules/netstat" */

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Represents lines obtained from the /proc file system related to network
// activity, for example a line read from /proc/<pid>/net/tcp. The namespace
// identifier is associated with the entry so we can track it.
type procNetLine struct {
	line         string // The line from the proc file
	nsIdentifier string // A namespace identifier associated with the process
}

// HasSeenMac on linux looks for a matching mac address in /proc/net/arp or
// in individual processes <pid>/net/arp, and returns its MAC and IP address
// if found
func HasSeenMac(val string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasSeenMac(): %v", e)
		}
	}()
	found = false
	lns, err := procArpEntries()
	if err != nil {
		panic(err)
	}
	// /proc arp entries have a static format:
	// IP address       HW type     Flags       HW address            Mask     Device
	// we split the string on fields, and compare field #4 with our search regex
	re, err := regexp.Compile(val)
	if err != nil {
		panic(err)
	}
	for _, arpent := range lns {
		fields := strings.Fields(arpent.line)
		if len(fields) < 4 {
			continue
		}
		if re.MatchString(fields[3]) {
			found = true
			var el element
			el.RemoteAddr = fields[0]
			el.RemoteMACAddr = fields[3]
			el.Namespace = arpent.nsIdentifier
			elements = append(elements, el)
		}
		stats.Examined++
	}
	return
}

func procArpEntries() (ret []procNetLine, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("procArpEntries(): %v", e)
		}
	}()

	// If we have namespace resolution enabled, use that
	if namespaceMode {
		return procArpEntriesNS()
	}

	fd, err := os.Open("/proc/net/arp")
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	scanner := bufio.NewScanner(fd)
	scanner.Scan() // Skip the header
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		newnl := procNetLine{line: scanner.Text(), nsIdentifier: "default"}
		ret = append(ret, newnl)
	}

	return ret, nil
}

func procArpEntriesNS() (ret []procNetLine, err error) {
	return procNetNS("arp")
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
		// Also search for the IPv4 address as an IPv4-mapped
		// IPv6 address; IPv4 peers connected to a e.g. a tcp6 socket will
		// be found in the IPv6 related proc net files (e.g., /proc/net/tcp6)
		// if this is the case, so we also want to check there.
		//
		// net.IP already stores the IPv4 address internally in the
		// format we need, so just pass it into the IP6 function
		found2, elements2, err := hasIP6Connected(ip, ipnet)
		if err != nil {
			panic(err)
		}
		if found2 {
			found = true
			for _, x := range elements2 {
				elements = append(elements, x)
			}
		}
	} else { // or an ipv6
		found, elements, err = hasIP6Connected(ip, ipnet)
		if err != nil {
			panic(err)
		}
	}
	return
}

// hasIP4Connected parses the list of remote addresses in /proc/net/{tcp,udp} or in
// /proc/<pid>/net/{tcp,udp} and returns addresses that are contained within the
// ipnet submitted. It always uses CIDR inclusion, even when only
// searching for a single IP (but assuming a /32 bitmask).
// Remote addresses exposed in /proc are in hexadecimal notation, and converted into byte slices
// to use in ipnet.Contains()
func hasIP4Connected(ip net.IP, ipnet *net.IPNet) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("hasIP4Connected(): %v", e)
		}
	}()
	lns, err := procIP4Entries()
	if err != nil {
		panic(err)
	}
	// if the ipnet is nil, assume that its a full 32bits mask
	if ipnet == nil {
		ipnet = new(net.IPNet)
		ipnet.IP = ip
		ipnet.Mask = net.CIDRMask(net.IPv4len*8, net.IPv4len*8)
	}
	for _, ipent := range lns {
		fields := strings.Fields(ipent.line)
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
			el.Namespace = ipent.nsIdentifier
			elements = append(elements, el)
			found = true
		}
		stats.Examined++
	}
	return
}

func procIP4Entries() (ret []procNetLine, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("procIP4Entries(): %v", e)
		}
	}()

	// If we have namespace resolution mode enabled, use that
	if namespaceMode {
		return procIP4EntriesNS()
	}

	var connfiles = [...]string{"/proc/net/tcp", "/proc/net/udp"}
	for _, f := range connfiles {
		fd, err := os.Open(f)
		if err != nil {
			panic(err)
		}
		defer fd.Close()
		scanner := bufio.NewScanner(fd)
		scanner.Scan() // Skip the header
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			newnl := procNetLine{line: scanner.Text(), nsIdentifier: "default"}
			ret = append(ret, newnl)
		}
	}

	return ret, nil
}

func procIP4EntriesNS() (ret []procNetLine, err error) {
	var connfiles = [...]string{"tcp", "udp"}
	for _, f := range connfiles {
		ts, err := procNetNS(f)
		if err != nil {
			return ret, err
		}
		ret = append(ret, ts...)
	}
	return ret, nil
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

// hasIP6Connected parses the list of remote addresses in /proc/net/{tcp,udp}6 or in
// /proc/<pid>/net/{tcp,udp}6 and returns addresses that are contained within
// the ipnet submitted. It always uses CIDR inclusion, even when only
// searching for a single IP (but assuming a /128 bitmask).
// Remote addresses exposed in /proc are in hexadecimal notation, and converted into byte slices
// to use in ipnet.Contains()
func hasIP6Connected(ip net.IP, ipnet *net.IPNet) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("hasIP6Connected(): %v", e)
		}
	}()
	lns, err := procIP6Entries()
	if err != nil {
		panic(err)
	}
	// if the ipnet is nil, assume that its a full 128bits mask
	if ipnet == nil {
		ipnet = new(net.IPNet)
		ipnet.IP = ip
		ipnet.Mask = net.CIDRMask(net.IPv6len*8, net.IPv6len*8)
	}
	for _, ipent := range lns {
		fields := strings.Fields(ipent.line)
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
			el.Namespace = ipent.nsIdentifier
			elements = append(elements, el)
			found = true
		}
		stats.Examined++
	}
	return
}

func procIP6Entries() (ret []procNetLine, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("procIP6Entries(): %v", e)
		}
	}()

	// If we have namespace resolution mode enabled, use that
	if namespaceMode {
		return procIP6EntriesNS()
	}

	var connfiles = [...]string{"/proc/net/tcp6", "/proc/net/udp6"}
	for _, f := range connfiles {
		fd, err := os.Open(f)
		if err != nil {
			panic(err)
		}
		defer fd.Close()
		scanner := bufio.NewScanner(fd)
		scanner.Scan() // Skip the header
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			newnl := procNetLine{line: scanner.Text(), nsIdentifier: "default"}
			ret = append(ret, newnl)
		}
	}

	return ret, nil
}

func procIP6EntriesNS() (ret []procNetLine, err error) {
	var connfiles = [...]string{"tcp6", "udp6"}
	for _, f := range connfiles {
		ts, err := procNetNS(f)
		if err != nil {
			return ret, err
		}
		ret = append(ret, ts...)
	}
	return ret, nil
}

// hexToIP6 converts the hexadecimal representation of an IP address as found in
// /proc/net/tcp6, into a net.IP byte slice as defined in the net package
//
// the hex address as found in tcp6 is stored as 4 words of 4 bytes each, where in
// each word the bytes are in reverse order.
func hexToIP6(hexIP string) net.IP {
	ip := make(net.IP, net.IPv6len)
	ipPos := 0
	// Loop through the hex string, and grab 8 bytes of the hex string
	// (4 bytes of the address) at a time
	for i := 0; i < 32; i += 8 {
		// Reverse the byte order in the word and store it in ip
		for lctr := i + 8; lctr > i; lctr -= 2 {
			b, err := strconv.ParseUint(string(hexIP[lctr-2:lctr]), 16, 8)
			if err != nil {
				return nil
			}
			ip[ipPos] = uint8(b)
			ipPos++
		}
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
	lns := make([]procNetLine, 0)
	lnbuf, err := procIP4Entries()
	if err != nil {
		panic(err)
	}
	lns = append(lns, lnbuf...)
	lnbuf, err = procIP6Entries()
	if err != nil {
		panic(err)
	}
	lns = append(lns, lnbuf...)
	for _, ipent := range lns {
		fields := strings.Fields(ipent.line)
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
			el.Namespace = ipent.nsIdentifier
			elements = append(elements, el)
			found = true
		}
		stats.Examined++
	}
	return
}

// Given a network proc file (e.g., tcp would represent instances of
// <pid>/net/tcp visible by the agent, return a list of proc network entries
// for each active network namespace, including the host.
//
// This function works by building a cache of all current network namespaces
// on the host based on /proc. From this list, we try to collect data from the
// indicated file for at most one process per namespace. Since <pid>/net/*
// represents network activity for the namespace the process is in, and not
// the process itself we only need to be successful with one PID in each
// namespace.
func procNetNS(fname string) (ret []procNetLine, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("procNetNS(): %v", e)
		}
	}()
	procCache, err := procCandidates()
	if err != nil {
		panic(err)
	}

	// For each cached namespace, attempt to extract data from one
	// process. We try until we find one to help work around the
	// potential for the process to exit after we have cached the
	// list.
	for nsid, procpaths := range procCache {
		lns := make([]string, 0)
		for _, procpath := range procpaths {
			nsEntrych := make(chan string, 0)
			nsErrch := make(chan error, 0)
			go procNetNSEntries(path.Join(procpath, fname), nsEntrych, nsErrch)
			done := false
			err = nil
			for {
				if done {
					break
				}
				select {
				case newent, ok := <-nsEntrych:
					if !ok {
						done = true
					} else {
						lns = append(lns, newent)
					}
				case errent := <-nsErrch:
					// An error occurred with this
					// candidate, zero the slice and try
					// the next one.
					done = true
					err = errent
				}
			}
			if err != nil {
				// We don't treat this condition as fatal, just
				// continue with the next candidate
				lns = lns[:0]
				continue
			} else {
				break
			}
		}
		for _, x := range lns {
			ret = append(ret, procNetLine{line: x, nsIdentifier: nsid})
		}
	}

	return ret, nil
}

// Return process candidates organized by active network namespace, this will
// include the host namespace in addition to other namespaces
func procCandidates() (map[string][]string, error) {
	var err error
	ret := make(map[string][]string)

	dirents, err := ioutil.ReadDir("/proc")
	if err != nil {
		return ret, err
	}
	for _, x := range dirents {
		_, err = strconv.Atoi(x.Name())
		if err != nil {
			continue
		}
		nspath := path.Join("/proc", x.Name(), "ns", "net")
		nsname, err := os.Readlink(nspath)
		if err != nil {
			// We don't treat this condition as fatal
			continue
		}
		if _, ok := ret[nsname]; !ok {
			ret[nsname] = make([]string, 0)
		}
		ret[nsname] = append(ret[nsname], path.Join("/proc", x.Name(), "net"))
	}

	return ret, nil
}

// Read lines from a network proc file; used by procNetNS()
func procNetNSEntries(fname string, nsEntrych chan string, nsErrch chan error) {
	fd, err := os.Open(fname)
	if err != nil {
		nsErrch <- err
		return
	}
	defer fd.Close()
	scanner := bufio.NewScanner(fd)
	scanner.Scan() // Skip the header
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			nsErrch <- err
			return
		}
		nsEntrych <- scanner.Text()
	}
	close(nsEntrych)
}

// HasSeenIP on linux looks for a matching IP address in /proc/net/arp or
// in individual processes <pid>/net/arp, and returns its MAC and IP address
// if found
func HasSeenIP(val string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasSeenIP(): %v", e)
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
		found, elements, err = hasSeenIP4(ip, ipnet)
		if err != nil {
			panic(err)
		}
		// or an ipv6
	} else {
		found, elements, err = hasSeenIP6(ip, ipnet)
		if err != nil {
			panic(err)
		}
	}
	return
}

func hasSeenIP4(ip net.IP, ipnet *net.IPNet) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("hasSeenIP4(): %v", e)
		}
	}()
	lns, err := procArpEntries()
	if err != nil {
		panic(err)
	}
	// if the ipnet is nil, assume that its a full 32bits mask
	if ipnet == nil {
		ipnet = new(net.IPNet)
		ipnet.IP = ip
		ipnet.Mask = net.CIDRMask(net.IPv4len*8, net.IPv4len*8)
	}
	for _, arpent := range lns {
		fields := strings.Fields(arpent.line)
		if len(fields) < 4 {
			continue
		}
		remoteIP := net.ParseIP(fields[0])
		if remoteIP == nil {
			panic("failed to convert remote IP")
		}
		// if we've got a match, store the element
		if ipnet.Contains(remoteIP) {
			var el element
			el.RemoteAddr = remoteIP.String()
			el.RemoteMACAddr = fields[3]
			el.Namespace = arpent.nsIdentifier
			elements = append(elements, el)
			found = true
		}
		stats.Examined++
	}
	return
}

func hasSeenIP6(ip net.IP, ipnet *net.IPNet) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("hasSeenIP6(): %v", e)
		}
	}()
	lns, err := procArpEntries()
	if err != nil {
		panic(err)
	}
	// if the ipnet is nil, assume that its a full 128bits mask
	if ipnet == nil {
		ipnet = new(net.IPNet)
		ipnet.IP = ip
		ipnet.Mask = net.CIDRMask(net.IPv6len*8, net.IPv6len*8)
	}
	for _, arpent := range lns {
		fields := strings.Fields(arpent.line)
		if len(fields) < 4 {
			continue
		}
		remoteIP := net.ParseIP(fields[0])
		if remoteIP == nil {
			panic("failed to convert remote IP")
		}
		// if we've got a match, store the element
		if ipnet.Contains(remoteIP) {
			var el element
			el.RemoteAddr = remoteIP.String()
			el.RemoteMACAddr = fields[3]
			el.Namespace = arpent.nsIdentifier
			elements = append(elements, el)
			found = true
		}
		stats.Examined++
	}
	return
}
