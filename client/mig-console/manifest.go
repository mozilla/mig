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
		var symbols = []string{"entry", "exit", "json", "r", "sign"}
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
		case "exit":
			fmt.Printf("exit\n")
			goto exit
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
		case "r":
			mr, err = cli.GetManifestRecord(mid)
			if err != nil {
				panic(err)
			}
			fmt.Println("reloaded")
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
