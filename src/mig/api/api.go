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
	"io/ioutil"
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
	r.HandleFunc("/api/action/create/", createAction).Methods("POST")
	r.HandleFunc("/api/action/cancel/{actionID:[0-9]{1,20}}", cancelAction).Methods("POST")
	r.HandleFunc("/api/action/{actionID:[0-9]{1,20}}", getAction).Methods("GET")
	r.HandleFunc("/api/action/{actionID:[0-9]{1,20}}/command/{commandID:[0-9]{1,20}}", getCommand).Methods("GET")
	r.HandleFunc("/api/agent/dashboard/", getAgentsDashboard).Methods("GET")
	r.HandleFunc("/api/agent/search/", searchAgents).Methods("POST")

	// all set, start the http handler
	http.Handle("/", r)
	listenAddr := fmt.Sprintf("%s:%d", ctx.Server.IP, ctx.Server.Port)
	err = http.ListenAndServe(listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

// createAction receives a signed action in a POST request, validates it,
// and write it into the scheduler spool
func createAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	var action mig.Action
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: fmt.Sprintf("%v", e)}.Err()
			// return an error to the client
			respWriter.Header().Set("Content-Type", "application/json")
			respWriter.WriteHeader(500)
			http_error := `{"error":{"opid":` + string(opid) + `,"reason":"` + fmt.Sprintf("%v", e) + `}}`
			respWriter.Write([]byte(http_error))
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

	// respond to the client
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.WriteHeader(201)
	respWriter.Write(newAction)
}

func cancelAction(respWriter http.ResponseWriter, request *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.WriteHeader(501)
	respWriter.Write([]byte(`{"Error": "Not Implemented"}`))
}

// getAction queries the database and retrieves the detail of an action
func getAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAction()"}.Debug()
	}()

	vars := mux.Vars(request)
	actionID, err := strconv.Atoi(vars["actionID"])
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
		respWriter.WriteHeader(404)
		respWriter.Write([]byte(`{"error": "not found"}`))
		panic("Action not found in the database")
	}
	actionsList, err := json.Marshal(eas)
	if err != nil {
		panic(err)
	}

	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.WriteHeader(200)
	respWriter.Write(actionsList)
}

func getCommand(respWriter http.ResponseWriter, request *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.WriteHeader(501)
	respWriter.Write([]byte(`{"Error": "Not Implemented"}`))
}

func getAgentsDashboard(respWriter http.ResponseWriter, request *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.Write([]byte(`{"Error": "Not Implemented"}`))
	respWriter.WriteHeader(501)
}

func searchAgents(respWriter http.ResponseWriter, request *http.Request) {
	respWriter.Header().Set("Content-Type", "application/json")
	respWriter.WriteHeader(501)
	respWriter.Write([]byte(`{"Error": "Not Implemented"}`))
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
