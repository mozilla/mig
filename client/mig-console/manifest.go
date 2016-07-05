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

	"github.com/bobappleyard/readline"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
)

// manifestReader is used to manipulate API manifests
func manifestReader(input string, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("manifestReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'manifest <manifestid>'")
	}
	mid, err := strconv.ParseFloat(inputArr[1], 64)
	if err != nil {
		panic(err)
	}
	mr, err := cli.GetManifestRecord(mid)
	if err != nil {
		panic(err)
	}

	fmt.Println("Entering manifest reader mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Manifest: '%s'.\nStatus '%s'.\n", mr.Name, mr.Status)

	prompt := fmt.Sprintf("\x1b[31;1mmanifest %d>\x1b[0m ", uint64(mid)%1000)
	for {
		var symbols = []string{"disable", "entry", "exit", "help", "json", "r", "reset", "sign"}
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
		case "disable":
			err = cli.ManifestRecordStatus(mr, "disabled")
			if err != nil {
				panic(err)
			}
			fmt.Println("Manifest record has been disabled")
		case "entry":
			mre, err := mr.ManifestResponse()
			if err != nil {
				panic(err)
			}
			buf, err := json.MarshalIndent(mre, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", buf)
		case "help":
			fmt.Printf(`The following orders are avialable:
disable         disables manifest and prevents future use

entry           show the manifest for this record as would be sent to a loader

help            show this help

exit            exit this mode (also works with ctrl+d)

fetch           fetch manifest content and save to disk

json            show json of manifest record stored in database

loaders         show known loader entries that will match this manifest

r               refresh the manifest (get latest version from database)

reset           reset manifest status (marks manifest as staged, removes signatures)

sign            add a signature to the manifest record
`)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "fetch":
			if len(orders) != 2 {
				fmt.Println("error: must be 'fetch <path>'. try 'help'")
				break
			}
			err = mr.FileFromContent(orders[1])
			if err != nil {
				panic(err)
			}
		case "json":
			tmpmr := mr
			if len(tmpmr.Content) > 0 {
				tmpmr.Content = "..."
			} else {
				tmpmr.Content = "None"
			}
			jsonmr, err := json.MarshalIndent(tmpmr, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", jsonmr)
		case "loaders":
			ldrs, err := cli.GetManifestLoaders(mid)
			if err != nil {
				panic(err)
			}
			for _, x := range ldrs {
				buf, err := json.Marshal(x)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%v\n", string(buf))
			}
		case "r":
			mr, err = cli.GetManifestRecord(mid)
			if err != nil {
				panic(err)
			}
			fmt.Println("reloaded")
		case "reset":
			err = cli.ManifestRecordStatus(mr, "staged")
			if err != nil {
				panic(err)
			}
			fmt.Println("Manifest record has been reset")
		case "sign":
			sig, err := cli.SignManifest(mr)
			if err != nil {
				panic(err)
			}
			err = cli.PostManifestSignature(mr, sig)
			if err != nil {
				panic(err)
			}
			fmt.Println("Manifest signature has been accepted")
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'. You are in manifest reader mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}

exit:
	fmt.Printf("\n")
	return
}

// Prompts for input and creates a new manifest record through the API
func manifestCreator(cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("manifestCreator() -> %v", e)
		}
	}()
	var newmr mig.ManifestRecord
	fmt.Println("Entering manifest creation mode.\nPlease provide the name" +
		" of the new manifest")
	newmr.Name, err = readline.String("name> ")
	if err != nil {
		panic(err)
	}
	if len(newmr.Name) < 3 {
		panic("input name too short")
	}
	fmt.Printf("Name: '%s'\n", newmr.Name)
	fmt.Println("Please provide loader targeting string for manifest.")
	newmr.Target, err = readline.String("target> ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Target: '%s'\n", newmr.Target)
	fmt.Println("Please enter path to new manifest archive content.")
	arcpath, err := readline.String("contentpath> ")
	if err != nil {
		panic(err)
	}
	// Load the content into the manifest record from the specified path,
	// we assume the archive is a gzip compressed tar file; if not it will
	// fail during validation later on.
	err = newmr.ContentFromFile(arcpath)
	if err != nil {
		panic(err)
	}
	newmr.Status = "staged"
	// Validate the new manifest record before sending it to the API
	err = newmr.Validate()
	if err != nil {
		panic(err)
	}
	tmpmr := newmr
	if len(tmpmr.Content) > 0 {
		tmpmr.Content = "..."
	} else {
		tmpmr.Content = "None"
	}
	jsonmr, err := json.MarshalIndent(tmpmr, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", jsonmr)
	input, err := readline.String("create manifest? (y/n)> ")
	if err != nil {
		panic(err)
	}
	if input != "y" {
		fmt.Println("abort")
		return
	}
	err = cli.PostNewManifest(newmr)
	if err != nil {
		panic(err)
	}
	fmt.Println("Manifest successfully created")
	return
}
