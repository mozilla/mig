// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

// Assists with creating scribe policies that are specific to package
// vulnerability checks
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mozilla/scribe/vulnpolicy"
)

func errExit(s string, args ...interface{}) {
	buf := fmt.Sprintf(s, args...)
	fmt.Fprintf(os.Stderr, "error: %v\n", buf)
	os.Exit(1)
}

func documentFromPolicy(polpath string) {
	fd, err := os.Open(polpath)
	if err != nil {
		errExit("%v", err)
	}
	defer fd.Close()
	buf, err := ioutil.ReadAll(fd)
	if err != nil {
		errExit("%v", err)
	}
	doc, err := vulnpolicy.DocumentFromPolicy(buf)
	if err != nil {
		errExit("%v", err)
	}
	jbuf, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		errExit("%v", err)
	}
	fmt.Fprintf(os.Stdout, "%v\n", string(jbuf))
}

func main() {
	var (
		polpath string
	)

	flag.StringVar(&polpath, "p", "", "generate scribe document from policy file")
	flag.Parse()

	if polpath != "" {
		documentFromPolicy(polpath)
	}
}
