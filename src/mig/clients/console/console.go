package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mig"
	migdb "mig/database"
	"net/http"
	"os"
	"os/user"
	"runtime"

	"code.google.com/p/gcfg"
	"github.com/jvehent/cljs"
)

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

	printStatus()
}

var ctx Context

type Context struct {
	API struct {
		URL string
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

func printStatus() {
	resp, err := http.Get(ctx.API.URL + "dashboard")
	perror(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	perror(err)
	st := cljs.New("")
	err = json.Unmarshal(body, &st)
	perror(err)
	agtout := make([]string, 4)
	agtout[0] = "~~~ Agents Summary ~~~"
	actout := make([]string, 11)
	actout[0] = "~~~ Actions Summary ~~~"
	actctr := 0
	for _, item := range st.Collection.Items {
		for _, data := range item.Data {
			switch data.Name {
			case "action":
				bData, err := json.Marshal(data.Value)
				perror(err)
				var a mig.Action
				err = json.Unmarshal(bData, &a)
				perror(err)
				var investigators string
				for ctr, i := range a.Investigators {
					if ctr > 0 {
						investigators += "; "
					}
					investigators += i.Name
				}
				if len(investigators) > 30 {
					investigators = investigators[0:27] + "..."
				}
				if len(a.Name) > 30 {
					a.Name = a.Name[0:27] + "..."
				}
				actout[actctr+1] = fmt.Sprintf("%s, %s launched '%s' with id %.0f on target '%s'",
					a.LastUpdateTime.Format("On Mon Jan 2 at 3:04pm (MST)"),
					investigators, a.Name, a.ID, a.Target)
				actctr++
			case "active agents":
				agtout[1] = fmt.Sprintf("%.0f agents have checked in during the last 5 minutes", data.Value)
			case "agents versions count":
				bData, err := json.Marshal(data.Value)
				perror(err)
				var sum []migdb.AgentsSum
				err = json.Unmarshal(bData, &sum)
				perror(err)
				agtout[3] = "count by version:"
				for _, asum := range sum {
					s := fmt.Sprintf("\t%.0f agents run version %s", asum.Count, asum.Version)
					agtout = append(agtout, s)
				}
			case "agents started in the last 24 hours":
				agtout[2] = fmt.Sprintf("%.0f agents started in the last 24 hours", data.Value)
			}
		}
	}
	for _, s := range agtout {
		fmt.Println(s)
	}
	for _, s := range actout {
		fmt.Println(s)
	}
}

func perror(err error) {
	if err != nil {
		panic(err)
	}
}
