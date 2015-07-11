// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"github.com/jvehent/cljs"
	"mig"
	migdb "mig/database"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// search runs searches
func search(respWriter http.ResponseWriter, request *http.Request) {
	var (
		err         error
		p           migdb.SearchParameters
		filterFound bool
	)
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			// on panic, log and return error to client, including the search parameters
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.AddItem(cljs.Item{
				Href: loc,
				Data: []cljs.Data{{Name: "search parameters", Value: p}},
			})
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving search()"}.Debug()
	}()

	p, filterFound, err = parseSearchParameters(request.URL.Query())
	if err != nil {
		panic(err)
	}

	// run the search based on the type
	var results interface{}
	switch p.Type {
	case "action":
		results, err = ctx.DB.SearchActions(p)
	case "agent":
		if p.Target != "" {
			results, err = ctx.DB.ActiveAgentsByTarget(p.Target)
		} else {
			results, err = ctx.DB.SearchAgents(p)
		}
	case "command":
		results, err = ctx.DB.SearchCommands(p, filterFound)
	case "investigator":
		results, err = ctx.DB.SearchInvestigators(p)
	default:
		panic("search type is invalid")
	}
	if err != nil {
		panic(err)
	}

	// prepare the output in the requested format
	switch p.Report {
	case "complianceitems":
		if p.Type != "command" {
			panic("compliance items reporting is only available for the 'command' type")
		}
		items, err := commandsToComplianceItems(results.([]mig.Command))
		if err != nil {
			panic(err)
		}
		for i, item := range items {
			err = resource.AddItem(cljs.Item{
				Href: fmt.Sprintf("%s%s/search?type=command?agentname=%s&commandid=%s&actionid=%s&threatfamily=compliance&report=complianceitems",
					ctx.Server.Host, ctx.Server.BaseRoute, item.Target, p.CommandID, p.ActionID),
				Data: []cljs.Data{{Name: "compliance item", Value: item}},
			})
			if err != nil {
				panic(err)
			}
			if uint64(i) > p.Limit {
				break
			}
		}
	case "geolocations":
		if p.Type != "command" {
			panic("geolocations reporting is only available for the 'command' type")
		}
		items, err := commandsToGeolocations(results.([]mig.Command))
		if err != nil {
			panic(err)
		}
		for i, item := range items {
			err = resource.AddItem(cljs.Item{
				Href: fmt.Sprintf("%s%s/search?type=command?agentname=%s&commandid=%s&actionid=%s&report=geolocations",
					ctx.Server.Host, ctx.Server.BaseRoute, item.Endpoint, p.CommandID, p.ActionID),
				Data: []cljs.Data{{Name: "geolocation", Value: item}},
			})
			if err != nil {
				panic(err)
			}
			if uint64(i) > p.Limit {
				break
			}
		}
	default:
		switch p.Type {
		case "action":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d actions", len(results.([]mig.Action)))}
			for i, r := range results.([]mig.Action) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/action?actionid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if uint64(i) > p.Limit {
					break
				}
			}
		case "agent":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d agents", len(results.([]mig.Agent)))}
			for i, r := range results.([]mig.Agent) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/agent?agentid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if uint64(i) > p.Limit {
					break
				}
			}
		case "command":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d commands", len(results.([]mig.Command)))}
			for i, r := range results.([]mig.Command) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/command?commandid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if uint64(i) > p.Limit {
					break
				}
			}
		case "investigator":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d investigators", len(results.([]mig.Investigator)))}
			for i, r := range results.([]mig.Investigator) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/investigator?investigatorid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if uint64(i) > p.Limit {
					break
				}
			}
		}
	}
	// add search parameters at the end of the response
	err = resource.AddItem(cljs.Item{
		Href: loc,
		Data: []cljs.Data{{Name: "search parameters", Value: p}},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request)
}

// truere is a case insensitive regex that matches the string 'true'
var truere = regexp.MustCompile("(?i)^true$")

// false is a case insensitive regex that matches the string 'false'
var falsere = regexp.MustCompile("(?i)^false$")

// parseSearchParameters transforms a query string into search parameters in the migdb format
func parseSearchParameters(qp url.Values) (p migdb.SearchParameters, filterFound bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseSearchParameters()-> %v", e)
		}
	}()
	p = migdb.NewSearchParameters()
	for queryParams, _ := range qp {
		switch queryParams {
		case "actionname":
			p.ActionName = qp["actionname"][0]
		case "actionid":
			p.ActionID = qp["actionid"][0]
		case "after":
			p.After, err = time.Parse(time.RFC3339, qp["after"][0])
			if err != nil {
				panic("after date not in RFC3339 format")
			}
		case "agentid":
			p.AgentID = qp["agentid"][0]
		case "agentname":
			p.AgentName = qp["agentname"][0]
		case "before":
			p.Before, err = time.Parse(time.RFC3339, qp["before"][0])
			if err != nil {
				panic("before date not in RFC3339 format")
			}
		case "commandid":
			p.CommandID = qp["commandid"][0]
		case "foundanything":
			if truere.MatchString(qp["foundanything"][0]) {
				p.FoundAnything = true
			} else if falsere.MatchString(qp["foundanything"][0]) {
				p.FoundAnything = false
			} else {
				panic("foundanything parameter must be true or false")
			}
			filterFound = true
		case "investigatorid":
			p.InvestigatorID = qp["investigatorid"][0]
		case "investigatorname":
			p.InvestigatorName = qp["investigatorname"][0]
		case "limit":
			p.Limit, err = strconv.ParseUint(qp["limit"][0], 10, 64)
			if err != nil {
				panic("invalid limit parameter")
			}
		case "report":
			switch qp["report"][0] {
			case "complianceitems":
				p.Report = qp["report"][0]
			case "geolocations":
				p.Report = qp["report"][0]
			default:
				panic("report not implemented")
			}
		case "status":
			p.Status = qp["status"][0]
		case "target":
			p.Target = qp["target"][0]
		case "threatfamily":
			p.ThreatFamily = qp["threatfamily"][0]
		case "type":
			p.Type = qp["type"][0]
		}
	}
	return
}
