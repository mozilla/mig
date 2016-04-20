// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/jvehent/cljs"
	"mig.ninja/mig"
	migdbsearch "mig.ninja/mig/database/search"
)

type pagination struct {
	Limit  float64 `json:"limit"`
	Offset float64 `json:"offset"`
	Next   string  `json:"next"`
}

// search runs searches
func search(respWriter http.ResponseWriter, request *http.Request) {
	var (
		err         error
		p           migdbsearch.Parameters
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
			if fmt.Sprintf("%v", e) == "no results found" {
				respond(http.StatusNotFound, resource, respWriter, request)
			} else {
				respond(http.StatusInternalServerError, resource, respWriter, request)
			}

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
	case "manifest":
		results, err = ctx.DB.SearchManifests(p)
	case "loader":
		results, err = ctx.DB.SearchLoaders(p)
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
			if float64(i) > p.Limit {
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
			if float64(i) > p.Limit {
				break
			}
		}
	default:
		switch p.Type {
		case "action":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d actions", len(results.([]mig.Action)))}
			if len(results.([]mig.Action)) == 0 {
				panic("no results found")
			}
			for i, r := range results.([]mig.Action) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/action?actionid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if float64(i) > p.Limit {
					break
				}
			}
		case "agent":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d agents", len(results.([]mig.Agent)))}
			if len(results.([]mig.Agent)) == 0 {
				panic("no results found")
			}
			for i, r := range results.([]mig.Agent) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/agent?agentid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if float64(i) > p.Limit {
					break
				}
			}
		case "command":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d commands", len(results.([]mig.Command)))}
			if len(results.([]mig.Command)) == 0 {
				panic("no results found")
			}
			for i, r := range results.([]mig.Command) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/command?commandid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if float64(i) > p.Limit {
					break
				}
			}
		case "investigator":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d investigators", len(results.([]mig.Investigator)))}
			if len(results.([]mig.Investigator)) == 0 {
				panic("no results found")
			}
			for i, r := range results.([]mig.Investigator) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/investigator?investigatorid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if float64(i) > p.Limit {
					break
				}
			}
		case "manifest":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d manifests", len(results.([]mig.ManifestRecord)))}
			if len(results.([]mig.ManifestRecord)) == 0 {
				panic("no results found")
			}
			for i, r := range results.([]mig.ManifestRecord) {
				err = resource.AddItem(cljs.Item{
					Href: fmt.Sprintf("%s%s/manifest?manifestid=%.0f",
						ctx.Server.Host, ctx.Server.BaseRoute, r.ID),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if float64(i) > p.Limit {
					break
				}
			}
		case "loader":
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("returning search results with %d loaders", len(results.([]mig.LoaderEntry)))}
			if len(results.([]mig.LoaderEntry)) == 0 {
				panic("no results found")
			}
			for i, r := range results.([]mig.LoaderEntry) {
				err = resource.AddItem(cljs.Item{
					// XXX This should be an Href to fetch the entry
					Href: fmt.Sprintf("%s%s", ctx.Server.Host,
						ctx.Server.BaseRoute),
					Data: []cljs.Data{{Name: p.Type, Value: r}},
				})
				if err != nil {
					panic(err)
				}
				if float64(i) > p.Limit {
					break
				}
			}
		}
	}
	// if needed, add pagination info
	if p.Offset > 0 {
		nextP := p
		nextP.Offset += p.Limit
		page := pagination{
			Limit:  p.Limit,
			Offset: p.Offset,
			Next:   ctx.Server.BaseURL + "/search?" + nextP.String(),
		}
		err = resource.AddItem(cljs.Item{
			Href: loc,
			Data: []cljs.Data{{Name: "pagination", Value: page}},
		})
		if err != nil {
			panic(err)
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
	respond(http.StatusOK, resource, respWriter, request)
}

// truere is a case insensitive regex that matches the string 'true'
var truere = regexp.MustCompile("(?i)^true$")

// false is a case insensitive regex that matches the string 'false'
var falsere = regexp.MustCompile("(?i)^false$")

// parseSearchParameters transforms a query string into search parameters in the migdb format
func parseSearchParameters(qp url.Values) (p migdbsearch.Parameters, filterFound bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseSearchParameters()-> %v", e)
		}
	}()
	p = migdbsearch.NewParameters()
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
		case "agentversion":
			p.AgentVersion = qp["agentversion"][0]
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
			p.Limit, err = strconv.ParseFloat(qp["limit"][0], 64)
			if err != nil {
				panic("invalid limit parameter")
			}
		case "loadername":
			p.LoaderName = qp["loadername"][0]
		case "loaderid":
			p.LoaderID = qp["loaderid"][0]
		case "manifestname":
			p.ManifestName = qp["manifestname"][0]
		case "manifestid":
			p.ManifestID = qp["manifestid"][0]
		case "offset":
			p.Offset, err = strconv.ParseFloat(qp["offset"][0], 64)
			if err != nil {
				panic("invalid offset parameter")
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
