package main

import (
	"flag"
	"fmt"
	"mig/client"
	"os"
	"strings"
	"time"
)

var version string

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s - Search for MIG Agents\n"+
			"Usage: %s name='some.agent.example.net' OR name='some.other.agent.example.com'\n",
			os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	var err error
	homedir := client.FindHomedir()
	var config = flag.String("c", homedir+"/.migrc", "Load configuration from file")
	flag.Parse()

	// instanciate an API client
	conf, err := client.ReadConfiguration(*config)
	if err != nil {
		panic(err)
	}
	cli, err = client.NewClient(conf, "agent-search-"+version)
	if err != nil {
		panic(err)
	}
	agents, err := cli.EvaluateAgentTarget(strings.Join(flag.Args(), " "))
	if err != nil {
		panic(err)
	}
	fmt.Println("name; id; status; version; mode; os; arch; pid; starttime; heartbeattime; operator; ident; publicip; addresses")
	for _, agt := range agents {
		operator := "unknown"
		if _, ok := agt.Tags.(map[string]interface{})["operator"]; ok {
			operator = agt.Tags.(map[string]interface{})["operator"].(string)
		}
		fmt.Printf("\"%s\"; \"%.0f\"; \"%s\"; \"%s\"; \"%s\"; \"%s\"; \"%s\"; \"%d\"; \"%s\"; \"%s\"; \"%s\"; \"%s\"; \"%s\"; \"%s\"\n",
			agt.Name, agt.ID, agt.Status, agt.Version, agt.Mode, agt.Env.OS, agt.Env.Arch, agt.PID, agt.StartTime.Format(time.RFC3339),
			agt.HeartBeatTS.Format(time.RFC3339), operator, agt.Env.Ident, agt.Env.PublicIP, agt.Env.Addresses)
	}
}
