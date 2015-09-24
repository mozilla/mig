// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"os"

	"mig.ninja/mig"
	"mig.ninja/mig/client"
)

func main() {
	var err error
	var Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Mozilla InvestiGator Action Verifier\n"+
				"usage: %s <-a action file> <-c command file>\n\n"+
				"Command line to verify an action *or* command.\n"+
				"Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}

	hasaction := false
	hascommand := false
	homedir := client.FindHomedir()

	// command line options
	var actionfile = flag.String("a", "/path/to/action", "Load action from file")
	var commandfile = flag.String("c", "/path/to/command", "Load command from file")
	var config = flag.String("conf", homedir+"/.migrc", "Load configuration from file")
	var showversion = flag.Bool("V", false, "Show build version and exit")
	flag.Parse()

	if *showversion {
		fmt.Println(mig.Version)
		os.Exit(0)
	}

	conf, err := client.ReadConfiguration(*config)
	if err != nil {
		panic(err)
	}

	// if a file is defined, load action from that
	if *actionfile != "/path/to/action" {
		hasaction = true
	}
	if *commandfile != "/path/to/command" {
		hascommand = true
	}
	if (hasaction && hascommand) || (!hasaction && !hascommand) {
		fmt.Println("[error] either an action file or a command file must be provided")
		Usage()
		os.Exit(1)
	}

	var a mig.Action
	if hasaction {
		a, err = mig.ActionFromFile(*actionfile)
		if err != nil {
			panic(err)
		}
	} else {
		c, err := mig.CmdFromFile(*commandfile)
		if err != nil {
			panic(err)
		}
		a = c.Action
	}

	err = a.Validate()
	if err != nil {
		fmt.Println(err)
	}

	pubringFile, err := os.Open(conf.GPG.Home + "/pubring.gpg")
	if err != nil {
		panic(err)
	}
	defer pubringFile.Close()

	// syntax checking
	err = a.VerifySignatures(pubringFile)
	if err != nil {
		fmt.Println("[error]", err)
	} else {
		fmt.Println("Valid signature")
	}

}
