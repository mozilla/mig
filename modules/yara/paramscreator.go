// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package yara /* import "github.com/mozilla/mig/modules/yara" */

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
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
%srules <path>     - yara rules path
		    ex: path ./myrules.yar
		    processes yara rules on agent

%sfiles <spec>     - scan files using rules
                    ex: files '-path /bin -path /sbin -name ssh'
		    indicate files that should be scanned, argument is
		    parameters as supplied to the file module for scanning,
		    each matching file will be scanned using rules. see the
		    help output for the file module for available options.
`, dash, dash)
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
		case "rules":
			rulebuf, err := ioutil.ReadFile(checkValue)
			if err != nil {
				fmt.Printf("%v\n", err)
				continue
			}
			p.YaraRules = string(rulebuf)
		case "files":
			p.FileSearch = checkValue
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
		fs         flag.FlagSet
		yaraPath   string
		fileSearch string
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("yara", flag.ContinueOnError)
	fs.StringVar(&yaraPath, "rules", "", "see help")
	fs.StringVar(&fileSearch, "files", "", "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()

	if yaraPath == "" {
		return nil, fmt.Errorf("-rules option is required")
	}
	rulebuf, err := ioutil.ReadFile(yaraPath)
	if err != nil {
		return nil, err
	}
	p.YaraRules = string(rulebuf)
	p.FileSearch = fileSearch
	// Right now file searching is the only supported search, so this
	// option must be specified.
	if p.FileSearch == "" {
		return nil, fmt.Errorf("-files option is required")
	}
	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}
