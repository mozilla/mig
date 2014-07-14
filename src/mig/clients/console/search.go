/* Mozilla InvestiGator Console

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
Portions created by the Initial Developer are Copyright (C) 2014
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
	"strings"
	"time"

	"github.com/jvehent/cljs"
)

// search runs a search for actions, commands or agents
func search(input string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("search() -> %v", e)
		}
	}()
	orders := strings.Split(input, " ")
	if len(orders) < 2 {
		orders = append(orders, "help")
	}
	sType := ""
	switch orders[1] {
	case "action", "agent", "command":
		sType = orders[1]
	case "", "help":
		fmt.Printf(`usage: search <action|agent|command> where <parameters> [<and|or> <parameters>]
The following search parameters are available:
`)
		return nil
	default:
		return fmt.Errorf("Invalid search '%s'. Try `search help`.\n", input)
	}
	query, err := parseSearchQuery(orders)
	if err != nil {
		panic(err)
	}
	items, err := runSearchQuery(query, ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("----    ID      ---- + ----         Name         ---- + ---- Last Update ----")
	for _, item := range items {
		for _, data := range item.Data {
			if data.Name != sType {
				continue
			}
			switch data.Name {
			case "action":
				idstr, name, datestr, _, err := actionPrintShort(data.Value)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%s   %s   %s\n", idstr, name, datestr)
			case "command":
				cmd, err := valueToCommand(data.Value)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%20.0f   %s   %s\n", cmd.ID, cmd.Agent.Name, cmd.FinishTime.Format(time.RFC3339))
			case "agent":
				agt, err := valueToAgent(data.Value)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%20.0f   %s   %s\n", agt.ID, agt.Name[0:30], agt.HeartBeatTS.Format(time.RFC3339))
			}
		}
	}
	return
}

// parseSearchQuery transforms a search string into an API query
func parseSearchQuery(orders []string) (query string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseSearchQuery() -> %v", e)
		}
	}()
	sType := orders[1]
	query = "search?type=" + sType
	if len(orders) < 4 {
		panic("Invalid search syntax. try `search help`.")
	}
	if orders[2] != "where" {
		panic(fmt.Sprintf("Expected keyword 'where' after search type. Got '%s'", orders[2]))
	}
	for _, order := range orders[3:len(orders)] {
		if order == "and" || order == "or" {
			continue
		}
		params := strings.Split(order, "=")
		if len(params) != 2 {
			panic(fmt.Sprintf("Invalid `key=value` for in parameter '%s'", order))
		}
		key := params[0]
		// if the string contains % characters, used in postgres's pattern matching,
		// escape them properly
		value := strings.Replace(params[1], "%", "%25", -1)
		// wildcards are converted to postgres's % pattern matching
		value = strings.Replace(value, "*", "%25", -1)
		switch key {
		case "and", "or":
			continue
		case "agentname":
			query += "&agentname=" + value
		case "after":
			query += "&after=" + value
		case "before":
			query += "&before=" + value
		case "id":
			panic("If you already know the ID, don't use the search. Use (action|command|agent) <id> directly")
		case "actionid":
			query += "&actionid=" + value
		case "commandid":
			query += "&commandid=" + value
		case "agentid":
			query += "&agentid=" + value
		case "name":
			switch sType {
			case "action", "command":
				query += "&actionname=" + value
			case "agent":
				query += "&agentname=" + value
			}
		case "status":
			switch sType {
			case "action":
				panic("'status' is not a valid action search parameter")
			case "command", "agent":
				query += "&status=" + value
			}
		case "limit":
			query += "&limit=" + value
		default:
			panic(fmt.Sprintf("Unknown search key '%s'", key))
		}
	}
	return
}

// runSearchQuery executes a search string against the API
func runSearchQuery(query string, ctx Context) (items []cljs.Item, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("runSearchQuery() -> %v", e)
		}
	}()
	fmt.Println("Search query:", query)
	targetURL := ctx.API.URL + query
	resource, err := getAPIResource(targetURL, ctx)
	if err != nil {
		panic(err)
	}
	items = resource.Collection.Items
	return
}
