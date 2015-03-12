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
	"github.com/streadway/amqp"
	"mig"
	"mig/event"
	"mig/workers"
	"os"
	"regexp"
	"time"
)

const workerName = "mozdef_asset"

type Config struct {
	Mq      workers.MqConf
	MozDef  workers.MqConf
	Logging mig.Logging
}

func main() {
	var (
		err  error
		conf Config
		hint hint
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
	mozdefChan, err := workers.InitMQ(conf.MozDef)
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
		err = publishHintToMozdef(hint, mozdefChan)
		if err != nil {
			mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("failed to publish to mozdef: %v", err)}.Err())
		}
		mig.ProcessLog(logctx, mig.Log{Desc: "published asset hint for agent '" + hint.Details.Name + "' to mozdef"}.Debug())
	}
	return
}

// A hint describes informations about an endpoint as gathered by MIG
// The format uses Mozdef's standard event format described in
// http://mozdef.readthedocs.org/en/latest/usage.html#json-format
// {
//   "timestamp": "2014-02-14T11:48:19.035762739-05:00",
//   "summary": "mig discovered host server1.net.example.com",
//   "hostname": "mig-worker1.use1.opsec.mozilla.com",
//   "severity": "INFO",
//   "category": "asset_hint",
//   "tags": [
//     "MIG",
//     "Asset"
//   ],
//   "details": {
//     "type": "host",
//     "name": "opsec1.private.phx1.mozilla.com",
//     "ipv4": [
//           "10.8.75.110/24"
//     ],
//     "ipv6": [
//       "fe80::250:56ff:febd:6850/64"
//     ],
//     "arch": "amd64",
//     "ident": "Red Hat Enterprise Linux Server release 6.6 (Santiago)",
//     "init": "upstart",
//     "isproxied": false,
//       "operator": "IT"
//   }
// }
type hint struct {
	Timestamp time.Time   `json:"timestamp"`
	Summary   string      `json:"summary"`
	Hostname  string      `json:"hostname"`
	Severity  string      `json:"severity"`
	Category  string      `json:"category"`
	Tags      []string    `json:"tags"`
	Details   hintDetails `json:"details"`
}

type hintDetails struct {
	Type      string   `json:"type"`
	Name      string   `json:"name"`
	IPv4      []string `json:"ipv4"`
	IPv6      []string `json:"ipv6"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	Ident     string   `json:"ident"`
	Init      string   `json:"init"`
	IsProxied bool     `json:"isproxied"`
	Operator  string   `json:"operator"`
}

func makeHintFromAgent(agt mig.Agent) (hint hint, err error) {
	hint.Timestamp = time.Now().UTC()
	hint.Summary = "mig discovered host " + agt.Name
	hint.Hostname, err = os.Hostname()
	if err != nil {
		return
	}
	hint.Severity = "INFO"
	hint.Category = "asset_hint"
	hint.Tags = append(hint.Tags, "mig")
	hint.Tags = append(hint.Tags, "asset")
	hint.Details.Type = "host"
	hint.Details.Name = agt.Name
	reipv4 := regexp.MustCompile(`([0-9]{1,3}\.){3}([0-9]{1,3})`)
	for _, ip := range agt.Env.Addresses {
		if reipv4.MatchString(ip) {
			hint.Details.IPv4 = append(hint.Details.IPv4, ip)
		} else {
			hint.Details.IPv6 = append(hint.Details.IPv6, ip)
		}
	}
	hint.Details.OS = agt.Env.OS
	hint.Details.Arch = agt.Env.Arch
	hint.Details.Ident = agt.Env.Ident
	hint.Details.Init = agt.Env.Init
	hint.Details.IsProxied = agt.Env.IsProxied
	if _, ok := agt.Tags.(map[string]interface{})["operator"]; ok {
		hint.Details.Operator = agt.Tags.(map[string]interface{})["operator"].(string)
	}
	return
}

func publishHintToMozdef(hint hint, mozdefChan *amqp.Channel) (err error) {
	data, err := json.Marshal(hint)
	if err != nil {
		return
	}
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/plain",
		Expiration:   "6000000", // events expire after 100 minutes if not consumed
		Body:         data,
	}
	err = mozdefChan.Publish("mozdef", "event", false, false, msg)
	return
}
