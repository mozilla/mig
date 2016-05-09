// +build linux
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributors:
// Alexandru Tudorica <tudalex@gmail.com>
// vladimirdiaconescu <vladimirdiaconescu@users.noreply.github.com>
// Teodora Baluta <teobaluta@gmail.com>


package sandbox

//#include "signal_handler.h"
import "C"

import (
	"github.com/seccomp/libseccomp-golang"
	"log"
	"syscall"
)

var ActTrap = seccomp.ActTrap
var ActAllow = seccomp.ActAllow

type FilterAction string

type FilterOperation struct {
	FilterOn   []string
	Action     seccomp.ScmpAction
	Conditions []seccomp.ScmpCondition
}

type SandboxProfile struct {
	DefaultPolicy seccomp.ScmpAction
	Filters       []FilterOperation
}

func Jail(sandboxProfile SandboxProfile) {
	C.install_sighandler()
	filter, err := seccomp.NewFilter(sandboxProfile.DefaultPolicy)
	if err != nil {
		log.Fatalf("Error creating filter: %s\n", err)
	} else {
		log.Printf("Created filter\n")
	}
	defer filter.Release()
	action, err := filter.GetDefaultAction()
	if err != nil {
		log.Fatal("Error getting default action of filter\n")
	} else if action != seccomp.ActTrap {
		log.Printf("Default action of filter was set incorrectly!\n")
	}
	for _, profileFilter := range sandboxProfile.Filters {
		for _, callName := range profileFilter.FilterOn {
			call, err := seccomp.GetSyscallFromName(callName)
			if err != nil {
				log.Fatalf("Error getting syscall number of %s: %s\n", callName, err)
			} else {
				log.Printf("Got hook to syscall %d\n", call)
			}
			// if there are conditions, construct a conditional rule
			if len(profileFilter.Conditions) > 0 {
				err = filter.AddRuleConditional(call, profileFilter.Action, profileFilter.Conditions)
				if err != nil {
					log.Fatalf("Error adding conditional rule: %s", err)
				}
			} else {
				err = filter.AddRule(call, profileFilter.Action)
				if err != nil {
					log.Fatalf("Error adding rule to restrict syscall: %s\n", err)
				} else {
					log.Printf("Added rule to restrict syscall %s\n", callName)
				}
			}
		}
	}
	filter.SetTsync(true)
	filter.SetNoNewPrivsBit(true)
	err = filter.Load()
	errBadFilter, ok := err.(syscall.Errno)
	if ok {
		log.Printf("Failed to load seccomp filter, seccomp (filter mode) not supported by kernel: %d. Sandbox features disabled.",
			int(errBadFilter))
	} else {
		log.Fatalf("Filter context is invalid: %s", errBadFilter)
	}
}
