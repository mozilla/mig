// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Rob Murtha robmurtha@gmail.com [:robmurtha]
package main

import (
	"flag"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"log"

	"os/exec"

	"io/ioutil"

	"github.com/kardianos/service"
	"github.com/robfig/cron"
	"mig.ninja/mig"
)

// options mig-cron flag options
var options cronOptions

// c instance of the cron scheduling agent
var c cron.Cron

// w wait group for for waiting on commands
var w sync.WaitGroup

// logger service logger for sending output to the windows service log
var logger service.Logger

// cronOptions holds all of the individual flag and configuration options
type cronOptions struct {
	crontab   string
	loaderBin string
	// agnetKey options for updating the agent key
	agentKey     string
	agentKeyPath string
	agentKeyPerm os.FileMode
	keyPrompt    bool
	schedule     string
	interval     time.Duration
	install      bool
	foreground   bool
	debug        bool
	uninstall    bool
}

type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go start()
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	stop()
	return nil
}

func main() {
	if err := setCronDefaults(&options); err != nil {
		log.Fatal(err)
	}

	flag.BoolVar(&options.install, "i", options.install, "Install mig-cron as a service.")
	flag.StringVar(&options.crontab, "c", options.crontab, "Load configuration from file.")
	flag.DurationVar(&options.interval, "s", options.interval, "Schedule interval to run (2h, 30m, 1h).")
	flag.StringVar(&options.loaderBin, "l", options.loaderBin, "Path to loader binary.")
	flag.BoolVar(&options.foreground, "f", options.foreground, "Run in foreground instead of installing service.")
	flag.BoolVar(&options.debug, "d", options.debug, "Log debug info.")
	flag.BoolVar(&options.uninstall, "u", options.uninstall, "Uninstall mig-cron as a service.")
	flag.StringVar(&options.agentKey, "k", options.agentKey, "Update agent API key.")
	flag.BoolVar(&options.keyPrompt, "p", options.keyPrompt, "Prompt for inputting API key.")

	flag.Parse()

	//if options.foreground {
	//	// run instead of install as a service
	//	// start the cron scheduler in the background
	//	start()
	//
	//	// catch interrupts
	//	c := make(chan os.Signal, 1)
	//	signal.Notify(c, os.Interrupt)
	//	go func() {
	//		// wait for signal
	//		<-c
	//		// stop the scheduler
	//		stop()
	//	}()
	//
	//	// wait for Stop
	//	wait()
	//	return
	//}

	svcConfig := &service.Config{
		Name:        "mig-cron",
		DisplayName: "mig-cron",
		Description: "Mozilla mig cron service.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	if options.uninstall {
		err = s.Uninstall()
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		return
	}

	if options.agentKey != "" {
		log.Println("mig-cron: updating loader key")
		if err := ioutil.WriteFile(options.agentKeyPath, []byte(options.agentKey), options.agentKeyPerm); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	}

	if _, err := os.Stat(options.loaderBin); os.IsNotExist(err) {
		logger.Errorf("mig-cron: error accessing loader - %s", err.Error())
		os.Exit(1)
	}

	if options.install {
		err = s.Install()
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		return
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
	return
}

func start() {
	options.schedule = "@every " + options.interval.String()
	logger.Infof("mig-cron: starting")
	logger.Infof("mig-cron: jobs - running %s %s", options.loaderBin, options.schedule)
	c := cron.New()

	if err := c.AddFunc(options.schedule, runLoader); err != nil {
		logger.Errorf("mig-cron: invalid schedule entry %v", err)
		panic(err)
	}

	w.Add(1)
	c.Start()

	// run loader on start
	runLoader()
}

func stop() {
	logger.Info("mig-cron: stopping")
	c.Stop()
	w.Done()
}

func wait() {
	w.Wait()
}

func runLoader() {
	logger.Info("mig-cron: running ", options.loaderBin)
	out, err := exec.Command(options.loaderBin).CombinedOutput()
	if err != nil {
		logger.Errorf("mig-cron: error running %s %v", options.loaderBin, err)
	}
	if options.debug {
		logger.Infof("mig-cron: output - %s", out)
	}
}

func setCronDefaults(options *cronOptions) error {

	bundles, err := mig.GetHostBundle()
	if err != nil {
		return err
	}
	if entry, ok := getEntry("configuration-cron", bundles); ok {
		options.crontab = entry.Path
	}
	if entry, ok := getEntry("mig-loader", bundles); ok {
		options.loaderBin = entry.Path
	}
	if entry, ok := getEntry("agentkey", bundles); ok {
		options.agentKeyPath = entry.Path
		options.agentKeyPerm = entry.Perm
	}

	options.interval = time.Duration(7200 * time.Second)

	if runtime.GOOS == `windows` {
		setCronWindowsDefaults(options)
	}
	return nil
}

func setCronWindowsDefaults(options *cronOptions) {
	root := os.Getenv(mig.Env_Win_Root)
	if root != "" && root != mig.Env_Win_Root_Defaut {
		// setup non default paths
		options.crontab = path.Join(root, "mig", "mig-cron.cfg")
		options.loaderBin = path.Join(root, "mig", "mig-loader.exe")
		options.agentKeyPath = path.Join(root, "mig", "agent.key")
	}
	return
}

func getEntry(name string, bundle []mig.BundleDictionaryEntry) (mig.BundleDictionaryEntry, bool) {
	for i := 0; i < len(bundle); i++ {
		if bundle[i].Name == name {
			return bundle[i], true
		}
	}
	return mig.BundleDictionaryEntry{}, false
}
