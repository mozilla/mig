// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// Connected is a module that looks for IP addresses currently connected
// to the system. It does so by reading conntrack data on Linux. MacOS and
// Windows are not yet implemented.
package connected

import (
	"bufio"
	"encoding/json"
	"fmt"
	"mig"
	"os"
	"regexp"
	"runtime"
	"strings"
)

func init() {
	mig.RegisterModule("connected", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters params
	Results    results
	conns      params
}

type params map[string][]string

type results struct {
	FoundAnything bool                               `json:"foundanything"`
	Elements      map[string]map[string]singleresult `json:"elements,omitempty"`
	Errors        []string                           `json:"errors,omitempty"`
	Statistics    statistics                         `json:"statistics,omitempty"`
}

type statistics struct {
	OpenFailed int `json:"openfailed"`
	TotalConn  int `json:"totalconn"`
}

// singleresult contains information on the result of a single test
type singleresult struct {
	MatchCount  int      `json:"matchcount,omitempty"`
	Connections []string `json:"connections,omitempty"`
}

func newResults() *results {
	return &results{Elements: make(map[string]map[string]singleresult), FoundAnything: false}
}

func (r Runner) Run(args []byte) string {
	err := json.Unmarshal(args, &r.Parameters)
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}

	err = r.ValidateParameters()
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}

	switch runtime.GOOS {
	case "linux":
		r.conns = r.checkLinuxConnectedIPs()
	default:
		panic("OS not supported")
	}
	return r.buildResults()
}

// Validate ensures that the parameters contain valid IPv4 addresses
func (r Runner) ValidateParameters() (err error) {
	for _, values := range r.Parameters {
		for _, value := range values {
			ipre := regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)
			if !ipre.MatchString(value) {
				return fmt.Errorf("Parameter '%s' isn't a valid IP", value)
			}
		}
	}
	return
}

// checkLinuxConnectedIPs checks the content of /proc/net/ip_conntrack
// and /proc/net/nf_conntrack
func (r Runner) checkLinuxConnectedIPs() map[string][]string {
	var list []string
	connections := make(map[string][]string)
	for _, ips := range r.Parameters {
		for _, newIP := range ips {
			addit := true
			for _, ip := range list {
				if newIP == ip {
					addit = false
				}
			}
			if addit {
				list = append(list, newIP)
			}
		}
	}
	// TODO: read connection data from /proc/net/{tcp,udp} instead
	sources := []string{"/proc/net/ip_conntrack", "/proc/net/nf_conntrack"}
	for _, srcfile := range sources {
		// check those regexes against conntrack
		fd, err := os.Open(srcfile)
		if err != nil {
			r.Results.Statistics.OpenFailed++
		}
		defer fd.Close()
		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				panic(err)
			}
			for _, ip := range list {
				if strings.Contains(scanner.Text(), ip) {
					connections[ip] = append(connections[ip], scanner.Text())
				}
			}
			r.Results.Statistics.TotalConn++
		}
	}
	return connections
}

// buildResults transforms the connectedIPs map into a Results
// map that is serialized in JSON and returned as a string
func (r Runner) buildResults() string {
	results := newResults()
	for ip, lines := range r.conns {
		// find mapping between IP and test name, and store the result
		for name, testips := range r.Parameters {
			for _, testip := range testips {
				if testip == ip {
					if _, ok := results.Elements[name]; !ok {
						results.Elements[name] = map[string]singleresult{
							ip: singleresult{
								MatchCount:  len(lines),
								Connections: lines,
							},
						}
					} else {
						results.Elements[name][ip] = singleresult{
							MatchCount:  len(lines),
							Connections: lines,
						}
					}
				}
			}
		}
		results.FoundAnything = true
	}
	jsonOutput, err := json.Marshal(*results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
