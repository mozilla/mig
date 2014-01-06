package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	"github.com/streadway/amqp"
	"labix.org/v2/mgo"
	"mig"
	"os"
)

// Context contains all configuration variables as well as handlers for
// database and message brokers. It also contains some statistics.
// Context is intended as a single structure that can be passed around easily.
type Context struct {
	OpID uint64	// ID of the current operation, used for tracking
	Agent struct {
		// configuration
		TimeOut, Whitelist string
	}
	Channels struct {
		// internal
		Terminate chan error
		Log chan mig.Log
		NewAction, ActionDone, CommandReady, UpdateCommand, CommandReturned, CommandDone chan string
	}
	Directories struct {
		// configuration
		Spool string
		Tmp string
		// internal
		Action struct {
			New, InFlight, Done, Invalid string
		}
		Command struct {
			Ready, InFlight, Returned, Done string
		}
	}
	DB struct {
		// configuration
		Host, User, Pass string
		Port int
		UseTLS bool
		// internal
		session *mgo.Session
		Col struct {
			Reg, Action, Cmd *mgo.Collection
		}
	}
	MQ struct {
		// configuration
		Host, User, Pass string
		Port int
		UseTLS bool
		// internal
		conn *amqp.Connection
		Chan *amqp.Channel
	}
	Stats struct {
	}
	Logging mig.Logging

}

// Init() initializes a context from a configuration file into an
// existing context struct
func Init(path string) (ctx Context, err error){
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

	ctx.Logging, err = mig.InitLogger(ctx.Logging)
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


// initDB() sets up the connection to the MongoDB backend database
func initDB(orig_ctx Context) (ctx Context, err error){
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initDB() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initDB()"}.Debug()
	}()

	ctx = orig_ctx
	ctx.DB.session, err = mgo.Dial(ctx.DB.Host)
	if err != nil {
		panic(err)
	}

	ctx.DB.session.SetSafe(&mgo.Safe{}) // make safe writes only

	// create handlers to collections
	ctx.DB.Col.Reg		= ctx.DB.session.DB("mig").C("registrations")
	ctx.DB.Col.Action	= ctx.DB.session.DB("mig").C("actions")
	ctx.DB.Col.Cmd		= ctx.DB.session.DB("mig").C("commands")

	ctx.Channels.Log <- mig.Log{Sev: "info", Desc: "MongoDB connection opened"}
	return
}

// initBroker() sets up the connection to the RabbitMQ broker
func initBroker(orig_ctx Context) (ctx Context, err error){
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initBroker() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initBroker()"}.Debug()
	}()

	ctx = orig_ctx
	// dialing address use format "<scheme>://<user>:<pass>@<host>:<port>/"
	var scheme, user, pass, host, port string
	if ctx.MQ.UseTLS {
		panic("TLS AMQPS mode not supported")
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

	dialaddr := scheme + "://" + user + ":" + pass + "@" + host + ":" + port + "/"

	// Setup the AMQP broker connection
	ctx.MQ.conn, err = amqp.Dial(dialaddr)
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
	ctx.Channels.NewAction		= make(chan string, 173)
	ctx.Channels.ActionDone		= make(chan string, 127)
	ctx.Channels.CommandReady	= make(chan string, 991)
	ctx.Channels.UpdateCommand	= make(chan string, 599)
	ctx.Channels.CommandReturned	= make(chan string, 271)
	ctx.Channels.CommandDone	= make(chan string, 641)
	ctx.Channels.Log		= make(chan mig.Log, 601)
	ctx.Channels.Log <- mig.Log{Desc: "leaving initChannels()"}.Debug()
	return
}

// Destroy closes all the connections
func Destroy(ctx Context) {
	// close rabbitmq
	ctx.MQ.conn.Close()
	ctx.Channels.Log <- mig.Log{Sev: "info", Desc: "AMQP connection closed"}
	// close mongodb
	ctx.DB.session.Close()
	ctx.Channels.Log <- mig.Log{Sev: "info", Desc: "MongoDB connection closed"}
	// close syslog
	ctx.Logging.Destroy()
	return
}
