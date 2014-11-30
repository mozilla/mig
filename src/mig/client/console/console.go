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
	"github.com/bobappleyard/readline"
	"io"
	"log"
	"mig"
	"mig/client"
	migdb "mig/database"
	"os"
	"strconv"
	"strings"
)

var useShortNames bool

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
	flag.Parse()

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
	cli := client.NewClient(conf)
	// print platform status
	err = printStatus(cli)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(out, "\nConnected to %s. Exit with \x1b[32;1mctrl+d\x1b[0m. Type \x1b[32;1mhelp\x1b[0m for help.\n", cli.Conf.API.URL)
	for {
		// completion
		var symbols = []string{"action", "agent", "create", "command", "help", "history",
			"exit", "showcfg", "status", "investigator", "search", "where", "and"}
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
action <id|new>		enter interactive action mode. if <id> is given, go to reader mode. if "new" is given, enter launcher mode.
agent <id>		enter interactive agent reader mode for agent <id>
create action		enter interactive action creation mode
create investigator	create a new investigator, will prompt for name and public key
command <id>		enter command reader mode for command <id>
exit			leave
help			show this help
history <count>		print last <count> entries in history. count=10 by default.
investigator <id>	enter interactive investigator management mode for investigator <id>
search			perform a search. see "search help" for more information.
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
		case "search":
			err = search(input, cli)
			if err != nil {
				log.Println(err)
			}
		case "showcfg":
			fmt.Printf("homedir = %s\n[api]\n    url = %s\n[gpg]\n    home = %s\n    keyid = %s\n",
				cli.Conf.API.URL, cli.Conf.Homedir, cli.Conf.GPG.Home, cli.Conf.GPG.KeyID)
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
	agtout := make([]string, 5)
	agtout[0] = "Agents Summary:"
	actout := make([]string, 2)
	actout[0] = "Latest Actions:"
	actout[1] = "----    ID      ---- + ----         Name         ---- + -Sent- + ----    Date     ---- + ---- Investigators ----"
	for _, item := range st.Collection.Items {
		for _, data := range item.Data {
			switch data.Name {
			case "action":
				idstr, name, datestr, invs, sent, err := actionPrintShort(data.Value)
				if err != nil {
					panic(err)
				}
				str := fmt.Sprintf("%s   %s   %6d   %s   %s", idstr, name, sent, datestr, invs)
				actout = append(actout, str)
			case "active agents":
				agtout[1] = fmt.Sprintf("* %.0f active agents", data.Value)
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
			case "endpoints running 2 or more agents":
				agtout[3] = fmt.Sprintf("* %.0f endpoints are running 2 or more agents", data.Value)
			case "endpoints that have disappeared over last 7 days":
				agtout[4] = fmt.Sprintf("* %.0f endpoints have disappeared over the last 7 days", data.Value)
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
