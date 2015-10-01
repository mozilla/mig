// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"sync"
	"time"

	"mig.ninja/mig"
)

var ctx Context
var wg sync.WaitGroup

func main() {
	var err error

	var config = flag.String("c", "/etc/mig/runner.cfg", "Load configuration from file")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	ctx, err = initContext(*config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(9)
	}

	wg.Add(1)
	go func() {
		var stop bool
		for event := range ctx.Channels.Log {
			stop, err = mig.ProcessLog(ctx.Logging, event)
			if err != nil {
				panic("unable to process log")
			}
			if stop {
				break
			}
		}
		wg.Done()
	}()
	mlog("logging routine started")

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, os.Kill)
	go func() {
		<-sigch
		mlog("signal, exiting")
		ctx.Channels.ExitNotify <- true
	}()

	err = loadPlugins()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		doExit(9)
	}
	// Start up the results processor
	go processResults()

	err = runnerScan()
	if err != nil {
		mlog("runner error: %v", err)
		doExit(9)
	}

	doExit(0)
}

// Scan the runner directory periodically, adding and removing jobs as the
// configuration changes in the runner spool.
func runnerScan() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("runnerScan() -> %v", e)
		}
	}()

	// Begin scanning the runner directory
	for {
		ents, err := ioutil.ReadDir(ctx.Runner.RunDirectory)
		if err != nil {
			panic(err)
		}
		haveents := make([]string, 0)
		for _, x := range ents {
			if !x.IsDir() {
				continue
			}
			haveents = append(haveents, x.Name())
			dirpath := path.Join(ctx.Runner.RunDirectory, x.Name())
			err = procDir(dirpath)
			if err != nil {
				panic(err)
			}
		}
		err = procReap(haveents)
		if err != nil {
			panic(err)
		}

		doexit := false
		select {
		case <-time.After(time.Duration(ctx.Runner.CheckDirectory) * time.Second):
		case <-ctx.Channels.ExitNotify:
			doexit = true
		}
		if doexit {
			break
		}
	}
	mlog("runner exiting due to notification")

	return nil
}

func doExit(rc int) {
	close(ctx.Channels.Log)
	wg.Wait()
	os.Exit(rc)
}

// Process a directory and generate a new scheduling entity if needed
func procDir(dirpath string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("procDir() -> %v", e)
		}
	}()

	ename := path.Base(dirpath)
	confpath := path.Join(dirpath, "entity.cfg")
	finfo, err := os.Stat(confpath)
	if err != nil {
		// Don't treat this as fatal; we will just try to load it again
		// next time.
		mlog("%v: %v", ename, err)
		return nil
	}

	var ent *entity
	// See if we already have an entity by this name, and if the
	// modification time on the configuration file is the same. If so
	// we just return.
	ent, ok := ctx.Entities[ename]
	if ok && (ent.modTime == finfo.ModTime()) {
		return
	}
	if ent != nil {
		delete(ctx.Entities, ename)
		ent.stop()
	}
	ent = &entity{}
	// Add the entity and start it.
	ent.name = ename
	ent.modTime = finfo.ModTime()
	ent.baseDir = dirpath
	ent.confPath = confpath
	err = ent.load()
	if err != nil {
		// Don't treat this as fatal; we will just try to load it again
		// next time.
		mlog("%v: %v", ename, err)
		return nil
	}
	ent.deadChan = make(chan bool, 1)
	ctx.Entities[ename] = ent
	mlog("added entity %v", ename)
	go ent.start()

	return nil
}

// Remove entities that are no longer present in the runner directory, or are
// stale
func procReap(ents []string) error {
	for k := range ctx.Entities {
		found := false
		for _, x := range ents {
			if x == k {
				found = true
			}
		}
		if !found {
			ctx.Entities[k].stop()
			delete(ctx.Entities, k)
			mlog("removed entity %v", k)
		}
	}
	// Also remove stale entities that are still present in the runner
	// directory but have encountered an error.
	for k := range ctx.Entities {
		dead := false
		select {
		case <-ctx.Entities[k].deadChan:
			dead = true
		default:
		}
		if dead {
			delete(ctx.Entities, k)
			mlog("removed stale entity %v", k)
		}
	}
	return nil
}

func mlog(s string, args ...interface{}) {
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf(s, args...)}
}
