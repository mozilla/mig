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
	targetURL := ctx.API.URL + "action?actionid=" + aid
	a, err := getAction(targetURL, ctx)
	if err != nil {
		panic(err)
	}
	investigators := investigatorsStringFromAction(a.Investigators)

	fmt.Println("Entering action reading mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Action: '%s'. Launched by '%s' on '%s'. Status '%s'.\n",
		a.Name, investigators, a.StartTime, a.Status)
	for {
		input, err := readline.String("\x1b[31;1maction>\x1b[0m ")
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("error: ", err)
			break
		}
		order := strings.Split(input, " ")[0]
		switch order {
		case "counters":
			fmt.Printf("Sent:\t\t%d\nReturned:\t%d\nDone:\t\t%d\n"+
				"Cancelled:\t%d\nFailed:\t\t%d\nTimeout:\t%d\n",
				a.Counters.Sent, a.Counters.Returned, a.Counters.Done,
				a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut)
		case "exit":
			goto exit
		case "meta":
			fmt.Printf("Action id %.0f named '%s'\nTarget '%s'\n"+
				"Description: Author '%s <%s>'; Revision '%.0f'; URL '%s'\n"+
				"Threat: Type '%s', Level '%s', Family '%s', Reference '%s'\n",
				a.ID, a.Name, a.Target, a.Description.Author, a.Description.Email,
				a.Description.Revision, a.Description.URL,
				a.Threat.Type, a.Threat.Level, a.Threat.Family, a.Threat.Ref)
			fmt.Printf("Operations: %d -> ", len(a.Operations))
			for _, op := range a.Operations {
				fmt.Printf("%s; ", op.Module)
			}
			fmt.Printf("\n")
		case "help":
			fmt.Printf(`The following orders are available:
counters	display the counters of the action
exit		exit this mode
help		show this help
investigators   print the list of investigators that signed the action
meta		display the action metadata
search		run a search based on this action. type 'search help' for details
raw		show raw action (serialized json)
re		refresh the action (reload latest version from API)
times		show the various timestamps of the action
`)
		case "investigators":
			for _, i := range a.Investigators {
				fmt.Println(i.Name, "- Key ID:", i.PGPFingerprint)
			}
		case "raw":
			astr, err := json.Marshal(a)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", astr)
		case "re":
			a, err = getAction(targetURL, ctx)
			if err != nil {
				panic(err)
			}
		case "search":
			err = runActionSearch(input, a, ctx)
			if err != nil {
				panic(err)
			}
		case "times":
			fmt.Printf("Valid from   '%s' until '%s'\nStarted on   '%s'\n"+
				"Last updated '%s'\nFinished on  '%s'\n",
				a.ValidFrom, a.ExpireAfter, a.StartTime, a.LastUpdateTime, a.FinishTime)
		default:
			fmt.Printf("Unknown order '%s'. You are in action reading mode. Try `help`.\n", order)
		}
		readline.AddHistory(input)
	}
exit:
	return
}

func getAction(targetURL string, ctx Context) (a mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getAction() -> %v", e)
		}
	}()
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

func investigatorsStringFromAction(invlist []mig.Investigator) (investigators string) {
	for ctr, i := range invlist {
		if ctr > 0 {
			investigators += "; "
		}
		investigators += i.Name
	}
	if len(investigators) > 30 {
		investigators = investigators[0:27] + "..."
	}
	return
}

func runActionSearch(input string, a mig.Action, ctx Context) (err error) {
	return
}
