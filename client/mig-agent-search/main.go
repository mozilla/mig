// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/client"
	migdbsearch "mig.ninja/mig/database/search"
)

func main() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", e)
			os.Exit(1)
		}
	}()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `%s <query> - Search for MIG Agents
Usage: %s -p "console style query" | -t "target style query"

The -p or -t flag must be specified.

CONSOLE MODE QUERY
------------------

The console mode query allows specific of a query string as would be passed
in mig-console using "search agent". It returns all matching agents.

EXAMPLE CONSOLE MODE QUERIES
----------------------------

All online agents:
  $ mig-agent-search -p "status=online"

All agents regardless of status:
  $ mig-agent-search -p "status=%%"

See the output of "search help" in mig-console for additional information on
how to format these queries.

TARGET MODE QUERY
-----------------

The target mode query allows specification of an agent targeting string as would
be passed to the -t flag using MIG command line. This evaluates agents using the
targeting string as the command line would, returning matching agents.

A search query is a SQL WHERE condition. It can filter on any field present in
the MIG Agents table.
	     Column      |           Type
	-----------------+-------------------------
	 id              | numeric
	 name            | character varying(2048)
	 queueloc        | character varying(2048)
	 mode            | character varying(2048)
	 version         | character varying(2048)
	 pid             | integer
	 starttime       | timestamp with time zone
	 destructiontime | timestamp with time zone
	 heartbeattime   | timestamp with time zone
	 status          | character varying(255)
	 environment     | json
	 tags            | json

The "environment" and "tags" fields are free JSON fields and can be queried using
Postgresql's JSON querying syntax.

Below is an example of environment document:
	{
	    "addresses": [
		"172.21.0.3/20",
		"fe80::3602:86ff:fe2b:6fdd/64"
	    ],
	    "arch": "amd64",
	    "ident": "Debian testing-updates sid",
	    "init": "upstart",
	    "isproxied": false,
	    "os": "linux",
	    "publicip": "172.21.0.3"
	}

Below is an example of tags document:
	{"operator":"linuxwall"}

EXAMPLE TARGET MODE QUERIES
---------------------------

Agent name "myserver.example.net"
  $ mig-agent-search -t "name='myserver.example.net'"

All Linux agents:
  $ mig-agent-search -t "environment->>'os'='linux'"

Ubuntu agents running 32 bits
  $ mig-agent-search -t "environment->>'ident' LIKE 'Ubuntu%%' AND environment->>'arch'='386'"

MacOS agents in datacenter SCL3
  $ mig-agent-search -t "environment->>'os'='darwin' AND name LIKE '%%\.scl3\.%%'"

Agents with uptime greater than 30 days
  $ mig-agent-search -t "starttime < NOW() - INTERVAL '30 days'"

Linux agents in checkin mode that are currently idle but woke up in the last hour
  $ mig-agent-search -t "mode='checkin' AND environment->>'os'='linux' AND status='idle' AND starttime > NOW() - INTERVAL '1 hour'"

Agents operated by team "opsec"
  $ mig-agent-search -t "tags->>'operator'='opsec'"

Command line flags:
`,
			os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	var err error
	homedir := client.FindHomedir()
	var config = flag.String("c", homedir+"/.migrc", "Load configuration from file")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	var paramSearch = flag.String("p", "", "Search using mig-console search style query")
	var targetSearch = flag.String("t", "", "Search using agent targeting string")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	// Instantiate an API client
	conf, err := client.ReadConfiguration(*config)
	if err != nil {
		panic(err)
	}
	cli, err := client.NewClient(conf, "agent-search-"+mig.Version)
	if err != nil {
		panic(err)
	}

	if *paramSearch != "" {
		// Search using mig-console style keywords
		p, err := parseSearchQuery(*paramSearch)
		if err != nil {
			panic(err)
		}
		resources, err := cli.GetAPIResource("search?" + p.String())
		if err != nil {
			panic(err)
		}
		fmt.Println("name; id; status; version; mode; os; arch; pid; starttime; heartbeattime; tags; environment")
		for _, item := range resources.Collection.Items {
			for _, data := range item.Data {
				if data.Name != "agent" {
					continue
				}
				agt, err := client.ValueToAgent(data.Value)
				if err != nil {
					panic(err)
				}
				err = printAgent(agt)
				if err != nil {
					panic(err)
				}
			}
		}
	} else if *targetSearch != "" {
		// Search using an agent targeting string
		agents, err := cli.EvaluateAgentTarget(*targetSearch)
		if err != nil {
			panic(err)
		}
		fmt.Println("name; id; status; version; mode; os; arch; pid; starttime; heartbeattime; tags; environment")
		for _, agt := range agents {
			err = printAgent(agt)
			if err != nil {
				panic(err)
			}
		}
	} else {
		panic("must specify -p or -t, see help")
	}
	os.Exit(0)
}

func printAgent(agt mig.Agent) error {
	tags, err := json.Marshal(agt.Tags)
	if err != nil {
		return err
	}
	env, err := json.Marshal(agt.Env)
	if err != nil {
		return err
	}
	fmt.Printf("%s; %.0f; %s; %s; %s; %s; %s; %d; %s; %s; %s; %s\n",
		agt.Name, agt.ID, agt.Status, agt.Version, agt.Mode, agt.Env.OS,
		agt.Env.Arch, agt.PID, agt.StartTime.Format(time.RFC3339),
		agt.HeartBeatTS.Format(time.RFC3339), tags, env)
	return nil
}

// Transform a mig-console style search query into a set of parameters to send to the API
//
// This function is similar to the function in mig-console, however we do not include
// parameters that are not relevant to agents.
func parseSearchQuery(querystring string) (p migdbsearch.Parameters, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("parseSearchQuery() -> %v", e)
		}
	}()

	p = migdbsearch.NewParameters()
	p.Type = "agent"

	orders := strings.Split(querystring, " ")
	if len(orders) == 0 {
		panic("no criteria specified")
	}

	for _, order := range orders {
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
		case "limit":
			p.Limit, err = strconv.ParseFloat(value, 64)
			if err != nil {
				panic("invalid limit parameter")
			}
		case "status":
			p.Status = value
		case "name":
			p.AgentName = value
		default:
			panic(fmt.Sprintf("Unknown search key '%s'", key))
		}
	}
	return
}
