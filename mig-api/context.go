// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	geo "github.com/oschwald/geoip2-golang"
	"gopkg.in/gcfg.v1"
	"io"
	"mig.ninja/mig"
	migdb "mig.ninja/mig/database"
	"strconv"
	"strings"
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
	DB      migdb.DB
	Keyring struct {
		Reader     io.ReadSeeker
		Mutex      sync.Mutex
		UpdateTime time.Time
	}
	Manifest struct {
		RequiredSignatures int
	}
	Postgres struct {
		Host, User, Password, DBName, SSLMode string
		Port, MaxConn                         int
	}
	Server struct {
		IP                       string
		Port                     int
		Host, BaseRoute, BaseURL string
		ClientPublicIP           string
		ClientPublicIPOffset     int
	}
	MaxMind struct {
		Path string
		r    *geo.Reader
	}
	Logging mig.Logging
}

// Init() initializes a context from a configuration file into an
// existing context struct
func Init(path string, debug bool) (ctx Context, err error) {
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

	// Set the mode we will use to determine a client's public IP address
	if ctx.Server.ClientPublicIP == "" {
		ctx.Server.ClientPublicIP = "peer"
	}
	ctx.Server.ClientPublicIPOffset, err = parseClientPublicIP(ctx.Server.ClientPublicIP)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	if debug {
		ctx.Logging.Level = "debug"
		ctx.Logging.Mode = "stdout"
	}
	ctx.Logging, err = mig.InitLogger(ctx.Logging, "mig-api")
	if err != nil {
		panic(err)
	}

	if ctx.Manifest.RequiredSignatures < 1 {
		panic("manifest:requiredsignatures must be at least 1 in config file")
	}

	ctx, err = initDB(ctx)
	if err != nil {
		panic(err)
	}

	if ctx.MaxMind.Path != "" {
		ctx.MaxMind.r, err = geo.Open(ctx.MaxMind.Path)
		if err != nil {
			panic(err)
		}
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
	ctx.DB.SetMaxOpenConns(ctx.Postgres.MaxConn)
	ctx.Channels.Log <- mig.Log{Desc: "Database connection opened"}
	return
}

func parseClientPublicIP(s string) (ret int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseClientPublicIP() -> %v", e)
		}
	}()

	if s == "peer" {
		return -1, nil
	}
	args := strings.Split(s, ":")
	if len(args) != 2 || args[0] != "x-forwarded-for" {
		panic("argument must be peer or x-forwarded-for:<int>")
	}
	ret, err = strconv.Atoi(args[1])
	if err != nil || ret < 0 {
		panic("x-forwarded-for argument must be positive integer or zero")
	}
	return ret, nil
}
