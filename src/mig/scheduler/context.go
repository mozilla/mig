/* Mozilla InvestiGator Scheduler

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

package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"mig"
	migdb "mig/database"
	"net"
	"os"
	"time"

	"code.google.com/p/gcfg"
	"github.com/streadway/amqp"
)

// Context contains all configuration variables as well as handlers for
// database and message brokers. It also contains some statistics.
// Context is intended as a single structure that can be passed around easily.
type Context struct {
	OpID  uint64 // ID of the current operation, used for tracking
	Agent struct {
		// configuration
		TimeOut, HeartbeatFreq, Whitelist string
		DetectMultiAgents                 bool
	}
	Channels struct {
		// internal
		Terminate                                                                        chan error
		Log                                                                              chan mig.Log
		NewAction, ActionDone, CommandReady, UpdateCommand, CommandReturned, CommandDone chan string
		DetectDupAgents                                                                  chan string
	}
	Collector struct {
		Freq, DeleteAfter string
	}
	Directories struct {
		// configuration
		Spool string
		Tmp   string
		// internal
		Action struct {
			New, InFlight, Done, Invalid string
		}
		Command struct {
			Ready, InFlight, Returned, Done string
		}
	}
	DB migdb.DB
	MQ struct {
		// configuration
		Host, User, Pass, Vhost string
		Port                    int
		UseTLS                  bool
		TLScert, TLSkey, CAcert string
		Timeout                 string
		// internal
		conn *amqp.Connection
		Chan *amqp.Channel
	}
	PGP struct {
		KeyID, Home string
	}
	Postgres struct {
		Host, User, Password, DBName, SSLMode string
		Port                                  int
	}
	Stats struct {
	}
	Logging mig.Logging
}

// Init() initializes a context from a configuration file into an
// existing context struct
func Init(path string) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Init() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving Init()"}.Debug()
	}()

	err = gcfg.ReadFileInto(&ctx, path)
	if err != nil {
		panic(err)
	}

	ctx, err = initChannels(ctx)
	if err != nil {
		panic(err)
	}

	ctx.Logging, err = mig.InitLogger(ctx.Logging, "mig-scheduler")
	if err != nil {
		panic(err)
	}

	ctx, err = initDirectories(ctx)
	if err != nil {
		panic(err)
	}

	ctx, err = initDB(ctx)
	if err != nil {
		panic(err)
	}

	ctx, err = initBroker(ctx)
	if err != nil {
		panic(err)
	}

	return
}

// initDirectories() creates the folders used by the scheduler on the local file system
func initDirectories(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initDirectories() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initDirectories()"}.Debug()
	}()

	ctx.Directories.Action.New = ctx.Directories.Spool + "/action/new/"
	err = os.MkdirAll(ctx.Directories.Action.New, 0750)
	if err != nil {
		panic(err)
	}

	ctx.Directories.Action.InFlight = ctx.Directories.Spool + "/action/inflight/"
	err = os.MkdirAll(ctx.Directories.Action.InFlight, 0750)
	if err != nil {
		panic(err)
	}

	ctx.Directories.Action.Done = ctx.Directories.Spool + "/action/done"
	err = os.MkdirAll(ctx.Directories.Action.Done, 0750)
	if err != nil {
		panic(err)
	}

	ctx.Directories.Action.Invalid = ctx.Directories.Spool + "/action/invalid"
	err = os.MkdirAll(ctx.Directories.Action.Invalid, 0750)
	if err != nil {
		panic(err)
	}

	ctx.Directories.Command.Ready = ctx.Directories.Spool + "/command/ready"
	err = os.MkdirAll(ctx.Directories.Command.Ready, 0750)
	if err != nil {
		panic(err)
	}

	ctx.Directories.Command.InFlight = ctx.Directories.Spool + "/command/inflight"
	err = os.MkdirAll(ctx.Directories.Command.InFlight, 0750)
	if err != nil {
		panic(err)
	}

	ctx.Directories.Command.Returned = ctx.Directories.Spool + "/command/returned"
	err = os.MkdirAll(ctx.Directories.Command.Returned, 0750)
	if err != nil {
		panic(err)
	}

	ctx.Directories.Command.Done = ctx.Directories.Spool + "/command/done"
	err = os.MkdirAll(ctx.Directories.Command.Done, 0750)
	if err != nil {
		panic(err)
	}

	return
}

// initDB sets up the connection to the Postgres backend database
func initDB(orig_ctx Context) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initDB() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initDB()"}.Debug()
	}()

	ctx = orig_ctx
	ctx.DB, err = migdb.Open(ctx.Postgres.DBName, ctx.Postgres.User, ctx.Postgres.Password,
		ctx.Postgres.Host, ctx.Postgres.Port, ctx.Postgres.SSLMode)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: "Database connection opened"}
	return
}

// initBroker() sets up the connection to the RabbitMQ broker
func initBroker(orig_ctx Context) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initBroker() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initBroker()"}.Debug()
	}()

	ctx = orig_ctx
	// dialing address use format "<scheme>://<user>:<pass>@<host>:<port><vhost>"
	var scheme, user, pass, host, port, vhost string
	if ctx.MQ.UseTLS {
		scheme = "amqps"
	} else {
		scheme = "amqp"
	}
	if ctx.MQ.User == "" {
		panic("MQ User is missing")
	}
	user = ctx.MQ.User
	if ctx.MQ.Pass == "" {
		panic("MQ Pass is missing")
	}
	pass = ctx.MQ.Pass
	if ctx.MQ.Host == "" {
		panic("MQ Host is missing")
	}
	host = ctx.MQ.Host
	if ctx.MQ.Port < 1 {
		panic("MQ Port is missing")
	}
	port = fmt.Sprintf("%d", ctx.MQ.Port)
	vhost = ctx.MQ.Vhost
	dialaddr := scheme + "://" + user + ":" + pass + "@" + host + ":" + port + "/" + vhost

	if ctx.MQ.Timeout == "" {
		ctx.MQ.Timeout = "600s"
	}
	timeout, err := time.ParseDuration(ctx.MQ.Timeout)
	if err != nil {
		panic("Failed to parse timeout duration")
	}

	// create an AMQP configuration with specific timers
	var dialConfig amqp.Config
	dialConfig.Heartbeat = timeout
	dialConfig.Dial = func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}
	// create the TLS configuration
	if ctx.MQ.UseTLS {
		// import the client certificates
		cert, err := tls.LoadX509KeyPair(ctx.MQ.TLScert, ctx.MQ.TLSkey)
		if err != nil {
			panic(err)
		}

		// import the ca cert
		data, err := ioutil.ReadFile(ctx.MQ.CAcert)
		ca := x509.NewCertPool()
		if ok := ca.AppendCertsFromPEM(data); !ok {
			panic("failed to import CA Certificate")
		}
		TLSconfig := tls.Config{Certificates: []tls.Certificate{cert},
			RootCAs:            ca,
			InsecureSkipVerify: false,
			Rand:               rand.Reader}

		dialConfig.TLSClientConfig = &TLSconfig

	}

	// Setup the AMQP broker connection
	ctx.MQ.conn, err = amqp.DialConfig(dialaddr, dialConfig)
	if err != nil {
		panic(err)
	}

	ctx.MQ.Chan, err = ctx.MQ.conn.Channel()
	if err != nil {
		panic(err)
	}
	// declare the "mig" exchange used for all publications
	err = ctx.MQ.Chan.ExchangeDeclare("mig", "topic", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{Sev: "info", Desc: "AMQP connection opened"}
	return
}

// initChannels creates Go channels used by the disk watcher
func initChannels(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	ctx.Channels.NewAction = make(chan string, 173)
	ctx.Channels.ActionDone = make(chan string, 127)
	ctx.Channels.CommandReady = make(chan string, 991)
	ctx.Channels.UpdateCommand = make(chan string, 599)
	ctx.Channels.CommandReturned = make(chan string, 271)
	ctx.Channels.CommandDone = make(chan string, 641)
	ctx.Channels.DetectDupAgents = make(chan string, 29)
	ctx.Channels.Log = make(chan mig.Log, 601)
	ctx.Channels.Log <- mig.Log{Desc: "leaving initChannels()"}.Debug()
	return
}

// Destroy closes all the connections
func Destroy(ctx Context) {
	// close rabbitmq
	ctx.MQ.conn.Close()
	ctx.Channels.Log <- mig.Log{Sev: "info", Desc: "AMQP connection closed"}
	// close database
	ctx.DB.Close()
	ctx.Channels.Log <- mig.Log{Sev: "info", Desc: "MongoDB connection closed"}
	// close syslog
	ctx.Logging.Destroy()
	return
}
