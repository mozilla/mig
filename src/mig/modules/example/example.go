// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// This is an example module. It doesn't do anything. It only serves as
// a template for writing modules.
// If you run it, it will return a JSON struct with the hostname and IPs
// of the current endpoint.
//
//$ ./bin/linux/amd64/mig-agent-latest -m example '{"gethostname": true, "getaddresses": true, "lookuphost": "www.google.com"}' | python -mjson.tool
//{
//    "elements": {
//        "addresses": [
//            "172.21.0.3/20",
//            "fe80::8e70:5aff:fec8:be50/64"
//        ],
//        "hostname": "fedbox2.subdomain.example.net",
//        "lookeduphost": "www.google.com=173.194.37.115, 173.194.37.113, 173.194.37.116, 173.194.37.114, 173.194.37.112, 2607:f8b0:4008:805::1010"
//    },
//    "foundanything": true,
//    "statistics": {
//        "stufffound": 9
//    },
//    "success": true
//}

package example

import (
	"encoding/json"
	"fmt"
	"mig"
	"net"
	"os"
	"regexp"
)

// init is called by the Go runtime at startup. We use this function to
// register the module in a global array of available modules, so the
// agent knows we exist
func init() {
	mig.RegisterModule("example", func() interface{} {
		return new(Runner)
	})
}

// Runner gives access to the exported functions and structs of the module
type Runner struct {
	Parameters params
	Results    results
}

// a simple parameters structure, the format is arbitrary
type params struct {
	GetHostname  bool   `json:"gethostname"`
	GetAddresses bool   `json:"getaddresses"`
	LookupHost   string `json:"lookuphost"`
}

// results is the structure that is returned back to the agent.
// the fields are arbitrary
type results struct {
	// Elements contains the information retrieved by the agent
	Elements data `json:"elements"`
	// when the module performs a search, it is useful to return FoundAnything=true if _something_ was found
	FoundAnything bool `json:"foundanything"`
	// Success=true would mean that the module ran without major errors
	Success bool `json:"success"`
	// a list of errors can be returned
	Errors []string `json:"errors,omitempty"`
	// it may be interesting to include stats on execution
	Statistics statistics `json:"statistics,omitempty"`
}

type data struct {
	Hostname     string   `json:"hostname,omitempty"`
	Addresses    []string `json:"addresses,omitempty"`
	LookedUpHost string   `json:"lookeduphost,omitempty"`
}

// some execution statistics
var stats statistics

type statistics struct {
	StuffFound int64 `json:"stufffound"`
}

// ValidateParameters *must* be implemented by a module. It provides a method
// to verify that the parameters passed to the module conform the expected format.
// It must return an error if the parameters do not validate.
func (r Runner) ValidateParameters() (err error) {
	fqdn := regexp.MustCompilePOSIX(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$`)
	if !fqdn.MatchString(r.Parameters.LookupHost) {
		return fmt.Errorf("ValidateParameters: LookupHost parameter is not a valid FQDN.")
	}
	return
}

// Run *must* be implemented by a module. Its the function that executes the module.
// It must return a string, that is typically a marshalled json struct that contains
// the results of the execution.
func (r Runner) Run(Args []byte) string {
	// arguments are passed as an array of bytes, the module has to unmarshal that
	// into the proper structure of parameters, then validate it.
	err := json.Unmarshal(Args, &r.Parameters)
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}
	err = r.ValidateParameters()
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}

	// ---
	// From here on, we would normally do something useful, like:

	stats.StuffFound = 0 // count for stuff

	// grab the hostname of the endpoint
	if r.Parameters.GetHostname {
		hostname, err := os.Hostname()
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
			return r.buildResults()
		}
		r.Results.Elements.Hostname = hostname
		stats.StuffFound++
	}

	// grab the local ip addresses
	if r.Parameters.GetAddresses {
		addresses, err := net.InterfaceAddrs()
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
			return r.buildResults()
		}
		for _, addr := range addresses {
			if addr.String() == "127.0.0.1/8" || addr.String() == "::1/128" {
				continue
			}
			r.Results.Elements.Addresses = append(r.Results.Elements.Addresses, addr.String())
			stats.StuffFound++
		}
	}

	// look up a host
	if r.Parameters.LookupHost != "" {
		r.Results.Elements.LookedUpHost = r.Parameters.LookupHost + "="
		addresses, err := net.LookupHost(r.Parameters.LookupHost)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
			return r.buildResults()
		}
		for ctr, addr := range addresses {
			if ctr > 0 {
				r.Results.Elements.LookedUpHost += ", "
			}
			r.Results.Elements.LookedUpHost += addr
			stats.StuffFound++
		}

	}

	// return the results as a string (a marshalled json struct)
	return r.buildResults()
}

// buildResults marshals the results
func (r Runner) buildResults() string {
	if len(r.Results.Errors) == 0 {
		r.Results.Success = true
	}
	r.Results.Statistics = stats
	if stats.StuffFound > 0 {
		r.Results.FoundAnything = true
	}
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}

// PrintResults() is an *optional* method that returns results in a human-readable format.
// if matchOnly is set, only results that have at least one match are returned.
// If matchOnly is not set, all results are returned, along with errors and statistics.
func (r Runner) PrintResults(rawResults []byte, matchOnly bool) (prints []string, err error) {
	var results results
	err = json.Unmarshal(rawResults, &results)
	if err != nil {
		panic(err)
	}
	if results.Elements.Hostname != "" {
		fmt.Println("hostname", results.Elements.Hostname)
	}
	for _, addr := range results.Elements.Addresses {
		fmt.Println("address", addr)
	}
	if results.Elements.LookedUpHost != "" {
		fmt.Println(results.Elements.LookedUpHost)
	}
	for _, e := range results.Errors {
		fmt.Println("error:", e)
	}
	fmt.Println("stat:", results.Statistics.StuffFound, "stuff found")
	return
}
