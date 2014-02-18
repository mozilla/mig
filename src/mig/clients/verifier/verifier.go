/* Mozilla InvestiGator Action Verifier

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]
Guillaume Destuynder gdestuynder@mozilla.com

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

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
	err = a.VerifySignature(keyring)
	if err != nil {
		panic(err)
	}

	fmt.Println("Valid signature")

}
