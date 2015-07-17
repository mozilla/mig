// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package scribe

import (
	"bufio"
	"flag"
	"fmt"
	scribelib "github.com/ameihm0912/scribe/src/scribe"
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
%spath <path>      - scribe processor
		    ex: scribe ./mytests.json
		    process scribe document on agent

%spkgmatch <regex> - package query
                    ex: package '^openssl'
		    scribe package query for matching string

%sonlytrue <bool>  - only true evaluations
                    ex: onlytrue true
		    just return document tests that evaluate to true
`, dash, dash, dash)
}

func loadScribeDocument(path string) (*scribelib.Document, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	dp, err := scribelib.LoadDocument(fd)
	if err != nil {
		return nil, err
	}
	return &dp, nil
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
		case "path":
			dp, err := loadScribeDocument(checkValue)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
			p.ScribeDoc = *dp
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
		fs        flag.FlagSet
		scribeDoc string
		onlyTrue  bool
		pkgMatch  string
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("scribe", flag.ContinueOnError)
	fs.StringVar(&scribeDoc, "path", "", "see help")
	fs.BoolVar(&onlyTrue, "onlytrue", false, "see help")
	fs.StringVar(&pkgMatch, "pkgmatch", "", "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()

	if scribeDoc != "" {
		dp, err := loadScribeDocument(scribeDoc)
		if err != nil {
			return nil, err
		}
		p.ScribeDoc = *dp
		p.RunMode = modeScribe
	} else if pkgMatch != "" {
		p.PkgMatch = pkgMatch
		p.RunMode = modePackage
	}

	p.OnlyTrue = onlyTrue

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
