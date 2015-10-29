package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/kardianos/osext"
	"github.com/kardianos/service"

	"mig.ninja/mig"
)

func (agt *Agent) Start(s service.Service) error {
	// start in background.
	go m.Init()
}
func (agt *Agent) Init() {

}
func (agt *Agent) Stop(s service.Service) error {
	return nil
}

// Init prepare the AMQP connections to the broker and launches the
// goroutines that will process commands received by the MIG Scheduler
func (agt *Agent) Init() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("run() -> %v", e)
		}
		agt.Channels.Log <- mig.Log{Desc: "leaving run()"}.Debug()
	}()

	// TODO(mvanotti): What does this do?
	agt.Context.Tags = TAGS
	agt.initChannels()

	if err = agt.initLogging(); err != nil {
		panic(err)
	}

	// defines whether the agent should respawn itself or not
	// this value is overriden in the daemonize calls if the agent
	// is controlled by systemd, upstart or launchd
	agt.Context.Respawn = ISIMMORTAL

	if err = agt.initEnvInfo(); err != nil {
		panic(err)
	}

	// get the agent ID
	if err := agt.initAgentID(); err != nil {
		panic(err)
	}

	// build the agent message queue location
	agt.Context.QueueLoc = fmt.Sprintf("%s.%s.%s", m.ctx.Agent.Env.OS, m.ctx.Agent.Hostname, m.ctx.Agent.UID)

	agt.Sleeper = HEARTBEATFREQ

	// parse the ACLs
	if err = agt.initACL(); err != nil {
		panic(err)
	}

	// connect to MQ Relay
	if err = agt.connectMQ(); err != nil {
		panic(err)
	}

	agt.catchInterrupts()

	// try to connect the stat socket until it works
	// this may fail if one agent is already running
	if SOCKET != "" {
		go agt.connectSocket(SOCKET)
	}

	return
}

// connectSocket will try to connect to the given socket, and it will retry once a minute.
// This function blocks, so it might be useful to run it as a goroutine.
func (agt *Agent) connectSocket(socket string) {
	for {
		agt.Socket.Bind = socket
		if err = agt.initSocket(); err != nil {
			agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect stat socket: '%v'", err)}.Err()
			time.Sleep(60 * time.Second)
			continue
		}

		agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Stat socket connected successfully on %s", agt.Socket.Bind)}.Info()
		break
	}
}

func (agt *Agent) catchInterrupts() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		agt.Channels.Terminate <- sig.String()
	}()
}

// connectMQ will try to connect to the MQ relay, trying to connect
// first without proxies, then checking for a proxy in the environment, and finally checking for proxies in config.
func (agt *migAgent) connectMQ() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("connectMQ() -> %v", e)
		}
		agt.Channels.Log <- mig.Log{Desc: "leaving connectMQ()"}.Debug()
	}()

	// connect to the message broker
	if err = agt.initMQ(false, ""); err == nil {
		return
	}
	agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect to relay directly: '%v'", err)}.Debug()

	// if the connection failed, look for a proxy
	// in the environment variables, and try again
	if err = agt.initMQ(true, ""); err == nil {
		return
	}
	agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect to relay using HTTP_PROXY: '%v'", err)}.Debug()

	// still failing, try connecting using the proxies in the configuration
	for _, proxy := range PROXIES {
		if err = agt.initMQ(true, proxy); err == nil {
			return
		}
		agt.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to connect to relay using proxy %s: '%v'", proxy, err)}.Debug()
	}

	panic("Failed to connect to the relay")
}

func (agt *Agent) initLogging() (err error) {
	agt.Logging, err = mig.InitLogger(LOGGINGCONF, "mig-agent")
	if err != nil {
		panic(err)
	}

	// Logging GoRoutine,
	go func() {
		for event := range agt.Channels.Log {
			_, err := mig.ProcessLog(m.ctx.Logging, event)
			if err != nil {
				fmt.Println("Unable to process logs")
			}
		}
	}()
	agt.Channels.Log <- mig.Log{Desc: "Logging routine initialized."}.Debug()
}

// initEnvInfo tries to load information related to the environment in which the mig agent is running.
func (agt *Agent) initEnvInfo() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getEnvInfo() -> %v", e)
		}
		agt.Channels.Log <- mig.Log{Desc: "leaving getEnvInfo()"}.Debug()
	}()

	// get the path of the executable
	agt.Context.BinPath, err = osext.Executable()
	if err != nil {
		panic(err)
	}

	// retrieve the hostname
	if err = agt.getHostname(ctx); err != nil {
		panic(err)
	}

	// retrieve information about the operating system
	m.ctx.Agent.Env.OS = runtime.GOOS
	m.ctx.Agent.Env.Arch = runtime.GOARCH
	m.ctx, err = findOSInfo(ctx)
	if err != nil {
		panic(err)
	}
	m.ctx, err = findLocalIPs(ctx)
	if err != nil {
		panic(err)
	}

	// Attempt to discover the public IP
	if DISCOVERPUBLICIP {
		m.ctx, err = findPublicIP(ctx)
		if err != nil {
			panic(err)
		}
	}

	// find the run directory
	m.ctx.Agent.RunDir = getRunDir()
}
