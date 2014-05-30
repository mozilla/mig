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
	"fmt"
	"mig"
	migdb "mig/database"
	"net/http"
	"strconv"
	"time"

	"github.com/jvehent/cljs"
)

// search runs searches
func search(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := mig.GenID()
	loc := fmt.Sprintf("http://%s:%d%s", ctx.Server.IP, ctx.Server.Port, request.URL.String())
	resource := cljs.New(loc)
	p := migdb.NewSearchParameters()
	defer func() {
		if e := recover(); e != nil {
			// on panic, log and return error to client, including the search parameters
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.AddItem(cljs.Item{
				Href: loc,
				Data: []cljs.Data{{Name: "search parameters", Value: p}},
			})
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%d", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request, opid)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving search()"}.Debug()
	}()
	timeLayout := time.RFC3339
	for queryParams, _ := range request.URL.Query() {
		switch queryParams {
		case "actionname":
			p.ActionName = request.URL.Query()["actionname"][0]
		case "after":
			p.After, err = time.Parse(timeLayout, request.URL.Query()["after"][0])
			if err != nil {
				panic("after date not in RFC3339 format")
			}
		case "agentname":
			p.AgentName = request.URL.Query()["agentname"][0]
		case "before":
			p.Before, err = time.Parse(timeLayout, request.URL.Query()["before"][0])
			if err != nil {
				panic("before date not in RFC3339 format")
			}
		case "report":
			switch request.URL.Query()["report"][0] {
			case "complianceitems":
				p.Report = request.URL.Query()["report"][0]
			default:
				panic("report not implemented")
			}
		case "limit":
			p.Limit, err = strconv.ParseUint(request.URL.Query()["limit"][0], 10, 64)
			if err != nil {
				panic("invalid limit parameter")
			}
		case "status":
			p.Status = request.URL.Query()["status"][0]
		case "threatfamily":
			p.ThreatFamily = request.URL.Query()["threatfamily"][0]
		}
	}
	// run the search based on the type
	var results interface{}
	if _, ok := request.URL.Query()["type"]; ok {
		p.Type = request.URL.Query()["type"][0]
		switch p.Type {
		case "command":
			results, err = ctx.DB.SearchCommands(p)
		case "action":
			results, err = ctx.DB.SearchActions(p)
		case "agent":
			results, err = ctx.DB.SearchAgents(p)
		default:
			panic("search type is invalid")
		}
		if err != nil {
			panic(err)
		}
	} else {
		panic("search type is missing")
	}

	// prepare the output in the requested format
	switch p.Report {
	case "complianceitems":
		var items interface{}
		switch p.Type {
		case "command":
			items, err = commandsToComplianceItems(results.([]mig.Command))
		default:
			panic("compliance items not available for this type")
		}
		if err != nil {
			panic(err)
		}
		err = resource.AddItem(cljs.Item{
			Href: loc,
			Data: []cljs.Data{{Name: "compliance items", Value: items}},
		})
	default:
		// no transformation, just return the results
		err = resource.AddItem(cljs.Item{
			Href: loc,
			Data: []cljs.Data{{Name: "search results", Value: results}},
		})
	}
	if err != nil {
		panic(err)
	}
	// add search parameters at the end of the response
	err = resource.AddItem(cljs.Item{
		Href: loc,
		Data: []cljs.Data{{Name: "search parameters", Value: p}},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request, opid)
}
