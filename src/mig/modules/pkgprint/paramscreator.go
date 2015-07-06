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

%sdepth <int>       - Specify maximum directory search depth
                    ex: depth 2
		    default depth is 10

%sroot <path>       - Specify search root
                    ex: root /usr/local
		    default root is /

Available templates:

linuxkernel         - Running Linux kernel information
linuxmodules        - Loaded Linux modules
pythonegg           - Python package versions
django              - Django framework versions
mediawiki           - MediaWiki framework versions
`, dash, dash, dash)
}

func (r Runner) ParamsParser(args []string) (interface{}, error) {
	var (
		fs           flag.FlagSet
		templateName string
		depth        int
		root         string
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("pkgprint", flag.ContinueOnError)
	fs.StringVar(&templateName, "template", "", "see help")
	fs.IntVar(&depth, "depth", 10, "see help")
	fs.StringVar(&root, "root", "/", "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	if templateName != "" {
		p.TemplateParams.Name = templateName
		p.TemplateMode = true
	}
	p.SearchDepth = depth
	p.SearchRoot = root

	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}
