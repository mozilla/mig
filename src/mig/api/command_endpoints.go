// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"mig"
	"net/http"
	"strconv"

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
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getCommand()"}.Debug()
	}()
	commandID, err := strconv.ParseUint(request.URL.Query()["commandid"][0], 10, 64)
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
				respond(404, resource, respWriter, request)
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
		respond(400, resource, respWriter, request)
		return
	}
	// store the results in the resource
	commandItem, err := commandToItem(cmd)
	if err != nil {
		panic(err)
	}
	resource.AddItem(commandItem)
	respond(200, resource, respWriter, request)
}

// describeCancelCommand returns a resource that describes how to cancel a command
func describeCancelCommand(respWriter http.ResponseWriter, request *http.Request) {
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
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving describeCancelCommand()"}.Debug()
	}()
	err = resource.SetTemplate(cljs.Template{
		Data: []cljs.Data{
			{Name: "actionid", Value: "[0-9]{1,20}", Prompt: "Action ID"},
			{Name: "commandid", Value: "[0-9]{1,20}", Prompt: "Command ID"},
		},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request)
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
