// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package sshkey /* import "mig.ninja/mig/modules/sshkey" */

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func printHelp(isCmd bool) {
	dash := " "
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`Query parameters
----------------
%vpath <string>  - specify a path to search for keys, more than one path option
                  can be specified on the command line. if the path option is not
                  present an OS specific default will be used (/root and /home for
                  Linux and Darwin, and c:\Users for Windows).

%vmaxdepth <int> - override default search depth of 8.
`, dash, dash)
}

// ParamsCreator is used by mig-console to create parameters for the module
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
		checkType := arr[0]
		checkValue := ""
		if len(arr) > 1 {
			checkValue = arr[1]
		}
		switch checkType {
		case "path":
			if checkValue == "" {
				fmt.Println("Missing parameter, try again")
				continue
			}
			p.Paths = append(p.Paths, checkValue)
		case "maxdepth":
			var err error
			if checkValue == "" {
				fmt.Println("Missing parameter, try again")
			}
			p.MaxDepth, err = strconv.Atoi(checkValue)
			if err != nil {
				fmt.Println("maxdepth argument must be an integer")
				continue
			}
		default:
			fmt.Printf("Invalid command, try help\n")
			continue
		}
	}

exit:
	r.Parameters = *p
	return r.Parameters, r.ValidateParameters()
}

// ParamsParser is used by the mig command line tool to parse parameters for the module
func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		paths    flagParam
		fs       flag.FlagSet
		maxdepth int
	)
	if len(args) > 0 && args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("sshkey", flag.ContinueOnError)
	fs.Var(&paths, "path", "see help")
	fs.IntVar(&maxdepth, "maxdepth", 0, "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	p.Paths = paths
	p.MaxDepth = maxdepth
	r.Parameters = *p
	return r.Parameters, r.ValidateParameters()
}

type flagParam []string

func (f *flagParam) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *flagParam) Set(value string) error {
	*f = append(*f, value)
	return nil
}
