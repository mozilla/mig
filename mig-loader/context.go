// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"mig.ninja/mig"
	"mig.ninja/mig/mig-agent/agentcontext"
	"mig.ninja/mig/service"
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

// initKeyring loads public key material from the keys directory in the loader configuration
// directory if present. Each file should contain a single PGP public key, and these keys override
// any keys present in the MANIFESTKEYS configuration variable.
func initKeyring(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initKeyring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initKeyring()"}.Debug()
	}()

	krdir := path.Join(agentcontext.GetConfDir(), "loaderkeys")
	files, err := ioutil.ReadDir(krdir)
	if err != nil && os.IsNotExist(err) {
		logInfo("key directory %v not found, continuing with built-in keyring", krdir)
		return ctx, nil
	} else if err != nil {
		panic(err)
	}
	MANIFESTPGPKEYS = MANIFESTPGPKEYS[:0]
	for _, x := range files {
		keypath := path.Join(krdir, x.Name())
		logInfo("loading key from %v", keypath)
		buf, err := ioutil.ReadFile(keypath)
		if err != nil {
			panic(err)
		}
		MANIFESTPGPKEYS = append(MANIFESTPGPKEYS, string(buf))
	}

	return
}
