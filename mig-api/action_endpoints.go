// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jvehent/cljs"
	"mig.ninja/mig"
	"mig.ninja/mig/pgp"
)

// createAction receives a signed action in a POST request, validates it,
// and write it into the scheduler spool
func createAction(respWriter http.ResponseWriter, request *http.Request) {
	var (
		err    error
		action mig.Action
	)
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "leaving createAction()"}.Debug()
	}()

	// parse the POST body into a mig action
	err = request.ParseForm()
	if err != nil {
		panic(err)
	}
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
	action.Status = "pending"

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
		k, err := getKeyring()
		if err != nil {
			panic(err)
		}
		fp, err := pgp.GetFingerprintFromSignature(astr, sig, k)
		if err != nil {
			panic(err)
		}
		inv, err := ctx.DB.InvestigatorByFingerprint(fp)
		if err != nil {
			panic(err)
		}
		err = ctx.DB.InsertSignature(action.ID, inv.ID, sig)
		if err != nil {
			panic(err)
		}
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, ActionID: action.ID, Desc: "Action written to database"}
	err = resource.AddItem(cljs.Item{
		Href: fmt.Sprintf("%s/action?actionid=%.0f", ctx.Server.BaseURL, action.ID),
		Data: []cljs.Data{{Name: "action ID " + fmt.Sprintf("%.0f", action.ID), Value: action}},
	})
	if err != nil {
		panic(err)
	}
	// return a 202 Accepted. the action will be processed asynchronously, and may fail later.
	respond(http.StatusAccepted, resource, respWriter, request)
}

// getAction queries the database and retrieves the detail of an action
func getAction(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(http.StatusInternalServerError, resource, respWriter, request)
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
				respond(http.StatusNotFound, resource, respWriter, request)
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
		respond(http.StatusBadRequest, resource, respWriter, request)
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
	respond(http.StatusOK, resource, respWriter, request)
}

// actionToItem receives an Action and returns an Item
// in the Collection+JSON format
func actionToItem(a mig.Action, addCommands bool, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/action?actionid=%.0f", ctx.Server.BaseURL, a.ID)
	item.Data = []cljs.Data{
		{Name: "action", Value: a},
	}
	/*
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
	*/
	return
}
