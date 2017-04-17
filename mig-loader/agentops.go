// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"fmt"
	"mig.ninja/mig"
	"os/exec"
)

func runTriggers() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("runTriggers() -> %v", e)
		}
	}()
	logInfo("running triggers due to modification")
	err = terminateAgent()
	if err != nil {
		logInfo("%v (ignored)", err)
	}
	err = agentServices()
	if err != nil {
		panic(err)
	}
	// Finally, handle any platform specific service related triggers
	// we may want to run. Generally, this will only apply where the
	// loader is replaced and it is running as a service.
	err = serviceTriggers()
	if err != nil {
		panic(err)
	}
	return nil
}

// Inform the agent to set itself up as a service.
func agentServices() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("agentServices() -> %v", e)
		}
	}()

	hb, err := mig.GetHostBundle()
	if err != nil {
		panic(err)
	}
	var abe mig.BundleDictionaryEntry
	found := false
	for _, x := range hb {
		if x.Name == "mig-agent" {
			abe = x
			found = true
			break
		}
	}
	if !found {
		panic("no agent entry in host bundle")
	}

	// Call mig-agent to request it set itself up as a service. This will replace
	// any existing MIG agent service present on the system.
	//
	// Pass the -u flag here to mig-agent (upgrading) as well, to help notify the
	// agent that it should reconfigure it's service setup and exit and should not
	// just be running as an agent normally for this invocation.
	migcomm := exec.Command(abe.Path, "-u")
	err = migcomm.Run()
	if err != nil {
		panic(err)
	}

	return
}

// Terminate any running agent on the system.
func terminateAgent() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("terminateAgent() -> %v", e)
		}
	}()
	hb, err := mig.GetHostBundle()
	if err != nil {
		panic(err)
	}
	var abe mig.BundleDictionaryEntry
	found := false
	for _, x := range hb {
		if x.Name == "mig-agent" {
			abe = x
			found = true
			break
		}
	}
	if !found {
		panic("no agent entry in host bundle")
	}

	migcomm := exec.Command(abe.Path, "-q", "shutdown")
	err = migcomm.Run()
	if err != nil {
		panic(err)
	}

	return
}

// Check if the agent is running or not using the pid status function, err
// is non-nil if not
func agentRunning() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("agentRunning() -> %v", e)
		}
	}()
	hb, err := mig.GetHostBundle()
	if err != nil {
		panic(err)
	}
	var abe mig.BundleDictionaryEntry
	found := false
	for _, x := range hb {
		if x.Name == "mig-agent" {
			abe = x
			found = true
			break
		}
	}
	if !found {
		panic("no agent entry in host bundle")
	}

	migcomm := exec.Command(abe.Path, "-q", "pid")
	err = migcomm.Run()
	if err != nil {
		panic(err)
	}

	return
}
