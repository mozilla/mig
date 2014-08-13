// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"mig"
	migdb "mig/database"

	"github.com/jvehent/cljs"
)

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

// agentToItem receives an agent and returns an Item in Collection+JSON
func agentToItem(agt mig.Agent) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/agent?agentid=%.0f", ctx.Server.BaseURL, agt.ID)
	item.Data = []cljs.Data{
		{Name: "agent", Value: agt},
	}
	return
}

// agentsSumToItem receives an AgentsSum and returns an Item
// in the Collection+JSON format
func agentsSummaryToItem(sum []migdb.AgentsSum, count, double, disappeared float64, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/dashboard", ctx.Server.BaseURL)
	var total float64 = 0
	for _, asum := range sum {
		total += asum.Count
	}
	item.Data = []cljs.Data{
		{Name: "active agents", Value: total},
		{Name: "agents versions count", Value: sum},
		{Name: "agents started in the last 24 hours", Value: count},
		{Name: "endpoints running 2 or more agents", Value: double},
		{Name: "endpoints that have disappeared over last 7 days", Value: disappeared},
	}
	links := make([]cljs.Link, 0)
	link := cljs.Link{
		Rel:  "agents dashboard",
		Href: fmt.Sprintf("%s/agents/dashboard", ctx.Server.BaseURL),
	}
	links = append(links, link)
	item.Links = links
	return
}
