// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package client

import (
	"bufio"
	"bytes"
	"code.google.com/p/gcfg"
	"code.google.com/p/go.crypto/openpgp"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/jvehent/cljs"
	"io"
	"io/ioutil"
	"mig"
	"mig/pgp"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"strings"
	"time"
)

var version string

// A Client provides all the needed functionalities to interact with the MIG API.
// It should be initialized with a proper configuration file.
type Client struct {
	API     *http.Client
	Token   string
	Conf    Configuration
	Version string
}

// Configuration stores the live configuration and global parameters of a client
type Configuration struct {
	API     ApiConf // location of the MIG API
	Homedir string  // location of the user's home directory
	GPG     GpgConf // location of the user's secring
}

type ApiConf struct {
	URL            string
	SkipVerifyCert bool
}
type GpgConf struct {
	Home      string
	KeyID     string
	Keyserver string
}

// NewClient initiates a new instance of a Client
func NewClient(conf Configuration, version string) Client {
	var cli Client
	cli.Version = version
	cli.Conf = conf
	tr := &http.Transport{
		DisableCompression: false,
		DisableKeepAlives:  false,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS10,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			InsecureSkipVerify: conf.API.SkipVerifyCert,
		},
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
	_, err = os.Stat(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no configuration file found at %s\n", file)
		err = MakeConfiguration(file)
		if err != nil {
			panic(err)
		}
	}
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
	// try to make a signed token, just to check that we can access the private key
	var cli Client
	cli.Conf = conf
	_, err = cli.MakeSignedToken()
	if err != nil {
		err = fmt.Errorf("failed to generate a security token using key %s from %s\n",
			conf.GPG.KeyID, conf.GPG.Home+"/secring.gpg")
		return
	}
	return
}

func FindHomedir() string {
	if os.Getenv("HOME") != "" {
		return os.Getenv("HOME")
	}
	// find keyring in default location
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return u.HomeDir
}

// MakeConfiguration generates a new configuration file for the current user
func MakeConfiguration(file string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("MakeConfiguration() -> %v", e)
		}
	}()
	var cfg Configuration
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("would you like to generate a new configuration file at %s? Y/n> ", file)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	if scanner.Text() != "y" && scanner.Text() != "Y" && scanner.Text() != "" {
		panic("abort")
	}
	cfg.Homedir = FindHomedir()
	cfg.GPG.Home = cfg.Homedir + "/.gnupg/"
	_, err = os.Stat(cfg.GPG.Home + "secring.gpg")
	if err != nil {
		panic("couldn't find secring at " + cfg.GPG.Home + "secring.gpg")
	}
	sr, err := os.Open(cfg.GPG.Home + "secring.gpg")
	if err != nil {
		panic(err)
	}
	defer sr.Close()
	keyring, err := openpgp.ReadKeyRing(sr)
	if err != nil {
		panic(err)
	}
	for _, entity := range keyring {
		fingerprint := strings.ToUpper(hex.EncodeToString(entity.PrivateKey.PublicKey.Fingerprint[:]))
		// get the first name from the key identity
		var name string
		for _, identity := range entity.Identities {
			name = identity.Name
			break
		}
		fmt.Printf("found key '%s' with fingerprint '%s'.\nuse this key? Y/n> ", name, fingerprint)
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		if scanner.Text() == "y" || scanner.Text() == "Y" || scanner.Text() == "" {
			fmt.Printf("using key %s\n", fingerprint)
			cfg.GPG.KeyID = fingerprint
			break
		}
	}
	if cfg.GPG.KeyID == "" {
		panic("no suitable key found")
	}
	for {
		fmt.Printf("what is the location of the API? (ex: https://mig.example.net/api/v1/) > ")
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		cfg.API.URL = scanner.Text()
		_, err := http.Get(cfg.API.URL)
		if err != nil {
			fmt.Println("API connection failed. Wrong address?")
			continue
		}
		break
	}
	fd, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(fd, "[api]\n\turl = \"%s\"\n", cfg.API.URL)
	fmt.Fprintf(fd, "[gpg]\n\thome = \"%s\"\n\tkeyid = \"%s\"\n", cfg.GPG.Home, cfg.GPG.KeyID)
	fd.Close()
	fmt.Println("MIG client configuration generated at", file)
	return
}

