// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"mig.ninja/mig/mig-agent/agentcontext"
	"net"
	"os"
	"strings"
	"time"
)

func initSocket(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initSocket() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initSocket()"}.Debug()
	}()
	ctx.Socket.Listener, err = net.Listen("tcp", ctx.Socket.Bind)
	if err != nil {
		panic(err)
	}
	go socketAccept(ctx)
	return
}

func socketAccept(ctx Context) {
	for {

		conn, err := ctx.Socket.Listener.Accept()
		if err != nil {
			continue
		}
		// lock publication, serve, then exit
		publication.Lock()
		socketServe(conn, ctx)
		publication.Unlock()
		// sleep 2 seconds between accepting connections, to prevent abuses
		time.Sleep(2 * time.Second)
	}
}

// serveConn processes the request and close the connection. Connections to
// the stat socket are single-request only.
func socketServe(fd net.Conn, ctx Context) {
	var (
		resp string
		n    int
		err  error
	)
	gotdata := make(chan bool)
	defer func() {
		fd.Close()
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("socketServe() -> %v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving socketServe()"}.Debug()
	}()
	buf := make([]byte, 8192)
	go func() {
		n, err = fd.Read(buf)
		gotdata <- true
	}()
	select {
	// don't wait for more than 1 second for data to come in
	case <-time.After(1 * time.Second):
		return
	case <-gotdata:
		break
	}
	if err != nil {
		return
	}
	if n > 8190 {
		resp = "Request too long. Max size is 8192 bytes\n"
	} else if n < 2 {
		resp = "Request too short. Min size is 1 byte\n"
	} else {
		// input data can have multiple fields, space separated
		// the first field is always the request, the rest are optional parameters
		fields := strings.Split(string(buf[0:n-1]), " ")
		switch fields[0] {
		case "pid":
			resp = fmt.Sprintf("%d", os.Getpid())
		case "help":
			resp = fmt.Sprintf(`
Welcome to the MIG agent socket. The commands are:
pid		returns the PID of the running agent
shutdown <id>	request agent shutdown. <id> is the agent's secret id
`)
		case "shutdown":
			// to request a shutdown, the caller must provide the agent id
			// as second argument. If the ID is valid, the string "shutdown requested"
			// is written back to the caller, and injected into the terminate
			// channel before returning
			if len(fields) < 2 {
				resp = "missing agent id, shutdown refused"
				break
			}
			if fields[1] != ctx.Agent.UID {
				resp = "invalid agent id '" + fields[1] + "', shutdown refused"
				break
			}
			resp = "shutdown requested"
		default:
			resp = "unknown command"
		}
	}
	n, err = fd.Write([]byte(resp + "\n"))
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("serveConn(): %d bytes written", n)}.Debug()
	fd.Close()
	if resp == "shutdown requested" {
		ctx.Channels.Terminate <- resp
	}
}

func socketQuery(bind, query string) (resp string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("socketQuery() -> %v", e)
		}
	}()
	conn, err := net.Dial("tcp", bind)
	if err != nil {
		panic(err)
	}
	if query == "shutdown" {
		// attempt to read the agent secret id and append it to the shutdown order
		id, err := ioutil.ReadFile(agentcontext.GetRunDir() + ".migagtid")
		if err != nil {
			panic(err)
		}
		query = fmt.Sprintf("%s %s", query, id)
	}
	fmt.Fprintf(conn, query+"\n")
	resp, err = bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}
	conn.Close()
	// remove newline
	resp = resp[0 : len(resp)-1]
	if len(query) > 8 && query[0:8] == "shutdown" {
		fmt.Printf("agent shutdown requested, waiting for completion...")
		for {
			// loop until socket query fails, indicating the agent is dead
			time.Sleep(353 * time.Millisecond)
			fmt.Printf(".")
			conn, err = net.Dial("tcp", bind)
			if err != nil {
				resp = "done"
				return resp, nil
			}
			conn.Close()
		}
	}
	return
}
