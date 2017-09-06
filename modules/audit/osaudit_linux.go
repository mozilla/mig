// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package audit /* import "mig.ninja/mig/modules/audit" */

import (
	"encoding/json"
	"fmt"

	libaudit "github.com/mozilla/libaudit-go"
	"io/ioutil"
	"syscall"
)

var auditsock *libaudit.NetlinkConnection

// initializeAudit initializes the auditing subsystem . It will load rules
// from the rule path in the configuration, and enable auditing with the specified
// parameters.
func initializeAudit(cfg config) error {
	var err error
	auditsock, err = libaudit.NewNetlinkConnection()
	if err != nil {
		return fmt.Errorf("NewNetlinkConnection: %v", err)
	}
	err = libaudit.AuditSetEnabled(auditsock, true)
	if err != nil {
		return fmt.Errorf("AuditSetEnabled: %v", err)
	}
	status, err := libaudit.AuditIsEnabled(auditsock)
	if err != nil {
		return fmt.Errorf("AuditIsEnabled: %v", err)
	}
	if !status {
		return fmt.Errorf("was not possible to enable audit")
	}
	logChan <- "auditing enabled"
	// Configure audit as required
	if cfg.Audit.RateLimit == 0 { // Set a default rate limit if needed
		cfg.Audit.RateLimit = 500
	}
	err = libaudit.AuditSetRateLimit(auditsock, cfg.Audit.RateLimit)
	if err != nil {
		return fmt.Errorf("AuditSetRateLimit: %v", err)
	}
	if cfg.Audit.BacklogLimit == 0 { // Set default backlog if needed
		cfg.Audit.BacklogLimit = 16384
	}
	err = libaudit.AuditSetBacklogLimit(auditsock, cfg.Audit.BacklogLimit)
	if err != nil {
		return fmt.Errorf("AuditSetBacklogLimit: %v", err)
	}
	err = libaudit.AuditSetPID(auditsock, syscall.Getpid())
	if err != nil {
		return fmt.Errorf("AuditSetPID: %v", err)
	}
	err = libaudit.DeleteAllRules(auditsock)
	if err != nil {
		return fmt.Errorf("DeleteAllRules: %v", err)
	}
	rulebuf, err := ioutil.ReadFile(cfg.Audit.RulesPath)
	if err != nil {
		return err
	}
	warnings, err := libaudit.SetRules(auditsock, rulebuf)
	if err != nil {
		return fmt.Errorf("SetRules: %v", err)
	}
	for _, x := range warnings {
		logChan <- fmt.Sprintf("ruleset warning: %v", x)
	}
	logChan <- "auditing configured"
	return nil
}

func runAudit() error {
	doneChan := make(chan bool, 0)
	logChan <- "listening for audit events"
	libaudit.GetAuditMessages(auditsock, callback, &doneChan)
	return nil
}

func callback(msg *libaudit.AuditEvent, callerr error) {
	// In our callback, we want to simply marshal the audit event and write it to the
	// modules alert channel
	//
	// If includeraw is off, remove the raw data from the AuditEvent before we marshal it.
	if !cfg.Audit.IncludeRaw {
		msg.Raw = ""
	}
	buf, err := json.Marshal(msg)
	if err != nil {
		return
	}
	alertChan <- string(buf)
}
