// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package client

import (
	"code.google.com/p/gcfg"
	"encoding/json"
	"fmt"
	"github.com/jvehent/cljs"
	"io/ioutil"
	"mig"
	"mig/pgp"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"time"
)

// A Client provides all the needed functionalities to interact with the MIG API.
// It should be initialized with a proper configuration file.
type Client struct {
	API   *http.Client
	Token string
	Conf  Configuration
}

// Configuration stores the live configuration and global parameters of a client
type Configuration struct {
	API struct {
		URL string
	}
	Homedir string
	GPG     struct {
		Home      string
		KeyID     string
		Keyserver string
	}
}

// NewClient initiates a new instance of a Client
func NewClient(conf Configuration) Client {
	var cli Client
	cli.Conf = conf
	tr := &http.Transport{
		// TODO: add TLS support
		//TLSClientConfig:    &tls.Config{RootCAs: pool},
		DisableCompression: false,
		DisableKeepAlives:  false,
	}
	cli.API = &http.Client{Transport: tr}
	return cli
}

// ReadConfiguration loads a client configuration from a local configuration file
// and verifies that GnuPG's secring is available
func ReadConfiguration(file string) (conf Configuration, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ReadConfiguration() -> %v", e)
		}
	}()
	err = gcfg.ReadFileInto(&conf, file)
	if conf.GPG.Home == "" {
		gnupgdir := os.Getenv("GNUPGHOME")
		if gnupgdir == "" {
			gnupgdir = "/.gnupg"
		}
		conf.GPG.Home = FindHomedir() + gnupgdir
	}
	_, err = os.Stat(conf.GPG.Home + "/secring.gpg")
	if err != nil {
		panic("secring.gpg not found")
	}
	// if trailing slash is missing from API url, add it
	if conf.API.URL[len(conf.API.URL)-1] != '/' {
		conf.API.URL += "/"
	}
	return
}

func FindHomedir() string {
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

// GetAPIResource retrieves a cljs resource from a target endpoint. The target must be the relative
// to the API URL passed in the configuration. For example, if the API URL is `http://localhost:12345/api/v1/`
// then target could only be set to `dashboard` to retrieve `http://localhost:12345/api/v1/dashboard`
func (cli Client) GetAPIResource(target string) (resource *cljs.Resource, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetAPIResource() -> %v", e)
		}
	}()
	r, err := http.NewRequest("GET", cli.Conf.API.URL+target, nil)
	if err != nil {
		panic(err)
	}
	if cli.Token == "" {
		cli.Token, err = cli.MakeSignedToken()
		if err != nil {
			panic(err)
		}
	}
	r.Header.Set("X-PGPAUTHORIZATION", cli.Token)
	// execute the request
	resp, err := cli.API.Do(r)
	if err != nil {
		panic(err)
	}
	// if the request failed because of an auth issue, it may be that the auth token has expired.
	// try the request again with a fresh token
	if resp.StatusCode == 401 {
		cli.Token, err = cli.MakeSignedToken()
		if err != nil {
			panic(err)
		}
		r.Header.Set("X-PGPAUTHORIZATION", cli.Token)
		// execute the request
		resp, err = cli.API.Do(r)
		if err != nil {
			panic(err)
		}
	}
	// unmarshal the body. don't attempt to interpret it, as long as it
	// fits into a cljs.Resource, it's acceptable
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("error: HTTP %d. API call failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// GetAction retrieves a MIG Action from the API using its Action ID
func (cli Client) GetAction(aid float64) (a mig.Action, links []cljs.Link, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetAction() -> %v", e)
		}
	}()
	target := fmt.Sprintf("action?actionid=%.0f", aid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "action" {
		panic("API returned something that is not an action... something's wrong.")
	}
	a, err = ValueToAction(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	links = resource.Collection.Items[0].Links
	return
}

// PostAction submits a MIG Action to the API and returns the reflected action with API ID
func (cli Client) PostAction(a mig.Action) (a2 mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostAction() -> %v", e)
		}
	}()
	// serialize
	ajson, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	actionstr := string(ajson)

	// http post the action to the posturl endpoint
	postUrl := cli.Conf.API.URL + "/action/create/"
	resp, err := cli.API.PostForm(postUrl, url.Values{"action": {actionstr}})
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 202 {
		err = fmt.Errorf("error: HTTP %d. action creation failed.", resp.StatusCode)
		panic(err)
	}
	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		panic(err)
	}
	a2, err = ValueToAction(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

func ValueToAction(v interface{}) (a mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ValueToAction() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &a)
	if err != nil {
		panic(err)
	}
	return
}

func (cli Client) GetCommand(cmdid float64) (cmd mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetCommand() -> %v", e)
		}
	}()
	target := "command?commandid=" + fmt.Sprintf("%.0f", cmdid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "command" {
		panic("API returned something that is not a command... something's wrong.")
	}
	cmd, err = ValueToCommand(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

func ValueToCommand(v interface{}) (cmd mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ValueToCommand() -> %v", e)
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

func (cli Client) GetAgent(agtid float64) (agt mig.Agent, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetAgent() -> %v", e)
		}
	}()
	target := "agent?agentid=" + fmt.Sprintf("%.0f", agtid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "agent" {
		panic("API returned something that is not an agent... something's wrong.")
	}
	agt, err = ValueToAgent(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

func ValueToAgent(v interface{}) (agt mig.Agent, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("valueToAgent() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &agt)
	if err != nil {
		panic(err)
	}
	return
}

func (cli Client) GetInvestigator(iid float64) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetInvestigator() -> %v", e)
		}
	}()
	target := "investigator?investigatorid=" + fmt.Sprintf("%.0f", iid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "investigator" {
		panic("API returned something that is not an investigator... something's wrong.")
	}
	inv, err = ValueToInvestigator(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

func ValueToInvestigator(v interface{}) (inv mig.Investigator, err error) {
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

// MakeSignedToken encrypts a timestamp and a random number with the users GPG key
// to use as an auth token with the API
func (cli Client) MakeSignedToken() (token string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("MakeSignedToken() -> %v", e)
		}
	}()
	str := fmt.Sprintf("%s;%.0f", time.Now().UTC().Format(time.RFC3339), mig.GenID())
	secringFile, err := os.Open(cli.Conf.GPG.Home + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	sig, err := pgp.Sign(str+"\n", cli.Conf.GPG.KeyID, secringFile)
	if err != nil {
		panic(err)
	}
	token = str + ";" + sig
	return
}

// SignAction takes a MIG Action, signs it with the key identified in the configuration
// and returns the signed action
func (cli Client) SignAction(a mig.Action) (signed_action mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("SignAction() -> %v", e)
		}
	}()
	secring, err := os.Open(cli.Conf.GPG.Home + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer secring.Close()
	sig, err := a.Sign(cli.Conf.GPG.KeyID, secring)
	if err != nil {
		panic(err)
	}
	a.PGPSignatures = append(a.PGPSignatures, sig)
	signed_action = a
	return
}
