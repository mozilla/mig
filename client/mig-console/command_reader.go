// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"github.com/bobappleyard/readline"
	"io"
	"mig.ninja/mig/client"
	"strconv"
	"strings"
)

// commandReader retrieves an command from the API using its numerical ID
// and enters prompt mode to analyze it
func commandReader(input string, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("commandReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'command <commandid>'")
	}
	cmdid, err := strconv.ParseFloat(inputArr[1], 64)
	if err != nil {
		panic(err)
	}
	cmd, err := cli.GetCommand(cmdid)
	if err != nil {
		panic(err)
	}

	fmt.Println("Entering command reader mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Command %.0f ran on agent '%s' based on action '%s'\n",
		cmd.ID, cmd.Agent.Name, cmd.Action.Name)
	prompt := fmt.Sprintf("\x1b[36;1mcommand %d>\x1b[0m ", uint64(cmdid)%1000)
	for {
		// completion
		var symbols = []string{"exit", "help", "json", "found", "pretty", "r", "results"}
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
		orders := strings.Split(strings.TrimSpace(input), " ")
		switch orders[0] {
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
exit		exit this mode
help		show this help
json		show the json of the command
r		refresh the command (get latest version from upstream)
results <found>	print the results. if "found" is set, only print results that have at least one found
`)
		case "json":
			var cjson []byte
			cjson, err = json.MarshalIndent(cmd, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", cjson)
		case "r":
			cmd, err = cli.GetCommand(cmdid)
			if err != nil {
				panic(err)
			}
			fmt.Println("Reload succeeded")
		case "results":
			found := false
			if len(orders) > 1 {
				if orders[1] == "found" {
					found = true
				} else {
					fmt.Printf("Unknown option '%s'\n", orders[1])
				}
			}
			err = client.PrintCommandResults(cmd, found, false)
			if err != nil {
				panic(err)
			}
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'. You are in command reader mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}
exit:
	fmt.Printf("\n")
	return
}

func commandPrintShort(data interface{}) (idstr, agtname, duration, status string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("commandPrintShort() -> %v", e)
		}
	}()
	cmd, err := client.ValueToCommand(data)
	if err != nil {
		panic(err)
	}
	idstr = fmt.Sprintf("%.0f", cmd.ID)
	if len(idstr) < 20 {
		for i := len(idstr); i < 20; i++ {
			idstr += " "
		}
	}

	agtname = cmd.Agent.Name
	if len(agtname) < 30 {
		for i := len(agtname); i < 30; i++ {
			agtname += " "
		}
	}
	if len(agtname) > 30 {
		agtname = agtname[0:27] + "..."
	}

	duration = cmd.FinishTime.Sub(cmd.StartTime).String()
	if len(duration) > 10 {
		duration = duration[0:8] + duration[len(duration)-3:len(duration)-1]
	}
	if len(duration) < 10 {
		for i := len(duration); i < 10; i++ {
			duration += " "
		}
	}

	status = cmd.Status
	if len(status) > 10 {
		status = status[0:9]
	}
	if len(status) < 10 {
		for i := len(status); i < 10; i++ {
			status += " "
		}
	}

	return
}
