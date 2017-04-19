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
	service "mig.ninja/mig/service"
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
	for {
		cmd := exec.Command(binpath)
		err := cmd.Start()
		if err != nil {
			exitCh <- true
			return
		}
		err = cmd.Wait()
		if err != nil {
			// On failure, sleep for a shorter period and retry
			time.Sleep(time.Minute * 5)
			continue
		}
		time.Sleep(time.Minute * 60)
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
