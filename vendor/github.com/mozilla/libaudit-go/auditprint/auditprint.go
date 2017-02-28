// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// auditprint is a simple command line tool that loads an audit rule set from a JSON file,
// applies it to the current kernel and begins printing any audit event the kernel sends in
// JSON format.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mozilla/libaudit-go"
)

func auditProc(e *libaudit.AuditEvent, err error) {
	if err != nil {
		// See if the error is libaudit.ErrorAuditParse, if so convert and also display
		// the audit record we could not parse
		if nerr, ok := err.(libaudit.ErrorAuditParse); ok {
			fmt.Printf("parser error: %v: %v\n", nerr, nerr.Raw)
		} else {
			fmt.Printf("callback received error: %v\n", err)
		}
		return
	}
	// Marshal the event to JSON and print
	buf, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("callback was unable to marshal event: %v\n", err)
		return
	}
	fmt.Printf("%v\n", string(buf))
}

func main() {
	s, err := libaudit.NewNetlinkConnection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "NetNetlinkConnection: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) != 2 {
		fmt.Printf("usage: %v path_to_rules.json\n", os.Args[0])
		os.Exit(0)
	}

	err = libaudit.AuditSetEnabled(s, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AuditSetEnabled: %v\n", err)
		os.Exit(1)
	}

	err = libaudit.AuditSetPID(s, os.Getpid())
	if err != nil {
		fmt.Fprintf(os.Stderr, "AuditSetPid: %v\n", err)
		os.Exit(1)
	}
	err = libaudit.AuditSetRateLimit(s, 1000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AuditSetRateLimit: %v\n", err)
		os.Exit(1)
	}
	err = libaudit.AuditSetBacklogLimit(s, 250)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AuditSetBacklogLimit: %v\n", err)
		os.Exit(1)
	}

	var ar libaudit.AuditRules
	buf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ReadFile: %v\n", err)
		os.Exit(1)
	}
	// Make sure we can unmarshal the rules JSON to validate it is the correct
	// format
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unmarshaling rules JSON: %v\n", err)
		os.Exit(1)
	}

	// Remove current rule set and send rules to the kernel
	err = libaudit.DeleteAllRules(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DeleteAllRules: %v\n", err)
		os.Exit(1)
	}
	warnings, err := libaudit.SetRules(s, buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SetRules: %v\n", err)
		os.Exit(1)
	}
	// Print any warnings we got back but still continue
	for _, x := range warnings {
		fmt.Fprintf(os.Stderr, "ruleset warning: %v\n", x)
	}

	doneCh := make(chan bool, 1)
	libaudit.GetAuditMessages(s, auditProc, &doneCh)
}
