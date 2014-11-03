// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"flag"
	"fmt"
	"mig"
	"os"
	"os/user"
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

	// command line options
	var actionfile = flag.String("a", "/path/to/action", "Load action from file")
	var commandfile = flag.String("c", "/path/to/command", "Load command from file")
	var pubring = flag.String("pubring", "/path/to/pubring", "Use pubring at <path>")
	flag.Parse()

	// if a file is defined, load action from that
	if *actionfile != "/path/to/action" {
		hasaction = true
	}
	if *commandfile != "/path/to/command" {
		hascommand = true
	}
	if (hasaction && hascommand) || (!hasaction && !hascommand) {
		Usage()
		panic(err)
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

	fmt.Printf("%s\n", a)
	err = a.Validate()
	if err != nil {
		fmt.Println(err)
	}

	// find keyring in default location
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	if *pubring != "/path/to/pubring" {
		// load keyring
		var gnupghome string
		gnupghome = os.Getenv("GNUPGHOME")
		if gnupghome == "" {
			gnupghome = "/.gnupg"
		}
		*pubring = u.HomeDir + gnupghome + "/pubring.gpg"
	}
	keyring, err := os.Open(*pubring)
	if err != nil {
		panic(err)
	}
	defer keyring.Close()

	// syntax checking
	err = a.VerifySignatures(keyring)
	if err != nil {
		panic(err)
	}

	fmt.Println("Valid signature")

}
