// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"bytes"
	"code.google.com/p/gcfg"
	"fmt"
	"io"
	"io/ioutil"
	"mig"
	migdb "mig/database"
	"mig/pgp"
	"os"
	"time"
)

// Context contains all configuration variables as well as handlers for
// database and logging. It also contains some statistics.
// Context is intended as a single structure that can be passed around easily.
type Context struct {
	OpID     float64 // ID of the current operation, used for tracking
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
	ctx.Channels.Log = make(chan mig.Log, 37)

	err = gcfg.ReadFileInto(&ctx, path)
	if err != nil {
		panic(err)
	}

	ctx.Server.BaseURL = ctx.Server.Host + ctx.Server.BaseRoute

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

// makeKeyring retrieves GPG keys of active investigators from the database and makes
// a GPG keyring out of it
func makeKeyring() (keyring io.ReadSeeker, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makeKeyring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving makeKeyring()"}.Debug()
	}()
	keys, err := ctx.DB.ActiveInvestigatorsKeys()
	if err != nil {
		panic(err)
	}
	keyring, keycount, err := pgp.ArmoredPubKeysToKeyring(keys)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("loaded %d keys from active investigators", keycount)}.Debug()
	return
}

// getKeyring copy an io.Reader from the master keyring. If the keyring hasn't been refreshed
// in a while, it also gets a fresh copy from the database
func getKeyring() (kr io.Reader, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getKeyring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving getKeyring()"}.Debug()
	}()
	// refresh keyring from DB every 5 minutes
	if ctx.Keyring.UpdateTime.Before(time.Now().Add(-5 * time.Minute)) {
		ctx.Keyring.Reader, err = makeKeyring()
		if err != nil {
			panic(err)
		}
		ctx.Keyring.UpdateTime = time.Now()
	} else {
		// rewind the master keyring
		_, err = ctx.Keyring.Reader.Seek(0, 0)
		if err != nil {
			panic(err)
		}
	}
	buf, err := ioutil.ReadAll(ctx.Keyring.Reader)
	if err != nil {
		panic(err)
	}
	kr = bytes.NewBuffer(buf)
	return
}
