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
	ovallib "github.com/ameihm0912/mozoval/go/src/oval"
	"io/ioutil"
	"mig/modules"
	"time"
)

var stats Statistics

// Various counters used to populate module statistics at the end of the
// run.
var counters struct {
	startTime time.Time
}

func startCounters() {
	counters.startTime = time.Now()
}

func endCounters() {
	stats.OvalRuntime = time.Now().Sub(counters.startTime)
}

func init() {
	modules.Register("oval", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters Parameters
	Results    modules.Result
}

func (r Runner) Run() (resStr string) {
	defer func() {
		if e := recover(); e != nil {
			// return error in json
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			endCounters()
			r.Results.Statistics = stats
			err, _ := json.Marshal(r.Results)
			resStr = string(err)
			return
		}
	}()

	startCounters()

	// Read module parameters from stdin
	err := modules.ReadInputParameters(&r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	ovallib.Init()
	ovallib.SetMaxChecks(r.Parameters.MaxConcurrentEval)

	e := &elements{}

	if len(r.Parameters.PkgMatch.Matches) > 0 {
		oresp := ovallib.PackageQuery(r.Parameters.PkgMatch.Matches)
		for _, x := range oresp {
			npi := &PkgInfo{PkgName: x.Name, PkgVersion: x.Version, PkgType: x.PkgType}
			e.Matches = append(e.Matches, *npi)
		}

		r.Results.Success = true
		if len(e.Matches) > 0 {
			r.Results.FoundAnything = true
		}
		r.Results.Elements = e
		endCounters()
		r.Results.Statistics = stats
		buf, err := json.Marshal(r.Results)
		if err != nil {
			panic(err)
		}
		resStr = string(buf)
		return
	} else if r.Parameters.OvalDef != "" {
		b := bytes.NewBufferString(r.Parameters.OvalDef)
		decoder := base64.NewDecoder(base64.StdEncoding, b)
		gz, err := gzip.NewReader(decoder)
		if err != nil {
			panic(err)
		}
		ovalbuf, err := ioutil.ReadAll(gz)
		if err != nil {
			panic(err)
		}

		od, err := ovallib.ParseBuffer(string(ovalbuf))
		if err != nil {
			panic(err)
		}
		ovalresults, err := ovallib.Execute(od)
		if err != nil {
			panic(err)
		}
		for _, x := range ovalresults {
			if !r.Parameters.IncludeFalse {
				if x.Status == ovallib.RESULT_FALSE {
					continue
				}
			}
			nmor := &MOResult{}
			nmor.Title = x.Title
			nmor.Status = x.StatusString()
			nmor.ID = x.ID
			e.OvalResults = append(e.OvalResults, *nmor)
		}

		r.Results.Success = true
		if len(e.OvalResults) > 0 {
			r.Results.FoundAnything = true
		}
		r.Results.Elements = e
		endCounters()
		r.Results.Statistics = stats
		buf, err := json.Marshal(r.Results)
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
	if r.Parameters.MaxConcurrentEval <= 0 || r.Parameters.MaxConcurrentEval > 10 {
		return fmt.Errorf("concurrent evaluation must be between > 0 and <= 10")
	}
	return
}

func (r Runner) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var elem elements

	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}

	for _, x := range elem.Matches {
		resStr := fmt.Sprintf("pkgmatch name=%v version=%v type=%v", x.PkgName, x.PkgVersion, x.PkgType)
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
	PkgType    string `json:"type"`
}

type Statistics struct {
	OvalRuntime time.Duration `json:"ovalruntime"`
}

type Parameters struct {
	// Package match mode, contains a list of strings to use as substring
	// matches
	PkgMatch PkgMatch `json:"pkgmatch"`

	// A compressed, base64 encoded OVAL definition file for processing
	// using OVAL library on agent.
	OvalDef string `json:"ovaldef"`

	// Concurrent checks to run on agent
	MaxConcurrentEval int `json:"maxconneval"`

	// Include false results for checks
	IncludeFalse bool `json:"includefalse"`
}

type PkgMatch struct {
	Matches []string `json:"matches"`
}

func newParameters() *Parameters {
	return &Parameters{}
}
