// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mig"
	"mig/pgp"
	"net/http"
	"strconv"
	"time"

	"github.com/jvehent/cljs"
)

// describeCreateAction returns a resource that describes how to POST new actions
func describeCreateAction(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%s", request.URL.String())}
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
	date0 := time.Date(0011, time.January, 11, 11, 11, 11, 11, time.UTC)
	date1 := time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
	action.StartTime = date0
	action.FinishTime = date1
	action.LastUpdateTime = date0
	action.Status = "init"

	// load keyring and validate action
	keyring, err := getKeyring()
	if err != nil {
		panic(err)
	}

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
		k, err := getKeyring()
		if err != nil {
			panic(err)
		}
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
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%s", request.URL.String())}
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
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%s", request.URL.String())}
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
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%s", request.URL.String())}
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

// actionToItem receives an Action and returns an Item
// in the Collection+JSON format
func actionToItem(a mig.Action, addCommands bool, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/action?actionid=%.0f", ctx.Server.BaseURL, a.ID)
	item.Data = []cljs.Data{
		{Name: "action", Value: a},
	}
	if addCommands {
		links := make([]cljs.Link, 0)
		commands, err := ctx.DB.CommandsByActionID(a.ID)
		if err != nil {
			err = fmt.Errorf("ActionToItem() -> '%v'", err)
			return item, err
		}
		for _, cmd := range commands {
			link := cljs.Link{
				Rel:  fmt.Sprintf("Command ID %.0f on agent %s", cmd.ID, cmd.Agent.Name),
				Href: fmt.Sprintf("%s/command?commandid=%.0f", ctx.Server.BaseURL, cmd.ID),
			}
			links = append(links, link)
		}
		item.Links = links
	}
	return
}
