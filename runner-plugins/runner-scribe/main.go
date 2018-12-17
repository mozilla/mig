// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
// Contributor: Tristan Weir tweir@mozilla.com [:weir]

// runner-scribe is a mig-runner plugin that processes results coming from automated
// actions and forwards the results as vulnerability events to MozDef
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/mozilla/gozdef"
	"github.com/mozilla/mig"
	scribemod "github.com/mozilla/mig/modules/scribe"
	"gopkg.in/gcfg.v1"
)

const sourceName = "runner-scribe"

// config represents the configuration used by runner-scribe, and is read in on
// initialization
//
// URL is mandatory
type config struct {
	MozDef struct {
		URL      string // URL to post events to MozDef
		UseProxy bool   // A switch to enable/disable the use of a system-configured proxy
	}
	api 		ServiceApi
}

type ServiceApiAsset struct {
	Id 				string `json:"id"`
	AssetType 		string `json:"asset_type"`
	AssetIdentifier string `json:"asset_identifier"`
	Team 			string `json:"team"`
	Operator 		string `json:"operator"`
	Zone 			string `json:"zone"`
	Timestamp 		string `json:"timestamp_utc"`
	Description 	string `json:"description"`
	Score 			int `json:"score"`
}

type ServiceApi struct {
	URL				string
	AuthEndpoint 	string
	ClientID 		string
	ClientSecret	string
	Token 			string // ephemeral token we generate to connect to ServiceAPI
}

type Auth0Token struct {
	AccessToken	string `json:"access_token"`
	Scope			string `json:"scope"`
	ExpiresIn		int `json:"expires_in"`
	TokenType		string `json:"token_type"`
}

const configPath string = "/etc/mig/runner-scribe.conf"

var conf config

func main() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", e)
			os.Exit(1)
		}
	}()

	var (
		err     error
		results mig.RunnerResult
	)

	err = gcfg.ReadFileInto(&conf, configPath)
	if err != nil {
		panic(err)
	}

	// generate a realtime auth0 auth token
	conf.api.Token = GetAuthToken(conf.api)

	// load a searchable map of assets from ServiceAPI
	var serviceApiAssets = make(map[string]ServiceApiAsset)
	err = GetAssets(serviceApiAssets, conf.api)
	if err != nil {
		panic(err)
	}


	buf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, &results)
	if err != nil {
		panic(err)
	}
	var items []gozdef.VulnEvent
	for _, x := range results.Commands {
		// Process the incoming commands, under normal circumstances we will have one
		// returned command per host. However, this function can handle cases where
		// more than one command applies to a given host. If data for a host already
		// exists in items, makeVulnerability should attempt to append this data to
		// the host rather than add a new item.
		var err error
		items, err = makeVulnerability(items, x, serviceApiAssets)
		if err != nil {
			panic(err)
		}
	}
	for _, y := range items {
		y.SourceName = sourceName
		err = sendVulnerability(y)
		if err != nil {
			panic(err)
		}
	}
}

func sendVulnerability(item gozdef.VulnEvent) (err error) {
	ac := gozdef.APIConf{
		URL:      conf.MozDef.URL,
		UseProxy: conf.MozDef.UseProxy,
	}
	pub, err := gozdef.InitAPI(ac)
	if err != nil {
		return
	}
	err = pub.Send(item)
	return
}

