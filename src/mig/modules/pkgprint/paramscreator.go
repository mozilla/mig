// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkgprint

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
%stemplate <name>   - Scan using template
                    ex: template mediawiki
		    query for specific module supplied template
`, dash)
}

func (r Runner) ParamsParser(args []string) (interface{}, error) {
	var (
		fs           flag.FlagSet
		templateName string
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("pkgprint", flag.ContinueOnError)
	fs.StringVar(&templateName, "template", "", "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	if templateName != "" {
		p.TemplateParams.Name = templateName
		p.TemplateMode = true
	}

	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}
