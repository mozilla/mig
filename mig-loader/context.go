// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"fmt"
	"github.com/jvehent/service-go"
	"mig.ninja/mig"
	"runtime"
)

const runInterval = 7200

type Context struct {
	AgentIdentifier mig.Agent
	LoaderKey       string

	Channels struct {
		Log chan mig.Log
	}
	Logging mig.Logging
}

func getLoggingConf() (ret mig.Logging, err error) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		return getLoggingConfPosix()
	}
	err = fmt.Errorf("unable to obtain logging configuration for platform")
	return
}

func getLoggingConfPosix() (ret mig.Logging, err error) {
	ret.Mode = "file"
	ret.Level = "info"
	ret.File = "/var/log/mig-loader.log"
	ret.MaxFileSize = 10485760
	return
}

func serviceDeployInterval() error {
	svc, err := service.NewService("mig-loader", "MIG Loader", "Mozilla InvestiGator Loader")
	if err != nil {
		return err
	}
	err = svc.IntervalMode(runInterval)
	if err != nil {
		return err
	}
	// Ignore errors from stop and remove, as it may not be installed yet
	svc.Stop()
	svc.Remove()
	err = svc.Install()
	if err != nil {
		return err
	}
	err = svc.Start()
	if err != nil {
		return err
	}
	return nil
}

func serviceDeploy() error {
	// We deploy the loader as a launchd interval job on OSX, so only
	// target this platform here.
	if runtime.GOOS != "darwin" {
		return nil
	}
	return serviceDeployInterval()
}
