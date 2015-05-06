// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package migoval

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
)

func printHelp(isCmd bool) {
	dash := ""
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`Query parameters
----------------
%spkgmatch <string> - OS package search
                    ex: pkgmatch linux-image
		    query for installed OS packages containing substring

%soval <path>       - OVAL processor
		    ex: oval ./ovaldefs.xml
		    process oval definitions on agent
`, dash, dash)
}

func (r Runner) ParamsParser(args []string) (interface{}, error) {
	var (
		fs       flag.FlagSet
		ovalDefs string
		pkgMatch flagParam
	)

	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, nil
	}

	fs.Init("migoval", flag.ContinueOnError)
	fs.Var(&pkgMatch, "pkgmatch", "see help")
	fs.StringVar(&ovalDefs, "oval", "", "see help")
	err := fs.Parse(args)
	if err != nil {
		return nil, err
	}

	p := newParameters()
	p.PkgMatch.Matches = pkgMatch

	if ovalDefs != "" {
		var b bytes.Buffer
		fd, err := os.Open(ovalDefs)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		encoder := base64.NewEncoder(base64.StdEncoding, &b)
		gz := gzip.NewWriter(encoder)
		_, err = io.Copy(gz, fd)
		gz.Close()
		encoder.Close()
		if err != nil {
			return nil, err
		}
		p.OvalDef = b.String()
	}

	r.Parameters = *p

	return r.Parameters, r.ValidateParameters()
}

type flagParam []string

func (f *flagParam) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (f *flagParam) String() string {
	return fmt.Sprint([]string(*f))
}
