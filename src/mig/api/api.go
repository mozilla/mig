// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/jvehent/cljs"
	"mig"
	migdb "mig/database"
	"net/http"
	"os"
	"runtime"
	"time"
)

var ctx Context

func main() {
	var err error
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)

	// command line options
	var config = flag.String("c", "/etc/mig/api.cfg", "Load configuration from file")
	flag.Parse()

	// The context initialization takes care of parsing the configuration,
	// and creating connections to database, syslog, ...
	fmt.Fprintf(os.Stderr, "Initializing API context...")
	ctx, err = Init(*config) //ctx is a global variable
	if err != nil {
		fmt.Printf("\nFATAL: %v\n", err)
		os.Exit(9)
	}
	fmt.Fprintf(os.Stderr, "OK\n")
	ctx.Channels.Log <- mig.Log{Desc: "Context initialization done"}

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
	ctx.Channels.Log <- mig.Log{Desc: "Logger routine started"}

	// register routes
	r := mux.NewRouter()
	s := r.PathPrefix(ctx.Server.BaseRoute).Subrouter()
	s.HandleFunc("/", authenticate(getHome)).Methods("GET")
	s.HandleFunc("/search", authenticate(search)).Methods("GET")
	s.HandleFunc("/action", authenticate(getAction)).Methods("GET")
	s.HandleFunc("/action/create/", authenticate(describeCreateAction)).Methods("GET")
	s.HandleFunc("/action/create/", authenticate(createAction)).Methods("POST")
	s.HandleFunc("/command", authenticate(getCommand)).Methods("GET")
	s.HandleFunc("/agent", authenticate(getAgent)).Methods("GET")
	s.HandleFunc("/investigator", authenticate(getInvestigator)).Methods("GET")
	s.HandleFunc("/investigator/create/", authenticate(describeCreateInvestigator)).Methods("GET")
	s.HandleFunc("/investigator/create/", authenticate(createInvestigator)).Methods("POST")
	s.HandleFunc("/investigator/update/", authenticate(describeUpdateInvestigator)).Methods("GET")
	s.HandleFunc("/investigator/update/", authenticate(updateInvestigator)).Methods("POST")
	s.HandleFunc("/dashboard", authenticate(getDashboard)).Methods("GET")

	ctx.Channels.Log <- mig.Log{Desc: "Starting HTTP handler"}

	// all set, start the http handler
	http.Handle("/", context.ClearHandler(r))
	listenAddr := fmt.Sprintf("%s:%d", ctx.Server.IP, ctx.Server.Port)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

// invNameType defines a type to store the name of an investigator in the request context
type invNameType string

const authenticatedInvName invNameType = ""

// invIDType defines a type to store the ID of an investigator in the request context
type invIDType float64

const authenticatedInvID invIDType = 0

// opIDType defines a type for the operation ID
type opIDType float64

const opID opIDType = 0

// getOpID returns an operation ID from a request context, and if not found, generate one
func getOpID(r *http.Request) float64 {
	if opid := context.Get(r, opID); opid != nil {
		return opid.(float64)
	}
	return mig.GenID()
}

// handler defines the type returned by the authenticate function
type handler func(w http.ResponseWriter, r *http.Request)

// authenticate is called prior to processing incoming requests. it implements the client
// authentication logic, which mostly consist of validating GPG signed tokens and setting the
// identity of the signer in the request context
func authenticate(pass handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
			inv mig.Investigator
		)
		opid := getOpID(r)
		context.Set(r, opID, opid)
		if !ctx.Authentication.Enabled {
			inv.Name = "authdisabled"
			inv.ID = 0
			goto authorized
		}
		if r.Header.Get("X-PGPAUTHORIZATION") == "" {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("[auth missing] %s", r.URL.String())}.Err()
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: "X-PGPAUTHORIZATION header not found"})
			respond(401, resource, w, r)
			return
		}
		inv, err = verifySignedToken(r.Header.Get("X-PGPAUTHORIZATION"))
		if err != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("[auth failed] %s %v", r.URL.String(), err)}.Err()
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("Authorization verification failed with error '%v'", err)})
			respond(401, resource, w, r)
			return
		}
	authorized:
		// store investigator identity in request context
		context.Set(r, authenticatedInvName, inv.Name)
		context.Set(r, authenticatedInvID, inv.ID)
		ctx.Channels.Log <- mig.Log{
			OpID: opid,
			Desc: fmt.Sprintf("[authenticated name='%s' id='%.0f'] %s",
				inv.Name, inv.ID, r.URL.String()),
		}
		// accept request
		pass(w, r)
	}
}