// Do is a thin wrapper around http.Client.Do() that inserts an authentication header
// to the outgoing request
func (cli Client) Do(r *http.Request) (resp *http.Response, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Do() -> %v", e)
		}
	}()
	r.Header.Set("User-Agent", "MIG Client "+cli.Version)
	if cli.Token == "" {
		cli.Token, err = cli.MakeSignedToken()
		if err != nil {
			panic(err)
		}
	}
	r.Header.Set("X-PGPAUTHORIZATION", cli.Token)
	// execute the request
	resp, err = cli.API.Do(r)
	if err != nil {
		msg := fmt.Errorf("request failed error: %d %s (%v)", resp.StatusCode, resp.Status, err)
		panic(msg)
	}
	// if the request failed because of an auth issue, it may be that the auth token has expired.
	// try the request again with a fresh token
	if resp.StatusCode == 401 {
		resp.Body.Close()
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
	return
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
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	if resp.Body != nil {
		// unmarshal the body. don't attempt to interpret it, as long as it
		// fits into a cljs.Resource, it's acceptable
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		if len(body) > 1 {
			err = json.Unmarshal(body, &resource)
			if err != nil {
				panic(err)
			}
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
	a.SyntaxVersion = mig.ActionVersion
	// serialize
	ajson, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	actionstr := string(ajson)
	data := url.Values{"action": {actionstr}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"action/create/", strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
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

// PostInvestigator creates an Investigator and returns the reflected investigator
func (cli Client) PostInvestigator(name string, pubkey []byte) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostInvestigator() -> %v", e)
		}
	}()
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
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"investigator/create/", buf)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := cli.Do(r)
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
	inv, err = ValueToInvestigator(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// PostInvestigatorStatus updates the status of an Investigator
func (cli Client) PostInvestigatorStatus(iid float64, newstatus string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostInvestigatorStatus() -> %v", e)
		}
	}()
	data := url.Values{"id": {fmt.Sprintf("%.0f", iid)}, "status": {newstatus}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"investigator/update/", strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
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
	tokenVersion := 1
	str := fmt.Sprintf("%d;%s;%.0f", tokenVersion, time.Now().UTC().Format(time.RFC3339), mig.GenID())
	secringFile, err := os.Open(cli.Conf.GPG.Home + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer secringFile.Close()
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

// EvaluateAgentTarget runs a search against the api to find all agents that match an action target string
func (cli Client) EvaluateAgentTarget(target string) (agents []mig.Agent, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("EvaluateAgentTarget() -> %v", e)
		}
	}()
	query := "search?type=agent&target=" + url.QueryEscape(target)
	resource, err := cli.GetAPIResource(query)
	if err != nil {
		panic(err)
	}
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			if data.Name != "agent" {
				continue
			}
			agt, err := ValueToAgent(data.Value)
			if err != nil {
				panic(err)
			}
			agents = append(agents, agt)
		}
	}
	return
}

// FollowAction continuously loops over an action and prints its completion status in os.Stderr.
// when the action reaches its expiration date, FollowAction prints its final status and returns.
func (cli Client) FollowAction(a mig.Action) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("followAction() -> %v", e)
		}
	}()
	fmt.Fprintf(os.Stderr, "Following action ID %.0f. ", a.ID)
	sent := 0
	dotter := 0
	previousctr := 0
	status := ""
	attempts := 0
	var completion float64
	for {
		if completion > 97 {
			// if we got 97% completion, exit following mode.
			// there is always a couple % that are late and we don't want to block.
			goto finish
		}
		a, _, err = cli.GetAction(a.ID)
		if err != nil {
			attempts++
			time.Sleep(1 * time.Second)
			if attempts >= 30 {
				panic("failed to retrieve action after 30 seconds. launch may have failed")
			}
			continue
		}
		if status == "" {
			status = a.Status
		}
		if status != a.Status {
			fmt.Fprintf(os.Stderr, "\nstatus=%s", a.Status)
			status = a.Status
		}
		// exit follower mode if status isn't one we follow,
		// or enough commands have returned
		// or expiration time has passed
		if (status != "pending" && status != "scheduled" && status != "preparing" && status != "inflight") ||
			(a.Counters.Done > 0 && a.Counters.Done >= a.Counters.Sent) ||
			(time.Now().After(a.ExpireAfter)) {
			goto finish
			break
		}
		// init counters
		if sent == 0 {
			if a.Counters.Sent == 0 {
				time.Sleep(1 * time.Second)
				continue
			} else {
				sent = a.Counters.Sent
			}
		}
		if a.Counters.Done > 0 && a.Counters.Done > previousctr {
			completion = (float64(a.Counters.Done) / float64(a.Counters.Sent)) * 100
			if completion > 99 && a.Counters.Done != a.Counters.Sent {
				completion = 99.9
			}
			previousctr = a.Counters.Done
			fmt.Fprintf(os.Stderr, "%.0f%% ", completion)
		}
		fmt.Fprintf(os.Stderr, ".")
		time.Sleep(2 * time.Second)
		dotter++
	}
