// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

// runner-scribe is a mig-runner plugin that processes results coming from automated
// actions and forwards the results as vulnerability events to MozDef
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/jvehent/gozdef"
	"gopkg.in/gcfg.v1"
	"mig.ninja/mig"
	scribemod "mig.ninja/mig/modules/scribe"
)

// config represents the configuration used by runner-scribe, and is read in on
// initialization
//
// URL and Source are mandatory settings
type config struct {
	MozDef struct {
		URL    string // URL to post events to MozDef
		Source string // Source identifier for vulnerability events
	}
}

const configPath string = "/etc/mig/runner-scribe.conf"

var conf config

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
	var items []gozdef.VulnEvent
	for _, x := range results.Commands {
		// Process the incoming commands, under normal circumstances we will have one
		// returned command per host. However, this function can handle cases where
		// more than one command applies to a given host. If data for a host already
		// exists in items, makeVulnerability should attempt to append this data to
		// the host rather than add a new item.
		var err error
		items, err = makeVulnerability(items, x)
		if err != nil {
			panic(err)
		}
	}
	for _, y := range items {
		err = sendVulnerability(y)
		if err != nil {
			panic(err)
		}
	}
}

func sendVulnerability(item gozdef.VulnEvent) (err error) {
	ac := gozdef.APIConf{URL: conf.MozDef.URL}
	pub, err := gozdef.InitAPI(ac)
	if err != nil {
		return
	}
	err = pub.Send(item)
	return
}

func makeVulnerability(initems []gozdef.VulnEvent, cmd mig.Command) (items []gozdef.VulnEvent, err error) {
	var (
		itemptr                       *gozdef.VulnEvent
		assethostname, assetipaddress string
		insertNew                     bool
	)
	items = initems

	assethostname = cmd.Agent.Name
	for _, x := range cmd.Agent.Env.Addresses {
		if !strings.Contains(x, ".") {
			continue
		}
		ipt, _, err := net.ParseCIDR(x)
		if err != nil {
			continue
		}
		assetipaddress = ipt.String()
		break
	}

	// First, see if we can locate a preexisting item for this asset
	for i := range items {
		if items[i].Asset.Hostname == assethostname &&
			items[i].Asset.IPAddress == assetipaddress {
			itemptr = &items[i]
			break
		}
	}
	if itemptr == nil {
		// Initialize a new event we will insert later
		newevent, err := gozdef.NewVulnEvent()
		if err != nil {
			return items, err
		}
		newevent.Description = "MIG vulnerability identification"
		newevent.Zone = "mig"
		newevent.Asset.Hostname = assethostname
		newevent.Asset.IPAddress = assetipaddress
		newevent.Asset.OS = cmd.Agent.Env.OS
		if len(cmd.Agent.Tags) != 0 {
			if _, ok := cmd.Agent.Tags["operator"]; ok {
				newevent.Asset.Owner.Operator = cmd.Agent.Tags["operator"]
			}
		}
		// Apply a v2bkey to the event. This should be set using integration
		// with service-map, but here for now we just apply it based on the operator
		// and team values which may be present in the event.
		if newevent.Asset.Owner.V2Bkey == "" {
			if newevent.Asset.Owner.Operator != "" {
				newevent.Asset.Owner.V2Bkey = newevent.Asset.Owner.Operator
			}
			if newevent.Asset.Owner.Team != "" {
				newevent.Asset.Owner.V2Bkey += "-" + newevent.Asset.Owner.Team
			}
		}
		// Always set credentialed checks here
		newevent.CredentialedChecks = true
		insertNew = true
		itemptr = &newevent
	}

	for _, result := range cmd.Results {
		var el scribemod.ScribeElements
		err = result.GetElements(&el)
		if err != nil {
			return items, err
		}
		for _, x := range el.Results {
			itemptr.SourceName = conf.MozDef.Source
			if !x.MasterResult {
				// Result was false (vulnerability did not match)
				continue
			}
			newve := gozdef.VulnVuln{}
			newve.Name = x.TestName
			for _, y := range x.Tags {
				if y.Key == "severity" {
					newve.Risk = y.Value
				} else if y.Key == "link" {
					newve.Link = y.Value
				}
			}
			// If no risk value is set on the vulnerability, we just treat this as
			// informational and ignore it. This will apply to things like the result
			// from platform dependency checks associated with real vulnerability checks.
			if newve.Risk == "" {
				continue
			}
			newve.Risk = normalizeRisk(newve.Risk)
			newve.LikelihoodIndicator = likelihoodFromRisk(newve.Risk)
			if newve.CVSS == "" {
				newve.CVSS = cvssFromRisk(newve.Risk)
			}
			// Use the identifier for each true subresult in the
			// test as a proof section
			for _, y := range x.Results {
				if y.Result {
					newve.Packages = append(newve.Packages, y.Identifier)
				}
			}
			itemptr.Vuln = append(itemptr.Vuln, newve)
		}
	}
	if insertNew {
		items = append(items, *itemptr)
	}
	return
}

// cvssFromRisk returns a synthesized CVSS score as a string given a risk label
func cvssFromRisk(risk string) string {
	switch risk {
	case "critical":
		return "10.0"
	case "high":
		return "8.0"
	case "medium":
		return "5.0"
	case "low":
		return "2.5"
	}
	return "0.0"
}

// likelihoodFromRisk returns a likelihood indicator value given a risk label
func likelihoodFromRisk(risk string) string {
	switch risk {
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	case "critical":
		return "maximum"
	}
	return "unknown"
}

// normalizeRisk converts known risk labels into a standardized form, if we can't identify
// the value we just return it as is
func normalizeRisk(in string) string {
	switch strings.ToLower(in) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	case "critical":
		return "critical"
	}
	return in
}
