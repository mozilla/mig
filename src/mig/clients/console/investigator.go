// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bobappleyard/readline"
	"github.com/jvehent/cljs"
	"io"
	"io/ioutil"
	"mig"
	"mig/pgp"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
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
lastactions <limit>	print the last actions ran by the investigator. limit=10 by default.
pubkey			show the armored public key of the investigator
r			refresh the investigator (get latest version from upstream)
setstatus <status>	changes the status of the investigator to <status> (can be 'active' or 'disabled')
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
		case "setstatus":
			if len(orders) != 2 {
				fmt.Println("error: must be 'setstatus <status>'. try 'help'")
				break
			}
			newstatus := orders[1]
			err = updateInvestigatorStatus(iid, newstatus)
			if err != nil {
				panic(err)
			} else {
				fmt.Println("Investigator status set to", newstatus)
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

// investigatorCreator prompt the user for a name and the path to an armored
// public key and calls the API to create a new investigator
func investigatorCreator(ctx Context) (err error) {
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
		"and the public key of the new investigator.")
	name, err = readline.String("name> ")
	if err != nil {
		panic(err)
	}
	if len(name) < 3 {
		panic("input name too short")
	}
	fmt.Printf("Name: '%s'\n", name)
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
		if ctx.GPG.Keyserver == "" {
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
	postUrl := ctx.API.URL + "investigator/create/"
	// build the body into buf using a multipart writer
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	// set the name form value
	err = writer.WriteField("name", name)
	if err != nil {
		panic(err)
	}
	// set the publickey form value
	part, err := writer.CreateFormFile("publickey", fmt.Sprintf("%s.asc", name))
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(part, bytes.NewReader(pubkey))
	if err != nil {
		panic(err)
	}
	// must close the writer to write trailing boundary
	err = writer.Close()
	if err != nil {
		panic(err)
	}
	// post the request
	request, err := http.NewRequest("POST", postUrl, buf)
	if err != nil {
		panic(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := ctx.HTTP.Client.Do(request)
	if err != nil {
		panic(err)
	}
	// get the investigator back from the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 201 {
		err = fmt.Errorf("HTTP %d: %v (code %s)", resp.StatusCode,
			resource.Collection.Error.Message, resource.Collection.Error.Code)
		return
	}
	inv, err := valueToInvestigator(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Investigator '%s' successfully created with ID %.0f\n",
		inv.Name, inv.ID)
	return
}

func updateInvestigatorStatus(iid, newstatus string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("updateInvestigatorStatus() -> %v", e)
		}
	}()
	postUrl := ctx.API.URL + "investigator/update/"
	resp, err := ctx.HTTP.Client.PostForm(postUrl,
		url.Values{"id": {iid}, "status": {newstatus}})
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("error: HTTP %d. status update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}
