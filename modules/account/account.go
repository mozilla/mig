// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Kishor Bhat kishorbhat@gmail.com [:kbhat]

package account /* import "mig.ninja/mig/modules/account" */

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"mig.ninja/mig/modules"
	"os"
	"regexp"
	"strings"
)

var debug bool = false

func debugprint(format string, a ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("account", new(module))
}

type run struct {
	Parameters params
	Results    modules.Result
}

type params struct {
	Group string `json:"group"`
	User  string `json:"user"`
}

type elements struct {
	FoundUser  bool `json:"founduser"`  // true if user is found, false otherwise
	FoundGroup bool `json:"foundgroup"` // true if group is found, false otherwise
}

func (r *run) ValidateParameters() (err error) {
	debugprint("validating user '%s'\n", r)
	err = validateName(r.Parameters.User)
	if err != nil {
		return fmt.Errorf("ERRROR: %v\n", err)
	}
	debugprint("validating group '%s'\n", r)
	err = validateName(r.Parameters.Group)
	if err != nil {
		return fmt.Errorf("ERROR: %v\n", err)
	}
	return
}

func validateName(name string) error {
	if len(name) < 1 {
		return fmt.Errorf("Empty names are not permitted")
	}
	nameregexp := `^[_a-z][-0-9_a-z]*\$?`
	namere := regexp.MustCompile(nameregexp)
	if !namere.MatchString(name) {
		return fmt.Errorf("The syntax of name '%s' is invalid, and must match regex %s", name, nameregexp)
	}
	return nil
}

func (r *run) Run(in io.Reader) (out string) {
	var (
		el  elements
		err error
	)
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			buf, _ := json.Marshal(r.Results)
			out = string(buf[:])
		}
	}()
	err = modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}
	if r.Parameters.User != "" {
		el.FoundUser, err = r.FindUser()
		if err != nil {
			return
		}
	}
	if r.Parameters.Group != "" {
		el.FoundGroup, err = r.FindGroup()
		if err != nil {
			return
		}
	}
	return r.buildResults(el)
}

// FindUser opens up /etc/passwd, and reads each line.
// If the supplied username is found, the user exists.
func (r *run) FindUser() (found bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("FindUser(): %v", e)
		}
	}()
	file, err := os.Open("/etc/passwd")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		uname := strings.SplitN(scanner.Text(), ":", 2)[0]
		found = strings.Contains(uname, r.Parameters.User)
		if found == true {
			break
		}
	}
	return
}

// FindGroup opens up /etc/group, and reads each line.
// If the supplied groupname is found, the user exists.
func (r *run) FindGroup() (found bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("FindGroup(): %v", e)
		}
	}()
	file, err := os.Open("/etc/group")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		gname := strings.SplitN(scanner.Text(), ":", 2)[0]
		found = strings.Contains(gname, r.Parameters.Group)
		if found == true {
			break
		}
	}
	return
}

func (r *run) buildResults(el elements) string {
	r.Results.Elements = el
	if len(r.Results.Errors) == 0 {
		r.Results.Success = true
	}
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}

func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var el elements
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Print Error: %v", e)
		}
	}()
	err = result.GetElements(&el)
	if err != nil {
		panic(err)
	}
	var phrase string
	if r.Parameters.User != "" {
		switch el.FoundUser {
		case true:
			phrase = "found"
		default:
			phrase = "not found"
		}
		prints = append(prints,
			fmt.Sprintf("User %s %s.", r.Parameters.User, phrase))
	}
	if r.Parameters.Group != "" {
		switch el.FoundGroup {
		case true:
			phrase = "found"
		default:
			phrase = "not found"
		}
		prints = append(prints,
			fmt.Sprintf("Group %s %s.", r.Parameters.Group, phrase))
	}

	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
	}
	return
}
