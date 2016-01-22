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

// The default expiry time for an action launched by the runner if the
// entity configuration does not include an expiry.
var defaultExpiry = "5m"

// Represents a scheduler entity, which is basically a job configuration that
// lives in the runner spool directory.
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
		Expiry   string
		SendOnly bool
	}
}

func (e *entityConfig) validate() error {
	if e.Configuration.Schedule == "" {
		return fmt.Errorf("missing schedule")
	}
	_, err := cronexpr.Parse(e.Configuration.Schedule)
	if err != nil {
		return fmt.Errorf("bad cron expression: %v", err)
	}
	return nil
}

// Launch an action represented by a scheduler entity, this function takes
// care of submitting the action to the API and making a note of when to
// attempt to retrieve the results of the action.
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
	act.Name = fmt.Sprintf("mig-runner: %v", e.name)

	cli, err := client.NewClient(ctx.ClientConf, "mig-runner")
	if err != nil {
		panic(err)
	}

	// Borrow some logic from the action generator. Set a validation
	// period starting in the past so our action starts immediately.
	window := time.Duration(-60 * time.Second)
	act.ValidFrom = time.Now().Add(window).UTC()
	exstring := defaultExpiry
	if e.cfg.Configuration.Expiry != "" {
		exstring = e.cfg.Configuration.Expiry
	}
	period, err := time.ParseDuration(exstring)
	if err != nil {
		panic(err)
	}
	// Add the window period to the desired expiry since our start
	// time begins in the past.
	period += -window
	act.ExpireAfter = act.ValidFrom.Add(period)
	act, err = cli.CompressAction(act)
	if err != nil {
		panic(err)
	}
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

	// If we only dispatch this action we are done here.
	if e.cfg.Configuration.SendOnly {
		return nil
	}

	// Notify the results processor an action is in-flight
	re := mig.RunnerResult{}
	re.EntityName = e.name
	re.Action = res
	re.UsePlugin = e.cfg.Configuration.Plugin
	ctx.Channels.Results <- re

	return nil
}

// Start a scheduler entity. This is normally run in it's own go-routine
// and will wait until the configured time to execute.
func (e *entity) start() {
	xr := func(s string, args ...interface{}) {
		mlog(s, args...)
		e.deadChan <- true
	}

	e.abortRun = make(chan bool, 1)
	for {
		cexpr, err := cronexpr.Parse(e.cfg.Configuration.Schedule)
		if err != nil {
			xr("%v: bad cron expression: %v", e.name, err)
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

// Abort a scheduler entity, for example if the job has been removed from the
// runner configuration.
func (e *entity) stop() {
	close(e.abortRun)
}

// Load the configuration of a scheduler entity from the runner spool
// directory.
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
