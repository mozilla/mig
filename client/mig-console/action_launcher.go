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
	"mig.ninja/mig/modules"
)

// default expiration is 300 seconds
var defaultExpiration = "300s"

// actionLauncher prepares an action for launch, either by starting with an empty
// template, or by loading an existing action from the api or the local disk
func actionLauncher(tpl mig.Action, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("actionLauncher() -> %v", e)
		}
	}()
	var (
		a                mig.Action
		paramCompression bool
		tcount           int
	)
	if tpl.ID == 0 {
		fmt.Println("Entering action launcher with empty template")
	} else {
		// reinit the fields that we don't reuse
		a.Name = tpl.Name
		a.Target = tpl.Target
		a.Description = tpl.Description
		a.Threat = tpl.Threat
		a.Operations = tpl.Operations
		fmt.Printf("Entering action launcher using template '%s'\n", a.Name)
	}
	hasTimes := false
	hasSignatures := false
	hasEvaluatedTarget := false
	fmt.Println("Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	prompt := "\x1b[33;1mlauncher>\x1b[0m "
	for {
		// completion
		var symbols = []string{"addoperation", "compress", "deloperation", "exit", "help", "init",
			"json", "launch", "listagents", "load", "details", "filechecker", "netstat",
			"setname", "settarget", "settimes", "sign", "times"}
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
		case "addoperation":
			if len(orders) != 2 {
				fmt.Println("Wrong arguments. Expects 'addoperation <module_name>'")
				fmt.Println("example: addoperation filechecker")
				break
			}
			// attempt to call ParamsCreator from the requested module
			// ParamsCreator takes care of retrieving using input
			var operation mig.Operation
			operation.Module = orders[1]
			if _, ok := modules.Available[operation.Module]; ok {
				// instanciate and call module parameters creation function
				run := modules.Available[operation.Module].NewRun()
				if _, ok := run.(modules.HasParamsCreator); !ok {
					fmt.Println(operation.Module, "module does not provide a parameters creator.")
					fmt.Println("You can write your action by hand and import it using 'load <file>'")
					break
				}
				operation.Parameters, err = run.(modules.HasParamsCreator).ParamsCreator()
				if err != nil {
					fmt.Printf("Parameters creation failed with error: %v\n", err)
					break
				}
				if paramCompression {
					operation.WantCompressed = true
				}
				a.Operations = append(a.Operations, operation)
				opjson, err := json.MarshalIndent(operation, "", "  ")
				if err != nil {
					panic(err)
				}
				fmt.Printf("Inserting %s operation with parameters:\n%s\n", operation.Module, opjson)
			} else {
				fmt.Println("Module", operation.Module, "is not available in this console...")
				fmt.Println("You can write your action by hand and import it using 'load <file>'")
			}
		case "compress":
			if len(orders) != 2 {
				fmt.Println("Wrong arguments: Expects 'compress <true|false>'")
				fmt.Println("example: compress true")
				break
			}
			switch strings.ToLower(orders[1]) {
			case "false":
				paramCompression = false
				// Disable compression on all existing operations
				for i := range a.Operations {
					a.Operations[i].WantCompressed = false
					err = a.Operations[i].DecompressOperationParam()
					if err != nil {
						panic(err)
					}
				}
				// Invalidate any signatures applied to the action at this point
				hasSignatures = false
				a.PGPSignatures = nil
			case "true":
				paramCompression = true
				// Enable compression on all existing operations
				for i := range a.Operations {
					a.Operations[i].WantCompressed = true
				}
			default:
				fmt.Println("Argument to compress must be true or false")
			}
		case "deloperation":
			if len(orders) != 2 {
				fmt.Println("Wrong arguments. Expects 'deloperation <opnum>'")
				fmt.Println("example: deloperation 0")
				break
			}
			opnum, err := strconv.Atoi(orders[1])
			if err != nil || opnum < 0 || opnum > len(a.Operations)-1 {
				fmt.Println("error: <opnum> must be a positive integer between 0 and", len(a.Operations)-1)
				break
			}
			a.Operations = append(a.Operations[:opnum], a.Operations[opnum+1:]...)
		case "details":
			fmt.Printf("ID       %.0f\nName     %s\nTarget   %s\nAuthor   %s <%s>\n"+
				"Revision %.0f\nURL      %s\nThreat Type %s, Level %s, Family %s, Reference %s\n",
				a.ID, a.Name, a.Target, a.Description.Author, a.Description.Email,
				a.Description.Revision, a.Description.URL,
				a.Threat.Type, a.Threat.Level, a.Threat.Family, a.Threat.Ref)
			fmt.Printf("%d operations: ", len(a.Operations))
			for i, op := range a.Operations {
				fmt.Printf("%d=%s; ", i, op.Module)
			}
			fmt.Printf("\n")
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
addoperation <module>	append a new operation of type <module> to the action operations
compress <false|true>   request parameter compression in operations stored in action
listagents		list agents targetted by an action
deloperation <opnum>	remove operation numbered <opnum> from operations array, count starts at zero
details			display the action details
exit			exit this mode
help			show this help
json <pack>		show the json of the action
launch <nofollow>	launch the action. to return before completion, add "nofollow"
load <path>		load an action from a file at <path>
setname <name>		set the name of the action
settarget <target>	set the target
settimes <start> <stop>	set the validity and expiration dates
sign			PGP sign the action
times			show the various timestamps of the action
`)
		case "json":
			pack := false
			if len(orders) > 1 {
				if orders[1] == "pack" {
					pack = true
				} else {
					fmt.Printf("Unknown option '%s'\n", orders[1])
				}
			}
			tmpAction, err := getActionView(a)
			if err != nil {
				panic(err)
			}
			var ajson []byte
			if pack {
				ajson, err = json.Marshal(tmpAction)
			} else {
				ajson, err = json.MarshalIndent(tmpAction, "", "  ")
			}
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", ajson)
		case "launch":
			follow := true
			if len(orders) > 1 {
				if orders[1] == "nofollow" {
					follow = false
				} else {
					fmt.Printf("Unknown option '%s'\n", orders[1])
				}
			}
			if a.Name == "" {
				fmt.Println("Action has no name. Define one using 'setname <name>'")
				break
			}
			if a.Target == "" {
				fmt.Println("Action has no target. Define one using 'settarget <target>'")
				break
			}
			if !hasEvaluatedTarget {
				agents, err := cli.EvaluateAgentTarget(a.Target)
				if err != nil {
					panic(err)
				}
				tcount = len(agents)
				if tcount == 0 {
					fmt.Println("0 agents match this target. launch aborted")
					break
				}
				fmt.Printf("%d agents will be targeted by search \"%s\"\n", tcount, a.Target)
				input, err = readline.String("continue? (y/n)> ")
				if err != nil {
					panic(err)
				}
				if input != "y" {
					fmt.Println("launch aborted")
					break
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
				a, err = cli.CompressAction(a)
				if err != nil {
					panic(err)
				}
				asig, err := cli.SignAction(a)
				if err != nil {
					panic(err)
				}
				a = asig
				hasSignatures = true
			}
			a, err = cli.PostAction(a)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Action '%s' successfully launched with ID '%.0f' on target '%s'\n",
				a.Name, a.ID, a.Target)
			if follow {
				err = cli.FollowAction(a, tcount)
				if err != nil {
					panic(err)
				}
			}
			fmt.Println("")
			_ = actionReader(fmt.Sprintf("action %.0f", a.ID), cli)
			goto exit
		case "listagents":
			agents, err := cli.EvaluateAgentTarget(a.Target)
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("----    ID      ---- + ----         Name         -------")
			for _, agt := range agents {
				fmt.Printf("%20.0f   %s\n", agt.ID, agt.Name)
			}
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
			a, err = cli.CompressAction(a)
			if err != nil {
				panic(err)
			}
			asig, err := cli.SignAction(a)
			if err != nil {
				panic(err)
			}
			a = asig
			hasSignatures = true
		case "setname":
			if len(orders) < 2 {
				fmt.Println("Wrong arguments. Must be 'setname <some_name>'")
				break
			}
			a.Name = strings.Join(orders[1:], " ")
		case "settarget":
			if len(orders) < 2 {
				fmt.Println("Wrong arguments. Must be 'settarget <some_target_string>'")
				break
			}
			a.Target = strings.Join(orders[1:], " ")
			// Convert the target string to the desired value if the input was a
			// target macro
			a.Target = cli.ResolveTargetMacro(a.Target)
			agents, err := cli.EvaluateAgentTarget(a.Target)
			if err != nil {
				fmt.Println(err)
				break
			}
			tcount = len(agents)
			fmt.Printf("%d agents will be targetted. To get the list, use 'listagents'\n", tcount)
			hasEvaluatedTarget = true
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

func followAction(a mig.Action, cli client.Client) (err error) {
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
	attempts := 0
	var completion float64
	for {
		a, _, err = cli.GetAction(a.ID)
		if err != nil {
			attempts++
			time.Sleep(1 * time.Second)
			if attempts >= 30 {
				panic("failed to retrieve action after 30 seconds. launch may have failed")
			}
			continue
		}
		if status == "" {
			status = a.Status
		}
		if status != a.Status {
			fmt.Printf("action status is now '%s'\n", a.Status)
			status = a.Status
		}
		// exit follower mode if status isn't one we follow,
		// or enough commands have returned
		// or expiration time has passed
		if (status != "init" && status != "preparing" && status != "inflight") ||
			(a.Counters.Done > 0 && a.Counters.Done >= a.Counters.Sent) ||
			(time.Now().After(a.ExpireAfter)) {
			goto finish
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
		if a.Counters.Done > 0 && a.Counters.Done > previousctr {
			completion = (float64(a.Counters.Done) / float64(a.Counters.Sent)) * 100
			if completion > 99.9 && a.Counters.Done != a.Counters.Sent {
				completion = 99.9
			}
			previousctr = a.Counters.Done
		}
		time.Sleep(1 * time.Second)
		dotter++
		if dotter >= 5 {
			fmt.Printf("%.1f%% done - %d/%d - %s\n",
				completion, a.Counters.Done, a.Counters.Sent,
				time.Now().Sub(a.StartTime).String())
			dotter = 0
		}
	}
finish:
	fmt.Printf("leaving follower mode after %s\n", a.LastUpdateTime.Sub(a.StartTime).String())
	fmt.Printf("%d sent, %d done: %d returned, %d cancelled, %d expired, %d failed, %d timed out, %d still in flight\n",
		a.Counters.Sent, a.Counters.Done, a.Counters.Done, a.Counters.Cancelled, a.Counters.Expired,
		a.Counters.Failed, a.Counters.TimeOut, a.Counters.InFlight)
	return
}

// Return a view of an action suitable for JSON display in the console; this
// function essentially strips compression from the parameters, but leaves
// other fields intact -- the output should be used for display purposes only.
func getActionView(a mig.Action) (ret mig.Action, err error) {
	ret = a
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getActionView() -> %v", e)
		}
	}()
	// Create a copy of the original operations to modify, so we don't
	// change any of the original action parameters
	ret.Operations = make([]mig.Operation, len(a.Operations))
	copy(ret.Operations, a.Operations)
	for i := range ret.Operations {
		if !ret.Operations[i].IsCompressed {
			continue
		}
		err = ret.Operations[i].DecompressOperationParam()
		if err != nil {
			panic(err)
		}
		// Reset the IsCompressed flag, purely for visual purposes to
		// indicate the parameters are compressed when the JSON is
		// viewed
		ret.Operations[i].IsCompressed = true
	}
	return
}
