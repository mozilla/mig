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
	"io/ioutil"
	"mig"
	"mig/pgp/sign"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bobappleyard/readline"
	"github.com/jvehent/cljs"
)

// default expiration is 60 seconds
var defaultExpiration = "60s"

// actionLauncher prepares an action for launch, either by starting with an empty
// template, or by loading an existing action from the api or the local disk
func actionLauncher(tpl mig.Action, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("actionLauncher() -> %v", e)
		}
	}()
	var a mig.Action
	if tpl.ID == 0 {
		fmt.Println("Entering action launcher with empty template")
	} else {
		// reinit the fields that we don't reuse
		a.Name = tpl.Name
		a.Target = tpl.Target
		a.Description = tpl.Description
		a.Threat = tpl.Threat
		a.Operations = tpl.Operations
		a.SyntaxVersion = tpl.SyntaxVersion
		fmt.Printf("Entering action launcher using template '%s'\n", a.Name)
	}
	hasTimes := false
	hasSignatures := false

	fmt.Println("Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	prompt := "\x1b[33;1mlauncher>\x1b[0m "
	for {
		// completion
		var symbols = []string{"exit", "help", "json", "launch", "load", "details",
			"settarget", "settimes", "sign", "times"}
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
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
exit			exit this mode
help			show this help
json <pretty>		show the json of the action
launch <nofollow>	launch the action. to return before completion, add "nofollow"
load <path>		load an action from a file at <path>
details			display the action details
settarget <target>	set the target
settimes <start> <stop>	set the validity and expiration dates
sign			PGP sign the action
times			show the various timestamps of the action
`)
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
		case "launch":
			follow := true
			if len(orders) > 1 {
				if orders[1] == "follow" {
					follow = false
				} else {
					fmt.Printf("Unknown option '%s'\n", orders[1])
				}
			}
			if !hasTimes {
				fmt.Printf("Times are not defined. Setting validity from now until +%s\n", defaultExpiration)
				// for immediate execution, set validity one minute in the past
				a.ValidFrom = time.Now().Add(-60 * time.Second).UTC()
				period, err := time.ParseDuration(defaultExpiration)
				if err != nil {
					panic(err)
				}
				a.ExpireAfter = a.ValidFrom.Add(period)
				a.ExpireAfter = a.ExpireAfter.Add(60 * time.Second).UTC()
				hasTimes = true
			}
			if !hasSignatures {
				pgpsig, err := computeSignature(a, ctx)
				if err != nil {
					panic(err)
				}
				a.PGPSignatures = append(a.PGPSignatures, pgpsig)
				hasSignatures = true
			}
			a, err = postAction(a, follow, ctx)
			if err != nil {
				panic(err)
			}
			fmt.Println("")
			_ = actionReader(fmt.Sprintf("action %.0f", a.ID), ctx)
			goto exit
		case "load":
			if len(orders) != 2 {
				fmt.Println("Wrong arguments. Expects 'load <path_to_file>'")
				break
			}
			a, err = mig.ActionFromFile(orders[1])
			if err != nil {
				panic(err)
			}
			fmt.Printf("Loaded action '%s' from %s\n", a.Name, orders[1])
		case "sign":
			if !hasTimes {
				fmt.Println("Times must be set prior to signing")
				break
			}
			pgpsig, err := computeSignature(a, ctx)
			if err != nil {
				panic(err)
			}
			a.PGPSignatures = append(a.PGPSignatures, pgpsig)
			hasSignatures = true
		case "settarget":
			if len(orders) != 2 {
				fmt.Println("Wrong arguments. Must be 'settarget <some_target_string_without_spaces>'")
				break
			}
			a.Target = orders[1]
		case "settimes":
			// set the dates
			if len(orders) != 3 {
				fmt.Println(`Invalid times. Expects settimes <start> <stop.)
examples:
settimes 2014-06-30T12:00:00.0Z 2014-06-30T14:00:00.0Z
settimes now +60m
`)
				break
			}
			if orders[1] == "now" {
				// for immediate execution, set validity one minute in the past
				a.ValidFrom = time.Now().Add(-60 * time.Second).UTC()
				period, err := time.ParseDuration(orders[2])
				if err != nil {
					fmt.Println("Failed to parse duration '%s': %v", orders[2], err)
					break
				}
				a.ExpireAfter = a.ValidFrom.Add(period)
				a.ExpireAfter = a.ExpireAfter.Add(60 * time.Second).UTC()
			} else {
				a.ValidFrom, err = time.Parse("2014-01-01T00:00:00.0Z", orders[1])
				if err != nil {
					fmt.Println("Failed to parse time '%s': %v", orders[1], err)
					break
				}
				a.ExpireAfter, err = time.Parse("2014-01-01T00:00:00.0Z", orders[2])
				if err != nil {
					fmt.Println("Failed to parse time '%s': %v", orders[2], err)
					break
				}
			}
			hasTimes = true
		case "times":
			fmt.Printf("Valid from   '%s' until '%s'\nStarted on   '%s'\n"+
				"Last updated '%s'\nFinished on  '%s'\n",
				a.ValidFrom, a.ExpireAfter, a.StartTime, a.LastUpdateTime, a.FinishTime)
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'. You are in action launcher mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}
exit:
	fmt.Printf("\n")
	return
}

func computeSignature(a mig.Action, ctx Context) (pgpsig string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("computeSignature() -> %v", e)
		}
	}()
	secringFile, err := os.Open(ctx.GPG.Home + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer secringFile.Close()

	// compute the signature
	str, err := a.String()
	if err != nil {
		panic(err)
	}
	pgpsig, err = sign.Sign(str, ctx.GPG.KeyID, secringFile)
	if err != nil {
		panic(err)
	}
	fmt.Println("Signature computed successfully")
	return
}

func validateAction(a mig.Action, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("validateAction() -> %v", e)
		}
	}()
	// syntax checking
	err = a.Validate()
	if err != nil {
		panic(err)
	}
	// signature checking
	pubringFile, err := os.Open(ctx.GPG.Home + "/pubring.gpg")
	if err != nil {
		panic(err)
	}
	err = a.VerifySignatures(pubringFile)
	if err != nil {
		panic(err)
	}
	pubringFile.Close()
	return
}

func postAction(a mig.Action, follow bool, ctx Context) (a2 mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("postAction() -> %v", e)
		}
	}()
	err = validateAction(a, ctx)
	if err != nil {
		panic(err)
	}
	// serialize
	ajson, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	actionstr := string(ajson)

	// http post the action to the posturl endpoint
	postUrl := ctx.API.URL + "action/create/"
	resp, err := ctx.HTTP.Client.PostForm(postUrl, url.Values{"action": {actionstr}})
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		panic(err)
	}
	a2, err = valueToAction(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Action '%s' successfully launched with ID '%.0f' on target '%s'\n",
		a2.Name, a2.ID, a2.Target)
	if follow {
		err = followAction(a2, ctx)
		if err != nil {
			panic(err)
		}
	}
	return
}

func followAction(a mig.Action, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("followAction() -> %v", e)
		}
	}()
	fmt.Printf("Entering follower mode for action ID %.0f\n", a.ID)
	sent := 0
	dotter := 0
	previousctr := 0
	status := ""
	for {
		a, _, err = getAction(fmt.Sprintf("%.0f", a.ID), ctx)
		if err != nil {
			panic(err)
		}
		if status == "" {
			status = a.Status
		}
		if status != a.Status {
			fmt.Printf("action status is now '%s'\n", a.Status)
			status = a.Status
		}
		if status != "init" && status != "preparing" && status != "inflight" {
			fmt.Printf("action finished with status '%s' in %s\n",
				status, a.LastUpdateTime.Sub(a.StartTime).String())
			break
		}
		// init counters
		if sent == 0 {
			if a.Counters.Sent == 0 {
				time.Sleep(1 * time.Second)
				continue
			} else {
				sent = a.Counters.Sent
				fmt.Printf("%d commands have been sent\n", sent)
			}
		}
		if a.Counters.Returned > 0 && a.Counters.Returned > previousctr {
			if a.Counters.Returned == a.Counters.Sent {
				fmt.Printf("100%% done, completed in %s\n", a.FinishTime.Sub(a.StartTime).String())
				break
			}
			completion := (float64(a.Counters.Returned) / float64(a.Counters.Sent)) * 100
			if completion > 99.9 && a.Counters.Returned != a.Counters.Sent {
				completion = 99.9
			}
			fmt.Printf("%.1f%% done - %d/%d\n",
				completion, a.Counters.Returned, a.Counters.Sent)
			previousctr = a.Counters.Returned
		}
		time.Sleep(500 * time.Millisecond)
		dotter++
		if dotter%10 == 0 {
			fmt.Printf("elapsed: %s\n", time.Now().Sub(a.StartTime).String())
		}
	}
	return
}
