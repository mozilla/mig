// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
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

// loaderReader is used to manipulate loader entries
func loaderReader(input string, cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loaderReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) != 2 {
		panic("wrong order format. must be 'loader <loaderid>'")
	}
	lid, err := strconv.ParseFloat(inputArr[1], 64)
	if err != nil {
		panic(err)
	}
	le, err := cli.GetLoaderEntry(lid)
	if err != nil {
		panic(err)
	}

	fmt.Println("Entering loader reader mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Loader: '%v'.\nStatus '%v'.\n", le.Name, le.Enabled)

	prompt := fmt.Sprintf("\x1b[31;1mloader %v>\x1b[0m ", uint64(lid)%1000)
	for {
		reloadfunc := func() {
			le, err = cli.GetLoaderEntry(lid)
			if err != nil {
				panic(err)
			}
			fmt.Println("reloaded")
		}
		var symbols = []string{"disable", "enable", "exit", "help", "json", "r"}
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
			err = cli.LoaderEntryStatus(le, false)
			if err != nil {
				panic(err)
			}
			fmt.Println("Loader has been disabled")
			reloadfunc()
		case "enable":
			err = cli.LoaderEntryStatus(le, true)
			if err != nil {
				panic(err)
			}
			fmt.Println("Loader has been enabled")
			reloadfunc()
		case "help":
			fmt.Printf(`The following orders are avialable:
disable         disable loader entry

enable          enable loader entry

help            show this help

exit            exit this mode (also works with ctrl+d)

json            show json of loader entry stored in database

r               refresh the loader entry (get latest version from database)
`)
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "json":
			jsonle, err := json.MarshalIndent(le, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf("%v\n", string(jsonle))
		case "r":
			reloadfunc()
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'. You are in loader reader mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}

exit:
	fmt.Printf("\n")
	return
}

// Prompts for input and creates a new loader entry through the API
func loaderCreator(cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loaderCreator() -> %v", e)
		}
	}()
	var newle mig.LoaderEntry
	fmt.Println("Entering loader creation mode.\nPlease provide the name" +
		" of the new entry")
	newle.Name, err = readline.String("name> ")
	if err != nil {
		panic(err)
	}
	if len(newle.Name) < 3 {
		panic("input name too short")
	}
	fmt.Printf("Name: '%s'\n", newle.Name)
	fmt.Println("Please provide loader key for entry.")
	newle.Key, err = readline.String("key> ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Key: '%s'\n", newle.Key)
	// Validate the new loader entry before sending it to the API
	err = newle.Validate()
	if err != nil {
		panic(err)
	}
	jsonle, err := json.MarshalIndent(newle, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", jsonle)
	input, err := readline.String("create loader entry? (y/n)> ")
	if err != nil {
		panic(err)
	}
	if input != "y" {
		fmt.Println("abort")
		return
	}
	err = cli.PostNewLoader(newle)
	if err != nil {
		panic(err)
	}
	fmt.Println("New entry successfully created but is disabled")
	return
}
