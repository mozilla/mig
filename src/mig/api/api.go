// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mig"
	"mig/pgp"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jvehent/cljs"
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
	ctx, err := Init(*config)
	if err != nil {
		fmt.Printf("\nFATAL: %v\n", err)
		os.Exit(9)
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
	s.HandleFunc("/agent", getAgent).Methods("GET")
	s.HandleFunc("/dashboard", getDashboard).Methods("GET")

	// all set, start the http handler
	http.Handle("/", r)
	listenAddr := fmt.Sprintf("%s:%d", ctx.Server.IP, ctx.Server.Port)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

// respond builds a Collection+JSON body and sends it to the client
func respond(code int, response *cljs.Resource, respWriter http.ResponseWriter, request *http.Request, opid float64) (err error) {
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
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getHome()"}.Debug()
	}()

	resource.AddQuery(cljs.Query{
		Rel:  "Get dashboard",
		Href: fmt.Sprintf("%s/dashboard", ctx.Server.BaseURL),
	})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "create action",
		Href: fmt.Sprintf("%s/action/create/", ctx.Server.BaseURL),
		Name: "POST endpoint to create an action"})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "cancel action",
		Href: fmt.Sprintf("%s/action/cancel/", ctx.Server.BaseURL),
		Name: "POST endpoint to cancel an action"})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "cancel command",
		Href: fmt.Sprintf("%s/command/cancel/", ctx.Server.BaseURL),
		Name: "POST endpoint to cancel a command"})
	if err != nil {
		panic(err)
	}

	// Describe the queries that are exposed to the client
	err = resource.AddQuery(cljs.Query{
		Rel:    "Query action by ID",
		Href:   fmt.Sprintf("%s/action", ctx.Server.BaseURL),
		Prompt: "GET endpoint to query an action by ID, using url parameter ?actionid=<numerical id>",
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
		Prompt: "GET endpoint to query a command by ID, using url parameter ?commandid=<numerical id>",
		Data: []cljs.Data{
			{Name: "commandid", Value: "[0-9]{1,20}", Prompt: "Command ID"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:    "Query agent by ID",
		Href:   fmt.Sprintf("%s/agent", ctx.Server.BaseURL),
		Prompt: "GET endpoint to query an agent by ID, using url parameter ?agentid=<numerical id>",
		Data: []cljs.Data{
			{Name: "agentid", Value: "[0-9]{1,20}", Prompt: "Agent ID"},
		},
	})
	if err != nil {
		panic(err)
	}

	resource.AddQuery(cljs.Query{
		Rel:    "Search stuff",
		Href:   fmt.Sprintf("%s/search", ctx.Server.BaseURL),
		Prompt: "GET endpoint to search for stuff",
		Data: []cljs.Data{
			{Name: "before", Value: "9998-01-01 12:12:12.686438508-04:00", Prompt: "return results recorded before this RFC3339 date"},
			{Name: "after", Value: "11-01-01 12:12:12.686438508-04:00", Prompt: "return results recorded after this RFC3339 date"},
			{Name: "type", Value: "(command|action|agent|investigator)", Prompt: "type defines what the search is looking for"},
			{Name: "report", Value: "(compliancesummary|complianceitems)", Prompt: "if set, return results in the given report format"},
			{Name: "agentname", Value: "agent123.example.net", Prompt: "filter results on the agent name"},
			{Name: "actionname", Value: "some action name", Prompt: "filter results on the action name"},
			{Name: "actionid", Value: "123456789...", Prompt: "filter results on the action id"},
			{Name: "commandid", Value: "123456789...", Prompt: "filter results on the command id"},
			{Name: "status", Value: "(done|destroyed|cancelled|timeout|...)", Prompt: "filter commands or agents results on their status"},
			{Name: "threatfamily", Value: "(compliance|backdoor|...)", Prompt: "filter results of the threat family"},
			{Name: "limit", Value: "10", Prompt: "limit the number of results to 10 by default"},
			{Name: "foundanything", Value: "(true|false)", Prompt: "return commands that have results with foundanything flag set to true or false"},
		},
	})
	if err != nil {
		panic(err)
	}

	respond(200, resource, respWriter, request, opid)
}

// describeCreateAction returns a resource that describes how to POST new actions
func describeCreateAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
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
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
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
	keyring, err := os.Open(ctx.PGP.Home + "/pubring.gpg")
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
		k, err := os.Open(ctx.PGP.Home + "/pubring.gpg")
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
	destdir := fmt.Sprintf("%s/%.0f.json", ctx.Directories.Action.New, action.ID)
	newAction, err := json.Marshal(action)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(destdir, newAction, 0640)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "Action committed to spool"}

	err = resource.AddItem(cljs.Item{
		Href: fmt.Sprintf("%s/action?actionid=%.0f", ctx.Server.BaseURL, action.ID),
		Data: []cljs.Data{{Name: "action ID " + fmt.Sprintf("%.0f", action.ID), Value: action}},
	})
	if err != nil {
		panic(err)
	}
	respond(201, resource, respWriter, request, opid)
}

