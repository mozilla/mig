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
	"mig"
	"net"
	"os"
	"strings"
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
		fd, err := ctx.Socket.Listener.Accept()
		if err != nil {
			break
		}
		go socketServe(fd, ctx)
	}
}

// serveConn processes the request and close the connection. Connections to
// the stat socket are single-request only.
func socketServe(fd net.Conn, ctx Context) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("socketServe() -> %v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving socketServe()"}.Debug()
	}()
	var resp string
	buf := make([]byte, 8192)
	n, err := fd.Read(buf)
	if err != nil {
		return
	}
	if n > 8190 {
		resp = "Request too large. Max size is 8192 bytes\n"
	} else if n < 2 {
		resp = "Request too small. Min size is 1 byte\n"
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
			if len(fields) < 2 {
				resp = "missing agent id, shutdown refused"
				break
			}
			if fields[1] != ctx.Agent.UID {
				resp = "invalid agent id '" + fields[1] + "', shutdown refused"
				break
			}
			resp = "shutdown requested"
			ctx.Channels.Terminate <- fmt.Errorf(resp)
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
		id, err := ioutil.ReadFile(getRunDir() + ".migagtid")
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
	// remove newline
	resp = resp[0 : len(resp)-1]
	return
}
