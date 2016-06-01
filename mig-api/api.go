// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/jvehent/cljs"
	"mig.ninja/mig"
)

var ctx Context

func main() {
	var err error
	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)

	// command line options
	var config = flag.String("c", "/etc/mig/api.cfg", "Load configuration from file")
	var debug = flag.Bool("d", false, "Debug mode: run in foreground, log to stdout.")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	// The context initialization takes care of parsing the configuration,
	// and creating connections to database, syslog, ...
	fmt.Fprintf(os.Stderr, "Initializing API context...")
	ctx, err = Init(*config, *debug) //ctx is a global variable
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

	// Loader manifest endpoints, use loader specific authentication on
	// the request
	s.HandleFunc("/manifest/agent/",
		authenticateLoader(getAgentManifest)).Methods("POST")
	s.HandleFunc("/manifest/fetch/",
		authenticateLoader(getManifestFile)).Methods("POST")

	// Investigator resources that require authentication
	s.HandleFunc("/search",
		authenticate(search, false)).Methods("GET")
	s.HandleFunc("/action",
		authenticate(getAction, false)).Methods("GET")
	s.HandleFunc("/action/create/",
		authenticate(createAction, false)).Methods("POST")
	s.HandleFunc("/command",
		authenticate(getCommand, false)).Methods("GET")
	s.HandleFunc("/agent",
		authenticate(getAgent, false)).Methods("GET")
	s.HandleFunc("/dashboard",
		authenticate(getDashboard, false)).Methods("GET")

	// Administrator resources
	s.HandleFunc("/loader",
		authenticate(getLoader, true)).Methods("GET")
	s.HandleFunc("/loader/status/",
		authenticate(statusLoader, true)).Methods("POST")
	s.HandleFunc("/loader/key/",
		authenticate(keyLoader, true)).Methods("POST")
	s.HandleFunc("/loader/new/",
		authenticate(newLoader, true)).Methods("POST")
	s.HandleFunc("/manifest",
		authenticate(getManifest, true)).Methods("GET")
	s.HandleFunc("/manifest/sign/",
		authenticate(signManifest, true)).Methods("POST")
	s.HandleFunc("/manifest/status/",
		authenticate(statusManifest, true)).Methods("POST")
	s.HandleFunc("/manifest/new/",
		authenticate(newManifest, true)).Methods("POST")
	s.HandleFunc("/manifest/loaders/",
		authenticate(manifestLoaders, true)).Methods("GET")
	s.HandleFunc("/investigator",
		authenticate(getInvestigator, true)).Methods("GET")
	s.HandleFunc("/investigator/create/",
		authenticate(createInvestigator, true)).Methods("POST")
	s.HandleFunc("/investigator/update/",
		authenticate(updateInvestigator, true)).Methods("POST")

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

// invIsAdmin indicates if an investigator is an administrator or not
type invIsAdminType bool

const authenticatedInvIsAdmin invIsAdminType = false

// getInvIsAdmin returns true of an authenticated investigator is an administrator
func getInvIsAdmin(r *http.Request) bool {
	if f := context.Get(r, authenticatedInvIsAdmin); f != nil {
		return f.(bool)
	}
	return false
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

// loaderIDType defines a type to store the loader ID
type loaderIDType float64

const loaderID loaderIDType = 0

// getLoaderID returns the ID of the loader, 0 if not found
func getLoaderID(r *http.Request) float64 {
	if id := context.Get(r, loaderID); id != nil {
		return id.(float64)
	}
	return 0.0
}

// handler defines the type returned by the authenticate and authenticateLoader functions
type handler func(w http.ResponseWriter, r *http.Request)

// authenticate is called prior to processing incoming requests. it implements the client
// authentication logic, which mostly consist of validating GPG signed tokens and setting the
// identity of the signer in the request context
func authenticate(pass handler, adminRequired bool) handler {
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
			inv.IsAdmin = true
			goto authorized
		}
		if r.Header.Get("X-PGPAUTHORIZATION") == "" {
			inv.Name = "authmissing"
			inv.ID = -1
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: "X-PGPAUTHORIZATION header not found"})
			respond(http.StatusUnauthorized, resource, w, r)
			return
		}
		inv, err = verifySignedToken(r.Header.Get("X-PGPAUTHORIZATION"))
		if err != nil {
			inv.Name = "authfailed"
			inv.ID = -1
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("Authorization verification failed with error '%v'", err)})
			respond(http.StatusUnauthorized, resource, w, r)
			return
		}
	authorized:
		// store investigator identity in request context
		context.Set(r, authenticatedInvName, inv.Name)
		context.Set(r, authenticatedInvID, inv.ID)
		context.Set(r, authenticatedInvIsAdmin, inv.IsAdmin)
		// Validate investigator is an administrator if required
		if adminRequired {
			if !inv.IsAdmin {
				inv.Name = "authfailed"
				inv.ID = -1
				ctx.Channels.Log <- mig.Log{
					OpID: getOpID(r),
					Desc: fmt.Sprintf("Investigator '%v' %v has insufficient privileges to access API function", getInvName(r), getInvID(r)),
				}.Info()
				resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
				resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid),
					Message: "Insufficient privileges"})
				respond(http.StatusUnauthorized, resource, w, r)
				return
			}
		}
		// accept request
		pass(w, r)
	}
}

// authenticateLoader is used to authenticate requests that are made to the
// loader API endpoints. Rather than operate on GPG signatures, the
// authentication instead uses the submitted loader key
func authenticateLoader(pass handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			loaderid float64
			err      error
		)
		opid := getOpID(r)
		context.Set(r, opID, opid)
		lkey := r.Header.Get("X-LOADERKEY")
		if lkey == "" {
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: "X-LOADERKEY header not found"})
			respond(http.StatusUnauthorized, resource, w, r)
			return
		}
		// Do a sanity check here on the submitted loader string before
		// we attempt the authentication
		err = mig.ValidateLoaderKey(lkey)
		if err == nil {
			loaderid, err = ctx.DB.GetLoaderEntryID(lkey)
		}
		if err != nil {
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("Loader authorization failed")})
			respond(http.StatusUnauthorized, resource, w, r)
			return
		}
		context.Set(r, loaderID, loaderid)
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
			respond(http.StatusInternalServerError, resource, respWriter, request)
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
	respond(http.StatusOK, resource, respWriter, request)
}

// getIP returns a the public IP of the caller as read from X-Forwarded-For
func getIP(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	defer func() {
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getIP()"}.Debug()
	}()
	if request.Header.Get("X-FORWARDED-FOR") != "" {
		respond(http.StatusOK, []byte(request.Header.Get("X-FORWARDED-FOR")), respWriter, request)
	} else {
		// request.RemoteAddr contains IP:Port, so strip the port and return just the IP
		respond(http.StatusOK, []byte(request.RemoteAddr[:strings.LastIndex(request.RemoteAddr, ":")]), respWriter, request)
	}
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
			respond(http.StatusInternalServerError, resource, respWriter, request)
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
	respond(http.StatusOK, resource, respWriter, request)
}
