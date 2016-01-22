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

	"github.com/bobappleyard/readline"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
)

// actionReader retrieves an action from the API using its numerical ID
// and enters prompt mode to analyze it
func actionReader(input string, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("actionReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'action <actionid>'")
	}
	aid, err := strconv.ParseFloat(inputArr[1], 64)
	if err != nil {
		panic(err)
	}
	a, _, err := cli.GetAction(aid)
	if err != nil {
		panic(err)
	}
	investigators := investigatorsStringFromAction(a.Investigators, 80)

	fmt.Println("Entering action reader mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Action: '%s'.\nLaunched by '%s' on '%s'.\nStatus '%s'.\n",
		a.Name, investigators, a.StartTime, a.Status)
	a.PrintCounters()
	prompt := fmt.Sprintf("\x1b[31;1maction %d>\x1b[0m ", uint64(aid)%1000)
	for {
		// completion
		var symbols = []string{"command", "copy", "counters", "details", "exit", "grep", "help", "investigators",
			"json", "list", "all", "found", "notfound", "pretty", "r", "results", "times"}
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
		case "command":
			err = commandReader(input, cli)
			if err != nil {
				panic(err)
			}
		case "copy":
			err = actionLauncher(a, cli)
			if err != nil {
				panic(err)
			}
			goto exit
		case "counters":
			a, _, err = cli.GetAction(aid)
			if err != nil {
				panic(err)
			}
			a.PrintCounters()
		case "details":
			actionPrintDetails(a)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
command <id>	jump to command reader mode for command <id>

copy		enter action launcher mode using current action as template

counters	display the counters of the action

details		display the details of the action, including status & times

exit		exit this mode (also works with ctrl+d)

help		show this help

investigators   print the list of investigators that signed the action

json         	show the json of the action

list <show>	returns the list of commands with their status
		<show>: * set to "all" to get all results (default)
			* set to "found" to only display positive results
			* set to "notfound" for negative results
		list can be followed by a 'filter' pipe:
		ex: ls | grep server1.(dom1|dom2) | grep -v example.net

r		refresh the action (get latest version from upstream)

results <show> <render>	display results of all commands
			<show>: * set to "all" to get all results (default)
				* set to "found" to only display positive results
				* set to "notfound" for negative results
			<render>: * set to "text" to print results in console (default)
				  * set to "map" to generate an open a google map

times		show the various timestamps of the action
`)
		case "investigators":
			for _, i := range a.Investigators {
				fmt.Println(i.Name, "- Key ID:", i.PGPFingerprint)
			}
		case "json":
			tmpAction, err := getActionView(a)
			if err != nil {
				panic(err)
			}
			var ajson []byte
			ajson, err = json.MarshalIndent(tmpAction, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", ajson)
		case "list":
			err = actionPrintList(aid, orders, cli)
			if err != nil {
				panic(err)
			}
		case "r":
			a, _, err = cli.GetAction(aid)
			if err != nil {
				panic(err)
			}
			fmt.Println("reloaded")
		case "results":
			show := "all"
			if len(orders) > 1 {
				switch orders[1] {
				case "all", "found", "notfound":
					show = orders[1]
				default:
					panic("invalid show '" + orders[2] + "'")
				}
			}
			render := "text"
			if len(orders) > 2 {
				switch orders[2] {
				case "map", "text":
					render = orders[2]
				default:
					panic("invalid rendering '" + orders[2] + "'")
				}
			}
			err = cli.PrintActionResults(a, show, render)
			if err != nil {
				panic(err)
			}
		case "times":
			fmt.Printf("Valid from   '%s' until '%s'\nStarted on   '%s'\n"+
				"Last updated '%s'\nFinished on  '%s'\n",
				a.ValidFrom, a.ExpireAfter, a.StartTime, a.LastUpdateTime, a.FinishTime)
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'. You are in action reader mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}
exit:
	fmt.Printf("\n")
	return
}

func actionPrintShort(data interface{}) (idstr, name, target, datestr, invs, status string, sent int, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("actionPrintShort() -> %v", e)
		}
	}()
	a, err := client.ValueToAction(data)
	if err != nil {
		panic(err)
	}
	invs = investigatorsStringFromAction(a.Investigators, 23)

	idstr = fmt.Sprintf("%.0f", a.ID)
	if len(idstr) < 14 {
		for i := len(idstr); i < 14; i++ {
			idstr += " "
		}
	}

	name = a.Name
	if len(name) < 30 {
		for i := len(name); i < 30; i++ {
			name += " "
		}
	}
	if len(name) > 30 {
		name = name[0:27] + "..."
	}

	target = a.Target
	if len(target) < 30 {
		for i := len(target); i < 30; i++ {
			target += " "
		}
	}
	if len(target) > 30 {
		target = target[0:27] + "..."
	}

	status = a.Status
	if len(status) < 10 {
		for i := len(status); i < 10; i++ {
			status += " "
		}
	}

	datestr = a.LastUpdateTime.UTC().Format(time.RFC3339)
	if len(datestr) > 21 {
		datestr = datestr[0:21]
	}
	if len(datestr) < 20 {
		for i := len(datestr); i < 20; i++ {
			datestr += " "
		}
	}
	sent = a.Counters.Sent
	return
}

func investigatorsStringFromAction(invlist []mig.Investigator, strlen int) (investigators string) {
	for ctr, i := range invlist {
		if ctr > 0 {
			investigators += "; "
		}
		investigators += i.Name
	}
	if len(investigators) > strlen {
		investigators = investigators[0:(strlen-3)] + "..."
	}
	if len(investigators) < strlen {
		for i := len(investigators); i < strlen; i++ {
			investigators += " "
		}
	}
	return
}

func actionPrintDetails(a mig.Action) {
	fmt.Printf(`
ID             %.0f
Name           %s
Target         %s
Desc           author '%s <%s>'; revision '%.0f';
               url '%s'
Threat         type '%s'; level '%s'; family '%s'; reference '%s'
Status         %s
Times          valid from %s until %s
               started %s; last updated %s; finished %s
               duration: %s
`, a.ID, a.Name, a.Target, a.Description.Author, a.Description.Email, a.Description.Revision,
		a.Description.URL, a.Threat.Type, a.Threat.Level, a.Threat.Family, a.Threat.Ref, a.Status,
		a.ValidFrom, a.ExpireAfter, a.StartTime, a.LastUpdateTime, a.FinishTime, a.LastUpdateTime.Sub(a.StartTime).String())
	fmt.Printf("Investigators  ")
	for _, i := range a.Investigators {
		fmt.Println(i.Name, "- keyid:", i.PGPFingerprint)
	}
	fmt.Printf("Operations     count=%d => ", len(a.Operations))
	for _, op := range a.Operations {
		fmt.Printf("%s; ", op.Module)
	}
	fmt.Printf("\n")
	fmt.Printf("Counters       sent=%d; done=%d; in flight=%d\n"+
		"               success=%d; cancelled=%d; expired=%d; failed=%d; timeout=%d\n",
		a.Counters.Sent, a.Counters.Done, a.Counters.InFlight, a.Counters.Success,
		a.Counters.Cancelled, a.Counters.Expired, a.Counters.Failed, a.Counters.TimeOut)
	return
}

func actionPrintList(aid float64, orders []string, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("actionPrintList() -> %v", e)
		}
	}()
	show := "all"
	has_filter := false
	var filter []string
	if len(orders) > 1 {
		switch orders[1] {
		case "all", "found", "notfound":
			show = orders[1]
			if len(orders) > 2 && orders[2] == "|" {
				has_filter = true
				filter = orders[2:]
			}
		case "|":
			has_filter = true
			filter = orders[1:]
		}
	}
	cmds, err := searchCommands(aid, show, cli)
	if err != nil {
		panic(err)
	}
	ctr := 0
	if len(cmds) > 0 {
		fmt.Println("---- Command ID ----    ---- Agent Name & ID----")
		for _, cmd := range cmds {
			str := fmt.Sprintf("%20.0f    %s [%.0f]", cmd.ID, cmd.Agent.Name, cmd.Agent.ID)
			if has_filter {
				filtered, err := filterString(str, filter)
				if err != nil {
					fmt.Printf("Invalid filter '%s': '%v'\n", filter, err)
					break
				}
				if filtered != "" {
					fmt.Println(filtered)
					ctr++
				}
			} else {
				fmt.Println(str)
				ctr++
			}
		}
	}
	switch show {
	case "found":
		fmt.Printf("%d agents have found things\n", ctr)
	case "notfound":
		fmt.Printf("%d agents have found nothing\n", ctr)
	case "all":
		fmt.Printf("%d agents have returned\n", ctr)
	}
	return
}

func searchCommands(aid float64, show string, cli client.Client) (cmds []mig.Command, err error) {
	defer func() {
		fmt.Printf("\n")
		if e := recover(); e != nil {
			err = fmt.Errorf("searchCommands() -> %v", e)
		}
	}()
	base := fmt.Sprintf("search?type=command&actionid=%.0f", aid)
	switch show {
	case "found":
		base += "&foundanything=true"
	case "notfound":
		base += "&foundanything=false"
	}
	offset := 0
	// loop until all results have been retrieved using paginated queries
	for {
		fmt.Printf(".")
		target := fmt.Sprintf("%s&limit=50&offset=%d", base, offset)
		resource, err := cli.GetAPIResource(target)
		// because we query using pagination, the last query will return a 404 with no result.
		// When that happens, GetAPIResource returns an error which we do not report to the user
		if resource.Collection.Error.Message == "no results found" {
			err = nil
			break
		} else if err != nil {
			panic(err)
		}
		for _, item := range resource.Collection.Items {
			for _, data := range item.Data {
				if data.Name != "command" {
					continue
				}
				cmd, err := client.ValueToCommand(data.Value)
				if err != nil {
					panic(err)
				}
				cmds = append(cmds, cmd)
			}
		}
		// else increase limit and offset and continue
		offset += 50
	}
	return
}
