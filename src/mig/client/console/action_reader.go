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
	"github.com/jvehent/cljs"
	"io"
	"mig"
	"mig/client"
	"strconv"
	"strings"
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
	a, links, err := cli.GetAction(aid)
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
		var symbols = []string{"command", "copy", "counters", "details", "exit", "foundanything", "foundnothing",
			"grep", "help", "investigators", "json", "ls", "found", "pretty", "r", "results", "times"}
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
			a, links, err = cli.GetAction(aid)
			if err != nil {
				panic(err)
			}
			a.PrintCounters()
		case "details":
			actionPrintDetails(a)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "foundanything":
			err = searchFoundAnything(a, true, cli)
			if err != nil {
				panic(err)
			}
		case "foundnothing":
			err = searchFoundAnything(a, false, cli)
			if err != nil {
				panic(err)
			}
		case "help":
			fmt.Printf(`The following orders are available:
command <id>	jump to command reader mode for command <id>
copy		enter action launcher mode using current action as template
counters	display the counters of the action
details		display the details of the action, including status & times
exit		exit this mode (also works with ctrl+d)
foundanything	list commands and agents that have found something
foundnothing	list commands and agents that have found nothing
help		show this help
investigators   print the list of investigators that signed the action
json         	show the json of the action
ls <filter>	returns the list of commands with their status
		'filter' is a pipe separated string of filter:
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
			var ajson []byte
			ajson, err = json.MarshalIndent(a, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", ajson)
		case "ls":
			err = actionPrintLinks(links, orders)
			if err != nil {
				panic(err)
			}
		case "r":
			a, links, err = cli.GetAction(aid)
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

func searchFoundAnything(a mig.Action, wantFound bool, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("searchFoundAnything() -> %v", e)
		}
	}()
	target := "search?type=command&limit=1000000&actionid=" + fmt.Sprintf("%.0f", a.ID)
	if wantFound {
		target += "&foundanything=true"
	} else {
		target += "&foundanything=false"
	}
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	agents := make(map[float64]mig.Command)
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			if data.Name != "command" {
				continue
			}
			cmd, err := client.ValueToCommand(data.Value)
			if err != nil {
				panic(err)
			}
			agents[cmd.Agent.ID] = cmd
		}
	}
	if wantFound {
		fmt.Printf("%d agents have found things\n", len(agents))
	} else {
		fmt.Printf("%d agents have not found anything\n", len(agents))
	}
	if len(agents) > 0 {
		fmt.Println("---- Command ID ----    ---- Agent Name & ID----")
		for agtid, cmd := range agents {
			fmt.Printf("%20.0f    %s [%.0f]\n", cmd.ID, cmd.Agent.Name, agtid)
		}
	}
	return
}

func actionPrintShort(data interface{}) (idstr, name, datestr, invs string, sent int, err error) {
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
	if len(idstr) < 20 {
		for i := len(idstr); i < 20; i++ {
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

	datestr = a.LastUpdateTime.Format("Mon Jan 2 3:04pm MST")
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

func actionPrintLinks(links []cljs.Link, orders []string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("actionPrintLinks() -> %v", e)
		}
	}()
	has_filter := false
	var filter []string
	if len(orders) > 1 {
		has_filter = true
		filter = orders[1:len(orders)]
	}
	ctr := 0
	for _, link := range links {
		if has_filter {
			str, err := filterString(link.Rel, filter)
			if err != nil {
				fmt.Printf("Invalid filter '%s': '%v'\n", filter, err)
				break
			}
			if str != "" {
				fmt.Println(str)
				ctr++
			}
		} else {
			fmt.Println(link.Rel)
			ctr++
		}
	}
	fmt.Printf("%d command", ctr)
	if ctr > 1 {
		fmt.Printf("s")
	}
	fmt.Printf(" found\n")
	return
}
