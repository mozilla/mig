// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	"io"
	"mig"
	migdb "mig/database"
	"os"
	"sync"
	"time"
)

// Context contains all configuration variables as well as handlers for
// database and logging. It also contains some statistics.
// Context is intended as a single structure that can be passed around easily.
type Context struct {
	Authentication struct {
		Enabled       bool
		TokenDuration string
		duration      time.Duration
	}
	Channels struct {
		Log chan mig.Log
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
	DB      migdb.DB
	Keyring struct {
		Reader     io.ReadSeeker
		Mutex      sync.Mutex
		UpdateTime time.Time
	}
	Postgres struct {
		Host, User, Password, DBName, SSLMode string
		Port                                  int
	}
	Server struct {
		IP                       string
		Port                     int
		Host, BaseRoute, BaseURL string
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
	ctx.Channels.Log = make(chan mig.Log, 37)

	err = gcfg.ReadFileInto(&ctx, path)
	if err != nil {
		panic(err)
	}

	ctx.Server.BaseURL = ctx.Server.Host + ctx.Server.BaseRoute
	ctx.Authentication.duration, err = time.ParseDuration(ctx.Authentication.TokenDuration)
	if err != nil {
		panic(err)
	}

	ctx.Logging, err = mig.InitLogger(ctx.Logging, "mig-api")
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

	return
}

// initDirectories() stores the directories used by the scheduler spool
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
