// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
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
	OpID  float64 // ID of the current operation, used for tracking
	Agent struct {
		// configuration
		TimeOut, HeartbeatFreq, Whitelist string
		DetectMultiAgents                 bool
	}
	Channels struct {
		// internal
		Terminate                                             chan error
		Log                                                   chan mig.Log
		NewAction, ActionDone, UpdateCommand, CommandReturned chan string
		CommandReady, CommandDone                             chan mig.Command
		DetectDupAgents                                       chan string
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
			InFlight, Returned string
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
		Port, MaxConn                         int
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

	ctx, err = initRelay(ctx)
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
	ctx.DB.SetMaxOpenConns(ctx.Postgres.MaxConn)
	return
}

// initRelay() sets up the connection to the RabbitMQ broker
func initRelay(orig_ctx Context) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initRelay() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initRelay()"}.Debug()
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
	ctx.Channels.CommandReady = make(chan mig.Command, 13229)
	ctx.Channels.UpdateCommand = make(chan string, 6833)
	ctx.Channels.CommandReturned = make(chan string, 10559)
	ctx.Channels.CommandDone = make(chan mig.Command, 14153)
	ctx.Channels.DetectDupAgents = make(chan string, 29)
	ctx.Channels.Log = make(chan mig.Log, 21559)
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
