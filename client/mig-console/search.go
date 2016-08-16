// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mig.ninja/mig/client"
	migdbsearch "mig.ninja/mig/database/search"
)

// search runs a search for actions, commands or agents
func search(input string, cli client.Client) (err error) {
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
	case "action", "agent", "command", "investigator", "manifest", "loader":
		sType = orders[1]
	case "", "help":
		fmt.Printf(`usage: search <action|agent|command|investigator|loader|manifest> where <key>=<value> [and <key>=<value>...]

Example:
mig> search command where agentname=%%khazad%% and investigatorname=%%vehent%% and actionname=%%memory%% and after=2015-09-09T17:00:00Z
	----    ID      ---- + ----         Name         ---- + --- Last Updated ---
	       4886304327951   memory -c /home/ulfr/.migrc...   2015-09-09T13:01:03-04:00

The following search parameters are available, per search type:
* action:
	- name=<str>		search actions by name <str>
	- before=<rfc3339>	search actions that expired before <rfc3339 date>
	- after=<rfc3339>	search actions were valid after <rfc3339 date>
	- commandid=<id>	search action that spawned a given command
	- agentid=<id>		search actions that ran on a given agent
	- agentname=<str>	search actions that ran on an agent named <str>
	- investigatorid=<id>	search actions signed by a given investigator
	- investigatorname=<str>search actions signed by investigator named <str>
	- status=<str>		search actions with a given status amongst:
				pending, scheduled, preparing, invalid, inflight, completed
* command:
	- name=<str>		search commands by action name <str>
	- before=<rfc3339>	search commands that started before <rfc3339 date>
	- after=<rfc3339>	search commands that started after <rfc3339 date>
	- actionid=<id>		search commands spawned action <id>
	- actionname=<str>	search commands spawned by an action named <str>
	- agentname=<str>	search commands that ran on an agent named <str>
	- agentid=<id>		search commands that ran on a given agent
	- investigatorid=<id>	search commands signed by investigator <id>
	- investigatorname=<str>search commands signed by investigator named <str>
	- status=<str>		search commands with a given status amongst:
				prepared, sent, success, timeout, cancelled, expired, failed
* agent:
	- name=<str>		search agents by hostname
	- before=<rfc3339>	search agents that have sent a heartbeat before <rfc3339 date>
	- after=<rfc3339>	search agents that have sent a heartbeat after <rfc3339 date>
	- actionid=<id>		search agents that ran action <id>
	- actionname=<str>	search agents that ran action named <str>
	- commandid=<id>	search agents that ran command <id>
	- investigatorid=<id>	search agents that ran an action signed by investigator <id>
	- investigatorname=<str>search agents that ran an action signed by investigator named <str>
	- version=<str>		search agents by version <str>
	- status=<str>		search agents with a given status amongst:
				online, upgraded, destroyed, offline, idle
* investigator:
	- name=<str>		search investigators by name
	- before=<rfc3339>	search investigators created or modified before <rfc3339 date>
	- after=<rfc3339>	search investigators created or modified after <rfc3339 date>
	- actionid=<id>		search investigators that signed action <id>
	- actionname=<str>	search investigators that signed action named <str>
	- commandid=<id>	search investigators that ran command <id>
	- agentid=<id>		search investigators that ran a command on a given agent
	- agentname=<str>	search investigators that ran actions on an agent named <str>,
	- status=<str>		search investigators by status amongst: active, disabled

* manifest:
        - manifestid=<id>       search manifests by id
        - manifestname=<str>    search manifests by name
	- status=<str>          search manifests by status amongst: active, staged, disabled

* loader:
        - loaderid=<id>         search loaders by id
        - loadername=<str>      search loaders by loader name
        - agentname=<str>       search loaders for associated agent names

All searches accept the 'limit=<num>' parameter to limits the number of results returned by a search, defaults to 100
Parameters that accept a <str> can use wildcards * and % (ex: name=jul%veh% ).
No spaces are permitted within parameters. Spaces are used to separate search parameters.
`)
		return nil
	default:
		return fmt.Errorf("Invalid search '%s'. Try `search help`.\n", input)
	}
	p, err := parseSearchQuery(orders)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Searching %s after %s and before %s, limited to %.0f results\n", p.Type,
		p.After.Format(time.RFC3339), p.Before.Format(time.RFC3339), p.Limit)
	resources, err := cli.GetAPIResource("search?" + p.String())
	if err != nil {
		if strings.Contains(fmt.Sprintf("%v", err), "HTTP 404") {
			panic("No results found for search query: " + p.String())
		}
		panic(err)
	}
	switch sType {
	case "agent":
		fmt.Println("---   ID  ---- + ----         Name         ---- + -- Status -- + -- Last Heartbeat --")
	case "action":
		fmt.Println("----- ID ----- + --------   Action Name ------- + ----------- Target  ---------- + ---- Investigators ---- + - Sent - + - Status - + --- Last Updated --- ")
	case "command":
		fmt.Println("----  ID  ---- + ----         Name         ---- + --- Last Updated ---")
	case "investigator":
		fmt.Println("- ID - + ----         Name         ---- + --- Status --- + --- Permissions ---")
	case "manifest":
		fmt.Println("- ID - + ----      Name      ---- + -- Status -- + -------------- Target -------- + ---- Timestamp ---")
	case "loader":
		fmt.Println("- ID - + ----      Name      ---- + ----   Agent Name   ---- + -- Enabled - + -- Last Used ---")
	}
	for _, item := range resources.Collection.Items {
		for _, data := range item.Data {
			if data.Name != sType {
				continue
			}
			switch data.Name {
			case "action":
				idstr, name, target, datestr, invs, status, sent, err := actionPrintShort(data.Value)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%s   %s   %s   %s   %8d   %s   %s\n", idstr, name, target, invs, sent,
					status, datestr)
			case "command":
				cmd, err := client.ValueToCommand(data.Value)
				if err != nil {
					panic(err)
				}
				name := cmd.Action.Name
				if len(name) < 30 {
					for i := len(name); i < 30; i++ {
						name += " "
					}
				}
				if len(name) > 30 {
					name = name[0:27] + "..."
				}
				fmt.Printf("%14.0f   %s   %s\n", cmd.ID, name,
					cmd.FinishTime.UTC().Format(time.RFC3339))

			case "agent":
				agt, err := client.ValueToAgent(data.Value)
				if err != nil {
					panic(err)
				}
				name := agt.Name
				if len(name) < 30 {
					for i := len(name); i < 30; i++ {
						name += " "
					}
				}
				if len(name) > 30 {
					name = name[0:27] + "..."
				}
				status := agt.Status
				if len(status) < 12 {
					for i := len(status); i < 12; i++ {
						status += " "
					}
				}
				if len(status) > 12 {
					status = status[0:12]
				}
				fmt.Printf("%20.0f   %s   %s   %s\n", agt.ID, name, status,
					agt.HeartBeatTS.UTC().Format(time.RFC3339))
			case "investigator":
				inv, err := client.ValueToInvestigator(data.Value)
				if err != nil {
					panic(err)
				}
				name := inv.Name
				if len(name) < 30 {
					for i := len(name); i < 30; i++ {
						name += " "
					}
				}
				if len(name) > 30 {
					name = name[0:27] + "..."
				}
				sts := inv.Status
				if len(sts) < 17 {
					for i := len(sts); i < 16; i++ {
						sts += " "
					}
				}
				fmt.Printf("%6.0f   %s   %s %s\n", inv.ID, name, sts,
					inv.Permissions.ToDescriptive())
			case "manifest":
				mr, err := client.ValueToManifestRecord(data.Value)
				if err != nil {
					panic(err)
				}
				name := mr.Name
				if len(name) < 24 {
					for i := len(name); i < 24; i++ {
						name += " "
					}
				}
				if len(name) > 24 {
					name = name[0:21] + "..."
				}
				status := mr.Status
				if len(status) < 12 {
					for i := len(status); i < 12; i++ {
						status += " "
					}
				}
				if len(status) > 12 {
					status = status[0:12]
				}
				target := mr.Target
				if len(target) < 30 {
					for i := len(target); i < 30; i++ {
						target += " "
					}
				}
				if len(target) > 30 {
					target = target[0:27] + "..."
				}
				fmt.Printf("%6.0f   %s   %s   %s   %s\n", mr.ID, name,
					status, target, mr.Timestamp.UTC().Format(time.RFC3339))
			case "loader":
				le, err := client.ValueToLoaderEntry(data.Value)
				if err != nil {
					panic(err)
				}
				loadername := le.Name
				if len(loadername) < 24 {
					for i := len(loadername); i < 24; i++ {
						loadername += " "
					}
				}
				if len(loadername) > 24 {
					loadername = loadername[0:21] + "..."
				}
				agtname := le.AgentName
				if len(agtname) < 24 {
					for i := len(agtname); i < 24; i++ {
						agtname += " "
					}
				}
				if len(agtname) > 24 {
					agtname = agtname[0:21] + "..."
				}
				loaderstatus := fmt.Sprintf("%v", le.Enabled)
				for i := len(loaderstatus); i < 12; i++ {
					loaderstatus += " "
				}
				fmt.Printf("%6.0f   %s   %s   %s   %s\n", le.ID, loadername,
					agtname, loaderstatus,
					le.LastSeen.UTC().Format(time.RFC3339))
			}
		}
	}
	return
}

