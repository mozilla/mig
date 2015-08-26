// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package gozdef

import (
	"fmt"
	"os"
	"regexp"
	"time"
)

// Generic MozDef event handling

// an Event represent a piece of information being passed to MozDef
type Event struct {
	Timestamp   time.Time   `json:"timestamp"`   // Full date plus time timestamp of the event in ISO format including the timezone offset
	Category    string      `json:"category"`    // General category/type of event
	Hostname    string      `json:"hostname"`    // The fully qualified domain name of the host sending the message
	ProcessID   float64     `json:"processid"`   // The PID of the process sending the log
	ProcessName string      `json:"processname"` // The name of the process sending the log
	Severity    string      `json:"severity"`    // RFC5424 severity level of the event in all caps: DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY
	Source      string      `json:"source"`      // Source of the event (file name, system name, component name)
	Summary     string      `json:"summary"`     // Short human-readable version of the event suitable for IRC, SMS, etc.
	Tags        []string    `json:"tags"`        // An array or list of any tags you would like applied to the event
	Details     interface{} `json:"details"`     // Additional, event-specific fields included with the event
}

func NewEvent() (e Event, err error) {
	e.Timestamp = time.Now().UTC()
	e.Hostname, err = os.Hostname()
	if err != nil {
		return
	}
	e.ProcessID = float64(os.Getpid())
	e.ProcessName = os.Args[0]
	return
}

const SEV_REGEX string = "^(DEBUG|INFO|NOTICE|WARNING|ERROR|CRITICAL|ALERT|EMERGENCY)$"

// set the severity level of the event to DEBUG
func (e *Event) Debug() {
	e.Severity = "DEBUG"
}

// set the severity level of the event to INFO
func (e *Event) Info() {
	e.Severity = "INFO"
}

// set the severity level of the event to NOTICE
func (e *Event) Notice() {
	e.Severity = "NOTICE"
}

// set the severity level of the event to WARNING
func (e *Event) Warning() {
	e.Severity = "WARNING"
}

// set the severity level of the event to ERROR
func (e *Event) Error() {
	e.Severity = "ERROR"
}

// set the severity level of the event to CRITICAL
func (e *Event) Critical() {
	e.Severity = "CRITICAL"
}

// set the severity level of the event to ALERT
func (e *Event) Alert() {
	e.Severity = "ALERT"
}

// set the severity level of the event to EMERGENCY
func (e *Event) Emergency() {
	e.Severity = "EMERGENCY"
}

// Validate verifies that an event is formatted correctly
func (e Event) Validate() error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	if hostname != e.Hostname {
		return fmt.Errorf("event hostname does not match the host's fqdn")
	}
	if float64(os.Getpid()) != e.ProcessID {
		return fmt.Errorf("event processid does not match the id of the current process")
	}
	if os.Args[0] != e.ProcessName {
		return fmt.Errorf("event processname does not match the name of the current process")
	}
	resev := regexp.MustCompile(SEV_REGEX)
	if !resev.MatchString(e.Severity) {
		return fmt.Errorf("invalid severity '%s', must be one of %s", e.Severity, SEV_REGEX)
	}
	return nil
}
