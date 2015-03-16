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
	"net/http"
	"os"
	"runtime"
	"strings"
)

// build version
var version string

var ctx Context

func main() {
	var err error
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)

	// command line options
	var config = flag.String("c", "/etc/mig/api.cfg", "Load configuration from file")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()

	if *showversion {
		fmt.Println(version)
		os.Exit(0)
	}

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
	// unauthenticated endpoints
	s.HandleFunc("/heartbeat", getHeartbeat).Methods("GET")
	s.HandleFunc("/ip", getIP).Methods("GET")
	// all other resources require authentication
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

// Context variables:
// invNameType defines a type to store the name of an investigator in the request context
type invNameType string

const authenticatedInvName invNameType = ""

// getInvName returns the Name of the investigator, "noauth" if not found, an error string if auth failed
func getInvName(r *http.Request) string {
	if name := context.Get(r, authenticatedInvName); name != nil {
		return name.(string)
	}
	return "noauth"
}

// invIDType defines a type to store the ID of an investigator in the request context
type invIDType float64

const authenticatedInvID invIDType = 0

// getInvID returns the ID of the investigator, 0 if not found, -1 if auth failed
func getInvID(r *http.Request) float64 {
	if id := context.Get(r, authenticatedInvID); id != nil {
		return id.(float64)
	}
	return 0.0
}

// opIDType defines a type for the operation ID
type opIDType float64

const opID opIDType = 0

// getOpID returns an operation ID from a request context, and if not found, generates one
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
			inv.Name = "authmissing"
			inv.ID = -1
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: "X-PGPAUTHORIZATION header not found"})
			respond(401, resource, w, r)
			return
		}
		inv, err = verifySignedToken(r.Header.Get("X-PGPAUTHORIZATION"))
		if err != nil {
			inv.Name = "authfailed"
			inv.ID = -1
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("Authorization verification failed with error '%v'", err)})
			respond(401, resource, w, r)
			return
		}
	authorized:
		// store investigator identity in request context
		context.Set(r, authenticatedInvName, inv.Name)
		context.Set(r, authenticatedInvID, inv.ID)
		// accept request
		pass(w, r)
	}
}

func remoteAddresses(r *http.Request) (ips string) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: getOpID(r), Desc: fmt.Sprintf("%v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{OpID: getOpID(r), Desc: "leaving remoteAddresses()"}.Debug()
	}()
	if r.Header.Get("X-FORWARDED-FOR") != "" {
		ips += r.Header.Get("X-FORWARDED-FOR") + ","
	}
	// strip port from remoteaddr received from request
	pos := strings.LastIndex(r.RemoteAddr, ":")
	ips += r.RemoteAddr[0:pos]
	return
}

// respond builds a Collection+JSON body and sends it to the client
func respond(code int, response interface{}, respWriter http.ResponseWriter, r *http.Request) (err error) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: getOpID(r), Desc: fmt.Sprintf("%v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{OpID: getOpID(r), Desc: "leaving respond()"}.Debug()
	}()
	var body []byte
	// if the response is a cljs resource, marshal it, other treat it as a slice of bytes
	if _, ok := response.(*cljs.Resource); ok {
		body, err = response.(*cljs.Resource).Marshal()
		if err != nil {
			panic(err)
		}
	} else {
		body = []byte(response.([]byte))
	}

	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.Header().Set("Cache-Control", "no-cache")
	respWriter.WriteHeader(code)
	respWriter.Write(body)

	ctx.Channels.Log <- mig.Log{
		OpID: getOpID(r),
		Desc: fmt.Sprintf("src=%s auth=[%s %.0f] %s %s %s resp_code=%d resp_size=%d user-agent=%s",
			remoteAddresses(r), getInvName(r), getInvID(r), r.Method, r.Proto,
			r.URL.String(), code, len(body), r.UserAgent()),
	}
	return
}

// getHeartbeat returns a 200
func getHeartbeat(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getHeartbeat()"}.Debug()
	}()
	err := resource.AddItem(cljs.Item{
		Href: request.URL.String(),
		Data: []cljs.Data{
			{
				Name:  "heartbeat",
				Value: "gatorz say hi",
			},
		}})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request)
}

// getIP returns a the public IP of the caller as read from X-Forwarded-For
func getIP(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	defer func() {
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getIP()"}.Debug()
	}()
	if request.Header.Get("X-FORWARDED-FOR") != "" {
		respond(200, []byte(request.Header.Get("X-FORWARDED-FOR")), respWriter, request)
	} else {
		respond(200, []byte(request.RemoteAddr), respWriter, request)
	}
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
	var (
		err         error
		agentsStats mig.AgentsStats
	)
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
	stats, err := ctx.DB.GetAgentsStats(1)
	if err != nil {
		panic(err)
	}
	if len(stats) > 1 {
		panic(fmt.Sprintf("expected 1 set of agents stats, got %d", len(stats)))
	}
	if len(stats) == 1 {
		agentsStats = stats[0]
		sumItem, err := agentsSummaryToItem(agentsStats, ctx)
		if err != nil {
			panic(err)
		}
		resource.AddItem(sumItem)
	}

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