// parseSearchQuery transforms a search string into an API query
func parseSearchQuery(orders []string) (p migdbsearch.Parameters, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseSearchQuery() -> %v", e)
		}
	}()
	p = migdbsearch.NewParameters()
	p.Type = orders[1]
	if len(orders) < 4 {
		panic("Invalid search syntax. try `search help`.")
	}
	if orders[2] != "where" {
		panic(fmt.Sprintf("Expected keyword 'where' after search type. Got '%s'", orders[2]))
	}
	for _, order := range orders[3:len(orders)] {
		if order == "and" {
			continue
		}
		params := strings.Split(order, "=")
		if len(params) != 2 {
			panic(fmt.Sprintf("Invalid `key=value` in search parameter '%s'", order))
		}
		key := params[0]
		value := params[1]
		// if the string contains % characters, used in postgres's pattern matching,
		// escape them properly
		switch key {
		case "actionname":
			p.ActionName = value
		case "actionid":
			p.ActionID = value
		case "after":
			p.After, err = time.Parse(time.RFC3339, value)
			if err != nil {
				panic("after date not in RFC3339 format, ex: 2015-09-23T14:14:16Z")
			}
		case "agentid":
			p.AgentID = value
		case "agentname":
			p.AgentName = value
		case "agentversion":
			p.AgentVersion = value
		case "before":
			p.Before, err = time.Parse(time.RFC3339, value)
			if err != nil {
				panic("before date not in RFC3339 format, ex: 2015-09-23T14:14:16Z")
			}
		case "commandid":
			p.CommandID = value
		case "investigatorid":
			p.InvestigatorID = value
		case "investigatorname":
			p.InvestigatorName = value
		case "limit":
			p.Limit, err = strconv.ParseFloat(value, 64)
			if err != nil {
				panic("invalid limit parameter")
			}
		case "loadername":
			p.LoaderName = value
		case "loaderid":
			p.LoaderID = value
		case "manifestname":
			p.ManifestName = value
		case "manifestid":
			p.ManifestID = value
		case "status":
			p.Status = value
		case "name":
			switch p.Type {
			case "action", "command":
				p.ActionName = value
			case "agent":
				p.AgentName = value
			case "investigator":
				p.InvestigatorName = value
			}
		default:
			panic(fmt.Sprintf("Unknown search key '%s'", key))
		}
	}
	return
}

