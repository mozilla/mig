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
	"encoding/json"
	"fmt"
	"github.com/ameihm0912/mozoval/go/src/oval"
	"io"
	"io/ioutil"
	"mig"
	"os"
)

func init() {
	mig.RegisterModule("migoval", func() interface{} {
		return new(Runner)
	}, true) // Register for STDIN parameter submission
}

type Runner struct {
	Parameters Parameters
	Results    mig.ModuleResult
}

func (r Runner) Run(CommandArgs []byte) (resStr string) {
	defer func() {
		if e := recover(); e != nil {
			// return error in json
			res := newResults()
			res.Errors = append(res.Errors, fmt.Sprintf("%v", e))
			res.Success = false
			err, _ := json.Marshal(res)
			resStr = string(err)
			return
		}
	}()

	// This module reads parameters from STDIN; any arguments passed via
	// the command line are ignored.
	buf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(buf, &r.Parameters)
	if err != nil {
		panic(err)
	}

	oval.Init()

	e := &elements{}

	if len(r.Parameters.PkgMatch.Matches) > 0 {
		oresp := oval.PackageQuery(r.Parameters.PkgMatch.Matches)
		for _, x := range oresp {
			npi := &PkgInfo{PkgName: x.Name, PkgVersion: x.Version}
			e.Matches = append(e.Matches, *npi)
		}

		res := newResults()
		res.Success = true
		if len(e.Matches) > 0 {
			res.FoundAnything = true
		}
		res.Elements = e
		buf, err := json.Marshal(res)
		if err != nil {
			panic(err)
		}
		resStr = string(buf)
		return
	} else if r.Parameters.OvalDef != "" {
		var ovalbuf bytes.Buffer
		b := bytes.NewBufferString(r.Parameters.OvalDef)
		decoder := base64.NewDecoder(base64.StdEncoding, b)
		gz, err := gzip.NewReader(decoder)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(&ovalbuf, gz)

		od, err := oval.ParseBuffer(ovalbuf.String())
		ovalresults := oval.Execute(od)
		for _, x := range ovalresults {
			nmor := &MOResult{}
			nmor.Title = x.Title
			nmor.Status = x.StatusString()
			nmor.ID = x.ID
			e.OvalResults = append(e.OvalResults, *nmor)
		}

		res := newResults()
		res.Success = true
		if len(e.OvalResults) > 0 {
			res.FoundAnything = true
		}
		res.Elements = e
		buf, err := json.Marshal(res)
		if err != nil {
			panic(err)
		}
		resStr = string(buf)
		return
	}

	panic("no function specified")
	return
}

func (r Runner) ValidateParameters() (err error) {
	return
}

func (r Runner) PrintResults(rawResults []byte, foundOnly bool) (prints []string, err error) {
	var results mig.ModuleResult
	var elem elements

	err = json.Unmarshal(rawResults, &results)
	if err != nil {
		panic(err)
	}
	newelements, err := json.Marshal(results.Elements)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(newelements, &elem)
	if err != nil {
		panic(err)
	}

	for _, x := range elem.Matches {
		resStr := fmt.Sprintf("pkgmatch name=%v version=%v", x.PkgName, x.PkgVersion)
		prints = append(prints, resStr)
	}

	for _, x := range elem.OvalResults {
		resStr := fmt.Sprintf("ovalresult id=%v title=\"%v\" outcome=%v", x.ID, x.Title, x.Status)
		prints = append(prints, resStr)
	}

	return
}

type elements struct {
	// In package match mode, the packages the agent has found that match
	// the query parameters.
	Matches []PkgInfo `json:"matches"`

	// Results of OVAL definition checks in OVAL mode
	OvalResults []MOResult `json:"ovalresults"`
}

type MOResult struct {
	Title  string `json:"title"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

type PkgInfo struct {
	PkgName    string `json:"name"`
	PkgVersion string `json:"version"`
}

type Parameters struct {
	// Package match mode, contains a list of strings to use as substring
	// matches
	PkgMatch PkgMatch `json:"pkgmatch"`

	// A compressed, base64 encoded OVAL definition file for processing
	// using OVAL library on agent.
	OvalDef string `json:"ovaldef"`
}

type PkgMatch struct {
	Matches []string `json:"matches"`
}

func newParameters() *Parameters {
	return &Parameters{}
}

func newResults() *mig.ModuleResult {
	return &mig.ModuleResult{}
}
