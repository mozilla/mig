// +build linux darwin

// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"sync"
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

// Custom type to satisfy io.Writer to use as file logging output, handles
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
func initLogFile(orig_logctx Logging) (logctx Logging, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.InitLogFile() -> %v", e)
		}
	}()

	logctx = orig_logctx
	err = logctx.rotateWriter.new(logctx.File, logctx.MaxFileSize)
	if err != nil {
		panic(err)
	}
	log.SetOutput(&logctx.rotateWriter)
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
	case MODE_SYSLOG:
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
}

func (logctx Logging) Destroy() {
	if logctx.Mode == "syslog" {
		logctx.syslogfd.Close()
	}
}