// filterString matches an input string against a filter that's an array of string in the form
// ['|', 'grep', 'something', '|', 'grep', '-v', 'notsomething']
func filterString(input string, filter []string) (output string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("filterString() -> %v", e)
		}
	}()
	const (
		modeNull = 1 << iota
		modePipe
		modeGrep
		modeInverseGrep
		modeConsumed
	)
	mode := modeNull
	for _, comp := range filter {
		switch comp {
		case "|":
			if mode != modeNull {
				panic("Invalid pipe placement")
			}
			mode = modePipe
			continue
		case "grep":
			if mode != modePipe {
				panic("grep must be preceded by a pipe")
			}
			mode = modeGrep
		case "-v":
			if mode != modeGrep {
				panic("-v is an option of grep, but grep is missing")
			}
			mode = modeInverseGrep
		default:
			if mode == modeNull {
				panic("unknown filter mode")
			} else if (mode == modeGrep) || (mode == modeInverseGrep) {
				re, err := regexp.CompilePOSIX(comp)
				if err != nil {
					panic(err)
				}
				if re.MatchString(input) {
					// the string matches, but we want inverse grep
					if mode == modeInverseGrep {
						return "", err
					}
				} else {
					// the string doesn't match, and we want grep
					if mode == modeGrep {
						return "", err
					}
				}
			} else {
				panic("unrecognized filter syntax")
			}
			// reset the mode
			mode = modeNull
		}
	}
	output = input
	return
}
