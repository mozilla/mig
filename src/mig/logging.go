/* Mozilla InvestiGator

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package mig

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
)

const (
	MODE_STDOUT = 1 << iota
	MODE_FILE
	MODE_SYSLOG
)

// Logging stores the attributes needed to perform the logging
type Logging struct {
	// configuration
	Mode, Level, File, Host, Protocol, Facility string
	Port                                        int
	// internal
	logmode  int
	maxlvl   syslog.Priority
	syslogfd *syslog.Writer
}

// Log defines a log entry
type Log struct {
	OpID, ActionID, CommandID float64
	Sev, Desc                 string
	Priority                  syslog.Priority
}

func (l Log) Emerg() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_EMERG
	mlog.Sev = "emergency"
	return
}

func (l Log) Alert() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_ALERT
	mlog.Sev = "alert"
	return
}

func (l Log) Crit() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_CRIT
	mlog.Sev = "critical"
	return
}

func (l Log) Err() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_ERR
	mlog.Sev = "error"
	return
}

func (l Log) Warning() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_WARNING
	mlog.Sev = "warning"
	return
}

func (l Log) Notice() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_NOTICE
	mlog.Sev = "notice"
	return
}

func (l Log) Info() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_INFO
	mlog.Sev = "info"
	return
}

func (l Log) Debug() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_DEBUG
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
		logctx.logmode = MODE_SYSLOG
		logctx, err = initSyslog(logctx, progname)
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
		logctx.maxlvl = syslog.LOG_EMERG
	case "alert":
		logctx.maxlvl = syslog.LOG_ALERT
	case "crit":
		logctx.maxlvl = syslog.LOG_CRIT
	case "err":
		logctx.maxlvl = syslog.LOG_ERR
	case "warning":
		logctx.maxlvl = syslog.LOG_WARNING
	case "notice":
		logctx.maxlvl = syslog.LOG_NOTICE
	case "info":
		logctx.maxlvl = syslog.LOG_INFO
	case "debug":
		logctx.maxlvl = syslog.LOG_DEBUG
	}
	return
}

// initSyslog creates a connection to syslog and stores the handler in ctx
func initSyslog(orig_logctx Logging, progname string) (logctx Logging, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.initSyslog() -> %v", e)
		}
	}()

	logctx = orig_logctx
	if logctx.Host == "" {
		panic("Syslog host is missing")
	}
	if logctx.Port < 1 {
		panic("Syslog port is missing")
	}
	if logctx.Protocol == "" {
		panic("Syslog protocol is missing")
	}
	dialaddr := logctx.Host + ":" + fmt.Sprintf("%d", logctx.Port)
	logctx.syslogfd, err = syslog.Dial(logctx.Protocol, dialaddr, syslog.LOG_LOCAL3|syslog.LOG_INFO, progname)
	if err != nil {
		panic(err)
	}
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
		l.Priority = syslog.LOG_INFO
	}

	// discard logs that have a priority that's higher than the
	// minimal llogel we are configured to log
	if l.Priority > logctx.maxlvl {
		return
	}

	if l.OpID > 0 {
		logline += fmt.Sprintf("%d ", l.OpID)
	} else {
		logline += "- "
	}

	if l.ActionID > 0 {
		logline += fmt.Sprintf("%d ", l.ActionID)
	} else {
		logline += "- "
	}

	if l.CommandID > 0 {
		logline += fmt.Sprintf("%d ", l.CommandID)
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
	case MODE_SYSLOG:
		switch l.Priority {
		// emergency logging causes the scheduler to shut down
		case syslog.LOG_EMERG:
			err = logctx.syslogfd.Emerg(logline)
			stop = true
		case syslog.LOG_ALERT:
			err = logctx.syslogfd.Alert(logline)
		case syslog.LOG_CRIT:
			err = logctx.syslogfd.Crit(logline)
		case syslog.LOG_ERR:
			err = logctx.syslogfd.Err(logline)
		case syslog.LOG_WARNING:
			err = logctx.syslogfd.Warning(logline)
		case syslog.LOG_NOTICE:
			err = logctx.syslogfd.Notice(logline)
		case syslog.LOG_INFO:
			err = logctx.syslogfd.Info(logline)
		case syslog.LOG_DEBUG:
			err = logctx.syslogfd.Debug(logline)
		default:
			err = logctx.syslogfd.Info(logline)
		}
	case MODE_STDOUT:
		log.Println(logline)
	case MODE_FILE:
		log.Println(logline)
	default:
		log.Println(logline)
	}
	return
}

func (logctx Logging) Destroy() {
	if logctx.Mode == "syslog" {
		logctx.syslogfd.Close()
	}
}
