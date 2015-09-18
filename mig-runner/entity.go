// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"fmt"
	"github.com/gorhill/cronexpr"
	"gopkg.in/gcfg.v1"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
	"path"
	"time"
)

type entity struct {
	name     string
	baseDir  string
	confPath string
	modTime  time.Time

	deadChan chan bool
	abortRun chan bool
	cfg      entityConfig
}

type entityConfig struct {
	Configuration struct {
		Schedule string
		Plugin   string
	}
}

func (e *entityConfig) validate() error {
	if e.Configuration.Schedule == "" {
		return fmt.Errorf("missing schedule")
	}
	_, err := cronexpr.Parse(e.Configuration.Schedule)
	if err != nil {
		return fmt.Errorf("cron expression: %v", err)
	}
	return nil
}

func (e *entity) launchAction() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("launchAction() -> %v", e)
		}
	}()
	// Load the action from the entity run directory
	actpath := path.Join(e.baseDir, "action.json")
	act, err := mig.ActionFromFile(actpath)
	if err != nil {
		panic(err)
	}

	cli, err := client.NewClient(ctx.ClientConf, "mig-runner")
	if err != nil {
		panic(err)
	}

	// Borrow some logic from the action generator.
	act.ValidFrom = time.Now().Add(-60 * time.Second).UTC()
	period, err := time.ParseDuration("2m")
	if err != nil {
		panic(err)
	}
	act.ExpireAfter = act.ValidFrom.Add(period)
	asig, err := cli.SignAction(act)
	if err != nil {
		panic(err)
	}
	act = asig

	res, err := cli.PostAction(act)
	if err != nil {
		panic(err)
	}
	mlog("%v: launched action %.0f", e.name, res.ID)

	// Notify the results processor an action is in-flight
	re := mig.RunnerResult{}
	re.EntityName = e.name
	re.Action = res
	re.UsePlugin = e.cfg.Configuration.Plugin
	ctx.Channels.Results <- re

	return nil
}

func (e *entity) start() {
	xr := func(s string, args ...interface{}) {
		mlog(s, args...)
		e.deadChan <- true
	}

	e.abortRun = make(chan bool, 1)
	for {
		cexpr, err := cronexpr.Parse(e.cfg.Configuration.Schedule)
		if err != nil {
			xr("%v: %v", e.name, err)
			return
		}
		nrun := cexpr.Next(time.Now())
		waitduration := nrun.Sub(time.Now())
		mlog("%v: will run at %v (in %v)", e.name, nrun, waitduration)
		select {
		case <-e.abortRun:
			mlog("%v: asked to terminate, stopping", e.name)
			return
		case <-time.After(waitduration):
		}
		mlog("%v: running", e.name)
		err = e.launchAction()
		if err != nil {
			xr("%v: %v", e.name, err)
			return
		}
	}
	mlog("%v: job exiting", e.name)
	e.deadChan <- true
}

func (e *entity) stop() {
	close(e.abortRun)
}

func (e *entity) load() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("load() -> %v", e)
		}
	}()
	err = gcfg.ReadFileInto(&e.cfg, e.confPath)
	if err != nil {
		panic(err)
	}

	// Make sure an action file exists and is valid before we
	// schedule this.
	actpath := path.Join(e.baseDir, "action.json")
	_, err = mig.ActionFromFile(actpath)
	if err != nil {
		panic(err)
	}

	err = e.cfg.validate()
	if err != nil {
		panic(err)
	}
	return nil
}
