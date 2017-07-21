// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/bobappleyard/readline"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
)

func main() {
	var err error
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %v\n", e)
		}
	}()
	homedir := client.FindHomedir()
	// command line options
	var config = flag.String("c", homedir+"/.migrc", "Load configuration from file")
	var quiet = flag.Bool("q", false, "don't display banners and prompts")
	var showversion = flag.Bool("V", false, "show build version and exit")
	var verbose = flag.Bool("v", false, "verbose output, includes debug information and raw queries")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	// silence extra output
	out := os.Stdout
	if *quiet {
		out.Close()
		out, err = os.Open(os.DevNull)
		if err != nil {
			panic(err)
		}
	}
	defer out.Close()

	fmt.Fprintf(out, "\x1b[32;1m"+banner+"\x1b[0m")

	// append a space after completion
	readline.CompletionAppendChar = 0x20
	// load history
	historyfile := homedir + "/.mig_history"
	fi, err := os.Stat(historyfile)
	if err == nil && fi.Size() > 0 {
		err = readline.LoadHistory(historyfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load history from %s\n", historyfile)
		}
	}
	// instanciate an API client
	conf, err := client.ReadConfiguration(*config)
	if err != nil {
		panic(err)
	}
	conf, err = client.ReadEnvConfiguration(conf)
	if err != nil {
		panic(err)
	}
	cli, err := client.NewClient(conf, "console-"+mig.Version)
	if err != nil {
		panic(err)
	}
	if *verbose {
		cli.EnableDebug()
	}
	// print platform status
	err = printStatus(cli)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(out, "\nConnected to %s. Exit with \x1b[32;1mctrl+d\x1b[0m. Type \x1b[32;1mhelp\x1b[0m for help.\n", cli.GetConfiguration().API.URL)
	for {
		// completion
		var symbols = []string{"action", "agent", "create", "command", "help", "history",
			"exit", "manifest", "showcfg", "status", "investigator", "search", "query",
			"where", "and", "loader"}
		readline.Completer = func(query, ctx string) []string {
			var res []string
			for _, sym := range symbols {
				if strings.HasPrefix(sym, query) {
					res = append(res, sym)
				}
			}
			return res
		}

		input, err := readline.String("\x1b[32;1mmig>\x1b[0m ")
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("error: ", err)
			break
		}
		orders := strings.Split(strings.TrimSpace(input), " ")
		switch orders[0] {
		case "action":
			if len(orders) == 2 {
				err = actionReader(input, cli)
				if err != nil {
					log.Println(err)
				}
			} else {
				fmt.Println("error: missing action id in 'action <id>'")
			}
		case "agent":
			err = agentReader(input, cli)
			if err != nil {
				log.Println(err)
			}
		case "create":
			if len(orders) == 2 {
				switch orders[1] {
				case "action":
					var a mig.Action
					err = actionLauncher(a, cli)
				case "investigator":
					err = investigatorCreator(cli)
				case "loader":
					err = loaderCreator(cli)
				case "manifest":
					err = manifestCreator(cli)
				default:
					fmt.Printf("unknown order 'create %s'\n", orders[1])
				}
				if err != nil {
					log.Println(err)
				}
			} else {
				fmt.Println("error: missing order, must be 'create <action|investigator>'")
			}
		case "command":
			err = commandReader(input, cli)
			if err != nil {
				log.Println(err)
			}
		case "exit":
			fmt.Printf("exit\n")
			goto exit
		case "help":
			fmt.Printf(`The following orders are available:
action <id>		enter interactive action reader mode for action <id>
agent <id>		enter interactive agent reader mode for agent <id>
create action		create a new action
create investigator	create a new investigator, will prompt for name and public key
create loader           create a new loader entry
create manifest         create a new manifest
command <id>		enter command reader mode for command <id>
exit			leave
help			show this help
history <count>		print last <count> entries in history. count=10 by default.
investigator <id>	enter interactive investigator management mode for investigator <id>
manifest <id>           enter manifest management mode for manifest <id>
query <uri>		send a raw query string, without the base url, to the api
search <search>		perform a search. see "search help" for more information.
showcfg			display running configuration
status			display platform status: connected agents, latest actions, ...
`)
		case "history":
			var count int64 = 10
			if len(orders) > 1 {
				count, err = strconv.ParseInt(orders[1], 10, 64)
				if err != nil {
					log.Println(err)
					break
				}
			}
			for i := readline.HistorySize(); i > 0 && count > 0; i, count = i-1, count-1 {
				fmt.Println(readline.GetHistory(i - 1))
			}
		case "investigator":
			err = investigatorReader(input, cli)
			if err != nil {
				log.Println(err)
			}
		case "loader":
			err = loaderReader(input, cli)
			if err != nil {
				log.Println(err)
			}
		case "manifest":
			err = manifestReader(input, cli)
			if err != nil {
				log.Println(err)
			}
		case "query":
			fmt.Println("querying", orders[1])
			r, err := http.NewRequest("GET", orders[1], nil)
			if err != nil {
				panic(err)
			}
			resp, err := cli.Do(r)
			if err != nil {
				panic(err)
			}
			if err != nil || resp.Body == nil {
				log.Println("query failed")
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					panic(err)
				}
				fmt.Printf("%s\n", body)
			}
		case "search":
			err = search(input, cli)
			if err != nil {
				log.Println(err)
			}
		case "showcfg":
			conf := cli.GetConfiguration()
			fmt.Printf("homedir = %s\n[api]\n    url = %s\n[gpg]\n    home = %s\n    keyid = %s\n",
				conf.API.URL, conf.Homedir, conf.GPG.Home, conf.GPG.KeyID)
		case "status":
			err = printStatus(cli)
			if err != nil {
				log.Println(err)
			}
		case "":
			break
		default:
			fmt.Printf("Unknown order '%s'\n", orders[0])
		}
		readline.AddHistory(input)
	}
