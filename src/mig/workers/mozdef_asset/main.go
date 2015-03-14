// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"code.google.com/p/gcfg"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jvehent/gozdef"
	"mig"
	"mig/event"
	"mig/workers"
	"os"
	"regexp"
)

const workerName = "mozdef_asset"

type Config struct {
	Mq      workers.MqConf
	MozDef  gozdef.MqConf
	Logging mig.Logging
}

func main() {
	var (
		err  error
		conf Config
		hint gozdef.HostAssetHint
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s - a worker that listens to new endpoints and sends them as assets to mozdef\n", os.Args[0])
		flag.PrintDefaults()
	}
	var configPath = flag.String("c", "/etc/mig/mozdef_asset_worker.cfg", "Load configuration from file")
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
	consumerChan, err := workers.InitMqWithConsumer(conf.Mq, workerQueue, event.Q_Agt_New)
	if err != nil {
		panic(err)
	}

	// bind to the mozdef relay exchange
	gp, err := gozdef.InitAmqp(conf.MozDef)
	if err != nil {
		panic(err)
	}

	mig.ProcessLog(logctx, mig.Log{Desc: "worker started, consuming queue " + workerQueue + " from key " + event.Q_Agt_New})
	for event := range consumerChan {
		var agt mig.Agent
		err = json.Unmarshal(event.Body, &agt)
		if err != nil {
			mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("invalid agent description: %v", err)}.Err())
		}
		hint, err = makeHintFromAgent(agt)
		if err != nil {
			mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("failed to build asset hint: %v", err)}.Err())
		}
		err = publishHintToMozdef(hint, gp)
		if err != nil {
			mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("failed to publish to mozdef: %v", err)}.Err())
		}
		mig.ProcessLog(logctx, mig.Log{Desc: "published asset hint for agent '" + hint.Name + "' to mozdef"}.Info())
	}
	return
}

func makeHintFromAgent(agt mig.Agent) (hint gozdef.HostAssetHint, err error) {
	hint.Type = "host"
	hint.Name = agt.Name
	reipv4 := regexp.MustCompile(`([0-9]{1,3}\.){3}([0-9]{1,3})`)
	for _, ip := range agt.Env.Addresses {
		if reipv4.MatchString(ip) {
			hint.IPv4 = append(hint.IPv4, ip)
		} else {
			hint.IPv6 = append(hint.IPv6, ip)
		}
	}
	hint.OS = agt.Env.OS
	hint.Arch = agt.Env.Arch
	hint.Ident = agt.Env.Ident
	hint.Init = agt.Env.Init
	hint.IsProxied = agt.Env.IsProxied
	if _, ok := agt.Tags.(map[string]interface{})["operator"]; ok {
		hint.Operator = agt.Tags.(map[string]interface{})["operator"].(string)
	}
	return
}

func publishHintToMozdef(hint gozdef.HostAssetHint, gp gozdef.Publisher) error {
	// create a new event and set values in the fields
	ev, err := gozdef.NewEvent()
	if err != nil {
		return err
	}
	ev.Category = "asset_hint"
	ev.Source = "mig"
	ev.Summary = fmt.Sprintf("mig discovered endpoint %s", hint.Name)
	ev.Tags = append(ev.Tags, "mig")
	ev.Tags = append(ev.Tags, "asset")
	ev.Info()
	ev.Details = hint
	return gp.Send(ev)
}
