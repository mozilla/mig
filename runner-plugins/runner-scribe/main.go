// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

// This plugin is intended to process the results of scribe actions and
// dispatch those results to MozDef. The plugin should be used with
// mig-runner and not run directly.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jvehent/gozdef"
	"gopkg.in/gcfg.v1"
	"mig.ninja/mig"
	scribemod "mig.ninja/mig/modules/scribe"
)

// Configuration structure for runner-scribe
//
// URL and Source are mandatory settings
//
// Note that if the returned scribe event contains a "category" tag, this
// tag will be appended to the source identifier like "<source>-<tag>"
type Config struct {
	MozDef struct {
		URL    string // URL to post events to MozDef
		Source string // Source identifier for vulnerability events
	}
	Vmintgr struct {
		// Location of vmintgr wrapper executable
		Bin string
	}
}

const configPath string = "/etc/mig/runner-scribe.conf"

var conf Config

func main() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", e)
			os.Exit(1)
		}
	}()

	var (
		err     error
		results mig.RunnerResult
	)

	err = gcfg.ReadFileInto(&conf, configPath)
	if err != nil {
		panic(err)
	}

	buf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, &results)
	if err != nil {
		panic(err)
	}
	for _, x := range results.Commands {
		items, err := makeVulnerability(x)
		if err != nil {
			panic(err)
		}
		for _, y := range items {
			err = sendVulnerability(y)
			if err != nil {
				panic(err)
			}
		}
	}
}

// Send a vulnerability event to MozDef
func sendVulnerability(item gozdef.VulnEvent) (err error) {
	ac := gozdef.ApiConf{Url: conf.MozDef.URL}
	pub, err := gozdef.InitApi(ac)
	if err != nil {
		return
	}
	err = pub.Send(item)
	return
}

// Convert a MIG command result into a MozDef compliance event
func makeVulnerability(cmd mig.Command) (items []gozdef.VulnEvent, err error) {
	var ve gozdef.VulnEvent
	ve.UTCTimestamp = time.Now().UTC()
	ve.Description = "MIG vulnerability identification"
	ve.OS = cmd.Agent.Env.OS
	ve.Asset.AssetID = int(cmd.Agent.ID)
	ve.Asset.Hostname = cmd.Agent.Name
	for _, x := range cmd.Agent.Env.Addresses {
		if !strings.Contains(x, ".") {
			continue
		}
		ipt, _, errt := net.ParseCIDR(x)
		if errt != nil {
			continue
		}
		ve.Asset.IPv4 = ipt.String()
		break
	}
	if cmd.Agent.Tags != nil {
		operator := ""
		if _, ok := cmd.Agent.Tags.(map[string]interface{})["operator"]; ok {
			operator = cmd.Agent.Tags.(map[string]interface{})["operator"].(string)
		}
		team := getTeam(cmd.Agent, conf)
		ve.Asset.Operator = operator
		ve.Asset.Autogroup = team
	}
	for _, result := range cmd.Results {
		var el scribemod.ScribeElements
		err = result.GetElements(&el)
		if err != nil {
			return items, err
		}
		for _, x := range el.Results {
			ve.SourceName = conf.MozDef.Source
			if !x.MasterResult {
				continue
			}
			ve.Vuln.Title = x.TestID
			ve.Vuln.Description = x.Description
			ve.Vuln.Status = "open"
			ve.Vuln.VulnID = x.TestID
			for _, y := range x.Tags {
				if y.Key == "cve" {
					ve.Vuln.CVE = strings.Split(y.Value, ",")
				} else if y.Key == "cvss" {
					var cvss float64
					cvss, err = strconv.ParseFloat(y.Value, 64)
					if err != nil {
						continue
					}
					ve.Vuln.CVSS = cvss
				} else if y.Key == "category" {
					ve.SourceName += "-" + y.Value
				}
			}
			// Set the impact label based on the CVSS score
			if ve.Vuln.CVSS >= 9.0 {
				ve.Vuln.ImpactLabel = "maximum"
			} else if ve.Vuln.CVSS >= 7.0 {
				ve.Vuln.ImpactLabel = "high"
			} else {
				ve.Vuln.ImpactLabel = "mediumlow"
			}
			// Use the identifier for each true subresult in the
			// test as a proof section
			for _, y := range x.Results {
				if y.Result {
					ve.Vuln.Proof = "Object " + y.Identifier + " is vulnerable"
					items = append(items, ve)
				}
			}
		}
	}
	return
}

type VmintgrOutput struct {
	Host string `json:"host"`
	Ip   string `json:"ip"`
	Team string `json:"team"`
}

func getTeam(agt mig.Agent, conf Config) string {
	var vmout VmintgrOutput
	if conf.Vmintgr.Bin == "" {
		return ""
	}
	for i := 0; i <= len(agt.Env.Addresses); i++ {
		query := "host:" + agt.Name
		if i > 0 {
			query = "ip:" + agt.Env.Addresses[i-1]
		}
		out, err := exec.Command(conf.Vmintgr.Bin, query).Output()
		if err != nil {
			return ""
		}
		err = json.Unmarshal(out, &vmout)
		if err != nil {
			return ""
		}
		if vmout.Team != "default" {
			return vmout.Team
		}
	}
	return "default"
}
