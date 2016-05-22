// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package pkg /* import "mig.ninja/mig/modules/pkg" */

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func printHelp(isCmd bool) {
	dash := ""
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`Query parameters
----------------
%sname <regexp>     - OS package search
                    ex: name linux-image
		    query for installed OS packages matching expression

%sversion <regexp>  - Version string search
                    ex: version ^1\..*
		    optionally filter returned packages to include version

%snotver <bool>     - Invert version logic
                    ex: notver true
		    return matching packages that do not match version string
`, dash, dash, dash)
}

func checkBoolFlag(flag string) bool {
	if strings.ToLower(flag) == "true" {
		return true
	}
	return false
}

func (r *run) ParamsCreator() (interface{}, error) {
	p := newParameters()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("search> ")
		scanmore := scanner.Scan()
		if err := scanner.Err(); err != nil {
			fmt.Println("Invalid input. Try again")
			continue
		}
		if !scanmore {
			goto exit
		}
		input := scanner.Text()
		if input == "done" {
			goto exit
		} else if input == "help" {
			printHelp(false)
			continue
		}
		arr := strings.SplitN(input, " ", 2)
		if len(arr) != 2 {
			fmt.Printf("Invalid input format!\n")
			printHelp(false)
			continue
		}
		checkType := arr[0]
		checkValue := arr[1]
		switch checkType {
		case "name":
			p.PkgMatch.Matches = append(p.PkgMatch.Matches, checkValue)
		case "version":
			p.VerMatch = checkValue
		case "notver":
			p.NotVersion = checkBoolFlag(checkValue)
		default:
			fmt.Printf("Invalid method!\nTry 'help'\n")
			continue
		}
	}

exit:
	r.Parameters = *p
	return r.Parameters, r.ValidateParameters()
}

func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		fs       flag.FlagSet
		pkgMatch flagParam
		verMatch string
		notVer   bool
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("pkg", flag.ContinueOnError)
	fs.Var(&pkgMatch, "name", "see help")
	fs.StringVar(&verMatch, "version", "", "see help")
	fs.BoolVar(&notVer, "notver", false, "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	p.PkgMatch.Matches = pkgMatch
	if verMatch != "" {
		p.VerMatch = verMatch
	}
	p.NotVersion = notVer

	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}

type flagParam []string

func (f *flagParam) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (f *flagParam) String() string {
	return fmt.Sprint([]string(*f))
}
