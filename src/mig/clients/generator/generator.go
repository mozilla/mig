/* Mozilla InvestiGator Action Generator

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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mig"
	//"mig/modules/filechecker"
	"mig/pgp/sign"
	"net/url"
	"os"
	"os/user"
	"time"
)

func main() {

	var Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Mozilla InvestiGator Action Generator\n"+
				"usage: %s -k=<key id> (-i <input file)\n\n"+
				"Command line to generate and sign MIG Actions.\n"+
				"The resulting actions are display on stdout.\n\n"+
				"Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}

	// command line options
	var key = flag.String("k", "key identifier", "Key identifier used to sign the action (ex: B75C2346)")
	var pretty = flag.Bool("p", false, "Print signed action in pretty JSON format")
	var urlencode = flag.Bool("urlencode", false, "URL Encode marshalled JSON before output")
	var file = flag.String("i", "/path/to/file", "Load action from file")
	var validfrom = flag.String("validfrom", "now", "(optional) set an ISO8601 date the action will be valid from. If unset, use 'now'.")
	var expireafter = flag.String("expireafter", "30m", "(optional) set a validity duration for the action. If unset, use '30m'.")
	flag.Parse()

	// We need a key, if none is set on the command line, fail
	if *key == "key identifier" {
		Usage()
		os.Exit(-1)
	}

	var a mig.Action
	var err error

	// if a file is defined, load action from that
	if *file != "/path/to/file" {
		a, err = mig.ActionFromFile(*file)
	} else {
		// otherwise, use interactive mode
		a, err = getActionFromTerminal()
	}
	if err != nil {
		panic(err)
	}

	// set the dates
	if *validfrom == "now" {
		a.ValidFrom = time.Now().UTC()
	} else {
		a.ValidFrom, err = time.Parse("2014-01-01T00:00:00.0Z", *validfrom)
		if err != nil {
			panic(err)
		}
	}
	period, err := time.ParseDuration(*expireafter)
	if err != nil {
		log.Fatal(err)
	}
	a.ExpireAfter = a.ValidFrom.Add(period)

	// compute the signature
	str, err := a.String()
	if err != nil {
		panic(err)
	}
	a.PGPSignature, err = sign.Sign(str, *key)
	if err != nil {
		panic(err)
	}

	a.PGPSignatureDate = time.Now().UTC()

	var jsonAction []byte
	if *pretty {
		jsonAction, err = json.MarshalIndent(a, "", "\t")
	} else {
		jsonAction, err = json.Marshal(a)
	}
	if err != nil {
		panic(err)
	}

	// if asked, url encode the action before marshaling it
	if *urlencode {
		strJsonAction := string(jsonAction)
		urlEncodedAction := url.QueryEscape(strJsonAction)
		fmt.Printf("%s\n", urlEncodedAction)
	} else {
		fmt.Printf("%s\n", jsonAction)
	}

	// find keyring in default location
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	// load keyring
	var gnupghome string
	gnupghome = os.Getenv("GNUPGHOME")
	if gnupghome == "" {
		gnupghome = "/.gnupg"
	}
	keyring, err := os.Open(u.HomeDir + gnupghome + "/pubring.gpg")
	if err != nil {
		panic(err)
	}
	defer keyring.Close()

	// syntax checking
	err = a.Validate()
	if err != nil {
		panic(err)
	}

	// syntax checking
	err = a.VerifySignature(keyring)
	if err != nil {
		panic(err)
	}
}

func getActionFromTerminal() (a mig.Action, err error) {
	err = nil
	fmt.Print("Action name> ")
	_, err = fmt.Scanln(&a.Name)
	if err != nil {
		panic(err)
	}
	fmt.Print("Action Target> ")
	_, err = fmt.Scanln(&a.Target)
	if err != nil {
		panic(err)
	}

	//	var checkArgs string
	//	switch ea.Action.Order {
	//	default:
	//		fmt.Print("Unknown check type, supply JSON arguments> ")
	//		_, err := fmt.Scanln(&checkArgs)
	//		if err != nil {
	//			panic(err)
	//		}
	//		err = json.Unmarshal([]byte(checkArgs), ea.Action.Arguments)
	//		if err != nil {
	//			panic(err)
	//		}
	//	case "filechecker":
	//		fmt.Println("Filechecker module parameters")
	//		var name string
	//		var fcargs filechecker.FileCheck
	//		fmt.Print("Filechecker Name> ")
	//		_, err := fmt.Scanln(&name)
	//		if err != nil {
	//			panic(err)
	//		}
	//		fmt.Print("Filechecker Type> ")
	//		_, err = fmt.Scanln(&fcargs.Type)
	//		if err != nil {
	//			panic(err)
	//		}
	//		fmt.Print("Filechecker Path> ")
	//		_, err = fmt.Scanln(&fcargs.Path)
	//		if err != nil {
	//			panic(err)
	//		}
	//		fmt.Print("Filechecker Value> ")
	//		_, err = fmt.Scanln(&fcargs.Value)
	//		if err != nil {
	//			panic(err)
	//		}
	//		fc := make(map[string]filechecker.FileCheck)
	//		fc[name] = fcargs
	//		ea.Action.Arguments = fc
	//	}
	return
}
