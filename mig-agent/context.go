// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	mrand "math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jvehent/service-go"
	"github.com/streadway/amqp"
	"mig.ninja/mig"
)

type Agent struct {
	ACL    mig.acl
	Config struct {
	}
	Context struct {
		Hostname, Queueloc, Mode, UID, BinPath, RunDir string
		Respawn                                        bool
		CheckIn                                        bool
		Env                                            mig.AgentEnv
		Tags                                           interface{} // tags are free structs that are used to mark a group of agents, used to identify agents families by business units.
	}
	Channels struct {
		// internal
		Terminate                           chan string
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

// initChannels creates all the channels needed by a mig agent.
func (agt *Agent) initChannels() {
	agt.Channels.Terminate = make(chan string)
	agt.Channels.NewCommand = make(chan []byte, 7)
	agt.Channels.RunAgentCommand = make(chan moduleOp, 5)
	agt.Channels.RunExternalCommand = make(chan moduleOp, 5)
	agt.Channels.Results = make(chan mig.Command, 5)
	agt.Channels.Log = make(chan mig.Log, 97)
	agt.Channels.Log <- mig.Log{Desc: "leaving initChannels()"}.Debug()
	return
}

// initAgentID will retrieve an ID from disk, or create one if missing
func (agt *Agent) initAgentID() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initAgentID() -> %v", e)
		}
		agt.Channels.Log <- mig.Log{Desc: "leaving initAgentID()"}.Debug()
	}()
	os.Chmod(agt.Context.RunDir, 0755)
	idFile := agt.Context.RunDir + ".migagtid"
	id, err := ioutil.ReadFile(idFile)
	if err != nil {
		agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("unable to read agent id from '%s': %v", idFile, err)}.Debug()
		// ID file doesn't exist, create it
		id, err = agt.createIDFile()
		if err != nil {
			panic(err)
		}
	}
	agt.Context.UID = fmt.Sprintf("%s", id)
	os.Chmod(idFile, 0400)
	return
}

// createIDFile will generate a new ID for this agent and store it on disk
// the location depends on the operating system
func (agt *Agent) createIDFile() (id []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createIDFile() -> %v", e)
		}
	}()
	// generate an ID
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	sid := strconv.FormatUint(uint64(r.Int63()), 36)
	sid += strconv.FormatUint(uint64(r.Int63()), 36)
	sid += strconv.FormatUint(uint64(r.Int63()), 36)
	sid += strconv.FormatUint(uint64(r.Int63()), 36)
	runDir := agt.Context.RunDir

	// check that the storage DIR exist, and that it's a dir
	tdir, err := os.Open(runDir)
	defer tdir.Close()
	if err != nil {
		// dir doesn't exist, create it
		agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("agent rundir is missing from '%s'. creating it", runDir)}.Debug()
		err = os.MkdirAll(runDir, 0755)
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
			agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("'%s' is not a directory. removing it", runDir)}.Debug()
			// not a valid dir. destroy whatever it is, and recreate
			err = os.Remove(runDir)
			if err != nil {
				panic(err)
			}
			err = os.MkdirAll(runDir, 0755)
			if err != nil {
				panic(err)
			}
		}
	}

	idFile := runDir + ".migagtid"

	// something exists at the location of the id file, just plain remove it
	_ = os.Remove(idFile)

	// write the ID file
	if err = ioutil.WriteFile(idFile, []byte(sid), 0400); err != nil {
		panic(err)
	}
	// read ID from disk
	if _, err = ioutil.ReadFile(idFile); err != nil {
		panic(err)
	}

	agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("agent id created in '%s'", idFile)}.Debug()
	return
}

// parse the permissions from the configuration into an ACL structure
func (agt *Agent) initACL() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initACL() -> %v", e)
		}
		agt.Channels.Log <- mig.Log{Desc: "leaving initACL()"}.Debug()
	}()
	for _, jsonPermission := range AGENTACL {
		var parsedPermission mig.Permission
		err = json.Unmarshal([]byte(jsonPermission), &parsedPermission)
		if err != nil {
			panic(err)
		}
		for permName, _ := range parsedPermission {
			desc := fmt.Sprintf("Loading permission named '%s'", permName)
			a.Channels.Log <- mig.Log{Desc: desc}.Debug()
		}
		agt.ACL = append(agt.ACL, parsedPermission)
	}
	return
}

