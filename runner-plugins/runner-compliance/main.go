// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

// This is a plugin for use with mig-runner that processes incoming compliance
// results from MIG and sends them to MozDef
//
// It replaces the historical compliance item worker process. This program
// generally is not run directly, but instead should be called by mig-runner
// as a plugin.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/jvehent/gozdef"
	"gopkg.in/gcfg.v1"
	"mig.ninja/mig"
	"mig.ninja/mig/modules"
	"mig.ninja/mig/modules/file"
)

type Config struct {
	MIG struct {
		// The MIG API URL
		API string
	}
	MozDef struct {
		// URL to post events to MozDef
		URL string
	}
	Vmintgr struct {
		// Location of vmintgr wrapper executable
		Bin string
	}
}

const configPath string = "/etc/mig/runner-compliance.conf"

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
		items, err := makeComplianceItem(x)
		if err != nil {
			panic(err)
		}
		for _, y := range items {
			err = sendItem(y)
			if err != nil {
				panic(err)
			}
		}
	}
}

// Send a compliance item to MozDef
func sendItem(item gozdef.ComplianceItem) (err error) {
	ev, err := gozdef.NewEvent()
	if err != nil {
		return
	}
	ev.Category = "complianceitems"
	ev.Source = "mig"
	cverb := "fails"
	if item.Compliance {
		cverb = "passes"
	}
	ev.Summary = fmt.Sprintf("%s %s compliance with %s", item.Target, cverb, item.Check.Ref)
	ev.Tags = append(ev.Tags, "mig")
	ev.Tags = append(ev.Tags, "compliance")
	ev.Info()
	ev.Details = item
	ac := gozdef.ApiConf{Url: conf.MozDef.URL}
	pub, err := gozdef.InitApi(ac)
	if err != nil {
		return
	}
	err = pub.Send(ev)
	return
}

// Convert a MIG command result into a MozDef compliance event
func makeComplianceItem(cmd mig.Command) (items []gozdef.ComplianceItem, err error) {
	var ci gozdef.ComplianceItem
	ci.Utctimestamp = time.Now().UTC().Format(time.RFC3339Nano)
	ci.Target = cmd.Agent.Name
	ci.Policy.Name = cmd.Action.Threat.Type
	ci.Policy.URL = cmd.Action.Description.URL
	ci.Policy.Level = cmd.Action.Threat.Level
	ci.Check.Ref = cmd.Action.Threat.Ref
	ci.Check.Description = cmd.Action.Name
	ci.Link = fmt.Sprintf("%s/command?commandid=%.0f", conf.MIG.API, cmd.ID)
	if cmd.Agent.Tags != nil {
		operator := ""
		if _, ok := cmd.Agent.Tags.(map[string]interface{})["operator"]; ok {
			operator = cmd.Agent.Tags.(map[string]interface{})["operator"].(string)
		}
		team := getTeam(cmd.Agent, conf)
		ci.Tags = struct {
			Operator string `json:"operator"`
			Team     string `json:"team"`
		}{
			Operator: operator,
			Team:     team,
		}
	}
	for i, result := range cmd.Results {
		buf, err := json.Marshal(result)
		if err != nil {
			return items, err
		}
		if i > (len(cmd.Action.Operations) - 1) {
			// skip this entry if the lookup fails
			continue
		}
		switch cmd.Action.Operations[i].Module {
		case "file":
			var r modules.Result
			var el file.SearchResults
			err = json.Unmarshal(buf, &r)
			if err != nil {
				return items, err
			}
			err = r.GetElements(&el)
			if err != nil {
				return items, err
			}
			for label, sr := range el {
				for _, mf := range sr {
					ci.Check.Location = mf.File
					ci.Check.Name = label
					ci.Check.Test.Type = "file"
					ci.Check.Test.Value = ""
					for _, v := range mf.Search.Names {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("name='%s'", v)
					}
					for _, v := range mf.Search.Sizes {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("size='%s'", v)
					}
					for _, v := range mf.Search.Modes {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("mode='%s'", v)
					}
					for _, v := range mf.Search.Mtimes {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("mtime='%s'", v)
					}
					for _, v := range mf.Search.Contents {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("content='%s'", v)
					}
					for _, v := range mf.Search.MD5 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("md5='%s'", v)
					}
					for _, v := range mf.Search.SHA1 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha1='%s'", v)
					}
					for _, v := range mf.Search.SHA2 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha2='%s'", v)
					}
					for _, v := range mf.Search.SHA3 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha3='%s'", v)
					}
					if mf.File == "" {
						for i, p := range mf.Search.Paths {
							if i > 0 {
								ci.Check.Location += ", "
							}
							ci.Check.Location += p
						}
						ci.Compliance = false
					} else {
						ci.Compliance = true
					}
					items = append(items, ci)
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
