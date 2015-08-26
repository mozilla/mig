// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkg /* import "mig.ninja/mig/modules/pkg" */

import (
	"encoding/json"
	"fmt"
	scribelib "github.com/mozilla/scribe/src/scribe"
	"io"
	"mig.ninja/mig/modules"
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

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("pkg", new(module))
}

type run struct {
	Parameters Parameters
	Results    modules.Result
}

func buildResults(e elements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	if len(e.Packages) > 0 {
		r.FoundAnything = true
	}
	endCounters()
	r.Statistics = stats
	buf, err = json.Marshal(r)
	return
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

	e := &elements{}
	e.Packages = make([]scribelib.PackageInfo, 0)
	pkglist := scribelib.QueryPackages()
	for _, x := range r.Parameters.PkgMatch.Matches {
		re, err := regexp.Compile(x)
		if err != nil {
			panic(err)
		}
		for _, y := range pkglist {
			if !re.MatchString(y.Name) {
				continue
			}
			e.Packages = append(e.Packages, y)
		}
	}
	buf, err := buildResults(*e, &r.Results)
	if err != nil {
		panic(err)
	}
	resStr = string(buf)
	return
}

func (r *run) ValidateParameters() (err error) {
	if len(r.Parameters.PkgMatch.Matches) == 0 {
		return fmt.Errorf("must specify at least one package to match")
	}
	// Make sure all package match parameters are valid expressions.
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

	for _, x := range elem.Packages {
		resStr := fmt.Sprintf("pkgmatch name=%v version=%v type=%v", x.Name, x.Version, x.Type)
		prints = append(prints, resStr)
	}

	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
		stats := fmt.Sprintf("Statistics: runtime %v", stats.ExecRuntime)
		prints = append(prints, stats)
	}

	return
}

type elements struct {
	Packages []scribelib.PackageInfo `json:"packages"` // Results of package query.
}

type Statistics struct {
	ExecRuntime string `json:"execruntime"` // Total execution time.
}

type Parameters struct {
	PkgMatch PkgMatch `json:"pkgmatch"` // List of strings to use as regexp package matches.
}

type PkgMatch struct {
	Matches []string `json:"matches"`
}

func newParameters() *Parameters {
	return &Parameters{}
}
