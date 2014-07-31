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
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mig"
	"net"
	"net/http"
	"os"
	"time"

	"bitbucket.org/kardianos/osext"
	"bitbucket.org/kardianos/service"
	"github.com/streadway/amqp"
)

// Context contains all configuration variables as well as handlers for
// logs and channels
// Context is intended as a single structure that can be passed around easily.
type Context struct {
	ACL   mig.ACL
	Agent struct {
		Hostname, OS, QueueLoc, UID, BinPath, RunDir string
		Respawn                                      bool
		Env                                          mig.AgentEnv
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
		Bind   struct {
			Queue, Key string
			Chan       <-chan amqp.Delivery
		}
	}
	OpID    float64       // ID of the current operation, used for tracking
	Sleeper time.Duration // timer used when the agent has to sleep for a while
	Socket  struct {
		Bind     string
		Listener net.Listener
	}
	Logging mig.Logging
}

// Init prepare the AMQP connections to the broker and launches the
// goroutines that will process commands received by the MIG Scheduler
func Init(foreground, upgrade bool) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initAgent() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initAgent()"}.Debug()
	}()

	ctx.Logging, err = mig.InitLogger(LOGGINGCONF, "mig-agent")
	if err != nil {
		panic(err)
	}

	// defines whether the agent should respawn itself or not
	// this value is overriden in the daemonize calls if the agent
	// is controlled by systemd, upstart or launchd
	ctx.Agent.Respawn = ISIMMORTAL

	// create the go channels
	ctx, err = initChannels(ctx)
	if err != nil {
		panic(err)
	}

	// Logging GoRoutine,
	go func() {
		for event := range ctx.Channels.Log {
			_, err := mig.ProcessLog(ctx.Logging, event)
			if err != nil {
				fmt.Println("Unable to process logs")
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "Logging routine initialized."}.Debug()

	// get the path of the executable
	ctx.Agent.BinPath, err = osext.Executable()
	if err != nil {
		panic(err)
	}

	// retrieve the hostname
	ctx, err = findHostname(ctx)
	if err != nil {
		panic(err)
	}

	// retrieve information about the operating system
	ctx, err = findOSInfo(ctx)
	if err != nil {
		panic(err)
	}

	// find the run directory
	ctx.Agent.RunDir = getRunDir()

	// get the agent ID
	ctx, err = initAgentID(ctx)
	if err != nil {
		panic(err)
	}

	// build the agent message queue location
	ctx.Agent.QueueLoc = fmt.Sprintf("%s.%s.%s", ctx.Agent.OS, ctx.Agent.Hostname, ctx.Agent.UID)

	// daemonize if not in foreground mode
	if !foreground {
		// give one second for the caller to exit
		time.Sleep(time.Second)
		ctx, err = daemonize(ctx, upgrade)
		if err != nil {
			panic(err)
		}
	}

	ctx.Sleeper = HEARTBEATFREQ
	if err != nil {
		panic(err)
	}

	// parse the ACLs
	ctx, err = initACL(ctx)
	if err != nil {
		panic(err)
	}

	connected := false
	// connect to the message broker
	ctx, err = initMQ(ctx, false, "")
	if err != nil {
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect to relay directly: '%v'", err)}.Debug()
		// if the connection failed, look for a proxy
		// in the environment variables, and try again
		ctx, err = initMQ(ctx, true, "")
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect to relay using HTTP_PROXY: '%v'", err)}.Debug()
			// still failing, try connecting using the proxies in the configuration
			for _, proxy := range PROXIES {
				ctx, err = initMQ(ctx, true, proxy)
				if err != nil {
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect to relay using proxy %s: '%v'", proxy, err)}.Debug()
					continue
				}
				connected = true
				goto mqdone
			}
		} else {
			connected = true
		}
	} else {
		connected = true
	}
mqdone:
	if !connected {
		panic("Failed to connect to the relay")
	}

	// try to connect the stat socket until it works
	// this may fail if one agent is already running
	if SOCKET != "" {
		go func() {
			for {
				ctx.Socket.Bind = SOCKET
				ctx, err = initSocket(ctx)
				if err == nil {
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Stat socket connected successfully on %s", ctx.Socket.Bind)}.Info()
					goto socketdone
				}
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect stat socket: '%v'", err)}.Err()
				time.Sleep(60 * time.Second)
			}
		socketdone:
			return
		}()
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

// initAgentID will retrieve an ID from disk, or request one if missing
func initAgentID(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initAgentID() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initAgentID()"}.Debug()
	}()
	os.Chmod(ctx.Agent.RunDir, 0755)
	idFile := ctx.Agent.RunDir + ".migagtid"
	id, err := ioutil.ReadFile(idFile)
	if err != nil {
		// ID file doesn't exist, create it
		id, err = createIDFile(ctx)
		if err != nil {
			panic(err)
		}
	}
	ctx.Agent.UID = fmt.Sprintf("%s", id)
	os.Chmod(idFile, 0644)
	return
}

// createIDFile will generate a new ID for this agent and store it on disk
// the location depends on the operating system
func createIDFile(ctx Context) (id []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createIDFile() -> %v", e)
		}
	}()
	// generate an ID
	sid := mig.GenB32ID()
	// check that the storage DIR exist, and that it's a dir
	tdir, err := os.Open(ctx.Agent.RunDir)
	defer tdir.Close()
	if err != nil {
		// dir doesn't exist, create it
		err = os.MkdirAll(ctx.Agent.RunDir, 0755)
		if err != nil {
			panic(err)
		}
	} else {
		// open worked, verify that it's a dir
		tdirMode, err := tdir.Stat()
		if err != nil {
			panic(err)
		}
		if !tdirMode.Mode().IsDir() {
			// not a valid dir. destroy whatever it is, and recreate
			err = os.Remove(ctx.Agent.RunDir)
			if err != nil {
				panic(err)
			}
			err = os.MkdirAll(ctx.Agent.RunDir, 0755)
			if err != nil {
				panic(err)
			}
		}
	}

	idFile := ctx.Agent.RunDir + ".migagtid"

	// something exists at the location of the id file, just plain remove it
	_ = os.Remove(idFile)

	// write the ID file
	err = ioutil.WriteFile(idFile, []byte(sid), 0644)
	if err != nil {
		panic(err)
	}
	// read ID from disk
	id, err = ioutil.ReadFile(idFile)
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

