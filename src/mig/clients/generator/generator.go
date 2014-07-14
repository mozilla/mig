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
	"io/ioutil"
	"log"
	"mig"
	"mig/pgp/sign"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"runtime"
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
	var posturl = flag.String("posturl", "", "POST action to <url> (enforces urlencode)")
	var file = flag.String("i", "/path/to/file", "Load action from file")
	var target = flag.String("t", "some.target.example.net", "Set the target of the action")
	var validfrom = flag.String("validfrom", "now", "(optional) set an ISO8601 date the action will be valid from. If unset, use 'now'.")
	var expireafter = flag.String("expireafter", "30m", "(optional) set a validity duration for the action. If unset, use '30m'.")
	flag.Parse()

	// We need a key, if none is set on the command line, fail
	if *key == "key identifier" {
		Usage()
		os.Exit(-1)
	}

	var err error

	// if a file is defined, load action from that
	if *file == "/path/to/file" {
		fmt.Println("Missing action file")
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

	if *target != "some.target.example.net" {
		a.Target = *target
	}

	// find homedir
	var homedir string
	if runtime.GOOS == "darwin" {
		homedir = os.Getenv("HOME")
	} else {
		// find keyring in default location
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		homedir = u.HomeDir
	}
	// load keyrings
	var gnupghome string
	gnupghome = os.Getenv("GNUPGHOME")
	if gnupghome == "" {
		gnupghome = "/.gnupg"
	}
	pubringFile, err := os.Open(homedir + gnupghome + "/pubring.gpg")

	if err != nil {
		panic(err)
	}
	defer pubringFile.Close()

	secringFile, err := os.Open(homedir + gnupghome + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer secringFile.Close()

	// compute the signature
	str, err := a.String()
	if err != nil {
		panic(err)
	}
	pgpsig, err := sign.Sign(str, *key, secringFile)
	if err != nil {
		panic(err)
	}

	// store the signature in the action signature array
	a.PGPSignatures = append(a.PGPSignatures, pgpsig)

	// syntax checking
	err = a.Validate()
	if err != nil {
		panic(err)
	}

	// signature checking
	err = a.VerifySignatures(pubringFile)
	if err != nil {
		panic(err)
	}

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

	// http post the action to the posturl endpoint
	if *posturl != "" {
		resp, err := http.PostForm(*posturl, url.Values{"action": {actionstr}})
		defer resp.Body.Close()
		if err != nil {
			panic(err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s", body)
	}
}