func makeVulnerability(initems []gozdef.VulnEvent, cmd mig.Command, serviceApiAssets map[string]ServiceApiAsset) (items []gozdef.VulnEvent, err error) {
	var (
		itemptr                       *gozdef.VulnEvent
		assethostname, assetipaddress string
		insertNew                     bool
		assetoperator, assetteam      string
	)
	items = initems

	assethostname = cmd.Agent.Name
	for _, x := range cmd.Agent.Env.Addresses {
		if !strings.Contains(x, ".") {
			continue
		}
		ipt, _, err := net.ParseCIDR(x)
		if err != nil {
			continue
		}
		assetipaddress = ipt.String()
		break
	}

	// First, see if we can locate a preexisting item for this asset
	for i := range items {
		if items[i].Asset.Hostname == assethostname &&
			items[i].Asset.IPAddress == assetipaddress {
			itemptr = &items[i]
			break
		}
	}
	if itemptr == nil {
		// Initialize a new event we will insert later
		newevent, err := gozdef.NewVulnEvent()
		if err != nil {
			return items, err
		}
		newevent.Description = "MIG vulnerability identification"
		newevent.Zone = "mig"
		newevent.Asset.Hostname = assethostname
		newevent.Asset.IPAddress = assetipaddress
		newevent.Asset.OS = cmd.Agent.Env.OS
		
		assetoperator, assetteam = LookupOperatorTeam(assethostname, serviceApiAssets)
		newevent.Asset.Owner.Operator = assetoperator
		newevent.Asset.Owner.Team = assetteam
		
		// if we didn't find an operator from ServiceAPI assets
		// set it based on the tag
		if len(cmd.Agent.Tags) != 0 && newevent.Asset.Owner.Operator == "" {
			if _, ok := cmd.Agent.Tags["operator"]; ok {
				newevent.Asset.Owner.Operator = cmd.Agent.Tags["operator"]
			}
		}
		// Apply a v2bkey to the event
		if newevent.Asset.Owner.V2Bkey == "" {
			if newevent.Asset.Owner.Operator != "" {
				newevent.Asset.Owner.V2Bkey = newevent.Asset.Owner.Operator
			}
			if newevent.Asset.Owner.Team != "" {
				newevent.Asset.Owner.V2Bkey += "-" + newevent.Asset.Owner.Team
			}
		}
		// Always set credentialed checks here
		newevent.CredentialedChecks = true
		insertNew = true
		itemptr = &newevent
	}

	for _, result := range cmd.Results {
		var el scribemod.ScribeElements
		err = result.GetElements(&el)
		if err != nil {
			return items, err
		}
		for _, x := range el.Results {
			if !x.MasterResult {
				// Result was false (vulnerability did not match)
				continue
			}
			newve := gozdef.VulnVuln{}
			newve.Name = x.TestName
			for _, y := range x.Tags {
				if y.Key == "severity" {
					newve.Risk = y.Value
				} else if y.Key == "link" {
					newve.Link = y.Value
				}
			}
			// If no risk value is set on the vulnerability, we just treat this as
			// informational and ignore it. This will apply to things like the result
			// from platform dependency checks associated with real vulnerability checks.
			if newve.Risk == "" {
				continue
			}
			newve.Risk = normalizeRisk(newve.Risk)
			newve.LikelihoodIndicator = likelihoodFromRisk(newve.Risk)
			if newve.CVSS == "" {
				newve.CVSS = cvssFromRisk(newve.Risk)
			}
			// Use the identifier for each true subresult in the
			// test as a proof section
			for _, y := range x.Results {
				if y.Result {
					newve.Packages = append(newve.Packages, y.Identifier)
				}
			}
			itemptr.Vuln = append(itemptr.Vuln, newve)
		}
	}
	if insertNew {
		items = append(items, *itemptr)
	}
	return
}

// given config for an API behind Auth0 (including client ID and Secret), 
// return an Auth0 access token beginning with "Bearer "
// pattern from https://auth0.com/docs/api-auth/tutorials/client-credentials
func GetAuthToken(api ServiceApi) (authToken string) {
	payload := strings.NewReader("{\"grant_type\":\"client_credentials\",\"client_id\": \"" + api.ClientID + "\",\"client_secret\": \"" + api.ClientSecret + "\",\"audience\": \"" + api.URL + "\"}")
	req, _ := http.NewRequest("POST", api.AuthEndpoint, payload)
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	bodyJSON, _ := ioutil.ReadAll(res.Body)
	
	// unpack the JSON into an Auth0 token struct
	var body Auth0Token
	err = json.Unmarshal(bodyJSON, &body)
	if err != nil {
		panic(err)
	}

	// serviceAPI expects the Access token in the form of "Bearer <token>"
	authToken = "Bearer " + body.AccessToken
	return
}

// query a ServiceAPI instance for the set of all assets
// load them into a searchable map, keyed to asset hostname
// the ServiceAPI object must already be loaded with a Bearer token
func GetAssets(m map[string]ServiceApiAsset, api ServiceApi) (err error){
	
	// get json array of assets from serviceapi
	requestURL := api.URL + "api/v1/assets/"
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", api.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// unpack the HTTP request response
	defer res.Body.Close()
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return readErr
	}

	// because of the way that ServiceAPI returns the JSON content,
	// we need to Unmarshal it twice
	var allAssetsJson string
	err  = json.Unmarshal(body, &allAssetsJson)
	if err != nil {
		return err
	}

	// convert json into array of ServiceApiAsset objects
	var allAssets []ServiceApiAsset
	err  = json.Unmarshal([]byte(allAssetsJson), &allAssets)
	if err != nil {
		return err
	}

	// build a searchable map, keyed on AssetIdentifier (which is usually hostname)
	for _, tempAsset := range allAssets {
		permanentAsset := tempAsset				//not sure if this is needed
		m[tempAsset.AssetIdentifier] = permanentAsset
	}

	return
}

// return the operator and team for a given hostname, provided they are in the map of 
// ServiceApiAssets. If they are not in the map or if the values are not present, 
// operator and/or team will return as an empty string ""
func LookupOperatorTeam(hostname string, m map[string]ServiceApiAsset) (operator string, team string) {
	
	operator = m[hostname].Operator
	team = m[hostname].Team

	return
}

// cvssFromRisk returns a synthesized CVSS score as a string given a risk label
func cvssFromRisk(risk string) string {
	switch risk {
	case "critical":
		return "10.0"
	case "high":
		return "8.0"
	case "medium":
		return "5.0"
	case "low":
		return "2.5"
	}
	return "0.0"
}

// likelihoodFromRisk returns a likelihood indicator value given a risk label
func likelihoodFromRisk(risk string) string {
	switch risk {
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	case "critical":
		return "maximum"
	}
	return "unknown"
}

// normalizeRisk converts known risk labels into a standardized form, if we can't identify
// the value we just return it as is
func normalizeRisk(in string) string {
	switch strings.ToLower(in) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	case "critical":
		return "critical"
	}
	return in
}