// describeCancelAction returns a resource that describes how to cancel an action
func describeCancelAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
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
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
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
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAction()"}.Debug()
	}()
	actionID, err := strconv.ParseFloat(request.URL.Query()["actionid"][0], 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'actionid': '%v'", err)
		panic(err)
	}

	// retrieve the action
	var a mig.Action
	if actionID > 0 {
		a, err = ctx.DB.ActionByID(actionID)
		if err != nil {
			if a.ID == -1 {
				// not found, return 404
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Action ID '%.0f' not found", actionID)})
				respond(404, resource, respWriter, request, opid)
				return
			} else {
				panic(err)
			}
		}
	} else {
		// bad request, return 400
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: fmt.Sprintf("Invalid Action ID '%.0f'", actionID)})
		respond(400, resource, respWriter, request, opid)
		return
	}

	// retrieve investigators
	a.Investigators, err = ctx.DB.InvestigatorByActionID(a.ID)
	if err != nil {
		panic(err)
	}
	// store the results in the resource
	actionItem, err := actionToItem(a, true, ctx)
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
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getCommand()"}.Debug()
	}()
	commandID, err := strconv.ParseFloat(request.URL.Query()["commandid"][0], 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'commandid': '%v'", err)
		panic(err)
	}

	// retrieve the command
	var cmd mig.Command
	if commandID > 0 {
		cmd, err = ctx.DB.CommandByID(commandID)
		if err != nil {
			if fmt.Sprintf("%v", err) == "Error while retrieving command: 'sql: no rows in result set'" {
				// not found, return 404
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Command ID '%.0f' not found", commandID)})
				respond(404, resource, respWriter, request, opid)
				return
			} else {
				panic(err)
			}
		}
	} else {
		// bad request, return 400
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: fmt.Sprintf("Invalid Command ID '%.0f'", commandID)})
		respond(400, resource, respWriter, request, opid)
		return
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
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
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
	opid := mig.GenID()
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving cancelCommand()"}.Debug()
	}()
	respond(501, resource, respWriter, request, opid)
}

func getAgent(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAgentsDashboard()"}.Debug()
	}()
	agentID, err := strconv.ParseFloat(request.URL.Query()["agentid"][0], 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'agentid': '%v'", err)
		panic(err)
	}

	// retrieve the command
	var agt mig.Agent
	if agentID > 0 {
		agt, err = ctx.DB.AgentByID(agentID)
		if err != nil {
			if fmt.Sprintf("%v", err) == "Error while retrieving agent: 'sql: no rows in result set'" {
				// not found, return 404
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Agent ID '%.0f' not found", agentID)})
				respond(404, resource, respWriter, request, opid)
				return
			} else {
				panic(err)
			}
		}
	} else {
		// bad request, return 400
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: fmt.Sprintf("Invalid Agent ID '%.0f'", agentID)})
		respond(400, resource, respWriter, request, opid)
		return
	}
	// store the results in the resource
	agentItem, err := agentToItem(agt)
	if err != nil {
		panic(err)
	}
	resource.AddItem(agentItem)
	respond(200, resource, respWriter, request, opid)
}

func getDashboard(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getDashboard()"}.Debug()
	}()

	// get summary of agents active in the last 5 minutes
	sum, err := ctx.DB.SumAgentsByVersion(time.Now().Add(-5 * time.Minute))
	if err != nil {
		panic(err)
	}
	count, err := ctx.DB.CountNewAgents(time.Now().Add(-24 * time.Hour))
	if err != nil {
		panic(err)
	}
	double, err := ctx.DB.CountDoubleAgents(time.Now().Add(-5 * time.Minute))
	if err != nil {
		panic(err)
	}
	disappeared, err := ctx.DB.CountDisappearedAgents(
		time.Now().Add(-7*24*time.Hour), time.Now().Add(-5*time.Minute))
	if err != nil {
		panic(err)
	}
	sumItem, err := agentsSummaryToItem(sum, count, double, disappeared, ctx)
	resource.AddItem(sumItem)

	// add the last 10 actions
	actions, err := ctx.DB.Last10Actions()
	if err != nil {
		panic(err)
	}
	for _, action := range actions {
		// retrieve investigators
		action.Investigators, err = ctx.DB.InvestigatorByActionID(action.ID)
		if err != nil {
			panic(err)
		}
		// store the results in the resource
		actionItem, err := actionToItem(action, false, ctx)
		if err != nil {
			panic(err)
		}
		resource.AddItem(actionItem)
	}
	respond(200, resource, respWriter, request, opid)
}
