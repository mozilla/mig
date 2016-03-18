// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com
package main

import (
	"bufio"
	"fmt"
	"os"
	"github.com/mozilla/scribe"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, "specify input test data file as argument\n")
		os.Exit(1)
	}
	fmt.Println("starting evr comparison tests")

	fd, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		buf := strings.TrimSpace(scanner.Text())
		if len(buf) == 0 {
			continue
		}
		fmt.Printf("%v\n", buf)

		var opmode int
		fields := strings.Fields(buf)
		if len(fields) != 3 {
			fmt.Fprintf(os.Stderr, "invalid test string '%s'\n", buf)
			os.Exit(1)
		}
		operator := fields[1]
		switch operator {
		case "=":
			opmode = scribe.EVROP_EQUALS
		case "<":
			opmode = scribe.EVROP_LESS_THAN
		case ">":
			opmode = scribe.EVROP_GREATER_THAN
		default:
			fmt.Fprintf(os.Stderr, "unknown operation %v\n", operator)
			os.Exit(1)
		}
		result, err := scribe.TestEvrCompare(opmode, fields[0], fields[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR %v\n", err)
			os.Exit(2)
		}
		if !result {
			fmt.Println("FAIL")
			os.Exit(2)
		}
		fmt.Println("PASS")
	}
	fd.Close()

	fmt.Println("end evr comparison tests")
}
