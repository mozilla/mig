// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package migoval

import (
	"flag"
	"fmt"
)

func printHelp() {
	fmt.Printf("No help is available - good luck!\n")
}

func (r Runner) ParamsParser(args []string) (interface{}, error) {
	var (
		fs      flag.FlagSet
		pkglist bool
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp()
		return nil, nil
	}

	fs.Init("migoval", flag.ContinueOnError)
	fs.BoolVar(&pkglist, "pkglist", true, "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}
