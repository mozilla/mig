// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkg

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
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

%soval <path>       - OVAL processor
		    ex: oval ./ovaldefs.xml
		    process oval definitions on agent

%sconcurrency <int> - Concurrent OVAL checks (default: 1)
                    ex: concurrency 5
		    Specify concurrent OVAL definitions to evaluate at once, this
		    can have a performance impact on the agent system so increase
		    this carefully.

%sincludefalse      - Include false evaluations
                    ex: includefalse
		    Also includes definitions in results that evaluated to false
`, dash, dash, dash, dash)
}

func loadOvalDefinitions(path string) (string, error) {
	var b bytes.Buffer
	fd, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	encoder := base64.NewEncoder(base64.StdEncoding, &b)
	gz := gzip.NewWriter(encoder)
	_, err = io.Copy(gz, fd)
	if err != nil {
		return "", err
	}
	gz.Close()
	encoder.Close()
	return b.String(), nil
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
		} else if input == "includefalse" {
			p.IncludeFalse = true
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
		case "oval":
			obuf, err := loadOvalDefinitions(checkValue)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
			p.OvalDef = obuf
		case "concurrency":
			mce, err := strconv.Atoi(checkValue)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
			p.MaxConcurrentEval = mce
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
		fs           flag.FlagSet
		ovalDefs     string
		pkgMatch     flagParam
		maxEval      int
		includeFalse bool
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("migoval", flag.ContinueOnError)
	fs.Var(&pkgMatch, "name", "see help")
	fs.StringVar(&ovalDefs, "oval", "", "see help")
	fs.IntVar(&maxEval, "concurrency", 1, "see help")
	fs.BoolVar(&includeFalse, "includefalse", false, "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	p.PkgMatch.Matches = pkgMatch
	p.MaxConcurrentEval = maxEval
	p.IncludeFalse = includeFalse

	if ovalDefs != "" {
		obuf, err := loadOvalDefinitions(ovalDefs)
		if err != nil {
			return nil, err
		}
		p.OvalDef = obuf
	}

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
