package main

import (
	"flag"
	"fmt"
	"mig/client"
	"os"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s - Find MIG agents by name\n"+
			"Usage: %s some.agent.example.net some.other.agent.example.com\n",
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
	cli := client.NewClient(conf)

	args := flag.Args()
	for _, arg := range args {
		// replace % and * wildcards with %25 (url encoded)
		search := strings.Replace(arg, "%", "%25", -1)
		search = strings.Replace(search, "*", "%25", -1)
		r, err := cli.GetAPIResource("search?type=agent&agentname=" + search)
		if err != nil {
			panic(err)
		}
		fmt.Printf("name id os pid status isproxied\n")
		for _, item := range r.Collection.Items {
			if item.Data[0].Name != "agent" {
				continue
			}
			agt, err := client.ValueToAgent(item.Data[0].Value)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s %.0f %s %d %s %t\n",
				agt.Name, agt.ID, agt.Env.OS, agt.PID,
				agt.Status, agt.Env.IsProxied)
		}
	}
}
