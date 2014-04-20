/* Mozilla InvestiGator Agent

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
	"bitbucket.org/kardianos/osext"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/VividCortex/godaemon"
	"github.com/streadway/amqp"
	"io/ioutil"
	"mig"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

// Context contains all configuration variables as well as handlers for
// logs and channels
// Context is intended as a single structure that can be passed around easily.
type Context struct {
	ACL   mig.ACL
	Agent struct {
		Hostname, OS, QueueLoc, UID, BinPath string
	}
	Channels struct {
		// internal
		Terminate                           chan error
		Log                                 chan mig.Log
		NewCommand                          chan []byte
		RunAgentCommand, RunExternalCommand chan moduleOp
		Results                             chan mig.Command
	}
	MQ struct {
		// configuration
		Host, User, Pass string
		Port             int
		// internal
		UseTLS bool
		conn   *amqp.Connection
		Chan   *amqp.Channel
		Bind   mig.Binding
	}
	OpID    uint64        // ID of the current operation, used for tracking
	Sleeper time.Duration // timer used when the agent has to sleep for a while
	Stats   struct {
	}
	Logging mig.Logging
}

// Init prepare the AMQP connections to the broker and launches the
// goroutines that will process commands received by the MIG Scheduler
func Init(foreground bool) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initAgent() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initAgent()"}.Debug()
	}()

	ctx.Agent.BinPath, err = osext.Executable()
	if err != nil {
		panic(err)
	}

	// daemonize, and force logging to stdout
	if !foreground && LOGGINGCONF.Mode != "stdout" {
		godaemon.MakeDaemon(&godaemon.DaemonAttr{})
	}

	// store heartbeat frequency
	ctx.Sleeper = HEARTBEATFREQ
	if err != nil {
		panic(err)
	}

	// create the go channels
	ctx, err = initChannels(ctx)
	if err != nil {
		panic(err)
	}

	// initiate logging configuration
	ctx.Logging, err = mig.InitLogger(LOGGINGCONF)
	if err != nil {
		panic(err)
	}

	// Goroutine that handles events, such as logs and panics,
	// and decides what to do with them
	go func() {
		for event := range ctx.Channels.Log {
			stop, err := mig.ProcessLog(ctx.Logging, event)
			if err != nil {
				fmt.Println("Unable to process logs")
			}
			// if ProcessLog says we should stop now, feed the Terminate chan
			if stop {
				ctx.Channels.Terminate <- fmt.Errorf(event.Desc)
			}
		}
	}()

	// retrieve information on agent environment
	ctx, err = initAgentEnv(ctx)
	if err != nil {
		panic(err)
	}

	// parse the ACLs
	ctx, err = initACL(ctx)
	if err != nil {
		panic(err)
	}

	// connect to the message broker
	ctx, err = initMQ(ctx)
	if err != nil {
		panic(err)
	}

	return
}

func initChannels(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	ctx.Channels.Terminate = make(chan error)
	ctx.Channels.NewCommand = make(chan []byte, 7)
	ctx.Channels.RunAgentCommand = make(chan moduleOp, 5)
	ctx.Channels.RunExternalCommand = make(chan moduleOp, 5)
	ctx.Channels.Results = make(chan mig.Command, 5)
	ctx.Channels.Log = make(chan mig.Log, 97)
	ctx.Channels.Log <- mig.Log{Desc: "leaving initChannels()"}.Debug()
	return
}

// initAgentEnv retrieves information about the running system
func initAgentEnv(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initAgentEnv() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initAgentEnv()"}.Debug()
	}()

	// get the hostname
	ctx.Agent.Hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}

	// get the operating system family
	ctx.Agent.OS = runtime.GOOS

	// get the agent ID
	ctx, err = initAgentID(ctx)
	if err != nil {
		panic(err)
	}

	// build the agent message queue location
	ctx.Agent.QueueLoc = fmt.Sprintf("%s.%s.%s", ctx.Agent.OS, ctx.Agent.Hostname, ctx.Agent.UID)

	return
}

// initAgentID will retrieve an ID from disk, or request one if missing
func initAgentID(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initAgentID() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initAgentID()"}.Debug()
	}()

	// ID file location depends on the operation system
	loc := ""
	switch ctx.Agent.OS {
	case "linux":
		loc = "/var/run/mig/"
	case "windows":
		loc = "%appdata%/mig/"
	case "darwin":
		loc = "/Library/Preferences/mig/"
	}

	id, err := ioutil.ReadFile(loc + ".migagtid")
	if err != nil {
		// ID file doesn't exist, create it
		id, err = createIDFile(loc)
		if err != nil {
			panic(err)
		}
	}

	ctx.Agent.UID = fmt.Sprintf("%s", id)
	return
}

// createIDFile will generate a new ID for this agent and store it on disk
// the location depends on the operating system
func createIDFile(loc string) (id []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createIDFile() -> %v", e)
		}
	}()

	// generate an ID
	sid := mig.GenB32ID()

	// check that the storage DIR exist, or create it
	tdir, err := os.Open(loc)
	if err != nil {
		err = os.MkdirAll(loc, 0x400)
		if err != nil {
			panic(err)
		}
	} else {
		tdirMode, err := tdir.Stat()
		if err != nil {
			panic(err)
		}
		if !tdirMode.Mode().IsDir() {
			panic("Not a valid directory")
		}
	}
	tdir.Close()

	// write the ID
	err = ioutil.WriteFile(loc+".migagtid", []byte(sid), 400)
	if err != nil {
		panic(err)
	}

	// read ID from disk
	id, err = ioutil.ReadFile(loc + ".migagtid")
	if err != nil {
		panic(err)
	}

	return
}

// parse the permissions from the configuration into an ACL structure
func initACL(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initACL() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initACL()"}.Debug()
	}()
	for _, jsonPermission := range AGENTACL {
		var parsedPermission mig.Permission
		err = json.Unmarshal([]byte(jsonPermission), &parsedPermission)
		if err != nil {
			panic(err)
		}
		for permName, _ := range parsedPermission {
			desc := fmt.Sprintf("Loading permission named '%s'", permName)
			ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
		}
		ctx.ACL = append(ctx.ACL, parsedPermission)
	}
	return
}

func initMQ(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initMQ() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initMQ()"}.Debug()
	}()

	//Define the AMQP binding
	ctx.MQ.Bind.Queue = fmt.Sprintf("mig.agt.%s", ctx.Agent.QueueLoc)
	ctx.MQ.Bind.Key = fmt.Sprintf("mig.agt.%s", ctx.Agent.QueueLoc)

	// parse the dial string and use TLS if using amqps
	if strings.Contains(AMQPBROKER, "amqps://") {
		ctx.MQ.UseTLS = true
	}

	// create an AMQP configuration with specific timers
	var dialConfig amqp.Config
	dialConfig.Heartbeat = 2 * ctx.Sleeper
	dialConfig.Dial = func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, 2*ctx.Sleeper)
	}

	if ctx.MQ.UseTLS {
		// import the client certificates
		cert, err := tls.X509KeyPair([]byte(AGENTCERT), []byte(AGENTKEY))
		if err != nil {
			panic(err)
		}

		// import the ca cert
		ca := x509.NewCertPool()
		if ok := ca.AppendCertsFromPEM([]byte(CACERT)); !ok {
			panic("failed to import CA Certificate")
		}
		TLSconfig := tls.Config{Certificates: []tls.Certificate{cert},
			RootCAs:            ca,
			InsecureSkipVerify: false,
			Rand:               rand.Reader}

		dialConfig.TLSClientConfig = &TLSconfig
	}
	// Open a non-encrypted AMQP connection
	ctx.MQ.conn, err = amqp.DialConfig(AMQPBROKER, dialConfig)
	if err != nil {
		panic(err)
	}

	ctx.MQ.Chan, err = ctx.MQ.conn.Channel()
	if err != nil {
		panic(err)
	}

	// Limit the number of message the channel will receive at once
	err = ctx.MQ.Chan.Qos(7, // prefetch count (in # of msg)
		0,     // prefetch size (in bytes)
		false) // is global

	_, err = ctx.MQ.Chan.QueueDeclare(ctx.MQ.Bind.Queue, // Queue name
		true,  // is durable
		false, // is autoDelete
		false, // is exclusive
		false, // is noWait
		nil)   // AMQP args
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind(ctx.MQ.Bind.Queue, // Queue name
		ctx.MQ.Bind.Key, // Routing key name
		"mig",           // Exchange name
		false,           // is noWait
		nil)             // AMQP args
	if err != nil {
		panic(err)
	}

	// Consume AMQP message into channel
	ctx.MQ.Bind.Chan, err = ctx.MQ.Chan.Consume(ctx.MQ.Bind.Queue, // queue name
		"",    // some tag
		false, // is autoAck
		false, // is exclusive
		false, // is noLocal
		false, // is noWait
		nil)   // AMQP args
	if err != nil {
		panic(err)
	}

	return
}

func Destroy(ctx Context) (err error) {
	ctx.MQ.conn.Close()
	return
}
