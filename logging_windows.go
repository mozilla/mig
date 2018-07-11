// +build windows
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "github.com/mozilla/mig" */

import (
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	logModeStdout = 1 << iota
	logModeFile
	logModeEventlog
)

// Logging stores the attributes needed to perform the logging
type Logging struct {
	// configuration
	Mode, Level, File, Host, Protocol, Facility string
	Port                                        int
	MaxFileSize                                 int64
	// internal
	logmode      int
	maxlvl       int
	fd           *eventlog.Log
	rotateWriter rotateLogWriter
}

// Log defines a log entry
type Log struct {
	OpID, ActionID, CommandID float64
	Sev, Desc                 string
	Priority                  int
}

// Emerg sets Log entry level to emergency
func (l Log) Emerg() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "emergency"
	return
}

// Alert sets Log entry level to alert
func (l Log) Alert() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "alert"
	return
}

// Crit sets Log entry level to critical
func (l Log) Crit() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "critical"
	return
}

// Err sets Log entry level to error
func (l Log) Err() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Error
	mlog.Sev = "error"
	return
}

// Warning sets Log entry level to warning
func (l Log) Warning() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Warning
	mlog.Sev = "warning"
	return
}

// Notice sets Log entry level to notice
func (l Log) Notice() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Warning
	mlog.Sev = "notice"
	return
}

// Info sets Log entry level to info
func (l Log) Info() (mlog Log) {
	mlog = l
	mlog.Priority = eventlog.Info
	mlog.Sev = "info"
	return
}

// Debug sets log entry level to debug, we don't log debug
// messages on Windows
func (l Log) Debug() (mlog Log) {
	mlog = l
	mlog.Priority = 999
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
	//
	// On Windows, close the existing logging file descriptor first before we
	// attempt to rename the file.
	r.fd.Close()
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
	// Reinitialize
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
		logctx.logmode = logModeEventlog
		logctx, err = initEventlog(logctx, progname)
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
func initEventlog(origLogctx Logging, progname string) (logctx Logging, err error) {
	logctx = origLogctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initEventlog() -> %v", e)
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
		err = fmt.Errorf("missing mandatory description in logent")
		return
	}

	switch logctx.logmode {
	case logModeEventlog:
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
	return
}

// Destroy can be used to indicate no further logging with the given logging context
// will take place
func (logctx Logging) Destroy() {
	if logctx.Mode == "syslog" {
		_ = logctx.fd.Close()
	}
}
