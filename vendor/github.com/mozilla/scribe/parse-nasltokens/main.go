// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

// Parse the output of nasltokens to generate scribe checks
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"github.com/mozilla/scribe"
	"strings"
)

type checkEntry struct {
	dist    string
	pkgname string
	version string
	op      string
}

type releaseProfile struct {
	fdir       string
	fname      string
	expression string
}

var rProfileUbuntu = releaseProfile{
	fdir:       "/etc",
	fname:      "lsb-release",
	expression: "DISTRIB_RELEASE=(\\d{1,2}\\.\\d{1,2})",
}

var rProfileRedHat = releaseProfile{
	fdir:       "/etc",
	fname:      "redhat-release",
	expression: "release (\\d)\\.",
}

type releaseInformation struct {
	nasldist   string
	identifier string
	lsbmatch   string
	defid      string
	profile    *releaseProfile
}

var releaseList = []releaseInformation{
	{"UBUNTU14.10", "utopic", "14.10", "", &rProfileUbuntu},
	{"UBUNTU15.04", "vivid", "15.04", "", &rProfileUbuntu},
	{"UBUNTU14.04 LTS", "trusty", "14.04", "", &rProfileUbuntu},
	{"UBUNTU12.04 LTS", "precise", "12.04", "", &rProfileUbuntu},
	{"UBUNTU10.04 LTS", "lucid", "10.04", "", &rProfileUbuntu},
	{"RHENT_7", "rh7", "7", "", &rProfileRedHat},
	{"RHENT_6", "rh6", "6", "", &rProfileRedHat},
	{"RHENT_5", "rh5", "5", "", &rProfileRedHat},
}

func addReleaseDefinition(o *scribe.Document, rinfo *releaseInformation) {
	identifier := fmt.Sprintf("reldef-%v", rinfo.identifier)
	rinfo.defid = identifier

	obj := scribe.Object{}
	obj.Object = identifier + "-object"
	obj.FileContent.Path = rinfo.profile.fdir
	obj.FileContent.File = rinfo.profile.fname
	obj.FileContent.Expression = rinfo.profile.expression

	test := scribe.Test{}
	test.TestID = identifier + "-test"
	test.Object = obj.Object
	test.EMatch.Value = rinfo.lsbmatch

	o.Tests = append(o.Tests, test)
	o.Objects = append(o.Objects, obj)
}

func addReleaseDefinitions(o *scribe.Document) {
	for x := range releaseList {
		addReleaseDefinition(o, &releaseList[x])
	}
}

func getReleaseDefinition(nasldist string) string {
	for _, x := range releaseList {
		if nasldist == x.nasldist {
			return x.defid
		}
	}
	return ""
}

func addDefinition(o *scribe.Document, prefix string, check checkEntry) {
	// Don't create a definition for anything that is not in our release
	// list.
	reldefid := getReleaseDefinition(check.dist)
	if reldefid == "" {
		return
	}

	// Create an object definition for the package
	objid := fmt.Sprintf("%v-object", prefix)
	obj := scribe.Object{}
	obj.Object = objid
	obj.Package.Name = check.pkgname

	// Create a test
	testid := fmt.Sprintf("%v-test", prefix)
	test := scribe.Test{}
	test.TestID = testid
	test.Object = obj.Object
	test.EVR.Value = check.version
	test.EVR.Operation = check.op
	disttestref := fmt.Sprintf("%v-test", reldefid)
	test.If = append(test.If, disttestref)

	o.Tests = append(o.Tests, test)
	o.Objects = append(o.Objects, obj)
}

func processEntries(fpath string) {
	root := scribe.Document{}

	addReleaseDefinitions(&root)

	fd, err := os.Open(fpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	lv := 0
	for scanner.Scan() {
		lv++
		tokens := strings.Split(scanner.Text(), "|")
		ce := checkEntry{}
		ce.dist = strings.Trim(tokens[0], "\"")
		ce.pkgname = strings.Trim(tokens[1], "\"")
		ce.op = tokens[2]
		ce.version = strings.Trim(tokens[3], "\"")
		prefix := fmt.Sprintf("%v-%v", lv, ce.pkgname)
		addDefinition(&root, prefix, ce)
	}
	if err = scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	buf, err := json.MarshalIndent(&root, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%v\n", string(buf))
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "specify path to nasltokens output\n")
		os.Exit(1)
	}

	processEntries(args[0])
}