func initMQ(orig_ctx Context, try_proxy bool, proxy string) (ctx Context, err error) {
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
	amqp_uri, err := amqp.ParseURI(AMQPBROKER)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("AMQP: host=%s, port=%d, vhost=%s", amqp_uri.Host, amqp_uri.Port, amqp_uri.Vhost)}.Debug()
	if amqp_uri.Scheme == "amqps" {
		ctx.MQ.UseTLS = true
	}

	// create an AMQP configuration with specific timers
	var dialConfig amqp.Config
	dialConfig.Heartbeat = 2 * ctx.Sleeper
	if try_proxy {
		// if in try_proxy mode, the agent will try to connect to the relay using a CONNECT proxy
		// but because CONNECT is a HTTP method, not available in AMQP, we need to establish
		// that connection ourselves, and give it back to the amqp.DialConfig method
		if proxy == "" {
			// try to get the proxy from the environemnt (variable HTTP_PROXY)
			target := "http://" + amqp_uri.Host + ":" + fmt.Sprintf("%d", amqp_uri.Port)
			req, err := http.NewRequest("GET", target, nil)
			if err != nil {
				panic(err)
			}
			proxy_url, err := http.ProxyFromEnvironment(req)
			if err != nil {
				panic(err)
			}
			if proxy_url == nil {
				panic("Failed to find a suitable proxy in environment")
			}
			proxy = proxy_url.Host
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Found proxy at %s", proxy)}.Debug()
		}
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Connecting via proxy %s", proxy)}.Debug()
		dialConfig.Dial = func(network, addr string) (conn net.Conn, err error) {
			// connect to the proxy
			conn, err = net.DialTimeout("tcp", proxy, 5*time.Second)
			if err != nil {
				return
			}
			// write a CONNECT request in the tcp connection
			fmt.Fprintf(conn, "CONNECT "+addr+" HTTP/1.1\r\nHost: "+addr+"\r\n\r\n")
			// verify status is 200, and flush the buffer
			status, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				return
			}
			if status == "" || len(status) < 12 {
				err = fmt.Errorf("Invalid status received from proxy: '%s'", status[0:len(status)-2])
				return
			}
			// 9th character in response should be "2"
			// HTTP/1.0 200 Connection established
			//          ^
			if status[9] != '2' {
				err = fmt.Errorf("Invalid status received from proxy: '%s'", status[0:len(status)-2])
				return
			}
			return
		}
	} else {
		dialConfig.Dial = func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 5*time.Second)
		}
	}

	if ctx.MQ.UseTLS {
		ctx.Channels.Log <- mig.Log{Desc: "Loading AMQPS TLS parameters"}.Debug()
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
	// Open AMQP connection
	ctx.Channels.Log <- mig.Log{Desc: "Establishing connection to relay"}.Debug()
	ctx.MQ.conn, err = amqp.DialConfig(AMQPBROKER, dialConfig)
	if err != nil {
		ctx.Channels.Log <- mig.Log{Desc: "Connection failed"}.Debug()
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
	close(ctx.Channels.Terminate)
	close(ctx.Channels.NewCommand)
	close(ctx.Channels.RunAgentCommand)
	close(ctx.Channels.RunExternalCommand)
	close(ctx.Channels.Results)
	close(ctx.Channels.Log)
	ctx.MQ.conn.Close()
	return
}

// serviceDeploy stops, removes, installs and start the mig-agent service in one go
func serviceDeploy(orig_ctx Context) (ctx Context, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("serviceDeploy() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving serviceDeploy()"}.Debug()
	}()
	svc, err := service.NewService("mig-agent", "MIG Agent", "Mozilla InvestiGator Agent")
	if err != nil {
		panic(err)
	}
	// if already running, stop it
	_ = svc.Stop()
	err = svc.Remove()
	if err != nil {
		// fail but continue, the service may not exist yet
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Service removal failed: '%v'", err)}.Err()
	}
	err = svc.Install()
	if err != nil {
		panic(err)
	}
	err = svc.Start()
	if err != nil {
		panic(err)
	}
	return
}

// cleanString removes spaces, quotes and newlines
func cleanString(str string) string {
	if len(str) < 1 {
		return str
	}
	if str[len(str)-1] == '\n' {
		str = str[0 : len(str)-1]
	}
	// remove heading whitespaces and quotes
	for {
		if len(str) < 2 {
			break
		}
		switch str[0] {
		case ' ', '"', '\'':
			str = str[1:len(str)]
		default:
			goto trailing
		}
	}
trailing:
	// remove trailing whitespaces, quotes and linebreaks
	for {
		if len(str) < 2 {
			break
		}
		switch str[len(str)-1] {
		case ' ', '"', '\'', '\r', '\n':
			str = str[0 : len(str)-1]
		default:
			goto exit
		}
	}
exit:
	return str
}
