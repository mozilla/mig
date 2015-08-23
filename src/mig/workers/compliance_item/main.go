// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"gopkg.in/gcfg.v1"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jvehent/gozdef"
	"mig"
	"mig/modules"
	"mig/modules/file"
	"mig/workers"
	"os"
	"os/exec"
	"regexp"
	"time"
)

const workerName = "compliance_item"

type Config struct {
	Mq      workers.MqConf
	MozDef  gozdef.MqConf
	Logging mig.Logging
	API     struct {
		Host string
	}
	Vmintgr struct {
		Bin string
	}
}

func main() {
	var (
		err   error
		conf  Config
		items []gozdef.ComplianceItem
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s - a worker that transform commands results into compliance items and publishes them to mozdef\n", os.Args[0])
		flag.PrintDefaults()
	}
	var configPath = flag.String("c", "/etc/mig/compliance-item-worker.cfg", "Load configuration from file")
	flag.Parse()
	err = gcfg.ReadFileInto(&conf, *configPath)
	if err != nil {
		panic(err)
	}

	logctx, err := mig.InitLogger(conf.Logging, workerName)
	if err != nil {
		panic(err)
	}

	// bind to the MIG even queue
	workerQueue := "migevent.worker." + workerName
	consumerChan, err := workers.InitMqWithConsumer(conf.Mq, workerQueue, mig.Ev_Q_Cmd_Res)
	if err != nil {
		panic(err)
	}

	// bind to the mozdef relay exchange
	gp, err := gozdef.InitAmqp(conf.MozDef)
	if err != nil {
		panic(err)
	}

	mig.ProcessLog(logctx, mig.Log{Desc: "worker started, consuming queue " + workerQueue + " from key " + mig.Ev_Q_Cmd_Res})
	tFamRe := regexp.MustCompile("(?i)^compliance$")
	for event := range consumerChan {
		var cmd mig.Command
		err = json.Unmarshal(event.Body, &cmd)
		if err != nil {
			mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("invalid command: %v", err)}.Err())
		}
		// discard actions that aren't threat.family=compliance
		if !tFamRe.MatchString(cmd.Action.Threat.Family) {
			continue
		}
		items, err = makeComplianceItem(cmd, conf)
		if err != nil {
			mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("failed to make compliance items: %v", err)}.Err())
		}
		for _, item := range items {
			// create a new event and set values in the fields
			ev, err := gozdef.NewEvent()
			if err != nil {
				mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("failed to make new mozdef event: %v", err)}.Err())
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
			err = gp.Send(ev)
			if err != nil {
				mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("failed to publish to mozdef: %v", err)}.Err())
				// if publication to mozdef fails, crash the worker. systemd/upstart will restart a new one
				panic(err)
			}
		}
		mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("published %d items from command %.0f to mozdef", len(items), cmd.ID)}.Info())
	}
	return
}

func makeComplianceItem(cmd mig.Command, conf Config) (items []gozdef.ComplianceItem, err error) {
	var ci gozdef.ComplianceItem
	ci.Utctimestamp = time.Now().UTC().Format(time.RFC3339Nano)
	ci.Target = cmd.Agent.Name
	ci.Policy.Name = cmd.Action.Threat.Type
	ci.Policy.URL = cmd.Action.Description.URL
	ci.Policy.Level = cmd.Action.Threat.Level
	ci.Check.Ref = cmd.Action.Threat.Ref
	ci.Check.Description = cmd.Action.Name
	ci.Link = fmt.Sprintf("%s/command?commandid=%.0f", conf.API.Host, cmd.ID)
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
					for _, v := range mf.Search.SHA256 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha256='%s'", v)
					}
					for _, v := range mf.Search.SHA384 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha384='%s'", v)
					}
					for _, v := range mf.Search.SHA512 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha512='%s'", v)
					}
					for _, v := range mf.Search.SHA3_224 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha3_224='%s'", v)
					}
					for _, v := range mf.Search.SHA3_256 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha3_256='%s'", v)
					}
					for _, v := range mf.Search.SHA3_384 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha3_384='%s'", v)
					}
					for _, v := range mf.Search.SHA3_512 {
						if len(ci.Check.Test.Value) > 0 {
							ci.Check.Test.Value += " and "
						}
						ci.Check.Test.Value += fmt.Sprintf("sha3_512='%s'", v)
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