finish:
	a, _, err = cli.GetAction(a.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[error] failed to retrieve action counters\n")
	} else {
		completion = (float64(a.Counters.Done) / float64(a.Counters.Sent)) * 100
		fmt.Fprintf(os.Stderr, "- %2.1f%% done in %s\n", completion, time.Now().Sub(a.StartTime).String())
	}
	a.PrintCounters()
	return
}

func (cli Client) PrintActionResults(a mig.Action, show, render string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PrintActionResults() -> %v", e)
		}
	}()
	var (
		found  bool
		report string
		locs   []CommandLocation
	)
	switch render {
	case "map":
		report = "&report=geolocations"
	}
	var resource *cljs.Resource
	switch show {
	case "all":
		target := fmt.Sprintf("search?type=command&limit=1000000&actionid=%.0f%s", a.ID, report)
		resource, err = cli.GetAPIResource(target)
		if err != nil {
			panic(err)
		}
	case "found":
		found = true
		target := fmt.Sprintf("search?type=command&limit=1000000&foundanything=true&actionid=%.0f%s", a.ID, report)
		resource, err = cli.GetAPIResource(target)
		if err != nil {
			panic(err)
		}
	case "notfound":
		target := fmt.Sprintf("search?type=command&limit=1000000&foundanything=false&actionid=%.0f%s", a.ID, report)
		resource, err = cli.GetAPIResource(target)
		if err != nil {
			panic(err)
		}
	default:
		return fmt.Errorf("invalid parameter '%s'", show)
	}
	count := 0
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			switch render {
			case "map":
				if data.Name != "geolocation" {
					continue
				}
				loc, err := ValueToLocation(data.Value)
				if err != nil {
					panic(err)
				}
				locs = append(locs, loc)
			default:
				if data.Name != "command" {
					continue
				}
				cmd, err := ValueToCommand(data.Value)
				if err != nil {
					panic(err)
				}
				err = PrintCommandResults(cmd, found, true)
				if err != nil {
					panic(err)
				}
				count++
			}
		}
	}
	switch render {
	case "map":
		title := fmt.Sprintf("Geolocation of %s results for action ID %.0f %s", show, a.ID, a.Name)
		err = PrintMap(locs, title)
		if err != nil {
			panic(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "%d agents have %s results\n", count, show)
	}
	return
}

func PrintCommandResults(cmd mig.Command, onlyFound, showAgent bool) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PrintCommandResults() -> %v", e)
		}
	}()
	var prefix string
	if showAgent {
		prefix = cmd.Agent.Name + " "
	}
	if cmd.Status != mig.StatusSuccess {
		if !onlyFound {
			fmt.Fprintf(os.Stderr, "%scommand did not succeed. status=%s\n", prefix, cmd.Status)
		}
		return
	}
	for i, result := range cmd.Results {
		if !onlyFound {
			for _, rerr := range cmd.Results.Errors {
				fmt.Fprintf(os.Stderr, "%s[error] %s\n", prefix, rerr)
			}
		}
		if len(cmd.Action.Operations) <= i {
			if !onlyFound {
				fmt.Fprintf(os.Stderr, "%s[error] operation %d did not return results\n", prefix, i)
			}
			continue
		}
		// verify that we know the module
		moduleName := cmd.Action.Operations[i].Module
		if _, ok := modules.Available[moduleName]; !ok {
			if !onlyFound {
				fmt.Fprintf(os.Stderr, "%s[error] unknown module '%s'\n", prefix, moduleName)
			}
			continue
		}
		modRunner := mig.AvailableModules[moduleName].Runner()
		// look for a result printer in the module
		if _, ok := modRunner.(mig.HasResultsPrinter); ok {
			outRes, err := modRunner.(mig.HasResultsPrinter).PrintResults(result, onlyFound)
			if err != nil {
				panic(err)
			}
			for _, resLine := range outRes {
				fmt.Printf("%s%s\n", prefix, resLine)
			}
		} else {
			if !onlyFound {
				fmt.Fprintf(os.Stderr, "%s[error] no printer available for module '%s'\n", prefix, moduleName)
			}
		}
	}
	if !onlyFound {
		fmt.Printf("%scommand %s\n", prefix, cmd.Status)
	}
	return
}
