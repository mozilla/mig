/* Mozilla InvestiGator API

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
Portions created by the Initial Developer are Copyright (C) 2013
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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mig"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jvehent/cljs"
	"github.com/mozilla/mig/src/mig/pgp"
)

var ctx Context

func main() {
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)

	// command line options
	var config = flag.String("c", "/etc/mig/api.cfg", "Load configuration from file")
	flag.Parse()

	// The context initialization takes care of parsing the configuration,
	// and creating connections to database, syslog, ...
	fmt.Fprintf(os.Stderr, "Initializing API context...")
	var err error
	ctx, err = Init(*config)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "OK\n")

	// Goroutine that handles events, such as logs and panics,
	// and decides what to do with them
	go func() {
		for event := range ctx.Channels.Log {
			stop, err := mig.ProcessLog(ctx.Logging, event)
			if err != nil {
				panic("Unable to process logs")
			}
			// if ProcessLog says we should stop
			if stop {
				panic("Logger routine asked to stop")
			}
		}
	}()
	ctx.Channels.Log <- mig.Log{Desc: "mig.ProcessLog() routine started"}

	// register routes
	r := mux.NewRouter()
	s := r.PathPrefix(ctx.Server.BaseRoute).Subrouter()
	s.HandleFunc("/", getHome).Methods("GET")
	s.HandleFunc("/search", search).Methods("GET")
	s.HandleFunc("/action", getAction).Methods("GET")
	s.HandleFunc("/action/create/", describeCreateAction).Methods("GET")
	s.HandleFunc("/action/create/", createAction).Methods("POST")
	s.HandleFunc("/action/cancel/", describeCancelAction).Methods("GET")
	s.HandleFunc("/action/cancel/", cancelAction).Methods("POST")
	s.HandleFunc("/command", getCommand).Methods("GET")
	s.HandleFunc("/command/cancel/", describeCancelCommand).Methods("GET")
	s.HandleFunc("/command/cancel/", cancelCommand).Methods("POST")
	s.HandleFunc("/agent/dashboard", getAgentsDashboard).Methods("GET")
	s.HandleFunc("/agent/search", searchAgents).Methods("GET")

	// all set, start the http handler
	http.Handle("/", r)
	listenAddr := fmt.Sprintf("%s:%d", ctx.Server.IP, ctx.Server.Port)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

// respond builds a Collection+JSON body and sends it to the client
func respond(code int, response *cljs.Resource, respWriter http.ResponseWriter, request *http.Request, opid uint64) (err error) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving respond()"}.Debug()
	}()
	body, err := response.Marshal()
	if err != nil {
		panic(err)
	}

	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.WriteHeader(code)
	respWriter.Write(body)

	return
}

// getHome returns a basic document that presents the different ressources
// available in the API, as well as some status information
func getHome(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getHome()"}.Debug()
	}()

	// List the creation URL. Those can be GET-ed to retrieve the creation templates
	err = resource.AddLink(cljs.Link{
		Rel:  "create action",
		Href: fmt.Sprintf("%s/action/create/", ctx.Server.BaseURL),
		Name: "Create an action"})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "cancel action",
		Href: fmt.Sprintf("%s/action/cancel/", ctx.Server.BaseURL),
		Name: "Cancel an action"})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "cancel command",
		Href: fmt.Sprintf("%s/command/cancel/", ctx.Server.BaseURL),
		Name: "Cancel a command"})
	if err != nil {
		panic(err)
	}

	// Describe the queries that are exposed to the client
	err = resource.AddQuery(cljs.Query{
		Rel:    "Query action by ID",
		Href:   fmt.Sprintf("%s/action", ctx.Server.BaseURL),
		Prompt: "Query action by ID",
		Data: []cljs.Data{
			{Name: "actionid", Value: "[0-9]{1,20}", Prompt: "Action ID"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:    "Query command by ID",
		Href:   fmt.Sprintf("%s/command", ctx.Server.BaseURL),
		Prompt: "Query command by ID",
		Data: []cljs.Data{
			{Name: "commandid", Value: "[0-9]{1,20}", Prompt: "Command ID"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:    "Search agent by name",
		Href:   fmt.Sprintf("%s/agent/search", ctx.Server.BaseURL),
		Prompt: "Search agent by name",
		Data: []cljs.Data{
			{Name: "name", Value: "agent123.example.net", Prompt: "Agent Name"},
		},
	})
	if err != nil {
		panic(err)
	}

	err = resource.AddQuery(cljs.Query{
		Rel:    "Query MIG data",
		Href:   fmt.Sprintf("%s/search", ctx.Server.BaseURL),
		Prompt: "Query MIG data",
		Data: []cljs.Data{
			{Name: "actionid", Value: "[0-9]{1,20}", Prompt: "Action ID"},
			{Name: "search", Value: "positiveresults, ...", Prompt: "Name of search query"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:  "Get agent dashboard",
		Href: fmt.Sprintf("%s/agent/dashboard", ctx.Server.BaseURL),
	})
	if err != nil {
		panic(err)
	}

	respond(200, resource, respWriter, request, opid)
}

// search is a generic function to run queries against mongodb
func search(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving describeCreateAction()"}.Debug()
	}()

	search := request.URL.Query()["search"][0]
	switch search {
	case "positiveresults":
		resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: "Not implemented"})
		respond(401, resource, respWriter, request, opid)
	default:
		resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: "Not implemented"})
		respond(401, resource, respWriter, request, opid)
	}
}

// describeCreateAction returns a resource that describes how to POST new actions
func describeCreateAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving describeCreateAction()"}.Debug()
	}()

	err := resource.SetTemplate(cljs.Template{
		Data: []cljs.Data{
			{Name: "action", Value: "URL encoded signed MIG action", Prompt: "Signed MIG Action"},
		},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request, opid)
}

// createAction receives a signed action in a POST request, validates it,
// and write it into the scheduler spool
func createAction(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := mig.GenID()
	var action mig.Action
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "leaving createAction()"}.Debug()
	}()

	// parse the POST body into a mig action
	request.ParseForm()
	postAction := request.FormValue("action")
	err = json.Unmarshal([]byte(postAction), &action)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received action for creation '%s'", action)}.Debug()

	// Init action fields
	action.ID = mig.GenID()
	date0 := time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
	action.StartTime = date0
	action.FinishTime = date0
	action.LastUpdateTime = date0
	action.Status = "init"

	// load keyring and validate action
	keyring, err := os.Open(ctx.PGP.PubRing)
	if err != nil {
		panic(err)
	}
	defer keyring.Close()

	err = action.Validate()
	if err != nil {
		panic(err)
	}
	err = action.VerifySignatures(keyring)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "Received new action with valid signature"}

	// write action to database
	err = ctx.DB.InsertAction(action)
	if err != nil {
		panic(err)
	}
	// write signatures to database
	astr, err := action.String()
	if err != nil {
		panic(err)
	}
	for _, sig := range action.PGPSignatures {
		// TODO: opening the keyring in a loop is really ugly. rewind!
		k, err := os.Open(ctx.PGP.PubRing)
		if err != nil {
			panic(err)
		}
		defer k.Close()
		fp, err := pgp.GetFingerprintFromSignature(astr, sig, k)
		if err != nil {
			panic(err)
		}
		iid, err := ctx.DB.InvestigatorByFingerprint(fp)
		if err != nil {
			panic(err)
		}
		err = ctx.DB.InsertSignature(action.ID, iid, sig)
		if err != nil {
			panic(err)
		}
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "Action written to database"}

	// write action to disk
	destdir := fmt.Sprintf("%s/%d.json", ctx.Directories.Action.New, action.ID)
	newAction, err := json.Marshal(action)
	if err != nil {
		panic(err)
	}
	err = safeWrite(opid, destdir, newAction)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "Action committed to spool"}

	err = resource.AddItem(cljs.Item{
		Href: fmt.Sprintf("%s/action?actionid=%d", ctx.Server.BaseURL, action.ID),
		Data: []cljs.Data{{Name: "action ID " + fmt.Sprintf("%d", action.ID), Value: action}},
	})
	if err != nil {
		panic(err)
	}
	respond(201, resource, respWriter, request, opid)
}

// describeCancelAction returns a resource that describes how to cancel an action
func describeCancelAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving describeCancelAction()"}.Debug()
	}()

	err := resource.SetTemplate(cljs.Template{
		Data: []cljs.Data{
			{Name: "id", Value: "[0-9]{1,20}", Prompt: "Action ID"},
		},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request, opid)
}

// cancelAction receives an action ID and issue a cancellation order
func cancelAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving cancelAction()"}.Debug()
	}()
	respond(501, resource, respWriter, request, opid)
}

// getAction queries the database and retrieves the detail of an action
func getAction(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAction()"}.Debug()
	}()
	actionID, err := strconv.ParseUint(request.URL.Query()["actionid"][0], 10, 64)
	if err != nil {
		panic(err)
	}

	// retrieve the action
	a, err := ctx.DB.ActionByID(actionID)
	if err != nil {
		panic(err)
	}
	// retrieve investigators
	a.Investigators, err = ctx.DB.InvestigatorByActionID(a.ID)
	if err != nil {
		panic(err)
	}
	// store the results in the resource
	actionItem, err := ActionToItem(a, ctx)
	if err != nil {
		panic(err)
	}
	resource.AddItem(actionItem)

	respond(200, resource, respWriter, request, opid)
}

// getCommand takes an actionid and a commandid and returns a command
func getCommand(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getCommand()"}.Debug()
	}()
	var commandID uint64
	cmdid := request.URL.Query()["commandid"][0]
	if cmdid != "" {
		commandID, err = strconv.ParseUint(cmdid, 10, 64)
		if err != nil {
			panic(err)
		}
	}
	// retrieve the action
	var cmd mig.Command
	if commandID > 0 {
		cmd, err = ctx.DB.CommandByID(commandID)
	} else {
		// nothing to search for, return 404
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%d", opid),
			Message: fmt.Sprintf("Invalid Command ID '%d'", commandID)})
		respond(400, resource, respWriter, request, opid)
	}
	// store the results in the resource
	commandItem, err := commandToItem(cmd)
	if err != nil {
		panic(err)
	}
	resource.AddItem(commandItem)
	respond(200, resource, respWriter, request, opid)
}

// describeCancelCommand returns a resource that describes how to cancel a command
func describeCancelCommand(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving describeCancelCommand()"}.Debug()
	}()
	err := resource.SetTemplate(cljs.Template{
		Data: []cljs.Data{
			{Name: "actionid", Value: "[0-9]{1,20}", Prompt: "Action ID"},
			{Name: "commandid", Value: "[0-9]{1,20}", Prompt: "Command ID"},
		},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request, opid)
}

// cancelCommand receives an action ID and a command ID and issues a cancellation order
func cancelCommand(respWriter http.ResponseWriter, request *http.Request) {
}

func getAgentsDashboard(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAgentsDashboard()"}.Debug()
	}()
	respond(501, resource, respWriter, request, opid)
}

func searchAgents(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.String())
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving searchAgents()"}.Debug()
	}()
	respond(501, resource, respWriter, request, opid)
}

// safeWrite performs a two steps write:
// 1) a temp file is written
// 2) the temp file is moved into the target folder
// this prevents the dir watcher from waking up before the file is fully written
func safeWrite(opid uint64, destination string, data []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("safeWrite() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving safeWrite()"}.Debug()
	}()

	// write the file temp dir
	tmp := fmt.Sprintf("%s/%d", ctx.Directories.Tmp, mig.GenID())
	err = ioutil.WriteFile(tmp, data, 0640)
	if err != nil {
		panic(err)
	}

	// move to destination
	err = os.Rename(tmp, destination)
	if err != nil {
		panic(err)
	}

	return
}
