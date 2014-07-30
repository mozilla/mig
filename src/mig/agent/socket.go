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
	"fmt"
	"mig"
	"net"
	"os"
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
	go acceptSockConn(ctx)
	return
}

func acceptSockConn(ctx Context) {
	for {
		fd, err := ctx.Socket.Listener.Accept()
		if err != nil {
			break
		}
		go serveConn(fd, ctx)
	}
}

// serveConn processes the request and close the connection. Connections to
// the stat socket are single-request only.
func serveConn(fd net.Conn, ctx Context) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("serveConn() -> %v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving serveConn()"}.Debug()
	}()
	var resp string
	buf := make([]byte, 8192)
	n, err := fd.Read(buf)
	if err != nil {
		return
	}
	if n > 8190 {
		resp = "Request too large. Max size is 8192 bytes\n"
	} else {
		data := buf[0 : n-1]
		switch string(data) {
		case "pid":
			resp = fmt.Sprintf("%d", os.Getpid())
		case "help":
			resp = fmt.Sprintf(`
Welcome to the MIG agent socket. The commands are:
pid	returns the PID of the running agent
`)
		default:
			resp = "unknown command"
		}
	}
	n, err = fd.Write([]byte(resp))
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("serveConn(): %d bytes written", n)}.Debug()
	fd.Close()
}
