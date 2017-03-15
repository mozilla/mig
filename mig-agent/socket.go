// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"mig.ninja/mig"
	"mig.ninja/mig/mig-agent/agentcontext"
	"net/http"
	"os"
	"time"
)

var sockCtx *Context

var statusTmpl = `<html>
<head>
<style>
body {
  background-color: linen;
  font-family: "Lucida Console", Monaco, monospace;
  font-size: 14px;
}
div {
  padding-top: 12px;
  padding-bottom: 12px;
}
th, td {
  padding: 7px;
  text-align: left;
}
table, td {
  border-collapse: collapse;
  border: 1px solid black;
  font-size: 14px;
}
td:nth-child(2) {
  background-color: #eeeeee;
}
</style>
</head>
<body>
<h1>mig-agent</h1>
<div>
<table>
  <tr><th colspan=2>Agent status</th></tr>
  <tr><td>Agent name</td><td>{{.Context.Agent.Hostname}}</td></tr>
  <tr><td>BinPath</td><td>{{.Context.Agent.BinPath}}</td></tr>
  <tr><td>RunDir</td><td>{{.Context.Agent.RunDir}}</td></tr>
  <tr><td>PID</td><td>{{.Pid}}</td></tr>
  <tr><td>Environment</td><td>{{.Env}}</td></tr>
  <tr><td>Tags</td><td>{{.Tags}}</td></tr>
</table>
</div>
<div>
<table>
  <tr><th colspan=2>Configuration</th></tr>
  <tr><td>Immortal</td><td>{{.Immortal}}</td></tr>
  <tr><td>Install as a service</td><td>{{.InstallService}}</td></tr>
  <tr><td>Discover public IP</td><td>{{.DiscoverPublicIP}}</td></tr>
  <tr><td>Discover AWS metadata</td><td>{{.DiscoverAWSMeta}}</td></tr>
  <tr><td>Checkin mode</td><td>{{.Checkin}}</td></tr>
  <tr><td>Only verify pubkeys (no ACL verification)</td><td>{{.OnlyVerifyPubkey}}</td></tr>
  <tr><td>Extra privacy mode</td><td>{{.ExtraPrivacyMode}}</td></tr>
  <tr><td>Environment refresh period</td><td>{{.RefreshEnv}}</td></tr>
  <tr><td>Module configuration directory</td><td>{{.ModuleConfigDir}}</td></tr>
  <tr><td>Spawn persistent modules</td><td>{{.SpawnPersistent}}</td></tr>
  <tr><td>Proxies</td><td>{{.Proxies}}</td></tr>
  <tr><td>Heartbeat frequency</td><td>{{.HeartBeatFreq}}</td></tr>
  <tr><td>Module timeout</td><td>{{.ModuleTimeout}}</td></tr>
</table>
</div>
</body>
</html>
`

type templateData struct {
	Context *Context
	Pid     int
	Env     string
	Tags    string

	Immortal         bool
	InstallService   bool
	DiscoverPublicIP bool
	DiscoverAWSMeta  bool
	Checkin          bool
	OnlyVerifyPubkey bool
	ExtraPrivacyMode bool
	RefreshEnv       time.Duration
	ModuleConfigDir  string
	SpawnPersistent  bool
	Proxies          []string
	HeartBeatFreq    time.Duration
	ModuleTimeout    time.Duration
}

func (t *templateData) importAgentConfig() {
	t.Immortal = ISIMMORTAL
	t.InstallService = MUSTINSTALLSERVICE
	t.DiscoverPublicIP = DISCOVERPUBLICIP
	t.DiscoverAWSMeta = DISCOVERAWSMETA
	t.Checkin = CHECKIN
	t.OnlyVerifyPubkey = ONLYVERIFYPUBKEY
	t.ExtraPrivacyMode = EXTRAPRIVACYMODE
	t.RefreshEnv = REFRESHENV
	t.ModuleConfigDir = MODULECONFIGDIR
	t.SpawnPersistent = SPAWNPERSISTENT
	t.Proxies = PROXIES
	t.HeartBeatFreq = HEARTBEATFREQ
	t.ModuleTimeout = MODULETIMEOUT
}

func initSocket(ctx *Context) {
	sockCtx = ctx
	for {
		http.HandleFunc("/pid", socketHandlePID)
		http.HandleFunc("/shutdown", socketHandleShutdown)
		http.HandleFunc("/", socketHandleStatus)
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

func socketHandleStatus(w http.ResponseWriter, req *http.Request) {
	tdata := templateData{Context: sockCtx}
	tdata.Pid = os.Getpid()
	tdata.importAgentConfig()
	buf, err := json.Marshal(sockCtx.Agent.Env)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	tdata.Env = string(buf)
	buf, err = json.Marshal(sockCtx.Agent.Tags)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	tdata.Tags = string(buf)
	t, err := template.New("status").Parse(statusTmpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, tdata)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
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
