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

// commandReader retrieves an command from the API using its numerical ID
// and enters prompt mode to analyze it
func commandReader(input string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("commandReader() -> %v", e)
		}
	}()
	inputArr := strings.Split(input, " ")
	if len(inputArr) < 2 {
		panic("wrong order format. must be 'command <commandid>'")
	}
	cmdid := inputArr[1]
	cmd, err := getCommand(cmdid, ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("Entering command reader mode. Type \x1b[32;1mexit\x1b[0m or press \x1b[32;1mctrl+d\x1b[0m to leave. \x1b[32;1mhelp\x1b[0m may help.")
	fmt.Printf("Command %.0f ran on agent '%s' based on action '%s'\n",
		cmd.ID, cmd.Agent.Name, cmd.Action.Name)
	for {
		input, err := readline.String("\x1b[36;1mcommand>\x1b[0m ")
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("error: ", err)
			break
		}
		orders := strings.Split(input, " ")
		switch orders[0] {
		case "exit":
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
exit		exit this mode
help		show this help
json <pretty>	show the json of the command
r		refresh the command (get latest version from upstream)
`)
		case "json":
			var cjson []byte
			if len(orders) > 1 {
				if orders[1] == "pretty" {
					cjson, err = json.MarshalIndent(cmd, "", "  ")
				} else {
					fmt.Printf("Unknown option '%s'\n", orders[1])
				}
			} else {
				cjson, err = json.Marshal(cmd)
			}
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s\n", cjson)
		case "r":
			cmd, err = getCommand(cmdid, ctx)
			if err != nil {
				panic(err)
			}
			fmt.Println("Reload succeeded")
		default:
			fmt.Printf("Unknown order '%s'. You are in command reader mode. Try `help`.\n", orders[0])
		}
		readline.AddHistory(input)
	}
exit:
	fmt.Printf("\n")
	return
}

func getCommand(cmdid string, ctx Context) (cmd mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getCommand() -> %v", e)
		}
	}()
	targetURL := ctx.API.URL + "command?commandid=" + cmdid
	resource, err := getAPIResource(targetURL, ctx)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "command" {
		panic("API returned something that is not a command... something's wrong.")
	}
	cmd, err = valueToCommand(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

func valueToCommand(v interface{}) (cmd mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("valueToCommand() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &cmd)
	if err != nil {
		panic(err)
	}
	return
}
