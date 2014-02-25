/* Look for IPs connected to a system

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package connected

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
)

type Parameters struct {
	Elements map[string][]string `json:"elements"`
}

func NewParameters() (p Parameters) {
	return
}

type Results struct {
	FoundAnything bool                               `json:"foundanything"`
	Elements      map[string]map[string]singleresult `json:"elements,omitempty"`
	Error         string                             `json:"error,omitempty"`
	Statistics    Statistics                         `json:"statistics,omitempty"`
}

func NewResults() *Results {
	return &Results{Elements: make(map[string]map[string]singleresult), FoundAnything: false}
}

// singleresult contains information on the result of a single test
type singleresult struct {
	Matchcount  int      `json:"matchcount,omitempty"`
	Connections []string `json:"connections,omitempty"`
}

func (p Parameters) Validate() (err error) {
	for _, values := range p.Elements {
		for _, value := range values {
			ipre := regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)
			if !ipre.MatchString(value) {
				return fmt.Errorf("Parameter '%s' isn't a valid IP", value)
			}
		}
	}
	return
}

var stats Statistics

type Statistics struct {
	Openfailed int `json:"openfailed"`
	Totalconn  int `json:"totalconn"`
}

type ConnectedIPs map[string][]string

func Run(Args []byte) string {
	var conns ConnectedIPs
	var errors string
	params := NewParameters()

	err := json.Unmarshal(Args, &params.Elements)
	if err != nil {
		panic(err)
	}

	err = params.Validate()
	if err != nil {
		panic(err)
	}

	switch runtime.GOOS {
	case "linux":
		conns = checkLinuxConnectedIPs(params)
	default:
		errors = fmt.Sprintf("'%s' isn't a supported OS", runtime.GOOS)
	}
	return buildResults(params, conns, errors)
}

// checkLinuxConnectedIPs checks the content of /proc/net/ip_conntrack
// and /proc/net/nf_conntrack
func checkLinuxConnectedIPs(params Parameters) ConnectedIPs {
	var list []string
	var conns ConnectedIPs
	for _, ips := range params.Elements {
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
	sources := []string{"/proc/net/ip_conntrack", "/proc/net/nf_conntrack"}
	for _, srcfile := range sources {
		// check those regexes against conntrack
		file, err := os.Open(srcfile)
		if err != nil {
			stats.Openfailed++
		}
		defer file.Close()
		conns = findInFile(file, list)
	}
	return conns
}

// iterate through a file and look for IP strings
func findInFile(fd *os.File, list []string) (conns ConnectedIPs) {
	conns = make(map[string][]string)
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		for _, ip := range list {
			if strings.Contains(scanner.Text(), ip) {
				conns[ip] = append(conns[ip], scanner.Text())
			}
		}
		stats.Totalconn++
	}
	return
}

// buildResults transforms the ConnectedIPs map into a Results
// map that is serialized in JSON and returned as a string
func buildResults(params Parameters, conns ConnectedIPs, errors string) string {
	results := NewResults()
	for ip, lines := range conns {
		// find mapping between IP and test name, and store the result
		for name, testips := range params.Elements {
			for _, testip := range testips {
				if testip == ip {
					if _, ok := results.Elements[name]; !ok {
						results.Elements[name] = map[string]singleresult{
							ip: singleresult{
								Matchcount:  len(lines),
								Connections: lines,
							},
						}
					} else {
						results.Elements[name][ip] = singleresult{
							Matchcount:  len(lines),
							Connections: lines,
						}
					}
				}
			}
		}
		results.FoundAnything = true
	}
	if errors != "" {
		results.Error = errors
	}
	results.Statistics = stats
	jsonOutput, err := json.Marshal(*results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
