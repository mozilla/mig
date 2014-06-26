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
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	migdb "mig/database"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"strings"
	"syscall"

	"code.google.com/p/gcfg"
	"github.com/jvehent/cljs"
)

type Context struct {
	API struct {
		URL string
	}
	HTTP struct {
		Client http.Client
	}
}

var ctx Context

func main() {
	fmt.Println("\x1b[32;1m" + `
## ##                                     _.---._     .---.
# # # /-\ ---||  |    /\         __...---' .---. '---'-.   '.
#   #|   | / ||  |   /--\    .-''__.--' _.'( | )'.  '.  '._ :
#   # \_/ ---| \_ \_/    \ .'__-'_ .--'' ._'---'_.-.  '.   '-'.
     ###                         ~ -._ -._''---. -.    '-._   '.
      # |\ |\    /---------|          ~ -.._ _ _ _ ..-_ '.  '-._''--.._
      # | \| \  / |- |__ | |                       -~ -._  '-.  -. '-._''--.._.--''.
     ###|  \  \/  ---__| | |                            ~ ~-.__     -._  '-.__   '. '.
          #####                                               ~~ ~---...__ _    ._ .' '.
          #      /\  --- /-\ |--|                                        ~  ~--.....--~
          # ### /--\  | |   ||-\
          #####/    \ |  \_/ |  \
` + "\x1b[0m")

	homedir := findHomedir()
	// command line options
	var config = flag.String("c", homedir+"/.migconsole", "Load configuration from file")
	flag.Parse()

	err := gcfg.ReadFileInto(&ctx, *config)
	if err != nil {
		panic(err)
	}
	// create a channel that receives signals, and capture
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for signal := range c {
			switch signal {
			case syscall.SIGINT:
				// signal is a ^C, handle it
				fmt.Printf(`
            .-._   _ _ _ _ _ _ _ _
 .-''-.__.-'00  '-' ' ' ' ' ' ' ' '-.
'.___ '    .   .--_'-' '-' '-' _'-' '._
 V: V 'vv-'   '_   '.       .'  _..' '.'.
   '=.____.=_.--'   :_.__.__:_   '.   : :
           (((____.-'        '-.  /   : :
                             (((-'\ .' /
                           _____..'  .'
                          '-._____.-'
              Gators are going back underwater.
`)
				os.Exit(0)
			}
		}
	}()

	err = printStatus(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nConnected to %s. Use ctrl+c to exit.\n", ctx.API.URL)
	for {
		fmt.Printf("\x1b[32;1mmig>\x1b[0m ")
		r := bufio.NewReader(os.Stdin)
		// read command line input, split on newlines
		input, err := r.ReadString(0x0A)
		if err != nil {
			panic(err)
		}
		// trim carriage return
		input = input[0 : len(input)-1]
		order := strings.Split(input, " ")[0]
		switch order {
		case "status":
			err = printStatus(ctx)
			if err != nil {
				log.Println(err)
			}
		case "action":
			err = actionReader(input, ctx)
			if err != nil {
				log.Println(err)
			}
		default:
			fmt.Printf("Unknown order '%s'\n", order)
		}
	}
}

func findHomedir() string {
	if runtime.GOOS == "darwin" {
		return os.Getenv("HOME")
	} else {
		// find keyring in default location
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		return u.HomeDir
	}
}

func getAPIResource(t string, ctx Context) (resource *cljs.Resource, err error) {
	resp, err := ctx.HTTP.Client.Get(t)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	resource = cljs.New("")
	err = json.Unmarshal(body, &resource)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("HTTP %d: %v (code %s)", resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		return
	}
	return
}

func printStatus(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("printStatus() -> %v", e)
		}
	}()
	targetURL := ctx.API.URL + "dashboard"
	st, err := getAPIResource(targetURL, ctx)
	if err != nil {
		panic(err)
	}
	agtout := make([]string, 3)
	agtout[0] = "Agents Summary:"
	actout := make([]string, 1)
	actout[0] = "Latest Actions:"
	for _, item := range st.Collection.Items {
		for _, data := range item.Data {
			switch data.Name {
			case "action":
				a, err := valueToAction(data.Value)
				if err != nil {
					panic(err)
				}
				investigators := investigatorsStringFromAction(a.Investigators)
				if len(a.Name) > 30 {
					a.Name = a.Name[0:27] + "..."
				}
				str := fmt.Sprintf("* %s, %s launched '%s' with id %.0f on target '%s'",
					a.LastUpdateTime.Format("On Mon Jan 2 at 3:04pm (MST)"),
					investigators, a.Name, a.ID, a.Target)
				actout = append(actout, str)
			case "active agents":
				agtout[1] = fmt.Sprintf("* %.0f agents have checked in during the last 5 minutes", data.Value)
			case "agents versions count":
				bData, err := json.Marshal(data.Value)
				if err != nil {
					panic(err)
				}
				var sum []migdb.AgentsSum
				err = json.Unmarshal(bData, &sum)
				if err != nil {
					panic(err)
				}
				for _, asum := range sum {
					s := fmt.Sprintf("* version %s: %.0f agent", asum.Version, asum.Count)
					if asum.Count > 1 {
						s += "s"
					}
					agtout = append(agtout, s)
				}
			case "agents started in the last 24 hours":
				agtout[2] = fmt.Sprintf("* %.0f agents (re)started in the last 24 hours", data.Value)
			}
		}
	}
	fmt.Println("\x1b[31;1m+------\x1b[0m")
	for _, s := range agtout {
		fmt.Println("\x1b[31;1m|\x1b[0m " + s)
	}
	fmt.Println("\x1b[31;1m|\x1b[0m")
	for _, s := range actout {
		fmt.Println("\x1b[31;1m|\x1b[0m " + s)
		if len(actout) < 2 {
			fmt.Println("\x1b[31;1m|\x1b[0m * None")
			break
		}
	}
	fmt.Println("\x1b[31;1m+------\x1b[0m")
	return
}
