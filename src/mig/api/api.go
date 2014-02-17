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
	"github.com/gorilla/mux"
	"github.com/jvehent/cljs"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"mig"
	"net/http"
	"os"
	"strconv"
)

var ctx Context

func main() {
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
	r.HandleFunc("/api/", getHome).Methods("GET")

	r.HandleFunc("/api/action", getAction).Methods("GET")
	r.HandleFunc("/api/action/search", getAction).Methods("GET")

	r.HandleFunc("/api/action/create/", describeCreateAction).Methods("GET")
	r.HandleFunc("/api/action/create/", createAction).Methods("POST")

	r.HandleFunc("/api/action/cancel/", describeCancelAction).Methods("GET")
	r.HandleFunc("/api/action/cancel/", cancelAction).Methods("POST")

	r.HandleFunc("/api/command", getCommand).Methods("GET")
	r.HandleFunc("/api/command/search", getCommand).Methods("GET")

	r.HandleFunc("/api/command/cancel", describeCancelCommand).Methods("GET")
	r.HandleFunc("/api/command/cancel", cancelCommand).Methods("POST")

	r.HandleFunc("/api/agent/dashboard/", getAgentsDashboard).Methods("GET")

	r.HandleFunc("/api/agent/search", searchAgents).Methods("GET")

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

	respWriter.Header().Set("Content-Type", cljs.ContentType)
	respWriter.WriteHeader(code)
	respWriter.Write(body)

	return
}

// getHome returns a basic document that presents the different ressources
// available in the API, as well as some status information
func getHome(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := mig.GenID()
	resource := cljs.New(request.URL.Path)
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
		Href: "/api/action/create/",
		Name: "Create an action"})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "cancel action",
		Href: "/api/action/cancel/",
		Name: "Cancel an action"})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "cancel command",
		Href: "/api/command/cancel/",
		Name: "Cancel a command"})
	if err != nil {
		panic(err)
	}

	// Describe the queries that are exposed to the client
	err = resource.AddQuery(cljs.Query{
		Rel:    "search action per id",
		Href:   "/api/action/search",
		Prompt: "Query action ID",
		Data: []cljs.Data{
			{Name: "actionid", Value: "[0-9]{1,20}", Prompt: "Action ID"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:    "search command per id",
		Href:   "/api/command/search",
		Prompt: "Query command ID",
		Data: []cljs.Data{
			{Name: "commandid", Value: "[0-9]{1,20}", Prompt: "Command ID"},
			{Name: "actionid", Value: "[0-9]{1,20}", Prompt: "Action ID"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:    "search agent per name",
		Href:   "/api/agent/search",
		Prompt: "Query agent name",
		Data: []cljs.Data{
			{Name: "name", Value: "agent123.example.net", Prompt: "Agent Name"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:  "get agent dashboard",
		Href: "/api/agent/dashboard/",
	})
	if err != nil {
		panic(err)
	}

	respond(200, resource, respWriter, request, opid)
}

// describeCreateAction returns a resource that describes how to POST new actions
func describeCreateAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.Path)
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
			{Name: "action", Value: "", Prompt: "Signed MIG Action"},
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
	opid := mig.GenID()
	var action mig.Action
	resource := cljs.New(request.URL.Path)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "leaving createAction()"}.Debug()
	}()

	// parse the POST body into a mig action
	data, err := ioutil.ReadAll(request.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal([]byte(data), &action)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "Received action for creation"}.Debug()

	// load keyring and validate action
	keyring, err := os.Open(ctx.OpenPGP.PubRing)
	if err != nil {
		panic(err)
	}
	defer keyring.Close()

	err = action.Validate(keyring)
	if err != nil {
		panic(err)
	}
	action.ID = mig.GenID()
	ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "Received new action with valid signature"}

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

	respond(201, resource, respWriter, request, opid)
}

// describeCancelAction returns a resource that describes how to cancel an action
func describeCancelAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.Path)
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
			{Name: "id", Value: "", Prompt: "Action ID"},
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
	resource := cljs.New(request.URL.Path)
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
	resource := cljs.New(request.URL.Path)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAction()"}.Debug()
	}()
	actionID, err := strconv.Atoi(request.URL.Query()["actionid"][0])
	if err != nil {
		panic(err)
	}

	// retrieve the action
	eas := []mig.ExtendedAction{}
	iter := ctx.DB.Col.Action.Find(bson.M{"action.id": actionID}).Iter()
	err = iter.All(&eas)
	if err != nil {
		panic(err)
	}
	if len(eas) == 0 {
		resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: "Action not found"})
		respond(404, resource, respWriter, request, opid)
	}
	// store the results in the resource
	for _, ea := range eas {
		actionItem, err := extendedActionToItem(ea)
		if err != nil {
			panic(err)
		}
		resource.AddItem(actionItem)
	}

	respond(200, resource, respWriter, request, opid)
}

// getCommand takes an actionid and a commandid and returns a command
func getCommand(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := mig.GenID()
	resource := cljs.New(request.URL.Path)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getCommand()"}.Debug()
	}()
	var actionID, commandID int
	aid := request.URL.Query()["actionid"][0]
	if aid != "" {
		actionID, err = strconv.Atoi(aid)
		if err != nil {
			panic(err)
		}
	}
	cmdid := request.URL.Query()["commandid"][0]
	if cmdid != "" {
		commandID, err = strconv.Atoi(cmdid)
		if err != nil {
			panic(err)
		}
	}

	// retrieve the action
	cmds := []mig.Command{}
	var iter *mgo.Iter
	if commandID > 0 {
		if actionID > 0 {
			iter = ctx.DB.Col.Cmd.Find(bson.M{"id": commandID, "action.id": actionID}).Iter()
		} else {
			iter = ctx.DB.Col.Cmd.Find(bson.M{"id": commandID}).Iter()
		}
	} else {
		// nothing to search for, return empty resource
		respond(200, resource, respWriter, request, opid)
	}
	err = iter.All(&cmds)
	if err != nil {
		panic(err)
	}
	if len(cmds) == 0 {
		resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: "No command found"})
		respond(404, resource, respWriter, request, opid)
	}
	// store the results in the resource
	for _, cmd := range cmds {
		commandItem, err := commandToItem(cmd)
		if err != nil {
			panic(err)
		}
		resource.AddItem(commandItem)
	}
	respond(200, resource, respWriter, request, opid)
}

// describeCancelCommand returns a resource that describes how to cancel a command
func describeCancelCommand(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	resource := cljs.New(request.URL.Path)
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
			{Name: "actionid", Value: "", Prompt: "Action ID"},
			{Name: "commandid", Value: "", Prompt: "Command ID"},
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
	resource := cljs.New(request.URL.Path)
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
	resource := cljs.New(request.URL.Path)
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
