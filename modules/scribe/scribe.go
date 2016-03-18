// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

// scribe module implementation for MIG.
package scribe /* import "mig.ninja/mig/modules/scribe" */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"time"

	scribelib "github.com/mozilla/scribe"
	"mig.ninja/mig/modules"
	"mig.ninja/mig/modules/file"
)

var stats statistics

// Counters to populate statistics at end of run.
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
	modules.Register("scribe", new(module))
}

type run struct {
	Parameters parameters
	Results    modules.Result
}

func buildResults(e ScribeElements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	if len(e.Results) > 0 {
		r.FoundAnything = true
	}
	// If any tests resulted in an error, store them as errors in the command.
	for _, x := range e.Results {
		if x.IsError {
			es := fmt.Sprintf("Error: %v in \"%v\"", x.Error, x.TestID)
			r.Errors = append(r.Errors, es)
		}
	}
	endCounters()
	r.Statistics = stats
	buf, err = json.Marshal(r)
	return
}

// This function is used to call the file module from this module. In order to
// avoid exporting types from the file module, we construct parameters for the
// file module using the parameter creation functions (passing command line
// arguments).
//
// We use the file modules file system location functions here to avoid
// duplicating functionality in this module.
func fileModuleLocator(pattern string, regex bool, root string, depth int) ([]string, error) {
	ret := make([]string, 0)

	// Build a pseudo-run struct to let us call the file module.
	run := modules.Available["file"].NewRun()
	args := make([]string, 0)
	args = append(args, "-path", root)
	args = append(args, "-name", pattern)
	args = append(args, "-maxdepth", strconv.Itoa(depth))
	param, err := run.(modules.HasParamsParser).ParamsParser(args)

	buf, err := modules.MakeMessage(modules.MsgClassParameters, param, false)
	if err != nil {
		return ret, nil
	}
	rdr := bytes.NewReader(buf)

	res := run.Run(rdr)
	var modresult modules.Result
	var sr file.SearchResults
	err = json.Unmarshal([]byte(res), &modresult)
	if err != nil {
		return ret, err
	}
	err = modresult.GetElements(&sr)
	if err != nil {
		return ret, err
	}

	p0, ok := sr["s1"]
	if !ok {
		return ret, fmt.Errorf("result in file module call was missing")
	}
	for _, x := range p0 {
		ret = append(ret, x.File)
	}

	return ret, nil
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

	// Initialize scribe
	scribelib.Bootstrap()

	// Install the file locator; this allows us to use the file module's
	// search functionality overriding the scribe built-in file system
	// walk function.
	scribelib.InstallFileLocator(fileModuleLocator)

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

	document := r.Parameters.ScribeDoc
	e := &ScribeElements{}
	e.HumanOutput = r.Parameters.HumanOutput
	e.JSONOutput = r.Parameters.JSONOutput

	e.Results = make([]scribelib.TestResult, 0)
	// Proceed with analysis here. ValidateParameters() will have already
	// validated the document.
	err = scribelib.AnalyzeDocument(document)
	if err != nil {
		panic(err)
	}
	for _, x := range document.GetTestIdentifiers() {
		tr, err := scribelib.GetResults(&document, x)
		if err != nil {
			panic(err)
		}
		if !tr.MasterResult && r.Parameters.OnlyTrue {
			continue
		}
		e.Results = append(e.Results, tr)
	}
	buf, err := buildResults(*e, &r.Results)
	if err != nil {
		panic(err)
	}
	resStr = string(buf)
	return
}

func (r *run) ValidateParameters() (err error) {
	return r.Parameters.ScribeDoc.Validate()
}

func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		elem  ScribeElements
		stats statistics
	)

	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}
	err = result.GetStatistics(&stats)
	if err != nil {
		panic(err)
	}
	for _, x := range elem.Results {
		if elem.HumanOutput {
			prints = append(prints, x.String())
		} else if elem.JSONOutput {
			prints = append(prints, x.JSON())
		} else {
			for _, y := range x.SingleLineResults() {
				prints = append(prints, y)
			}
		}
	}
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
		s := fmt.Sprintf("Statistics: runtime %v", stats.ExecRuntime)
		prints = append(prints, s)
	}
	return
}

type ScribeElements struct {
	Results     []scribelib.TestResult `json:"results"`     // Results of evaluation.
	HumanOutput bool                   `json:"humanoutput"` // Requested human output mode.
	JSONOutput  bool                   `json:"jsonoutput"`  // Requested JSON output mode.
}

type statistics struct {
	ExecRuntime string `json:"execruntime"` // Total execution time.
}

type parameters struct {
	ScribeDoc   scribelib.Document `json:"scribedoc"`   // The scribe document for analysis.
	OnlyTrue    bool               `json:"onlytrue"`    // Only return true evaluations
	HumanOutput bool               `json:"humanoutput"` // Use scribe extended output mode.
	JSONOutput  bool               `json:"jsonoutput"`  // Use JSON output mode.
}

func newParameters() *parameters {
	return &parameters{}
}
