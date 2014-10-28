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

func getAgent(respWriter http.ResponseWriter, request *http.Request) {
	opid := mig.GenID()
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%s", request.URL.String())}
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
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

// agentToItem receives an agent and returns an Item in Collection+JSON
func agentToItem(agt mig.Agent) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/agent?agentid=%.0f", ctx.Server.BaseURL, agt.ID)
	item.Data = []cljs.Data{
		{Name: "agent", Value: agt},
	}
	return
}
