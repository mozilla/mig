// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"net"
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
		authenticate(search, mig.PermSearch)).Methods("GET")
	s.HandleFunc("/action",
		authenticate(getAction, mig.PermAction)).Methods("GET")
	s.HandleFunc("/action/create/",
		authenticate(createAction, mig.PermActionCreate)).Methods("POST")
	s.HandleFunc("/command",
		authenticate(getCommand, mig.PermCommand)).Methods("GET")
	s.HandleFunc("/agent",
		authenticate(getAgent, mig.PermAgent)).Methods("GET")
	s.HandleFunc("/dashboard",
		authenticate(getDashboard, mig.PermDashboard)).Methods("GET")

	// Administrator resources
	s.HandleFunc("/loader",
		authenticate(getLoader, mig.PermLoader)).Methods("GET")
	s.HandleFunc("/loader/status/",
		authenticate(statusLoader, mig.PermLoaderStatus)).Methods("POST")
	s.HandleFunc("/loader/expect/",
		authenticate(expectLoader, mig.PermLoaderExpect)).Methods("POST")
	s.HandleFunc("/loader/key/",
		authenticate(keyLoader, mig.PermLoaderKey)).Methods("POST")
	s.HandleFunc("/loader/new/",
		authenticate(newLoader, mig.PermLoaderNew)).Methods("POST")
	s.HandleFunc("/manifest",
		authenticate(getManifest, mig.PermManifest)).Methods("GET")
	s.HandleFunc("/manifest/sign/",
		authenticate(signManifest, mig.PermManifestSign)).Methods("POST")
	s.HandleFunc("/manifest/status/",
		authenticate(statusManifest, mig.PermManifestStatus)).Methods("POST")
	s.HandleFunc("/manifest/new/",
		authenticate(newManifest, mig.PermManifestNew)).Methods("POST")
	s.HandleFunc("/manifest/loaders/",
		authenticate(manifestLoaders, mig.PermManifestLoaders)).Methods("GET")
	s.HandleFunc("/investigator",
		authenticate(getInvestigator, mig.PermInvestigator)).Methods("GET")
	s.HandleFunc("/investigator/create/",
		authenticate(createInvestigator, mig.PermInvestigatorCreate)).Methods("POST")
	s.HandleFunc("/investigator/update/",
		authenticate(updateInvestigator, mig.PermInvestigatorUpdate)).Methods("POST")

	ctx.Channels.Log <- mig.Log{Desc: "Starting HTTP handler"}

	// all set, start the http handler
	http.Handle("/", context.ClearHandler(r))
	listenAddr := fmt.Sprintf("%s:%d", ctx.Server.IP, ctx.Server.Port)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

// The category of request being made, this is set in the request context
const (
	_ = iota
	RequestCategoryInvestigator
	RequestCategoryLoader
)

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

// loaderNameType defines a type to store the loader name
type loaderNameType string

const loaderName loaderNameType = ""

// getLoaderName returns the name of the loader, "noauth" if not found
func getLoaderName(r *http.Request) string {
	if lname := context.Get(r, loaderName); lname != nil {
		return lname.(string)
	}
	return "noauth"
}

// apiRequestType defines a type to store the request type
type apiRequestCategoryType int

const apiRequestCategory apiRequestCategoryType = 0

// getAPIRequestType returns the type of request being made
func getAPIRequestCategory(r *http.Request) int {
	if rcat := context.Get(r, apiRequestCategory); rcat != nil {
		return rcat.(int)
	}
	return 0
}

// handler defines the type returned by the authenticate and authenticateLoader functions
type handler func(w http.ResponseWriter, r *http.Request)

// authenticate is called prior to processing incoming requests. it implements the client
// authentication logic, which mostly consist of validating GPG signed tokens and setting the
// identity of the signer in the request context. If requirePerm is not zero, this is the
// permission the investigator must have in order to access the endpoint.
func authenticate(pass handler, requirePerm int64) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
			inv mig.Investigator
		)
		opid := getOpID(r)
		context.Set(r, opID, opid)
		context.Set(r, apiRequestCategory, RequestCategoryInvestigator)
		if !ctx.Authentication.Enabled {
			inv.Name = "authdisabled"
			inv.ID = 0
			inv.Permissions.DefaultSet()
			inv.Permissions.ManifestSet()
			inv.Permissions.LoaderSet()
			inv.Permissions.AdminSet()
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

		// As a final phase, validate the investigator has permission to access
		// the endpoint
		if !inv.CheckPermission(requirePerm) {
			inv.Name = "authfailed"
			inv.ID = -1
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: "Insufficient permissions to access endpoint"})
			respond(http.StatusUnauthorized, resource, w, r)
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

