// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/gcfg.v1"
	"mig.ninja/mig"
	"mig.ninja/mig/workers"
)

const workerName = "agent_verif"

type Config struct {
	Mq      workers.MqConf
	Logging mig.Logging
}

func main() {
	var (
		err  error
		conf Config
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s - a worker verifying agents that fail to authenticate\n", os.Args[0])
		flag.PrintDefaults()
	}
	var configPath = flag.String("c", "/etc/mig/agent-verif-worker.cfg", "Load configuration from file")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()
	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	err = gcfg.ReadFileInto(&conf, *configPath)
	if err != nil {
		panic(err)
	}
	logctx, err := mig.InitLogger(conf.Logging, workerName)
	if err != nil {
		panic(err)
	}
	// set a binding to route events from mig.Ev_Q_Agt_Auth_Fail into the queue named after the worker
	// and return a channel that consumes the queue
	workerQueue := "migevent.worker." + workerName
	consumerChan, err := workers.InitMqWithConsumer(conf.Mq, workerQueue, mig.Ev_Q_Agt_Auth_Fail)
	if err != nil {
		panic(err)
	}
	fmt.Println("started worker", workerName, "consuming queue", workerQueue, "from key", mig.Ev_Q_Agt_Auth_Fail)
	for event := range consumerChan {
		mig.ProcessLog(logctx, mig.Log{Desc: fmt.Sprintf("unverified agent '%s'", event.Body)})
	}
	return
}
