// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// netstat is a module that retrieves network information about the endpoint,
// such as mac addresses, local and connected IPs, listening TCP and UDP
// sockets and peers
package netstat

import (
	"encoding/json"
	"fmt"
	"mig"
	"net"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	mig.RegisterModule("netstat", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters params
	Results    results
}

type params struct {
	LocalMAC      []string `json:"localmac,omitempty"`
	LocalIP       []string `json:"localip,omitempty"`
	NeighborMAC   []string `json:"neighbormac,omitempty"`
	NeighborIP    []string `json:"neighborip,omitempty"`
	ConnectedIP   []string `json:"connectedip,omitempty"`
	ListeningPort []string `json:"listeningport,omitempty"`
}

type results struct {
	LocalMAC      map[string]result `json:"localmac,omitempty"`
	LocalIP       map[string]result `json:"localip,omitempty"`
	NeighborMAC   map[string]result `json:"neighbormac,omitempty"`
	NeighborIP    map[string]result `json:"neighborip,omitempty"`
	ConnectedIP   map[string]result `json:"connectedip,omitempty"`
	ListeningPort map[string]result `json:"listeningport,omitempty"`
	FoundAnything bool              `json:"foundanything"`
	Success       bool              `json:"success"`
	Errors        []string          `json:"errors,omitempty"`
}

type result struct {
	Found    bool      `json:"found"`
	Elements []element `json:"element"`
}

type element struct {
	LocalMACAddr  string  `json:"localmacaddr,omitempty"`
	RemoteMACAddr string  `json:"remotemacaddr,omitempty"`
	LocalAddr     string  `json:"localaddr,omitempty"`
	LocalPort     float64 `json:"localport,omitempty"`
	RemoteAddr    string  `json:"remoteaddr,omitempty"`
	RemotePort    float64 `json:"remoteport,omitempty"`
}

func newResults() *results {
	var r results
	r.LocalMAC = make(map[string]result)
	r.LocalIP = make(map[string]result)
	r.NeighborMAC = make(map[string]result)
	r.NeighborIP = make(map[string]result)
	r.ConnectedIP = make(map[string]result)
	r.ListeningPort = make(map[string]result)
	return &r
}

func (r Runner) ValidateParameters() (err error) {
	for _, val := range r.Parameters.LocalMAC {
		err = validateMAC(val)
		if err != nil {
			return
		}
	}
	for _, val := range r.Parameters.NeighborMAC {
		err = validateMAC(val)
		if err != nil {
			return
		}
	}
	for _, val := range r.Parameters.LocalIP {
		err = validateIP(val)
		if err != nil {
			return
		}
	}
	for _, val := range r.Parameters.ConnectedIP {
		err = validateIP(val)
		if err != nil {
			return
		}
	}
	for _, val := range r.Parameters.ListeningPort {
		err = validatePort(val)
		if err != nil {
			return
		}
	}
	return
}

func validateMAC(regex string) (err error) {
	_, err = regexp.Compile(regex)
	if err != nil {
		return fmt.Errorf("Invalid MAC regexp '%s'. Compilation failed with '%v'. Must be a valid regular expression.", regex, err)
	}
	return
}

// if a '/' is found, validate as CIDR, otherwise validate as IP
func validateIP(val string) error {
	if strings.IndexAny(val, "/") > 0 {
		_, _, err := net.ParseCIDR(val)
		if err != nil {
			return fmt.Errorf("invalid IPv{4,6} CIDR %s: %v. Must be an IP or a CIDR.", val, err)
		}
		return nil
	}
	ip := net.ParseIP(val)
	if ip == nil {
		return fmt.Errorf("invalid IPv{4,6} %s. Must be an IP or a CIDR.", val)
	}
	return nil
}

func validatePort(val string) error {
	port, err := strconv.Atoi(val)
	if err != nil {
		return fmt.Errorf("%s is not a valid port", val)
	}
	if port < 0 || port > 65535 {
		return fmt.Errorf("port out of range. must be between 1 and 65535")
	}
	return nil
}

func (r Runner) Run(args []byte) (resStr string) {
	defer func() {
		if e := recover(); e != nil {
			// return error in json
			r.Results.Success = false
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			resJson, _ := json.Marshal(r.Results)
			resStr = string(resJson[:])
			return
		}
	}()

	err := json.Unmarshal(args, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	r.Results = *newResults()

	for _, val := range r.Parameters.LocalMAC {
		var result result
		result.Found, result.Elements, err = HasLocalMAC(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.LocalMAC[val] = result
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.NeighborMAC {
		var result result
		result.Found, result.Elements, err = HasSeenMac(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.NeighborMAC[val] = result
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.LocalIP {
		var result result
		result.Found, result.Elements, err = HasLocalIP(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.LocalIP[val] = result
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.ConnectedIP {
		var result result
		result.Found, result.Elements, err = HasIPConnected(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.ConnectedIP[val] = result
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, port := range r.Parameters.ListeningPort {
		var result result
		result.Found, result.Elements, err = HasListeningPort(port)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.ListeningPort[port] = result
		if result.Found {
			r.Results.FoundAnything = true
		}
	}

	r.Results.Success = true
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	resStr = string(jsonOutput[:])
	return
}

// HasLocalMac returns the mac addresses that match an input MAC regex
func HasLocalMAC(macstr string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasLocalMac() -> %v", e)
		}
	}()
	found = false
	re, err := regexp.Compile("(?i)" + macstr)
	if err != nil {
		panic(err)
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, iface := range ifaces {
		if re.MatchString(iface.HardwareAddr.String()) {
			found = true
			var el element
			el.LocalMACAddr = iface.HardwareAddr.String()
			elements = append(elements, el)
		}
	}
	return
}

// HasLocalIP compares an input ip address with the ip addresses
// of the local interfaces, and returns found=true when found
func HasLocalIP(ipStr string) (found bool, elements []element, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("HasLocalIP() -> %v", e)
		}
	}()
	found = false
	if strings.IndexAny(ipStr, "/") > 0 {
		_, ipnet, err := net.ParseCIDR(ipStr)
		if err != nil {
			panic(err)
		}
		ifaceAddrs, err := net.InterfaceAddrs()
		if err != nil {
			panic(err)
		}
		for _, ifaceAddr := range ifaceAddrs {
			addr := strings.Split(ifaceAddr.String(), "/")[0]
			if ipnet.Contains(net.ParseIP(addr)) {
				found = true
				var el element
				el.LocalAddr = addr
				elements = append(elements, el)
			}
		}
		return found, elements, err
	}
	ifaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, ifaceAddr := range ifaceAddrs {
		addr := strings.Split(ifaceAddr.String(), "/")[0]
		if ipStr == addr {
			found = true
			var el element
			el.LocalAddr = addr
			elements = append(elements, el)
		}
	}
	return
}
