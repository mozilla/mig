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
	LocalMAC      []result `json:"localmac,omitempty"`
	LocalIP       []result `json:"localip,omitempty"`
	NeighborMAC   []result `json:"neighbormac,omitempty"`
	NeighborIP    []result `json:"neighborip,omitempty"`
	ConnectedIP   []result `json:"connectedip,omitempty"`
	ListeningPort []result `json:"listeningport,omitempty"`
	FoundAnything bool     `json:"foundanything"`
	Success       bool     `json:"success"`
	Errors        []string `json:"errors,omitempty"`
}

type result struct {
	Item          string `json:"item"`
	Found         bool   `json:"found"`
	LocalMACAddr  string `json:"localmacaddr,omitempty"`
	RemoteMACAddr string `json:"remotemacaddr,omitempty"`
	LocalAddr     string `json:"localaddr,omitempty"`
	LocalPort     string `json:"localport,omitempty"`
	RemoteAddr    string `json:"remoteaddr,omitempty"`
	RemotePort    string `json:"remoteport,omitempty"`
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
		return err
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
	for _, val := range r.Parameters.LocalMAC {
		var result result
		result.Item = val
		result.Found, result.LocalMACAddr, err = HasLocalMAC(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.LocalMAC = append(r.Results.LocalMAC, result)
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.NeighborMAC {
		var result result
		result.Item = val
		result.Found, result.RemoteMACAddr, result.RemoteAddr, err = HasSeenMac(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.NeighborMAC = append(r.Results.NeighborMAC, result)
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.LocalIP {
		var result result
		result.Item = val
		result.Found, result.LocalAddr, err = HasLocalIP(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.LocalIP = append(r.Results.LocalIP, result)
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.ConnectedIP {
		result, err := HasIPConnected(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.ConnectedIP = append(r.Results.ConnectedIP, result)
		if result.Found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.ListeningPort {
		result, err := HasListeningPort(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		r.Results.ListeningPort = append(r.Results.ListeningPort, result)
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

// HasLocalMac compares an input mac address with the mac addresses
// of the local interfaces, and returns found=true when found
func HasLocalMAC(macstr string) (found bool, addr string, err error) {
	found = false
	re, err := regexp.Compile("(?i)" + macstr)
	if err != nil {
		return found, addr, err
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return found, addr, err
	}
	for _, iface := range ifaces {
		if re.MatchString(iface.HardwareAddr.String()) {
			found = true
			addr = iface.HardwareAddr.String()
			return found, addr, err
		}
	}
	return found, addr, err
}

// HasLocalIP compares an input ip address with the ip addresses
// of the local interfaces, and returns found=true when found
func HasLocalIP(ipStr string) (found bool, addr string, err error) {
	found = false
	if strings.IndexAny(ipStr, "/") > 0 {
		_, ipnet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return found, addr, err
		}
		ifaceAddrs, err := net.InterfaceAddrs()
		if err != nil {
			return found, addr, err
		}
		for _, ifaceAddr := range ifaceAddrs {
			addr = strings.Split(ifaceAddr.String(), "/")[0]
			if ipnet.Contains(net.ParseIP(addr)) {
				found = true
				return found, addr, err
			}
		}
		return found, addr, err
	}
	ifaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return found, addr, err
	}
	for _, ifaceAddr := range ifaceAddrs {
		addr = strings.Split(ifaceAddr.String(), "/")[0]
		if ipStr == addr {
			found = true
			return found, addr, err
		}
	}
	return found, addr, err
}
