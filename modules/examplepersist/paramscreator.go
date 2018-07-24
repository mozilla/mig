// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package examplepersist /* import "github.com/mozilla/mig/modules/examplepersist" */

import (
	"flag"
	"fmt"
)

func printHelp(isCmd bool) {
	dash := ""
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`Query parameters
----------------
%secho <string>     - String to echo back from module
                    ex: echo testing
		    Requests examplepersist echo the given string back
`, dash)
}

func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		fs   flag.FlagSet
		echo string
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("examplepersist", flag.ContinueOnError)
	fs.StringVar(&echo, "echo", "", "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	p.String = echo

	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}
