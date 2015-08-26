// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package memory /* import "mig.ninja/mig/modules/memory" */

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func printHelp(isCmd bool) {
	dash := ""
	ma := "off"
	notma := "on"
	if isCmd {
		dash = "-"
		ma = "on"
		notma = "off"
	}
	fmt.Printf(`Search parameters
-----------------
%sname <regex>	- regex to match against the name of a process
		  ex: %sname ^postgresql$

%slib <regex>	- regex to match processes that are linked to a given library
		  ex: %slib libssl.so.1.0.0

%scontent <regex> - regex to match against the memory of a process.
		  note that regexes match utf-8 character sets, and some processes
		  may use utf-16 or some other encoding internally
		  ex: %scontent "http://mig\.mozilla\.org"

%sbytes <hex>	- match an hex byte string against the memory of a process
		  ex: %sbyte "6d69672e6d6f7a696c6c612e6f7267"
		             (mig.mozilla.org)
Options
-------
%smatchall	- all search parameters must match on a given process for it to
		  return as a match. %s by default.
		  ex: %smatchall
%smatchany	- any search parameter must match on a given process for it to
		  return as a match. %s by default.
		  ex: %smatchany
%slogfailures	- return any failure encountered during scanning. There is usually
		  a fair amount of them due to memory regions that cannot be read.
		  ex: %slogfailures
%soffset <int>	- provide a memory offset to start the scan at
%smaxlength <int> - indicates if a search should stop after reading <int> bytes
		    from a process
detailled doc at http://mig.mozilla.org/doc/module_memory.html
`, dash, dash, dash, dash, dash, dash, dash, dash, dash, ma,
		dash, dash, notma, dash, dash, dash, dash, dash)
	return
}

// ParamsCreator implements an interactive parameters creation interface, which
// receives user input,  stores it into a Parameters structure, validates it,
// and returns that structure as an interface. It is mainly used by the MIG Console
func (r *run) ParamsCreator() (interface{}, error) {
	var err error
	p := newParameters()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		var label string
		var search search
		for {
			fmt.Println("Give a name to this search, or 'done' to exit")
			fmt.Printf("label> ")
			scanner.Scan()
			if err := scanner.Err(); err != nil {
				fmt.Println("Invalid input. Try again")
				continue
			}
			label = scanner.Text()
			if label == "done" {
				// no label to add, exit
				goto exit
			}
			if label == "help" {
				fmt.Println(`A search must first have a name before search parameters can be defined.`)
				continue
			}
			err = validateLabel(label)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			if _, exist := p.Searches[label]; exist {
				fmt.Printf("A search labelled", label, "already exist. Override it?\n(y/n)> ")
				scanner.Scan()
				if err := scanner.Err(); err != nil {
					fmt.Println("Invalid input.")
					continue
				}
				response := scanner.Text()
				if response == "y" {
					fmt.Println("Overriding search", label)
					break
				}
				fmt.Println("Not overriding search", label)
				continue
			}
			break
		}
		fmt.Printf("Creating new search with label '%s'\n"+
			"Enter 'help' to list available parameters.\n", label)

		for {
			fmt.Printf("search '%s'> ", label)
			scanner.Scan()
			if err := scanner.Err(); err != nil {
				fmt.Println("Invalid input. Try again")
				continue
			}
			input := scanner.Text()
			if input == "done" {
				break
			}
			if input == "help" {
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
			case "name":
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Names = append(search.Names, checkValue)
			case "lib":
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Libraries = append(search.Libraries, checkValue)
			case "bytes":
				err = validateBytes(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Bytes = append(search.Bytes, checkValue)
			case "content":
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Contents = append(search.Contents, checkValue)
			case "matchall":
				search.Options.MatchAll = true
			case "matchany":
				search.Options.MatchAll = false
			case "offset":
				search.Options.Offset, err = strconv.ParseFloat(checkValue, 10)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
			case "maxlength":
				search.Options.MaxLength, err = strconv.ParseFloat(checkValue, 10)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
			case "logfailures":
				search.Options.LogFailures = true
			default:
				fmt.Printf("Invalid method!\n")
				continue
			}
			fmt.Printf("Stored %s %s\nEnter a new parameter, or 'done' to exit.\n", checkType, checkValue)
		}
		p.Searches[label] = search
		fmt.Println("Stored search", label)
	}
exit:
	r.Parameters = *p
	return r.Parameters, r.ValidateParameters()
}

// ParamsParser implements a command line parameters parser that takes a string
// and returns a Parameters structure in an interface. It will display the module
// help if the arguments string spell the work 'help'
func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		err                               error
		names, libraries, bytes, contents flagParam
		offset, maxlength                 float64
		matchall, matchany, logfailures   bool
		fs                                flag.FlagSet
	)
	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}
	fs.Init("memory", flag.ContinueOnError)
	fs.Var(&names, "name", "see help")
	fs.Var(&libraries, "lib", "see help")
	fs.Var(&bytes, "bytes", "see help")
	fs.Var(&contents, "content", "see help")
	fs.Float64Var(&offset, "maxdepth", 0, "see help")
	fs.Float64Var(&maxlength, "matchlimit", 0, "see help")
	fs.BoolVar(&matchall, "matchall", true, "see help")
	fs.BoolVar(&matchany, "matchany", false, "see help")
	fs.BoolVar(&logfailures, "logfailures", false, "see help")
	err = fs.Parse(args)
	if err != nil {
		return nil, err
	}
	var s search
	s.Names = names
	s.Libraries = libraries
	s.Bytes = bytes
	s.Contents = contents
	s.Options.Offset = offset
	s.Options.MaxLength = maxlength
	s.Options.MatchAll = matchall
	if matchany {
		s.Options.MatchAll = false
	}
	s.Options.LogFailures = logfailures
	p := newParameters()
	p.Searches["s1"] = s
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
