// +build linux darwin

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "github.com/mozilla/mig" */

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"sync"
)

const (
	logModeStdout = 1 << iota
	logModeFile
	logModeSyslog
)

// Logging stores the attributes needed to perform the logging
type Logging struct {
	// configuration
	Mode, Level, File, Host, Protocol, Facility string
	Port                                        int
	MaxFileSize                                 int64
	// internal
	logmode      int
	maxlvl       syslog.Priority
	syslogfd     *syslog.Writer
	rotateWriter rotateLogWriter
}

// Log defines a log entry
type Log struct {
	OpID, ActionID, CommandID float64
	Sev, Desc                 string
	Priority                  syslog.Priority
}

// Emerg sets Log entry level to emergency
func (l Log) Emerg() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_EMERG
	mlog.Sev = "emergency"
	return
}

// Alert sets Log entry level to alert
func (l Log) Alert() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_ALERT
	mlog.Sev = "alert"
	return
}

// Crit sets Log entry level to critical
func (l Log) Crit() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_CRIT
	mlog.Sev = "critical"
	return
}

// Err sets Log entry level to error
func (l Log) Err() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_ERR
	mlog.Sev = "error"
	return
}

// Warning sets Log entry level to warning
func (l Log) Warning() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_WARNING
	mlog.Sev = "warning"
	return
}

// Notice sets Log entry level to notice
func (l Log) Notice() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_NOTICE
	mlog.Sev = "notice"
	return
}

// Info sets Log entry level to info
func (l Log) Info() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_INFO
	mlog.Sev = "info"
	return
}

// Debug sets log entry level to debug
func (l Log) Debug() (mlog Log) {
	mlog = l
	mlog.Priority = syslog.LOG_DEBUG
	mlog.Sev = "debug"
	return
}

// rotateLogWriter is a custom type to satisfy io.Writer to use as file logging output, handles
// log file rotation
type rotateLogWriter struct {
	sync.Mutex
	filename string
	fd       *os.File
	maxBytes int64 // Maximum size for log file, 0 means no rotation
}

func (r *rotateLogWriter) new(filename string, maxbytes int64) error {
	r.filename = filename
	r.maxBytes = maxbytes
	return r.initAndCheck()
}

func (r *rotateLogWriter) Write(output []byte) (int, error) {
	var err error
	err = r.initAndCheck()
	if err != nil {
		return 0, err
	}
	return r.fd.Write(output)
}

func (r *rotateLogWriter) initAndCheck() (err error) {
	defer func() {
		r.Unlock()
		if e := recover(); e != nil {
			err = fmt.Errorf("initAndCheck() -> %v", e)
		}
	}()
	r.Lock()
	// If we haven't initialized yet, open the log file
	if r.fd == nil {
		r.fd, err = os.OpenFile(r.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
		if err != nil {
			panic(err)
		}
	}
	// If the maximum file size is set to 0 we will never rotate
	if r.maxBytes == 0 {
		return nil
	}
	// Check the size of the existing log file, and rotate it if required.
	// We only keep the current log file and one older one.
	var fi os.FileInfo
	fi, err = r.fd.Stat()
	if err != nil {
		panic(err)
	}
	if fi.Size() < r.maxBytes {
		return
	}
	// Rotate the log and reinitialize it
	// If the old file already exists remove it
	rotatefile := r.filename + ".1"
	_, err = os.Stat(rotatefile)
	if err == nil || !os.IsNotExist(err) {
		err = os.Remove(rotatefile)
		if err != nil {
			panic(err)
		}
	}
	// Rename existing log file
	err = os.Rename(r.filename, rotatefile)
	if err != nil {
		panic(err)
	}
	// Close existing descriptor and reinitialize
	r.fd.Close()
	r.fd, err = os.OpenFile(r.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
	if err != nil {
		panic(err)
	}
	return
}

// InitLogger prepares the context for logging based on the configuration
// in Logging
func InitLogger(origLogctx Logging, progname string) (logctx Logging, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("InitLogger() -> %v", e)
		}
	}()

	logctx = origLogctx
	switch logctx.Mode {
	case "stdout":
		logctx.logmode = logModeStdout
		logctx, err = initLogStdOut(logctx)
		if err != nil {
			panic(err)
		}
	case "file":
		logctx.logmode = logModeFile
		logctx, err = initLogFile(logctx)
		if err != nil {
			panic(err)
		}
	case "syslog":
		logctx.logmode = logModeSyslog
		logctx, err = initSyslog(logctx, progname)
		if err != nil {
			panic(err)
		}
	default:
		log.Println("Logging mode is missing. Assuming stdout.")
		logctx.Mode = "stdout"
		logctx.logmode = logModeStdout
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
func initSyslog(origLogctx Logging, progname string) (logctx Logging, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initSyslog() -> %v", e)
		}
	}()

	logctx = origLogctx
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
	logctx.syslogfd, err = syslog.Dial(logctx.Protocol, dialaddr, syslog.LOG_DAEMON|syslog.LOG_INFO, progname)
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	return
}

// initLogFile creates a logfile and stores the descriptor in ctx
func initLogFile(origLogctx Logging) (logctx Logging, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initLogFile() -> %v", e)
		}
	}()

	logctx = origLogctx
	err = logctx.rotateWriter.new(logctx.File, logctx.MaxFileSize)
	if err != nil {
		panic(err)
	}
	log.SetOutput(&logctx.rotateWriter)
	return
}

// initLogStdOut does nothing except storing in ctx that logs should be
// sent to stdout directly
func initLogStdOut(origLogctx Logging) (logctx Logging, err error) {
	logctx = origLogctx
	return
}

// ProcessLog receives events and performs logging and evaluation of the log
// severity level, in the event of an emergency level entry stop will be true
func ProcessLog(logctx Logging, l Log) (stop bool, err error) {
	stop = false
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ProcessLog() -> %v", e)
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
		err = fmt.Errorf("missing mandatory description in logent")
		return
	}

	switch logctx.logmode {
	case logModeSyslog:
		switch l.Priority {
		// emergency logging causes the scheduler to shut down
		case syslog.LOG_EMERG:
			err = logctx.syslogfd.Emerg(logline)
			stop = true
			return
		case syslog.LOG_ALERT:
			err = logctx.syslogfd.Alert(logline)
			return
		case syslog.LOG_CRIT:
			err = logctx.syslogfd.Crit(logline)
			return
		case syslog.LOG_ERR:
			err = logctx.syslogfd.Err(logline)
			return
		case syslog.LOG_WARNING:
			err = logctx.syslogfd.Warning(logline)
			return
		case syslog.LOG_NOTICE:
			err = logctx.syslogfd.Notice(logline)
			return
		case syslog.LOG_INFO:
			err = logctx.syslogfd.Info(logline)
			return
		case syslog.LOG_DEBUG:
			err = logctx.syslogfd.Debug(logline)
			return
		default:
			err = logctx.syslogfd.Info(logline)
			return
		}
	case logModeStdout:
		log.Println(logline)
		return
	case logModeFile:
		log.Println(logline)
		return
	default:
		log.Println(logline)
		return
	}
}

// Destroy can be used to indicate no further logging with the given logging context
// will take place
func (logctx Logging) Destroy() {
	if logctx.Mode == "syslog" {
		logctx.syslogfd.Close()
	}
}