exit:
	fmt.Fprintf(out, footer)
	err = readline.SaveHistory(historyfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to save history to %s\n", historyfile)
	}
}

func printStatus(cli client.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("printStatus() -> %v", e)
		}
	}()
	st, err := cli.GetAPIResource("dashboard")
	if err != nil {
		panic(err)
	}
	var onlineagt, idleagt []string
	actout := make([]string, 2)
	actout[0] = "Latest Actions:"
	actout[1] = "----  ID  ---- + ----         Name         ---- + -Sent- + ----    Date    ---- + ---- Investigators ----"
	var onlineagents, onlineendpoints, idleagents, idleendpoints, newendpoints, doubleagents, disappearedendpoints, flappingendpoints float64
	for _, item := range st.Collection.Items {
		for _, data := range item.Data {
			switch data.Name {
			case "action":
				idstr, name, _, datestr, invs, _, sent, err := actionPrintShort(data.Value)
				if err != nil {
					panic(err)
				}
				str := fmt.Sprintf("%s   %s   %6d   %s   %s", idstr, name, sent, datestr, invs)
				actout = append(actout, str)
			case "online agents":
				onlineagents = data.Value.(float64)
			case "online endpoints":
				onlineendpoints = data.Value.(float64)
			case "idle agents":
				idleagents = data.Value.(float64)
			case "idle endpoints":
				idleendpoints = data.Value.(float64)
			case "new endpoints":
				newendpoints = data.Value.(float64)
			case "endpoints running 2 or more agents":
				doubleagents = data.Value.(float64)
			case "disappeared endpoints":
				disappearedendpoints = data.Value.(float64)
			case "flapping endpoints":
				flappingendpoints = data.Value.(float64)
			case "online agents by version":
				bData, err := json.Marshal(data.Value)
				if err != nil {
					panic(err)
				}
				var sum []mig.AgentsVersionsSum
				err = json.Unmarshal(bData, &sum)
				if err != nil {
					panic(err)
				}
				for _, asum := range sum {
					s := fmt.Sprintf("* version %s: %.0f agent", asum.Version, asum.Count)
					if asum.Count > 1 {
						s += "s"
					}
					onlineagt = append(onlineagt, s)
				}
			case "idle agents by version":
				bData, err := json.Marshal(data.Value)
				if err != nil {
					panic(err)
				}
				var sum []mig.AgentsVersionsSum
				err = json.Unmarshal(bData, &sum)
				if err != nil {
					panic(err)
				}
				for _, asum := range sum {
					s := fmt.Sprintf("* version %s: %.0f agent", asum.Version, asum.Count)
					if asum.Count > 1 {
						s += "s"
					}
					idleagt = append(idleagt, s)
				}
			}
		}
	}
	fmt.Println("\x1b[31;1m+------\x1b[0m")
	fmt.Printf("\x1b[31;1m| Agents & Endpoints summary:\n"+
		"\x1b[31;1m|\x1b[0m * %.0f online agents on %.0f endpoints\n"+
		"\x1b[31;1m|\x1b[0m * %.0f idle agents on %.0f endpoints\n"+
		"\x1b[31;1m|\x1b[0m * %.0f endpoints are running 2 or more agents\n"+
		"\x1b[31;1m|\x1b[0m * %.0f endpoints appeared over the last 7 days\n"+
		"\x1b[31;1m|\x1b[0m * %.0f endpoints disappeared over the last 7 days\n"+
		"\x1b[31;1m|\x1b[0m * %.0f endpoints have been flapping\n",
		onlineagents, onlineendpoints, idleagents, idleendpoints, doubleagents, newendpoints,
		disappearedendpoints, flappingendpoints)
	fmt.Println("\x1b[31;1m| Online agents by version:\x1b[0m")
	for _, s := range onlineagt {
		fmt.Println("\x1b[31;1m|\x1b[0m " + s)
	}
	fmt.Println("\x1b[31;1m| Idle agents by version:\x1b[0m")
	for _, s := range idleagt {
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

var banner string = `
## ##                                     _.---._     .---.
# # # /-\ ---||  |    /\         __...---' .---. '---'-.   '.
#   #|   | / ||  |   /--\    .-''__.--' _.'( | )'.  '.  '._ :
#   # \_/ ---| \_ \_/    \ .'__-'_ .--'' ._'---'_.-.  '.   '-'.
     ###                         ~ -._ -._''---. -.    '-._   '.
      # |\ |\    /---------|          ~ -.._ _ _ _ ..-_ '.  '-._''--.._
      # | \| \  / |- |__ | |                       -~ -._  '-.  -. '-._''--.._.--''.
     ###|  \  \/  ---__| | |                            ~ ~-.__     -._  '-.__   '. '.
          #####                                               ~~ ~---...__ _    ._ .' '.
          #      /\  --- /-\ |--|----                                    ~  ~--.....--~
          # ### /--\  | |   ||-\  //
          #####/    \ |  \_/ |  \//__
`

var footer string = `
            .-._   _ _ _ _ _ _ _ _
 .-''-.__.-'Oo  '-' ' ' ' ' ' ' ' '-.
'.___ '    .   .--_'-' '-' '-' _'-' '._
 V: V 'vv-'   '_   '.       .'  _..' '.'.
   '=.____.=_.--'   :_.__.__:_   '.   : :
           (((____.-'        '-.  /   : :
                             (((-'\ .' /
                           _____..'  .'
                          '-._____.-'
              Gators are going back underwater.
`
