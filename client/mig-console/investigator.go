// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bobappleyard/readline"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
	"mig.ninja/mig/pgp"
)

// investigatorReader retrieves an agent from the api
// and enters prompt mode to analyze it
func investigatorReader(input string, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("investigatorReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'investigator <investigatorid>'")
	}
	iid, err := strconv.ParseFloat(inputArr[1], 64)
	if err != nil {
		panic(err)
	}
	inv, err := cli.GetInvestigator(iid)
	if err != nil {
		panic(err)
	}

	fmt.Println("Entering investigator mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Investigator %.0f named '%s'\n", inv.ID, inv.Name)
	prompt := fmt.Sprintf("\x1b[35;1minv %.0f>\x1b[0m ", iid)
	for {
		// completion, for convenience also add permission categories here
		var symbols = []string{"details", "exit", "help", "pubkey", "r", "lastactions",
			"setperms", "PermManifest", "PermLoader", "PermAdmin"}
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
		case "details":
			fmt.Printf("Investigator ID %.0f\n"+
				"name           %s\n"+
				"status         %s\n"+
				"permissions    %v\n"+
				"key id         %s\n"+
				"created        %s\n"+
				"modified       %s\n",
				inv.ID, inv.Name, inv.Status, inv.Permissions.ToDescriptive(),
				inv.PGPFingerprint, inv.CreatedAt, inv.LastModified)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
details			  print the details of the investigator
exit			  exit this mode
help			  show this help
lastactions <limit>	  print the last actions ran by the investigator. limit=10 by default.
pubkey			  show the armored public key of the investigator
r			  refresh the investigator (get latest version from upstream)
setperms [permissions...] set permissions for investigator, no arguments to apply default
showperms                 display possible permission values
setadmin <true|false>     enable or disable admin flag for investigator
setstatus <status>	  changes the status of the investigator to <status> (can be 'active' or 'disabled')
`)
		case "lastactions":
			limit := 10
			if len(orders) > 1 {
				limit, err = strconv.Atoi(orders[1])
				if err != nil {
					panic(err)
				}
			}
			err = printInvestigatorLastActions(iid, limit, cli)
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
			inv, err = cli.GetInvestigator(iid)
			if err != nil {
				panic(err)
			}
			fmt.Println("Reload succeeded")
		case "setstatus":
			if len(orders) != 2 {
				fmt.Println("error: must be 'setstatus <status>'. try 'help'")
				break
			}
			newstatus := orders[1]
			err = cli.PostInvestigatorStatus(iid, newstatus)
			if err != nil {
				panic(err)
			} else {
				fmt.Println("Investigator status set to", newstatus)
			}
			inv, err = cli.GetInvestigator(iid)
			if err != nil {
				panic(err)
			}
		case "setperms":
			newperms := mig.InvestigatorPerms{}
			// Always apply the default permission set for now
			newperms.DefaultSet()
			if len(orders) >= 2 {
				err = newperms.FromSetList(orders[1:])
				if err != nil {
					panic(err)
				}
			}
			err = cli.PostInvestigatorPerms(iid, newperms)
			if err != nil {
				panic(err)
			}
			inv, err = cli.GetInvestigator(iid)
			if err != nil {
				panic(err)
			}
		case "showperms":
			for _, x := range mig.PermSets {
				fmt.Println(x)
			}
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

func printInvestigatorLastActions(iid float64, limit int, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("printInvestigatorLastActions() -> %v", e)
		}
	}()
	target := fmt.Sprintf("search?type=action&investigatorid=%.0f&limit=%d", iid, limit)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	fmt.Printf("----- ID ----- + --------    Action Name ------- + ----------- Target   ---------- + ----    Date    ---- +  -- Status --\n")
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			if data.Name != "action" {
				continue
			}
			a, err := client.ValueToAction(data.Value)
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

// investigatorCreator prompt the user for a name and the path to an armored
// public key and calls the API to create a new investigator
func investigatorCreator(cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("investigatorCreator() -> %v", e)
		}
	}()
	var (
		name   string
		pubkey []byte
	)
	fmt.Println("Entering investigator creation mode. Please provide the full name\n" +
		"additional permissions if desired, and the public key of the new investigator.")
	name, err = readline.String("name> ")
	if err != nil {
		panic(err)
	}
	if len(name) < 3 {
		panic("input name too short")
	}
	fmt.Printf("Name: '%s'\n", name)
	pset := mig.InvestigatorPerms{}
	pset.DefaultSet()
	fmt.Println("With no additional permissions, the investigator will be permitted\n" +
		"access to run investigations. Answer yes to any of the following to add\n" +
		"additional permissions to the investigator.\n\nIf this is the first " +
		"investigator being added, you should make this\ninvestigator an admin.")
	respv, err := readline.String("Allow investigator to manage users (admin)? (yes/no)> ")
	if err != nil {
		panic(err)
	}
	switch strings.ToLower(respv) {
	case "yes":
		fmt.Println("Investigator will have administrative permissions")
		pset.AdminSet()
	case "no":
		fmt.Println("Investigator will not have administrative permissions")
	default:
		panic("must specify yes or no")
	}
	respv, err = readline.String("Allow investigator to manage loaders? (yes/no)> ")
	if err != nil {
		panic(err)
	}
	switch strings.ToLower(respv) {
	case "yes":
		fmt.Println("Investigator will have loader management permissions")
		pset.LoaderSet()
	case "no":
		fmt.Println("Investigator will not have loader management permissions")
	default:
		panic("must specify yes or no")
	}
	respv, err = readline.String("Allow investigator to manage manifests? (yes/no)> ")
	if err != nil {
		panic(err)
	}
	switch strings.ToLower(respv) {
	case "yes":
		fmt.Println("Investigator will have manifest management permissions")
		pset.ManifestSet()
	case "no":
		fmt.Println("Investigator will not have manifest management permissions")
	default:
		panic("must specify yes or no")
	}
	fmt.Println("Please provide a public key. You can either provide a local path to the\n" +
		"armored public key file, or a full length PGP fingerprint.\n" +
		"example:\npubkey> 0x716CFA6BA8EBB21E860AE231645090E64367737B")
	input, err := readline.String("pubkey> ")
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile(`^0x[ABCDEF0-9]{8,64}$`)
	if re.MatchString(input) {
		var keyserver string
		if cli.Conf.GPG.Keyserver == "" {
			keyserver = "http://gpg.mozilla.org"
		}
		fmt.Println("retrieving public key from", keyserver)
		pubkey, err = pgp.GetArmoredKeyFromKeyServer(input, keyserver)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("retrieving public key from", input)
		pubkey, err = ioutil.ReadFile(input)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("%s\n", pubkey)
	input, err = readline.String("create investigator? (y/n)> ")
	if err != nil {
		panic(err)
	}
	if input != "y" {
		fmt.Println("abort")
		return
	}
	inv, err := cli.PostInvestigator(name, pubkey, pset)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Investigator '%s' successfully created with ID %.0f\n",
		inv.Name, inv.ID)
	return
}
