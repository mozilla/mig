// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Kishor Bhat kishorbhat@gmail.com [:kbhat]

package account /* import "mig.ninja/mig/modules/account" */

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
%scheckuser <username> - check whether given user exists on system
			ex: checkuser john

%scheckgroup <groupname>  - check whether given group exists on system
			ex: checkgroup docker
`, dash, dash)

	return
}

// ParamsCreator implements an interactive interface for the console
// It takes a string and returns a params structure in an interface
func (r *run) ParamsCreator() (interface{}, error) {
	var (
		err error
		p   params
	)
	scanner := bufio.NewScanner(os.Stdin)
	p.User = ""
	p.Group = ""
	for {
		fmt.Printf("account> ")
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
		option := arr[0]
		checkValue := arr[1]
		switch option {
		case "checkuser":
			if checkValue == "" {
				fmt.Println("Missing parameter, try again")
				continue
			}
			err = validateName(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.User = checkValue
		case "checkgroup":
			if checkValue == "" {
				fmt.Println("Missing parameter, try again")
				continue
			}
			err = validateName(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.Group = checkValue
		default:
			fmt.Printf("Invalid method!\nTry 'help'\n")
			continue
		}
	}

exit:
	r.Parameters = p
	return r.Parameters, r.ValidateParameters()
}

// ParamsParser implements a commandline parameter parser for the account module
func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		err         error
		p           params
		user, group string
		fs          flag.FlagSet
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("account", flag.ContinueOnError)
	fs.StringVar(&user, "user", "", "see help")
	fs.StringVar(&group, "group", "", "see help")
	err = fs.Parse(args)
	if err != nil {
		return nil, err
	}
	p.User = user
	p.Group = group
	r.Parameters = p
	return r.Parameters, r.ValidateParameters()
}
