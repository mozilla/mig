// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"os/exec"
	"time"

	"github.com/kardianos/osext"
	service "github.com/mozilla/mig/service"
)

// Run as a Windows service; the loader just runs in the background periodically calling
// itself.
func serviceMode() error {
	var (
		exitCh  chan bool
		binPath string
		err     error
	)

	exitCh = make(chan bool, 16)
	binPath, err = osext.Executable()
	if err != nil {
		return err
	}
	svc, err := service.NewService("mig-loader", "MIG Loader", "Mozilla InvestiGator Agent Loader")
	if err != nil {
		return err
	}
	ostart := func() error {
		return nil
	}
	ostop := func() error {
		exitCh <- true
		return nil
	}
	go svc.Run(ostart, ostop)
	go periodic(binPath, exitCh)
	<-exitCh
	return nil
}

// Periodically execute mig-loader
func periodic(binpath string, exitCh chan bool) {
	// Brief startup delay before we begin processing
	var (
		nextmanfetch     time.Time
		fetchingmanifest bool
		err              error
		cmd              *exec.Cmd
	)
	nextmanfetch = time.Now()
	time.Sleep(time.Second * 60)
	for {
		fetchingmanifest = false
		if time.Now().After(nextmanfetch) {
			// It's time to attempt a manifest fetch, so execute the loader
			// with no additional flags
			fetchingmanifest = true
			cmd = exec.Command(binpath)
			err = cmd.Start()
			if err != nil {
				exitCh <- true
				return
			}
		} else {
			// Request the loader validate the agent is running and start it if not
			cmd = exec.Command(binpath, "-c")
			err = cmd.Start()
			if err != nil {
				exitCh <- true
				return
			}
		}
		err = cmd.Wait()
		if err != nil {
			// If something goes wrong with command execution, sleep for an additional
			// period of time before we try again
			time.Sleep(time.Minute * 5)
			continue
		}
		if fetchingmanifest {
			nextmanfetch = time.Now().Add(60 * time.Minute)
		}
		time.Sleep(time.Minute * 2)
	}
}

// Run when the loader itself is replaced, this function will restart the existing mig-loader
// service on Windows so the new code is live
func serviceTriggers() error {
	svc, err := service.NewService("mig-loader", "MIG Loader", "Mozilla InvestiGator Agent Loader")
	if err != nil {
		return err
	}
	svc.Stop()
	// XXX This should be done properly using a StopWait() approach.
	time.Sleep(time.Second * 5)
	err = svc.Start()
	return err
}
