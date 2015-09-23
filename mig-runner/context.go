// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"fmt"
	"gopkg.in/gcfg.v1"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
	"path"
)

type Context struct {
	Channels struct {
		Log        chan mig.Log
		ExitNotify chan bool
		Results    chan mig.RunnerResult
	}
	Runner struct {
		Directory       string
		RunDirectory    string
		PluginDirectory string
		CheckDirectory  int
	}
	Client struct {
		ClientConfPath string
		Passphrase     string
		DelayResults   string
	}
	Logging mig.Logging

	Entities   map[string]*entity
	ClientConf client.Configuration
}

func initContext(config string) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initContext() -> %v", e)
		}
	}()

	ctx = Context{}
	ctx.Channels.Log = make(chan mig.Log, 37)
	ctx.Channels.Results = make(chan mig.RunnerResult, 64)
	ctx.Channels.ExitNotify = make(chan bool, 64)
	ctx.Entities = make(map[string]*entity)
	err = gcfg.ReadFileInto(&ctx, config)
	if err != nil {
		panic(err)
	}

	ctx.Runner.RunDirectory = path.Join(ctx.Runner.Directory, "runners")
	ctx.Runner.PluginDirectory = path.Join(ctx.Runner.Directory, "plugins")

	if ctx.Client.ClientConfPath == "default" {
		hdir := client.FindHomedir()
		ctx.Client.ClientConfPath = path.Join(hdir, ".migrc")
	}
	ctx.ClientConf, err = client.ReadConfiguration(ctx.Client.ClientConfPath)
	if err != nil {
		panic(err)
	}

	if ctx.Client.Passphrase != "" {
		client.ClientPassphrase(ctx.Client.Passphrase)
	}

	ctx.Logging, err = mig.InitLogger(ctx.Logging, "mig-runner")
	if err != nil {
		panic(err)
	}

	return ctx, nil
}
