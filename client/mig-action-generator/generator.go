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
	"log"
	"net/url"
	"os"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/client"
)

func main() {
	var err error
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("FATAL: %v\n", e)
		}
	}()
	homedir := client.FindHomedir()
	var Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Mozilla InvestiGator Action Generator\n"+
				"usage: %s -i <input file>\n\n"+
				"Command line to generate and sign MIG Actions.\n"+
				"Configuration is read from ~/.migrc by default.\n\n"+
				"Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}

	// command line options
	var config = flag.String("c", homedir+"/.migrc", "Load configuration from file")
	var pretty = flag.Bool("p", false, "Print signed action in pretty JSON format")
	var urlencode = flag.Bool("urlencode", false, "URL Encode marshalled JSON before printing it (implies '-p')")
	var file = flag.String("i", "/path/to/file", "Load action from file")
	var target = flag.String("t", "some.target.example.net", "Set the target of the action")
	var validfrom = flag.String("validfrom", "now", "(optional) set an ISO8601 date the action will be valid from. If unset, use 'now'.")
	var expireafter = flag.String("expireafter", "30m", "(optional) set a validity duration for the action. If unset, use '30m'.")
	var nolaunch = flag.Bool("nolaunch", false, "Don't launch the action. Print it and exit. (implies '-p')")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	if *nolaunch {
		*pretty = true
	}

	// instanciate an API client
	conf, err := client.ReadConfiguration(*config)
	if err != nil {
		panic(err)
	}
	cli, err := client.NewClient(conf, "generator-"+mig.Version)
	if err != nil {
		panic(err)
	}

	// We need a file to load the action from
	if *file == "/path/to/file" {
		fmt.Println("ERROR: Missing action file")
		Usage()
		os.Exit(1)
	}
	a, err := mig.ActionFromFile(*file)
	if err != nil {
		panic(err)
	}

	// set the dates
	if *validfrom == "now" {
		// for immediate execution, set validity one minute in the past
		a.ValidFrom = time.Now().Add(-60 * time.Second).UTC()
	} else {
		a.ValidFrom, err = time.Parse(time.RFC3339, *validfrom)
		if err != nil {
			panic(err)
		}
	}
	period, err := time.ParseDuration(*expireafter)
	if err != nil {
		log.Fatal(err)
	}
	a.ExpireAfter = a.ValidFrom.Add(period)

	if *target != "some.target.example.net" {
		a.Target = *target
	}

	a, err = cli.CompressAction(a)
	if err != nil {
		panic(err)
	}
	asig, err := cli.SignAction(a)
	if err != nil {
		panic(err)
	}
	a = asig

	// if asked, pretty print the action
	var jsonAction []byte
	if *pretty {
		jsonAction, err = json.MarshalIndent(a, "", "\t")
		fmt.Printf("%s\n", jsonAction)
	} else {
		jsonAction, err = json.Marshal(a)
	}
	if err != nil {
		panic(err)
	}

	// if asked, url encode the action before marshaling it
	actionstr := string(jsonAction)
	if *urlencode {
		strJsonAction := string(jsonAction)
		actionstr = url.QueryEscape(strJsonAction)
		if *pretty {
			fmt.Println(actionstr)
		}
	}

	if !*nolaunch {
		a2, err := cli.PostAction(a)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Successfully launched action %.0f\n", a2.ID)
	}
}
