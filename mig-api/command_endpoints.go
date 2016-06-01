// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"net/http"
	"strconv"

	"mig.ninja/mig"

	"github.com/jvehent/cljs"
)

// getCommand takes an actionid and a commandid and returns a command
func getCommand(respWriter http.ResponseWriter, request *http.Request) {
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
			Message: fmt.Sprintf("Invalid Command ID '%.0f'", commandID)})
		respond(http.StatusBadRequest, resource, respWriter, request)
		return
	}
	// store the results in the resource
	commandItem, err := commandToItem(cmd)
	if err != nil {
		panic(err)
	}
	resource.AddItem(commandItem)
	respond(http.StatusOK, resource, respWriter, request)
}

// commandToItem receives a command and returns an Item in Collection+JSON
func commandToItem(cmd mig.Command) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/command?commandid=%.0f", ctx.Server.BaseURL, cmd.ID)
	links := make([]cljs.Link, 0)
	link := cljs.Link{
		Rel:  "action",
		Href: fmt.Sprintf("%s/action?actionid=%.0f", ctx.Server.BaseURL, cmd.Action.ID),
	}
	links = append(links, link)
	item.Links = links
	item.Data = []cljs.Data{
		{Name: "command", Value: cmd},
	}
	return
}
