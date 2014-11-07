// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"mig/client"
)

func main() {
	var err error
	homedir := client.FindHomedir()
	// command line options
	var config = flag.String("c", homedir+"/.migconsole", "Load configuration from file")
	var aid = flag.Float64("aid", float64(1234567890), "Retrieve and print action by ID")
	flag.Parse()

	// instanciate an API client
	conf, err := client.ReadConfiguration(*config)
	if err != nil {
		panic(err)
	}
	cli := client.NewClient(conf)

	if *aid != float64(1234567890) {
		a, _, err := cli.GetAction(*aid)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("%.0f; %s; %s; %s\n", a.ID, a.Name, a.Target, a.Status)
		}
	}
}
