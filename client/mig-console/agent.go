// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"mig.ninja/mig/client"

	"github.com/bobappleyard/readline"
)

// agentReader retrieves an agent from the api
// and enters prompt mode to analyze it
func agentReader(input string, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("agentReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'agent <agentid>'")
	}
	agtid, err := strconv.ParseFloat(inputArr[1], 64)
	if err != nil {
		panic(err)
	}
	agt, err := cli.GetAgent(agtid)
	if err != nil {
		panic(err)
	}

	fmt.Println("Entering agent reader mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Agent %.0f named '%s'\n", agt.ID, agt.Name)
	prompt := fmt.Sprintf("\x1b[34;1magent %d>\x1b[0m ", uint64(agtid)%1000)
	for {
		// completion
		var symbols = []string{"details", "exit", "help", "json", "pretty", "r", "lastactions"}
		readline.Completer = func(query, ctx string) []string {
			var res []string
			for _, sym := range symbols {
				if strings.HasPrefix(sym, query) {
					res = append(res, sym)
				}
			}
			return res
		}

		input, err := readline.String(prompt)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("error: ", err)
			break
		}
		orders := strings.Split(input, " ")
		switch orders[0] {
		case "details":
			agt, err = cli.GetAgent(agtid)
			if err != nil {
				panic(err)
			}
			jEnv, err := json.MarshalIndent(agt.Env, "", "    ")
			if err != nil {
				panic(err)
			}
			jTags, err := json.MarshalIndent(agt.Tags, "", "    ")
			if err != nil {
				panic(err)
			}
			fmt.Printf(`Agent ID %.0f
name       %s
last seen  %s ago
version    %s
mode       %s
location   %s
platform   %s %s
pid        %d
starttime  %s
status     %s
environment %s
tags %s
`, agt.ID, agt.Name, time.Now().Sub(agt.HeartBeatTS).String(), agt.Version, agt.Mode, agt.QueueLoc,
				agt.Env.OS, agt.Env.Arch, agt.PID, agt.StartTime, agt.Status, jEnv, jTags)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
details			print the details of the agent
exit			exit this mode
help			show this help
json <pretty>		show the json of the agent registration
r			refresh the agent (get latest version from upstream)
lastactions <limit>	print the last actions that ran on the agent. limit=10 by default.
`)
		case "lastactions":
			limit := 10
			if len(orders) > 1 {
				limit, err = strconv.Atoi(orders[1])
				if err != nil {
					panic(err)
				}
			}
			err = printAgentLastCommands(agtid, limit, cli)
			if err != nil {
				panic(err)
			}
		case "json":
			var agtjson []byte
			if len(orders) > 1 {
				if orders[1] == "pretty" {
					agtjson, err = json.MarshalIndent(agt, "", "  ")
				} else {
					fmt.Printf("Unknown option '%s'\n", orders[1])
				}
			} else {
				agtjson, err = json.Marshal(agt)
			}
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", agtjson)
		case "r":
			agt, err = cli.GetAgent(agtid)
			if err != nil {
				panic(err)
			}
			fmt.Println("Reload succeeded")
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'. You are in agent reader mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}
exit:
	fmt.Printf("\n")
	return
}

func printAgentLastCommands(agtid float64, limit int, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("printAgentLastCommands() -> %v", e)
		}
	}()
	target := fmt.Sprintf("search?type=command&agentid=%.0f&limit=%d", agtid, limit)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	fmt.Printf("-------  ID  ------- + --------    Action Name ------- + ----    Date    ---- +  -- Status --\n")
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			if data.Name != "command" {
				continue
			}
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
			fmt.Printf("%.0f     %s   %s    %s\n", cmd.ID, name,
				cmd.StartTime.Format(time.RFC3339), cmd.Status)
		}
	}
	return
}
