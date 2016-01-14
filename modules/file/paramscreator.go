// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package file /* import "mig.ninja/mig/modules/file" */

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
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`Search parameters
-----------------
%spath <string>	- search path
		  ex: path /etc
		  note that the file module will follow symlinks, but only if the linked
		  path is located within the base path search
		  ex: if path is set to /sys/bus/usb/devices/, it will not follow symlinks
		  located in /sys/devices.

%sname <regex>	- regex to match against the name of a file. use !<regex> to inverse it.
		  ex: name \.sql$

%ssize <size>	- match files with a size smaller or greater that <size>
		  prefix with '<' for lower than, and '>' for greater than
		  suffix with k, m, g or t for kilo, mega, giga and terabytes
		  ex: size <10m (match files larger than 10 megabytes)

%smode <regex>	- filter on the filemode, provided as a regex on the mode string
		  ex: mode -r(w|-)xr-x---

%smtime <period>  - match files modified before or since <period>
		  prefix with '<' for modified since, and '>' for modified before
		  suffix with d, h, m for days, hours and minutes
		  ex: mtime <90d (match files modified since last 90 days)

%scontent <regex> - regex to match against file content. use !<regex> to inverse it.
		  ex: content ^root:\$1\$10CXRS19\$/h

%smd5 <hash>      .
%ssha1 <hash>     .
%ssha2 <hash>     .
%ssha3 <hash>     - compare file against given hash
 


Options
-------
%smaxdepth <int>	- limit search depth to <int> levels. default to 1000.
			  ex: %smaxdepth 3
%smatchall		- all search parameters must match on a given file for it to
			  return as a match. off by default. deactivates 'matchany' if set.
			  ex: %smatchall
%smatchany		- any search parameter must match on a given file for it to
			  return as a match. on by default. deactivates 'matchall' if set.
			  ex: %smatchany
%smacroal		- match all contents regexes on all lines. off by default.
			  ex: %smacroal
%smismatch=<filter>	- inverts the results for the given filter, used to list files
			  that did not match a given expression, instead of the default
			  instead of files that match it.
			  ex: %smismatch content
%smatchlimit <int>	- limit the number of files that can be matched by a search.
			  the default limit is set to 1000. search will stop once the limit
			  is reached.
%sreturnsha256		- include sha256 hash for matched files.
			  ex: %sreturnsha256

Module documentation is at http://mig.mozilla.org/doc/module_file.html
Cheatsheet and examples are at http://mig.mozilla.org/doc/cheatsheet.rst.html
`, dash, dash, dash, dash, dash, dash, dash, dash, dash, dash, dash,
		dash, dash, dash, dash, dash, dash, dash, dash, dash,
		dash, dash, dash, dash, dash, dash, dash, dash)

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
		// sane defaults
		search.Options.MatchAll = true
		search.Options.MaxDepth = 1000
		search.Options.MatchLimit = 1000
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
			case "path":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				search.Paths = append(search.Paths, checkValue)
			case "name":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Names = append(search.Names, checkValue)
			case "size":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateSize(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Sizes = append(search.Sizes, checkValue)
			case "mode":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Modes = append(search.Modes, checkValue)
			case "mtime":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateMtime(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Mtimes = append(search.Mtimes, checkValue)
			case "content":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Contents = append(search.Contents, checkValue)
			case "md5":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateHash(checkValue, checkMD5)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.MD5 = append(search.MD5, checkValue)
			case "sha1":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateHash(checkValue, checkSHA1)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA1 = append(search.SHA1, checkValue)
			case "sha2":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				var hashSize = len(checkValue)
				hashType := checkContent
				switch hashSize {
				case 64:
					hashType = checkSHA256
				case 96:
					hashType = checkSHA384
				case 128:
					hashType = checkSHA512
				default:
					fmt.Printf("ERROR: Invalid hash length")
				}
				err = validateHash(checkValue, hashType)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA2 = append(search.SHA2, checkValue)
			case "sha3":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				var hashSize = len(checkValue)
				hashType := checkContent
				switch hashSize {
				case 56:
					hashType = checkSHA3_224
				case 64:
					hashType = checkSHA3_256
				case 96:
					hashType = checkSHA3_384
				case 128:
					hashType = checkSHA3_512
				default:
					fmt.Printf("ERROR: Invalid hash length")
				}
				err = validateHash(checkValue, hashType)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA3 = append(search.SHA3, checkValue)
			case "maxdepth":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
			case "matchall":
				if checkValue != "" {
					fmt.Println("This option doesn't take arguments, try again")
					continue
				}
				search.Options.MatchAll = true

			case "matchany":
				if checkValue != "" {
					fmt.Println("This option doesn't take arguments, try again")
					continue
				}
				search.Options.MatchAll = false
			case "returnsha256":
				if checkValue != "" {
					fmt.Println("This option doesn't take arguments, try again")
					continue
				}
				search.Options.ReturnSHA256 = true
			case "macroal":
				if checkValue != "" {
					fmt.Println("This option doesn't take arguments, try again")
					continue
				}
				search.Options.Macroal = true
			case "mismatch":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				err = validateMismatch(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Options.Mismatch = append(search.Options.Mismatch, checkValue)
			case "matchlimit":
				if checkValue == "" {
					fmt.Println("Missing parameter, try again")
					continue
				}
				v, err := strconv.ParseFloat(checkValue, 64)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Options.MatchLimit = v
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
		err error
		paths, names, sizes, modes, mtimes, contents, md5s, sha1s, sha2s,
		sha3s, mismatch flagParam
		maxdepth, matchlimit                               float64
		returnsha256, matchall, matchany, macroal, verbose bool
		fs                                                 flag.FlagSet
	)
	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}
	fs.Init("file", flag.ContinueOnError)
	fs.Var(&paths, "path", "see help")
	fs.Var(&names, "name", "see help")
	fs.Var(&sizes, "size", "see help")
	fs.Var(&modes, "mode", "see help")
	fs.Var(&mtimes, "mtime", "see help")
	fs.Var(&contents, "content", "see help")
	fs.Var(&md5s, "md5", "see help")
	fs.Var(&sha1s, "sha1", "see help")
	fs.Var(&sha2s, "sha2", "see help")
	fs.Var(&sha3s, "sha3", "see help")
	fs.Var(&mismatch, "mismatch", "see help")
	fs.Float64Var(&maxdepth, "maxdepth", 1000, "see help")
	fs.Float64Var(&matchlimit, "matchlimit", 1000, "see help")
	fs.BoolVar(&matchall, "matchall", true, "see help")
	fs.BoolVar(&matchany, "matchany", false, "see help")
	fs.BoolVar(&macroal, "macroal", false, "see help")
	fs.BoolVar(&debug, "verbose", false, "see help")
	fs.BoolVar(&returnsha256, "returnsha256", false, "see help")
	err = fs.Parse(args)
	if err != nil {
		return nil, err
	}
	var s search
	s.Paths = paths
	s.Names = names
	s.Sizes = sizes
	s.Modes = modes
	s.Mtimes = mtimes
	s.Contents = contents
	s.MD5 = md5s
	s.SHA1 = sha1s
	s.SHA2 = sha2s
	s.SHA3 = sha3s
	s.Options.MaxDepth = maxdepth
	s.Options.MatchLimit = matchlimit
	s.Options.Macroal = macroal
	s.Options.Mismatch = mismatch
	s.Options.MatchAll = matchall
	s.Options.ReturnSHA256 = returnsha256
	if matchany {
		s.Options.MatchAll = false
	}
	if verbose {
		s.Options.Debug = "print"
	}
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
