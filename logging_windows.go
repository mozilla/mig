// +build windows
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"fmt"
	"log"
	"os"

	"code.google.com/p/winsvc/eventlog"
)

const (
	MODE_STDOUT = 1 << iota
	MODE_FILE
	MODE_EVENTLOG
)

// Logging stores the attributes needed to perform the logging
type Logging struct {
	// configuration
	Mode, Level, File, Host, Protocol, Facility string
	Port                                        int
	// internal
	logmode int
	maxlvl  int
	fd      *eventlog.Log
}

// Log defines a log entry
type Log struct {
	OpID, ActionID, CommandID float64
	Sev, Desc                 string
	Priority                  int
}

func (l Log) Emerg() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "emergency"
	return
}

func (l Log) Alert() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "alert"
	return
}

func (l Log) Crit() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "critical"
	return
}

func (l Log) Err() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "error"
	return
}

func (l Log) Warning() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Warning
	mlog.Sev = "warning"
	return
}

func (l Log) Notice() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Warning
	mlog.Sev = "notice"
	return
}

func (l Log) Info() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Info
	mlog.Sev = "info"
	return
}

// don't log debug messages on windows
func (l Log) Debug() (mlog Log) {
	mlog = l
	mlog.Priority = 999
	mlog.Sev = "debug"
	return
}

// InitLogger prepares the context for logging based on the configuration
// in Logging
func InitLogger(orig_logctx Logging, progname string) (logctx Logging, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.InitLogger() -> %v", e)
		}
	}()

	logctx = orig_logctx
	switch logctx.Mode {
	case "stdout":
		logctx.logmode = MODE_STDOUT
		logctx, err = initLogStdOut(logctx)
		if err != nil {
			panic(err)
		}
	case "file":
		logctx.logmode = MODE_FILE
		logctx, err = initLogFile(logctx)
		if err != nil {
			panic(err)
		}
	case "syslog":
		logctx.logmode = MODE_EVENTLOG
		logctx, err = initEventlog(logctx, progname)
		if err != nil {
			panic(err)
		}
	default:
		log.Println("Logging mode is missing. Assuming stdout.")
		logctx.Mode = "stdout"
		logctx.logmode = MODE_STDOUT
		logctx, err = initLogStdOut(logctx)
		if err != nil {
			panic(err)
		}
	}

	// set the minimal log level
	switch logctx.Level {
	case "emerg":
		logctx.maxlvl = eventlog.Error
	case "alert":
		logctx.maxlvl = eventlog.Error
	case "crit":
		logctx.maxlvl = eventlog.Error
	case "err":
		logctx.maxlvl = eventlog.Error
	case "warning":
		logctx.maxlvl = eventlog.Warning
	case "notice":
		logctx.maxlvl = eventlog.Warning
	case "info":
		logctx.maxlvl = eventlog.Info
	case "debug":
		// discard
		logctx.maxlvl = 999
	}
	return
}

// initEventlog creates a connection to event logs and stores the handler in ctx
func initEventlog(orig_logctx Logging, progname string) (logctx Logging, err error) {
	logctx = orig_logctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.initEventlog() -> %v", e)
		}
	}()
	const name = "mylog"
	const supports = eventlog.Error | eventlog.Warning | eventlog.Info
	err = eventlog.InstallAsEventCreate(name, supports)
	if err != nil {
		panic(err)
	}
	logctx.fd, err = eventlog.Open(progname)
	if err != nil {
		panic(err)
	}
	return
}

// initLogFile creates a logfile and stores the descriptor in ctx
func initLogFile(orig_logctx Logging) (logctx Logging, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.InitLogFile() -> %v", e)
		}
	}()

	logctx = orig_logctx
	fd, err := os.OpenFile(logctx.File, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
	if err != nil {
		panic(err)
	}
	log.SetOutput(fd)
	return
}

// initLogStdOut does nothing except storing in ctx that logs should be
// sent to stdout directly
func initLogStdOut(orig_logctx Logging) (logctx Logging, err error) {
	logctx = orig_logctx
	return
}

// processLog receives events and perform logging and evaluationg of the log
// if the log is too critical, Analyze will trigger a scheduler shutdown
func ProcessLog(logctx Logging, l Log) (stop bool, err error) {
	stop = false
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.ProcessLog() -> %v", e)
		}
	}()

	var logline string

	// if priority isn't set, use the default "info"
	if l.Priority < 1 {
		l.Priority = eventlog.Info
	}

	// discard logs that have a priority that's higher than the
	// minimal log level we are configured to log
	if l.Priority > logctx.maxlvl {
		return
	}

	if l.OpID > 0 {
		logline += fmt.Sprintf("%.0f ", l.OpID)
	} else {
		logline += "- "
	}

	if l.ActionID > 0 {
		logline += fmt.Sprintf("%.0f ", l.ActionID)
	} else {
		logline += "- "
	}

	if l.CommandID > 0 {
		logline += fmt.Sprintf("%.0f ", l.CommandID)
	} else {
		logline += "- "
	}

	if l.Sev != "" {
		logline += "[" + l.Sev + "] "
	} else {
		logline += "[info] "
	}

	if l.Desc != "" {
		logline += l.Desc
	} else {
		err = fmt.Errorf("Missing mandatory description in logent")
		return
	}

	switch logctx.logmode {
	case MODE_EVENTLOG:
		switch l.Priority {
		case eventlog.Error:
			err = logctx.fd.Error(1, logline)
			return
		case eventlog.Warning:
			err = logctx.fd.Warning(2, logline)
			return
		case eventlog.Info:
			err = logctx.fd.Info(3, logline)
			return
		case 999:
			// debug logs go to stdout
			log.Println(logline)
			return
		default:
			err = logctx.fd.Info(3, logline)
			return
		}
	case MODE_STDOUT:
		log.Println(logline)
		return
	case MODE_FILE:
		log.Println(logline)
		return
	default:
		log.Println(logline)
		return
	}
	return
}

func (logctx Logging) Destroy() {
	if logctx.Mode == "syslog" {
		_ = logctx.fd.Close()
	}
}
