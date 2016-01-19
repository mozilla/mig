package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/client"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `%s <query> - Search for MIG Agents
Usage: %s "name='some.agent.example.net' OR name='some.other.agent.example.com'"

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

EXAMPLE QUERIES
---------------

Agent name "myserver.example.net"
  $ mig-agent-search "name='myserver.example.net'"

All Linux agents:
  $ mig-agent-search "environment->>'os'='linux'"

Ubuntu agents running 32 bits
  $ mig-agent-search "environment->>'ident' LIKE 'Ubuntu%%' AND environment->>'arch'='386'

MacOS agents in datacenter SCL3
  $ mig-agent-search "environment->>'os'='darwin' AND name LIKE '%%\.scl3\.%%'

Agents with uptime greater than 30 days
  $ mig-agent-search "starttime < NOW() - INTERVAL '30 days'"

Linux agents in checkin mode that are currently idle but woke up in the last hour
  $ mig-agent-search "mode='checkin' AND environment->>'os'='linux' AND status='idle' AND starttime > NOW() - INTERVAL '1 hour'"

Agents operated by team "opsec"
  $ mig-agent-search "tags->>'operator'='opsec'"

Command line flags:
`,
			os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	var err error
	homedir := client.FindHomedir()
	var config = flag.String("c", homedir+"/.migrc", "Load configuration from file")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	// instanciate an API client
	conf, err := client.ReadConfiguration(*config)
	if err != nil {
		panic(err)
	}
	cli, err := client.NewClient(conf, "agent-search-"+mig.Version)
	if err != nil {
		panic(err)
	}
	agents, err := cli.EvaluateAgentTarget(strings.Join(flag.Args(), " "))
	if err != nil {
		panic(err)
	}
	fmt.Println("name; id; status; version; mode; os; arch; pid; starttime; heartbeattime; tags; environment")
	for _, agt := range agents {
		tags, err := json.Marshal(agt.Tags)
		if err != nil {
			panic(err)
		}
		env, err := json.Marshal(agt.Env)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s; %.0f; %s; %s; %s; %s; %s; %d; %s; %s; %s; %s\n",
			agt.Name, agt.ID, agt.Status, agt.Version, agt.Mode, agt.Env.OS, agt.Env.Arch, agt.PID, agt.StartTime.Format(time.RFC3339),
			agt.HeartBeatTS.Format(time.RFC3339), tags, env)
	}
}
