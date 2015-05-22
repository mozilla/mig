// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkg

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	ovallib "github.com/mozilla/mozoval/go/src/oval"
	"io"
	"io/ioutil"
	"mig/modules"
	"regexp"
	"runtime"
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
	stats.ExecRuntime = time.Now().Sub(counters.startTime).String()
}

type module struct {
}

func (m *module) NewRunner() interface{} {
	return new(run)
}

func init() {
	modules.Register("pkg", new(module))
}

type run struct {
	Parameters Parameters
	Results    modules.Result
}

func buildResults(e elements, r *modules.Result, matches int) (buf []byte, err error) {
	r.Success = true
	if matches > 0 {
		r.FoundAnything = true
	}
	r.Elements = e
	endCounters()
	r.Statistics = stats
	buf, err = json.Marshal(r)
	return
}

func makeOvalString(inbuf string) (string, error) {
	b := bytes.NewBufferString(inbuf)
	decoder := base64.NewDecoder(base64.StdEncoding, b)
	gz, err := gzip.NewReader(decoder)
	if err != nil {
		return "", err
	}
	ovalbuf, err := ioutil.ReadAll(gz)
	if err != nil {
		return "", err
	}
	return string(ovalbuf), nil
}

func (r *run) Run(in io.Reader) (resStr string) {
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

	// Restrict go runtime processor utilization here, this might be moved
	// into a more generic agent module function at some point.
	runtime.GOMAXPROCS(1)

	startCounters()

	// Read module parameters from stdin
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	ovallib.SetMaxChecks(r.Parameters.MaxConcurrentEval)

	e := &elements{}

	if len(r.Parameters.PkgMatch.Matches) > 0 {
		oresp := ovallib.PackageQuery(r.Parameters.PkgMatch.Matches, true)
		for _, x := range oresp {
			npi := &PkgInfo{PkgName: x.Name, PkgVersion: x.Version, PkgType: x.PkgType}
			e.Matches = append(e.Matches, *npi)
		}

		buf, err := buildResults(*e, &r.Results, len(e.Matches))
		if err != nil {
			panic(err)
		}
		resStr = string(buf)
		return
	} else if r.Parameters.OvalDef != "" {
		stats.InDefSize = len(r.Parameters.OvalDef)
		ovalbuf, err := makeOvalString(r.Parameters.OvalDef)
		if err != nil {
			panic(err)
		}
		pst := time.Now()
		od, err := ovallib.ParseBuffer(ovalbuf)
		stats.Parsetime = time.Now().Sub(pst).String()
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

		buf, err := buildResults(*e, &r.Results, len(e.OvalResults))
		if err != nil {
			panic(err)
		}
		resStr = string(buf)
		return
	}

	panic("no function specified")
	return
}

func (r *run) ValidateParameters() (err error) {
	if r.Parameters.MaxConcurrentEval <= 0 || r.Parameters.MaxConcurrentEval > 10 {
		return fmt.Errorf("concurrent evaluation must be between > 0 and <= 10")
	}

	// Supplying a definition with other modes is invalid
	if r.Parameters.OvalDef != "" && len(r.Parameters.PkgMatch.Matches) > 0 {
		return fmt.Errorf("cannot specify definition mode with other modes")
	}

	// Make sure a mode has been specified
	if r.Parameters.OvalDef == "" && len(r.Parameters.PkgMatch.Matches) == 0 {
		return fmt.Errorf("must specify a mode of operation")
	}

	// If an oval definition has been supplied, try parsing it to validate it
	if r.Parameters.OvalDef != "" {
		ovalbuf, err := makeOvalString(r.Parameters.OvalDef)
		if err != nil {
			return err
		}
		_, err = ovallib.ParseBuffer(ovalbuf)
		if err != nil {
			return err
		}
	}

	// If package match parameters have been specified, make sure they all
	// compile here.
	for _, x := range r.Parameters.PkgMatch.Matches {
		_, err = regexp.Compile(x)
		if err != nil {
			return err
		}
	}
	return
}

func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		elem  elements
		stats Statistics
	)

	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}
	err = result.GetStatistics(&stats)
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
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
		stats := fmt.Sprintf("Statistics: runtime %v, parsetime %v, defsize %v", stats.ExecRuntime, stats.Parsetime, stats.InDefSize)
		prints = append(prints, stats)
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
	ExecRuntime string `json:"execruntime"`
	Parsetime   string `json:"ovalparsetime"`
	InDefSize   int    `json:"inputdefinitionsize"`
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
	return &Parameters{MaxConcurrentEval: 1}
}
