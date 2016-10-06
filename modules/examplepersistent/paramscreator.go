// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package examplepersistent /* import "mig.ninja/mig/modules/examplepersistent" */

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
%secho <string>     - String to echo from module
                    ex: echo testing
		    String which will be echoed back from examplepersistent module
`, dash)
}

func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		fs flag.FlagSet
		s  string
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("examplepersistent", flag.ContinueOnError)
	fs.StringVar(&s, "echo", "", "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	p.String = s

	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}