func (agt *Agent) initMQ(try_proxy bool, proxy string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initMQ() -> %v", e)
		}
		agt.Channels.Log <- mig.Log{Desc: "leaving initMQ()"}.Debug()
	}()

	//Define the AMQP binding
	agt.MQ.Bind.Queue = fmt.Sprintf("mig.agt.%s", agt.Context.QueueLoc)
	agt.MQ.Bind.Key = fmt.Sprintf("mig.agt.%s", agt.Context.QueueLoc)

	// parse the dial string and use TLS if using amqps
	amqp_uri, err := amqp.ParseURI(AMQPBROKER)
	if err != nil {
		panic(err)
	}
	agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("AMQP: host=%s, port=%d, vhost=%s", amqp_uri.Host, amqp_uri.Port, amqp_uri.Vhost)}.Debug()
	if amqp_uri.Scheme == "amqps" {
		agt.MQ.UseTLS = true
	}

	// create an AMQP configuration with specific timers
	var dialConfig amqp.Config
	dialConfig.Heartbeat = 2 * agt.Sleeper
	if try_proxy {
		// if in try_proxy mode, the agent will try to connect to the relay using a CONNECT proxy
		// but because CONNECT is a HTTP method, not available in AMQP, we need to establish
		// that connection ourselves, and give it back to the amqp.DialConfig method
		if proxy == "" {
			// try to get the proxy from the environemnt (variable HTTP_PROXY)
			target := fmt.Sprintf("http://%s:%d", amqp_uri.Host, amqp_uri.Port)
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
			agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Found proxy at %s", proxy)}.Debug()
		}
		agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Connecting via proxy %s", proxy)}.Debug()
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
			agt.Context.Env.IsProxied = true
			agt.Context.Env.Proxy = proxy
			return
		}
	} else {
		dialConfig.Dial = func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 5*time.Second)
		}
	}

	if agt.MQ.UseTLS {
		agt.Channels.Log <- mig.Log{Desc: "Loading AMQPS TLS parameters"}.Debug()
		// import the client certificates
		cert, err := tls.X509KeyPair(AGENTCERT, AGENTKEY)
		if err != nil {
			panic(err)
		}

		// import the ca cert
		ca := x509.NewCertPool()
		if ok := ca.AppendCertsFromPEM(CACERT); !ok {
			panic("failed to import CA Certificate")
		}
		TLSconfig := tls.Config{Certificates: []tls.Certificate{cert},
			RootCAs:            ca,
			InsecureSkipVerify: false,
			Rand:               rand.Reader}

		dialConfig.TLSClientConfig = &TLSconfig
	}
	// Open AMQP connection
	agt.Channels.Log <- mig.Log{Desc: "Establishing connection to relay"}.Debug()
	agt.MQ.conn, err = amqp.DialConfig(AMQPBROKER, dialConfig)
	if err != nil {
		agt.Channels.Log <- mig.Log{Desc: "Connection failed"}.Debug()
		panic(err)
	}

	agt.MQ.Chan, err = agt.MQ.conn.Channel()
	if err != nil {
		panic(err)
	}

	// Limit the number of message the channel will receive at once
	err = agt.MQ.Chan.Qos(1, // prefetch count (in # of msg)
		0,     // prefetch size (in bytes)
		false) // is global

	_, err = agt.MQ.Chan.QueueDeclare(agt.MQ.Bind.Queue, // Queue name
		true,  // is durable
		false, // is autoDelete
		false, // is exclusive
		false, // is noWait
		nil)   // AMQP args
	if err != nil {
		panic(err)
	}

	err = agt.MQ.Chan.QueueBind(agt.MQ.Bind.Queue, // Queue name
		agt.MQ.Bind.Key,    // Routing key name
		mig.Mq_Ex_ToAgents, // Exchange name
		false,              // is noWait
		nil)                // AMQP args
	if err != nil {
		panic(err)
	}

	// Consume AMQP message into channel
	agt.MQ.Bind.Chan, err = agt.MQ.Chan.Consume(agt.MQ.Bind.Queue, // queue name
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

func (agt *Agent) Destroy() (err error) {
	close(agt.Channels.Terminate)
	close(agt.Channels.NewCommand)
	close(agt.Channels.RunAgentCommand)
	close(agt.Channels.RunExternalCommand)
	close(agt.Channels.Results)
	// give one second for the goroutines to close
	time.Sleep(1 * time.Second)
	agt.MQ.conn.Close()
	return
}

// serviceDeploy stops, removes, installs and start the mig-agent service in one go
func (agt *Agent) serviceDeploy() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("serviceDeploy() -> %v", e)
		}
		agt.Channels.Log <- mig.Log{Desc: "leaving serviceDeploy()"}.Debug()
	}()

	svcConfig := &service.Config{
		Name:        "mig-agent",
		DisplayName: "MIG Agent",
		Description: "Mozilla InvestiGator Agent",
	}

	svc, err := service.New(agt, svcConfig)
	if err != nil {
		panic(err)
	}

	// TODO: FIX THIS. it appears that stopping a service on upstart will kill both the agent
	// running as a service, and the agent currently upgrading which isn't yet running as a service.

	// if already running, stop it. don't panic on error
	err = svc.Stop()
	if err != nil {
		agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to stop service mig-agent: '%v'", err)}.Info()
	} else {
		agt.Channels.Log <- mig.Log{Desc: "Stopped running mig-agent service"}.Info()
	}

	err = svc.Remove()
	if err != nil {
		// fail but continue, the service may not exist yet
		agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to remove service mig-agent: '%v'", err)}.Info()
	} else {
		agt.Channels.Log <- mig.Log{Desc: "Removed existing mig-agent service"}.Info()
	}
	err = svc.Install()
	if err != nil {
		panic(err)
	}
	agt.Channels.Log <- mig.Log{Desc: "Installed mig-agent service"}.Info()
	err = svc.Start()
	if err != nil {
		panic(err)
	}
	agt.Channels.Log <- mig.Log{Desc: "Started mig-agent service"}.Info()
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
	// remove in-string linebreaks
	str = strings.Replace(str, "\n", " ", -1)
	str = strings.Replace(str, "\r", " ", -1)
	return str
}