// authenticateLoader is used to authenticate requests that are made to the
// loader API endpoints. Rather than operate on GPG signatures, the
// authentication instead uses the submitted loader key
func authenticateLoader(pass handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			err error
			ldr mig.LoaderEntry
		)
		opid := getOpID(r)
		context.Set(r, opID, opid)
		context.Set(r, apiRequestCategory, RequestCategoryLoader)
		lkey := r.Header.Get("X-LOADERKEY")
		if lkey == "" {
			resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: "X-LOADERKEY header not found"})
			respond(http.StatusUnauthorized, resource, w, r)
			return
		}
		err = mig.ValidateLoaderPrefixAndKey(lkey)
		if err != nil {
			goto authfailed
		}

		ldr, err = hashAuthenticateLoader(lkey)
		if err != nil {
			goto authfailed
		}
		context.Set(r, loaderID, ldr.ID)
		context.Set(r, loaderName, ldr.Name)
		// accept request
		pass(w, r)
		return

	authfailed:
		context.Set(r, loaderName, "authfailed")
		resource := cljs.New(fmt.Sprintf("%s%s", ctx.Server.Host, r.URL.String()))
		resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("Loader authorization failed")})
		respond(http.StatusUnauthorized, resource, w, r)
	}
}

// Extract the clients public IP from the Request using the method that
// has been defined in the API configuration using the clientpublicip
// option.
func remotePublicIP(r *http.Request) string {
	var useip string
	if ctx.Server.ClientPublicIPOffset == -1 {
		// Use the socket peer address
		useip = r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
	} else {
		// Use an offset of the X-Forwarded-For header
		xff := r.Header.Get("X-FORWARDED-FOR")
		if xff != "" {
			xargs := strings.Split(xff, ",")
			if ctx.Server.ClientPublicIPOffset >= len(xargs) {
				ctx.Channels.Log <- mig.Log{Desc: "warning: requested X-Forwarded-For offset is not possible, not enough elements"}.Warning()
				useip = strings.Trim(xargs[0], " ")
			} else {
				useip = strings.Trim(xargs[(len(xargs)-1)-ctx.Server.ClientPublicIPOffset], " ")
			}
		} else {
			ctx.Channels.Log <- mig.Log{Desc: "warning: API configured to use X-Forwarded-For but header not found"}.Warning()
			return "0.0.0.0"
		}
	}
	if net.ParseIP(useip) == nil {
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("warning: obtained client public IP %q invalid", useip)}.Warning()
		return "0.0.0.0"
	}
	return useip
}

// respond builds a Collection+JSON body and sends it to the client
func respond(code int, response interface{}, respWriter http.ResponseWriter, r *http.Request) (err error) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: getOpID(r), Desc: fmt.Sprintf("%v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{OpID: getOpID(r), Desc: "leaving respond()"}.Debug()
	}()
	var (
		body                []byte
		authfield, catfield string
	)
	// if the response is a cljs resource, marshal it, other treat it as a slice of bytes
	if _, ok := response.(*cljs.Resource); ok {
		body, err = response.(*cljs.Resource).Marshal()
		if err != nil {
			panic(err)
		}
	} else {
		body = []byte(response.([]byte))
	}

	switch getAPIRequestCategory(r) {
	case RequestCategoryInvestigator:
		authfield = fmt.Sprintf("[%s %.0f]", getInvName(r), getInvID(r))
		catfield = "investigator"
	case RequestCategoryLoader:
		authfield = fmt.Sprintf("[%s %.0f]", getLoaderName(r), getLoaderID(r))
		catfield = "loader"
	default:
		authfield = "[noauth 0]"
		catfield = "public"
	}

	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.Header().Set("Cache-Control", "no-cache")
	respWriter.WriteHeader(code)
	respWriter.Write(body)

	ctx.Channels.Log <- mig.Log{
		OpID: getOpID(r),
		Desc: fmt.Sprintf("src=%s category=%s auth=%s %s %s %s resp_code=%d resp_size=%d user-agent=%s",
			remotePublicIP(r), catfield, authfield, r.Method, r.Proto,
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
	respond(http.StatusOK, []byte(remotePublicIP(request)), respWriter, request)
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
