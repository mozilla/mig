// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package file

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

const help string = `To add search parameters, use the following syntax:
path <path>		add a path to search.
			example: > /etc/yum.*/*.repo

content <regex>		add a regex to check against files content
			example: > ^root:\$1\$10CXRS19\$/h

name <regex>		add a regex to search against filenames
			example: > \.sql$

<hashType> <hash>	add an hash to compare files against
			Available hash types:
				md5, sha1, sha256, sha384, sha512,
				sha3_224, sha3_256, sha3_384, sha3_512
			example: > md5 a12496cb3fd77a535df2d6ddc2e4ef53
When done, enter 'done'`

// ParamsCreator implements an interactive parameters creation interface, which
// receives user input,  stores it into a Parameters structure, validates it,
// and returns that structure as an interface. It is mainly used by the MIG Console
func (r Runner) ParamsCreator() (interface{}, error) {
	fmt.Println("initializing filechecker parameters creation")
	var err error
	p := newParameters()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		var label string
		var search search
		for {
			fmt.Println("create a new search by entering a search label, or 'done' to exit")
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
		fmt.Printf("creating new search with label: %s\n%s\n", label, help)

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
				fmt.Printf("%s\n", help)
				continue
			}
			arr := strings.SplitN(input, " ", 2)
			if len(arr) != 2 {
				fmt.Printf("Invalid input format!\n%s\n", help)
				continue
			}
			checkType := arr[0]
			checkValue := arr[1]
			switch checkType {
			case "path":
				search.Paths = append(search.Paths, checkValue)
			case "content":
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Contents = append(search.Contents, checkValue)
			case "name":
				err = validateRegex(checkValue)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.Names = append(search.Names, checkValue)
			case "md5":
				err = validateHash(checkValue, checkMD5)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.MD5 = append(search.MD5, checkValue)
			case "sha1":
				err = validateHash(checkValue, checkSHA1)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA1 = append(search.SHA1, checkValue)
			case "sha256":
				err = validateHash(checkValue, checkSHA256)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA256 = append(search.SHA256, checkValue)
			case "sha384":
				err = validateHash(checkValue, checkSHA384)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA384 = append(search.SHA384, checkValue)
			case "sha512":
				err = validateHash(checkValue, checkSHA512)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA512 = append(search.SHA512, checkValue)
			case "sha3_224":
				err = validateHash(checkValue, checkSHA3_224)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA3_224 = append(search.SHA3_224, checkValue)
			case "sha3_256":
				err = validateHash(checkValue, checkSHA3_256)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA3_256 = append(search.SHA3_256, checkValue)
			case "sha3_384":
				err = validateHash(checkValue, checkSHA3_384)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA3_384 = append(search.SHA3_384, checkValue)
			case "sha3_512":
				err = validateHash(checkValue, checkSHA3_512)
				if err != nil {
					fmt.Printf("ERROR: %v\nTry again.\n", err)
					continue
				}
				search.SHA3_512 = append(search.SHA3_512, checkValue)
			default:
				fmt.Printf("Invalid method!\n%s\n", help)
				continue
			}
			fmt.Printf("Stored %s %s\n", checkType, checkValue)
		}
		p.Searches[label] = search
		fmt.Println("Stored search", label)
	}
exit:
	return p, nil
}

const cmd_help string = `~~~ FILE module ~~~
-path <string>		inspects the given path recursively.
			At least one path must be given on invocation.

-name <regex>		looks for files that have a name that match this regex

-content <regex>	look for files that have a content that match this regex

-md5, -sha1, -sha256,
-sha384, -sha512,
-sha3_224, -sha3_256,
-sha3_384, -sha3_512 <hash>	look for files that match a given checksum
`

// ParamsParser implements a command line parameters parser that takes a string
// and returns a Parameters structure in an interface. It will display the module
// help if the arguments string spell the work 'help'
func (r Runner) ParamsParser(args []string) (interface{}, error) {
	var (
		err error
		paths, names, contents, md5s, sha1s, sha256s, sha384s,
		sha512s, sha3_224s, sha3_256s, sha3_384s, sha3_512s flagParam
		fs flag.FlagSet
	)
	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		fmt.Println(cmd_help)
		return nil, fmt.Errorf("help printed")
	}
	fs.Init("file", flag.ContinueOnError)
	fs.Var(&paths, "path", "see help")
	fs.Var(&names, "name", "see help")
	fs.Var(&contents, "content", "see help")
	fs.Var(&md5s, "md5", "see help")
	fs.Var(&sha1s, "sha1", "see help")
	fs.Var(&sha256s, "sha256", "see help")
	fs.Var(&sha384s, "sha384", "see help")
	fs.Var(&sha512s, "sha512", "see help")
	fs.Var(&sha3_224s, "sha3_224", "see help")
	fs.Var(&sha3_256s, "sha3_256", "see help")
	fs.Var(&sha3_384s, "sha3_384", "see help")
	fs.Var(&sha3_512s, "sha3_512", "see help")
	err = fs.Parse(args)
	if err != nil {
		return nil, err
	}
	var s search
	s.Paths = paths
	s.Names = names
	s.Contents = contents
	s.MD5 = md5s
	s.SHA1 = sha1s
	s.SHA256 = sha256s
	s.SHA384 = sha384s
	s.SHA512 = sha512s
	s.SHA3_224 = sha3_224s
	s.SHA3_256 = sha3_256s
	s.SHA3_384 = sha3_384s
	s.SHA3_512 = sha3_512s
	p := newParameters()
	p.Searches["s1"] = s
	return p, nil
}

type flagParam []string

func (f *flagParam) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *flagParam) Set(value string) error {
	*f = append(*f, value)
	return nil
}
