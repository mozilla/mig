// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package examplepersistent /* import "mig.ninja/mig/modules/examplepersistent" */

import (
	"encoding/json"
	"fmt"
	"runtime"

	"mig.ninja/mig/modules"
)

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("examplepersistent", new(module))
}

type run struct {
	Parameters Parameters
	Results    modules.Result
}

func buildResults(e elements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	if e.String != "" {
		r.FoundAnything = true
	}
	buf, err = json.Marshal(r)
	return
}

func (r *run) IsPersistent() bool {
	return false
}

func (r *run) Run(in modules.ModuleInput) (resStr string) {
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
	err := in.ReadInputParameters(&r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	e := &elements{}
	e.String = r.Parameters.String
	buf, err := buildResults(*e, &r.Results)
	if err != nil {
		panic(err)
	}
	resStr = string(buf)
	return
}

func (r *run) ValidateParameters() (err error) {
	if len(r.Parameters.String) == 0 {
		return fmt.Errorf("no string to echo specified")
	}
	return
}

func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		elem elements
	)
	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}
	prints = append(prints, elem.String)
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
	}
	return
}

type elements struct {
	String string `json:"string"`
}

type Parameters struct {
	String string `json:"string"`
}

func newParameters() *Parameters {
	return &Parameters{}
}
