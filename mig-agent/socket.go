// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"mig.ninja/mig/mig-agent/agentcontext"
	"net/http"
	"os"
	"time"
)

var sockCtx *Context

func initSocket(ctx *Context) {
	sockCtx = ctx
	for {
		http.HandleFunc("/pid", socketHandlePID)
		http.HandleFunc("/shutdown", socketHandleShutdown)
		err := http.ListenAndServe(ctx.Socket.Bind, nil)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Error from stat socket: %q", err)}.Err()
		}
		time.Sleep(60 * time.Second)
	}
}

func socketCheckQueueloc(req *http.Request) error {
	qv := req.Header.Get("AGENTID")
	if qv == "" {
		return fmt.Errorf("must set AGENTID header")
	}
	if sockCtx.Agent.UID != qv {
		return fmt.Errorf("invalid agent id")
	}
	return nil
}

func socketHandlePID(w http.ResponseWriter, req *http.Request) {
	publication.Lock()
	defer publication.Unlock()
	fmt.Fprintf(w, "%v", os.Getpid())
}

func socketHandleShutdown(w http.ResponseWriter, req *http.Request) {
	publication.Lock()
	defer publication.Unlock()
	err := socketCheckQueueloc(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusUnauthorized)
		return
	}
	sockCtx.Channels.Terminate <- "shutdown requested"
}

func socketQuery(bind, query string) (resp string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("socketQuery() -> %v", e)
		}
	}()

	var agtid string
	// attempt to read the agent secret id so we can append it to any orders that
	// require it, an error is not fatal here but just means we will not be able
	// to execute any privileged operations
	idbuf, _ := ioutil.ReadFile(agentcontext.GetRunDir() + ".migagtid")
	if len(idbuf) != 0 {
		agtid = string(idbuf)
	}
	client := &http.Client{}

	req, err := http.NewRequest("GET", "http://"+bind+"/"+query, nil)
	if err != nil {
		return "", err
	}
	switch query {
	case "shutdown":
		req.Header.Add("AGENTID", agtid)
		httpresp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		httpresp.Body.Close()
		// Poll the pid endpoint until a failure to wait for the agent to shutdown
		fmt.Printf("agent shutdown requested, waiting for completion...")
		resp = "done"
		for {
			time.Sleep(353 * time.Millisecond)
			fmt.Printf(".")
			req, err = http.NewRequest("GET", "http://"+bind+"/pid", nil)
			if err != nil {
				return resp, nil
			}
			httpresp, err := client.Do(req)
			if err != nil {
				return resp, nil
			}
			httpresp.Body.Close()
		}
	case "pid":
		httpresp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer httpresp.Body.Close()
		respbuf, err := ioutil.ReadAll(httpresp.Body)
		if err != nil {
			return "", err
		}
		resp = string(respbuf)
	default:
		return "", fmt.Errorf("unknown command %q", query)
	}
	return
}
