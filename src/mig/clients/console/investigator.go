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
	"mig"
	"mig/pgp"
	"strconv"
	"strings"
	"time"
)

// investigatorReader retrieves an agent from the api
// and enters prompt mode to analyze it
func investigatorReader(input string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("investigatorReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'investigator <investigatorid>'")
	}
	iid := inputArr[1]
	inv, err := getInvestigator(iid, ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("Entering investigator mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Investigator %.0f named '%s'\n", inv.ID, inv.Name)
	prompt := "\x1b[35;1minv " + iid + ">\x1b[0m "
	for {
		// completion
		var symbols = []string{"details", "exit", "help", "pubkey", "r", "lastactions"}
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
			inv, err = getInvestigator(iid, ctx)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Investigator ID %.0f\nname     %s\nstatus   %s\nkey id   %s\ncreated  %s\nmodified %s\n",
				inv.ID, inv.Name, inv.Status, inv.PGPFingerprint, inv.CreatedAt, inv.LastModified)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
details			print the details of the investigator
exit			exit this mode
help			show this help
pubkey			show the armored public key of the investigator
r			refresh the investigator (get latest version from upstream)
lastactions <limit>	print the last actions ran by the investigator. limit=10 by default.
`)
		case "lastactions":
			limit := 10
			if len(orders) > 1 {
				limit, err = strconv.Atoi(orders[1])
				if err != nil {
					panic(err)
				}
			}
			err = printInvestigatorLastActions(iid, limit)
			if err != nil {
				panic(err)
			}
		case "pubkey":
			armoredPubKey, err := pgp.ArmorPubKey(inv.PublicKey)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", armoredPubKey)
		case "r":
			inv, err = getInvestigator(iid, ctx)
			if err != nil {
				panic(err)
			}
			fmt.Println("Reload succeeded")
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'. You are in investigator mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}
exit:
	fmt.Printf("\n")
	return
}

func getInvestigator(iid string, ctx Context) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getInvestigator() -> %v", e)
		}
	}()
	targetURL := ctx.API.URL + "investigator?investigatorid=" + iid
	resource, err := getAPIResource(targetURL, ctx)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "investigator" {
		panic("API returned something that is not an investigator... something's wrong.")
	}
	inv, err = valueToInvestigator(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

func valueToInvestigator(v interface{}) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("valueToInvestigator) -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &inv)
	if err != nil {
		panic(err)
	}
	return
}

func printInvestigatorLastActions(iid string, limit int) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("printInvestigatorLastActions() -> %v", e)
		}
	}()
	targetURL := fmt.Sprintf("%s/search?type=action&investigatorid=%s&limit=%d", ctx.API.URL, iid, limit)
	resource, err := getAPIResource(targetURL, ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("-------  ID  ------- + --------    Action Name ------- + ----------- Target   ---------- + ----    Date    ---- +  -- Status --\n")
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			if data.Name != "action" {
				continue
			}
			a, err := valueToAction(data.Value)
			if err != nil {
				panic(err)
			}
			name := a.Name
			if len(name) < 30 {
				for i := len(name); i < 30; i++ {
					name += " "
				}
			}
			if len(name) > 30 {
				name = name[0:27] + "..."
			}
			target := a.Target
			if len(target) < 30 {
				for i := len(target); i < 30; i++ {
					target += " "
				}
			}
			if len(target) > 30 {
				target = target[0:27] + "..."
			}
			fmt.Printf("%.0f     %s   %s   %s    %s\n", a.ID, name, target,
				a.StartTime.Format(time.RFC3339), a.Status)
		}
	}
	return
}
