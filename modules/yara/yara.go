/// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

// yara module implementation for MIG.

package yara /* import "mig.ninja/mig/modules/yara" */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	yara "github.com/hillu/go-yara"

	"mig.ninja/mig/modules"
	"mig.ninja/mig/modules/file"
)

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("yara", new(module))
}

type run struct {
	Parameters parameters
	Results    modules.Result
}

func buildResults(e YaraElements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	if len(e.Matches) > 0 {
		r.FoundAnything = true
	}
	buf, err = json.Marshal(r)
	return
}

// Convert sub-module arguments supplied as a string in the arguments of the
// yara module query to a slice. For example, parses input like:
// -path /etc -name test -content "test file"
//
// into:
// ["-path", "/etc", "-name", "test", "content", "test file"]
func moduleArguments(args string) (ret []string) {
	r := regexp.MustCompile("'.+'|\".+\"|\\S+")
	m := r.FindAllString(args, -1)
	for _, x := range m {
		b := strings.Trim(x, "\"")
		b = strings.Trim(b, "'")
		ret = append(ret, b)
	}
	return
}

// If a file scan is being conducted, this locates any files we will include in the
// scan. This function makes use of the file module for this purpose.
func findFiles(args []string) ([]string, error) {
	ret := make([]string, 0)

	run := modules.Available["file"].NewRun()
	param, err := run.(modules.HasParamsParser).ParamsParser(args)
	if err != nil {
		return ret, err
	}
	buf, err := modules.MakeMessage(modules.MsgClassParameters, param, false)
	if err != nil {
		return ret, err
	}
	rdr := modules.NewModuleReader(bytes.NewReader(buf))

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

func (r *run) Run(in modules.ModuleReader) (resStr string) {
	defer func() {
		if e := recover(); e != nil {
			// return error in json
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			err, _ := json.Marshal(r.Results)
			resStr = string(err)
			return
		}
	}()

	// Restrict go runtime processor utilization here, this might be moved
	// into a more generic agent module function at some point.
	runtime.GOMAXPROCS(1)

	// Read module parameters from stdin
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	yaracomp, err := yara.NewCompiler()
	if err != nil {
		panic(err)
	}
	err = yaracomp.AddString(r.Parameters.YaraRules, "default")
	if err != nil {
		panic(err)
	}
	rules, err := yaracomp.GetRules()
	if err != nil {
		panic(err)
	}
	e := &YaraElements{}
	if r.Parameters.FileSearch != "" {
		flist, err := findFiles(moduleArguments(r.Parameters.FileSearch))
		if err != nil {
			panic(err)
		}
		for _, x := range flist {
			mr, err := rules.ScanFile(x, 0, time.Second*10)
			if err != nil {
				emsg := fmt.Sprintf("%v: %v", x, err)
				r.Results.Errors = append(r.Results.Errors, emsg)
				continue
			}
			if len(mr) != 0 {
				nm := YaraMatch{Object: x}
				for _, y := range mr {
					nm.MatchedRules = append(nm.MatchedRules, y.Rule)
				}
				e.Matches = append(e.Matches, nm)
			}
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
	if r.Parameters.YaraRules == "" {
		return fmt.Errorf("yara rules are a required option")
	}
	if r.Parameters.FileSearch != "" {
		run := modules.Available["file"].NewRun()
		_, err = run.(modules.HasParamsParser).ParamsParser(moduleArguments(r.Parameters.FileSearch))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		elem YaraElements
	)

	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}
	for _, x := range elem.Matches {
		var rn []string
		for _, y := range x.MatchedRules {
			rn = append(rn, y)
		}
		prints = append(prints, fmt.Sprintf("%v [%v]", x.Object, strings.Join(rn, ",")))
	}
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
	}
	return
}

type YaraMatch struct {
	Object       string   // Object matched (e.g., file name)
	MatchedRules []string // Matched rule
}

type YaraElements struct {
	Matches []YaraMatch // Module returns a list of matches
}

type Parameters struct {
	YaraRules  string `json:"yara"`       // Yara rules as a string
	FileSearch string `json:"filesearch"` // file module parameters for file search
}

func newParameters() *parameters {
	return &parameters{}
}
