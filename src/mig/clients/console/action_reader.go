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
	"encoding/json"
	"fmt"
	"io"
	"mig"
	"strings"

	"github.com/bobappleyard/readline"
)

// actionReader retrieves an action from the API using its numerical ID
// and enters prompt mode to analyze it
func actionReader(input string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("actionReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'action <actionid>'")
	}
	aid := inputArr[1]
	a, err := getAction(aid, ctx)
	if err != nil {
		panic(err)
	}
	investigators := investigatorsStringFromAction(a.Investigators, 80)

	fmt.Println("Entering action reader mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Action: '%s'.\nLaunched by '%s' on '%s'.\nStatus '%s'.\n",
		a.Name, investigators, a.StartTime, a.Status)
	prompt := "\x1b[31;1maction " + aid[len(aid)-3:len(aid)] + ">\x1b[0m "
	for {
		// completion
		var symbols = []string{"command", "copy", "counters", "details", "exit", "foundsomething",
			"foundnothing", "help", "investigators", "json", "pretty", "r", "times"}
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
		case "command":
			err = commandReader(input, ctx)
			if err != nil {
				panic(err)
			}
		case "copy":
			err = actionLauncher(a, ctx)
			if err != nil {
				panic(err)
			}
			goto exit
		case "counters":
			fmt.Printf("Sent:\t\t%d\nReturned:\t%d\nDone:\t\t%d\n"+
				"Cancelled:\t%d\nFailed:\t\t%d\nTimeout:\t%d\n",
				a.Counters.Sent, a.Counters.Returned, a.Counters.Done,
				a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "foundsomething":
			err = searchFoundAnything(a, true, ctx)
			if err != nil {
				panic(err)
			}
		case "foundnothing":
			err = searchFoundAnything(a, false, ctx)
			if err != nil {
				panic(err)
			}
		case "help":
			fmt.Printf(`The following orders are available:
command <id>	jump to command reader mode for command <id>
copy		enter action launcher mode using current action as template
counters	display the counters of the action
exit		exit this mode
foundsomething	list commands and agents that have found something
foundnothing	list commands and agents that have found nothing
help		show this help
investigators   print the list of investigators that signed the action
json <pretty>	show the json of the action
details		display the details of the action, including status & times
r		refresh the action (get latest version from upstream)
times		show the various timestamps of the action
`)
		case "investigators":
			for _, i := range a.Investigators {
				fmt.Println(i.Name, "- Key ID:", i.PGPFingerprint)
			}
		case "json":
			var ajson []byte
			if len(orders) > 1 {
				if orders[1] == "pretty" {
					ajson, err = json.MarshalIndent(a, "", "  ")
				} else {
					fmt.Printf("Unknown option '%s'\n", orders[1])
				}
			} else {
				ajson, err = json.Marshal(a)
			}
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", ajson)
		case "details":
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
			fmt.Printf("Counters       sent=%d; returned=%d; done=%d\n"+
				"               cancelled=%d; failed=%d; timeout=%d\n",
				a.Counters.Sent, a.Counters.Returned, a.Counters.Done,
				a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut)
		case "r":
			a, err = getAction(aid, ctx)
			if err != nil {
				panic(err)
			}
			fmt.Println("Reload succeeded")
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

func getAction(aid string, ctx Context) (a mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getAction() -> %v", e)
		}
	}()
	targetURL := ctx.API.URL + "action?actionid=" + aid
	resource, err := getAPIResource(targetURL, ctx)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "action" {
		panic("API returned something that is not an action... something's wrong.")
	}
	a, err = valueToAction(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

func valueToAction(v interface{}) (a mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("valueToAction() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &a)
	if err != nil {
		panic(err)
	}
	return
}

func searchFoundAnything(a mig.Action, wantFound bool, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("foundAnything() -> %v", e)
		}
	}()
	foundanything := "false"
	if wantFound {
		foundanything = "true"
	}
	targetURL := ctx.API.URL + "search?type=command&limit=1000000&actionid=" + fmt.Sprintf("%.0f", a.ID) + "&foundanything=" + foundanything
	resource, err := getAPIResource(targetURL, ctx)
	if resource.Collection.Items[0].Data[0].Name != "search results" {
		fmt.Println(targetURL)
		panic("API returned something that is not search results.")
	}
	var results []mig.Command
	bData, err := json.Marshal(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &results)
	if err != nil {
		panic(err)
	}
	agents := make(map[float64]mig.Command)
	for _, cmd := range results {
		agents[cmd.Agent.ID] = cmd
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
