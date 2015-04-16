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
	"time"
)

func init() {
	mig.RegisterModule("netstat", func() interface{} {
		return new(Runner)
	}, false)
}

type Runner struct {
	Parameters params
	Results    mig.ModuleResult
}

type params struct {
	LocalMAC      []string `json:"localmac,omitempty"`
	LocalIP       []string `json:"localip,omitempty"`
	NeighborMAC   []string `json:"neighbormac,omitempty"`
	NeighborIP    []string `json:"neighborip,omitempty"`
	ConnectedIP   []string `json:"connectedip,omitempty"`
	ListeningPort []string `json:"listeningport,omitempty"`
}

type elements struct {
	LocalMAC      map[string][]element `json:"localmac,omitempty"`
	LocalIP       map[string][]element `json:"localip,omitempty"`
	NeighborMAC   map[string][]element `json:"neighbormac,omitempty"`
	NeighborIP    map[string][]element `json:"neighborip,omitempty"`
	ConnectedIP   map[string][]element `json:"connectedip,omitempty"`
	ListeningPort map[string][]element `json:"listeningport,omitempty"`
}

type element struct {
	LocalMACAddr  string  `json:"localmacaddr,omitempty"`
	RemoteMACAddr string  `json:"remotemacaddr,omitempty"`
	LocalAddr     string  `json:"localaddr,omitempty"`
	LocalPort     float64 `json:"localport,omitempty"`
	RemoteAddr    string  `json:"remoteaddr,omitempty"`
	RemotePort    float64 `json:"remoteport,omitempty"`
}

func newElements() *elements {
	var e elements
	e.LocalMAC = make(map[string][]element)
	e.LocalIP = make(map[string][]element)
	e.NeighborMAC = make(map[string][]element)
	e.NeighborIP = make(map[string][]element)
	e.ConnectedIP = make(map[string][]element)
	e.ListeningPort = make(map[string][]element)
	return &e
}

// stats is a global variable
var stats statistics

type statistics struct {
	Examined  float64 `json:"examined"`
	Exectime  string  `json:"exectime"`
	Totalhits float64 `json:"totalhits"`
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
	t0 := time.Now()

	err := json.Unmarshal(args, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	els := *newElements()

	for _, val := range r.Parameters.LocalMAC {
		found, el, err := HasLocalMAC(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		els.LocalMAC[val] = el
		if found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.NeighborMAC {
		found, el, err := HasSeenMac(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		els.NeighborMAC[val] = el
		if found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.LocalIP {
		found, el, err := HasLocalIP(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		els.LocalIP[val] = el
		if found {
			r.Results.FoundAnything = true
		}
	}
	for _, val := range r.Parameters.ConnectedIP {
		found, el, err := HasIPConnected(val)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		els.ConnectedIP[val] = el
		if found {
			r.Results.FoundAnything = true
		}
	}
	for _, port := range r.Parameters.ListeningPort {
		found, el, err := HasListeningPort(port)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
		els.ListeningPort[port] = el
		if found {
			r.Results.FoundAnything = true
		}
	}
	r.Results.Elements = els
	// calculate execution time
	t1 := time.Now()
	stats.Exectime = t1.Sub(t0).String()
	r.Results.Statistics = stats

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
		stats.Examined++
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
			stats.Examined++
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
		stats.Examined++
	}
	return
}

func (r Runner) PrintResults(rawResults []byte, foundOnly bool) (prints []string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PrintResults() -> %v", e)
		}
	}()
	var modres mig.ModuleResult
	err = json.Unmarshal(rawResults, &modres)
	if err != nil {
		panic(err)
	}
	buf, err := json.Marshal(modres.Elements)
	if err != nil {
		panic(err)
	}
	els := *newElements()
	err = json.Unmarshal(buf, &els)
	if err != nil {
		panic(err)
	}
	for val, res := range els.LocalMAC {
		if foundOnly && len(res) < 1 {
			continue
		}
		for _, el := range res {
			resStr := fmt.Sprintf("found local mac %s for netstat localmac:'%s'", el.LocalMACAddr, val)
			prints = append(prints, resStr)
		}
	}
	for val, res := range els.NeighborMAC {
		if foundOnly && len(res) < 1 {
			continue
		}
		for _, el := range res {
			resStr := fmt.Sprintf("found neighbor mac %s %s for netstat neighbormac:'%s'",
				el.RemoteMACAddr, el.RemoteAddr, val)
			prints = append(prints, resStr)
		}
		if len(res) == 0 {
			resStr := fmt.Sprintf("did not find anything for netstat neighbormac:'%s'", val)
			prints = append(prints, resStr)
		}
	}
	for val, res := range els.LocalIP {
		if foundOnly && len(res) < 1 {
			continue
		}
		for _, el := range res {
			resStr := fmt.Sprintf("found local ip %s for netstat localip:'%s'", el.LocalAddr, val)
			prints = append(prints, resStr)
		}
		if len(res) == 0 {
			resStr := fmt.Sprintf("did not find anything for netstat localip:'%s'", val)
			prints = append(prints, resStr)
		}
	}
	for val, res := range els.ConnectedIP {
		if foundOnly && len(res) < 1 {
			continue
		}
		for _, el := range res {
			resStr := fmt.Sprintf("found connected tuple %s:%.0f with local tuple %s:%.0f for netstat connectedip:'%s'",
				el.RemoteAddr, el.RemotePort, el.LocalAddr, el.LocalPort, val)
			prints = append(prints, resStr)
		}
		if len(res) == 0 {
			resStr := fmt.Sprintf("did not find anything for netstat connectedip:'%s'", val)
			prints = append(prints, resStr)
		}
	}
	for val, res := range els.ListeningPort {
		if foundOnly && len(res) < 1 {
			continue
		}
		for _, el := range res {
			resStr := fmt.Sprintf("found listening port %.0f for netstat listeningport:'%s'", el.LocalPort, val)
			prints = append(prints, resStr)
		}
		if len(res) == 0 {
			resStr := fmt.Sprintf("did not find anything for netstat listeningport:'%s'", val)
			prints = append(prints, resStr)
		}
	}
	if !foundOnly {
		buf, err := json.Marshal(modres.Statistics)
		if err != nil {
			panic(err)
		}
		var stats statistics
		err = json.Unmarshal(buf, &stats)
		if err != nil {
			panic(err)
		}
		resStr := fmt.Sprintf("Statistics: total hits %.0f examined %.0f items exectime %s",
			stats.Totalhits, stats.Examined, stats.Exectime)
		prints = append(prints, resStr)
	}
	return
}