// respond builds a Collection+JSON body and sends it to the client
func respond(code int, response *cljs.Resource, respWriter http.ResponseWriter, request *http.Request) (err error) {
	opid := getOpID(request)
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
	respWriter.Header().Set("Cache-Control", "no-cache")
	respWriter.WriteHeader(code)
	respWriter.Write(body)

	return
}

// getHome returns a basic document that presents the different ressources
// available in the API, as well as some status information
func getHome(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
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
		Rel:  "create investigator",
		Href: fmt.Sprintf("%s/investigator/create/", ctx.Server.BaseURL),
		Name: "POST endpoint to create an investigator"})
	if err != nil {
		panic(err)
	}

	err = resource.AddLink(cljs.Link{
		Rel:  "update investigator",
		Href: fmt.Sprintf("%s/investigator/update/", ctx.Server.BaseURL),
		Name: "POST endpoint to update an investigator"})
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
		Rel:    "Query investigator by ID",
		Href:   fmt.Sprintf("%s/investigator", ctx.Server.BaseURL),
		Prompt: "GET endpoint to query an investigator by ID, using url parameter ?investigatorid=<numerical id>",
		Data: []cljs.Data{
			{Name: "investigatorid", Value: "[0-9]{1,20}", Prompt: "Investigator ID"},
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
			{Name: "type", Value: "(command|action|agent|investigator)", Prompt: "type defines what the search is looking for"},
			{Name: "actionid", Value: "123456789...", Prompt: "filter results on the action id"},
			{Name: "actionname", Value: "some action name", Prompt: "filter results on the action name"},
			{Name: "after", Value: "11-01-01 12:12:12.686438508-04:00", Prompt: "return results recorded after this RFC3339 date"},
			{Name: "agentid", Value: "123456789...", Prompt: "filter results on the agent id"},
			{Name: "agentname", Value: "agent123.example.net", Prompt: "filter results on the agent name"},
			{Name: "before", Value: "9998-01-01 12:12:12.686438508-04:00", Prompt: "return results recorded before this RFC3339 date"},
			{Name: "commandid", Value: "123456789...", Prompt: "filter results on the command id"},
			{Name: "foundanything", Value: "(true|false)", Prompt: "return commands that have results with foundanything flag set to true or false"},
			{Name: "investigatorid", Value: "123456789...", Prompt: "filter results on the investigator id"},
			{Name: "investigatorname", Value: "%bob%", Prompt: "filter results on the investigator name"},
			{Name: "limit", Value: "10000", Prompt: "limit the number of results to 10,000 by default"},
			{Name: "report", Value: "(compliancesummary|complianceitems)", Prompt: "if set, return results in the given report format"},
			{Name: "status", Value: "(sent|success|cancelled|expired|failed|timeout|...)", Prompt: "filter results on the type's status"},
			{Name: "threatfamily", Value: "(compliance|backdoor|...)", Prompt: "filter results of the threat family"},
		},
	})
	if err != nil {
		panic(err)
	}

	respond(200, resource, respWriter, request)
}

func getDashboard(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
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
	actions, err := ctx.DB.LastActions(10)
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
	respond(200, resource, respWriter, request)
}

// agentsSumToItem receives an AgentsSum and returns an Item
// in the Collection+JSON format
func agentsSummaryToItem(sum []migdb.AgentsSum, count, double, disappeared float64, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/dashboard", ctx.Server.BaseURL)
	var total float64 = 0
	for _, asum := range sum {
		total += asum.Count
	}
	item.Data = []cljs.Data{
		{Name: "active agents", Value: total},
		{Name: "agents versions count", Value: sum},
		{Name: "agents started in the last 24 hours", Value: count},
		{Name: "endpoints running 2 or more agents", Value: double},
		{Name: "endpoints that have disappeared over last 7 days", Value: disappeared},
	}
	return
}
