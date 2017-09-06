// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

// Package audit implements a persistent module which can read and parse the operating
// system audit trail. The module turns each audit event read from the kernel into a
// module alert, and writes it to the master agent process where the agent process can either
// write the audit event to it's log or send it to the dispatch module.
//
// The audit module is currently only supported on Linux.
package audit /* import "mig.ninja/mig/modules/audit" */

import (
	"encoding/json"
	"fmt"
	"runtime"

	"mig.ninja/mig/modules"
)

type module struct {
}

// NewRun returns a new instance of a modules.Runner for this module.
func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("audit", new(module))
}

type run struct {
	Parameters Parameters
	Results    modules.Result
}

func buildResults(e elements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	r.FoundAnything = true
	buf, err = json.Marshal(r)
	return
}

var logChan chan string
var alertChan chan string
var handlerErrChan chan error
var configChan chan modules.ConfigParams

func moduleMain() {
	var cfg config

	incfg := <-configChan
	buf, err := json.Marshal(incfg.Config)
	if err != nil {
		handlerErrChan <- err
		return
	}
	err = json.Unmarshal(buf, &cfg)
	if err != nil {
		handlerErrChan <- err
		return
	}
	logChan <- "module received configuration"

	err = initializeAudit(cfg)
	if err != nil {
		handlerErrChan <- err
		return
	}
	err = runAudit()
	if err != nil {
		handlerErrChan <- err
		return
	}
}

func requestHandler(p interface{}) (ret string) {
	var results modules.Result
	defer func() {
		if e := recover(); e != nil {
			results.Errors = append(results.Errors, fmt.Sprintf("%v", e))
			results.Success = false
			err, _ := json.Marshal(results)
			ret = string(err)
			return
		}
	}()
	e := elements{Ok: true}
	resp, err := buildResults(e, &results)
	if err != nil {
		panic(err)
	}
	return string(resp)
}

type config struct {
	Audit struct {
		RulesPath    string `json:"rulespath"`
		RateLimit    int    `json:"ratelimit"`
		BacklogLimit int    `json:"backloglimit"`
	} `json:"audit"`
}

// PersistModConfig returns a new configuration structure for this module.
func (r *run) PersistModConfig() interface{} {
	return &config{}
}

// RunPersist is the entry point for persistent execution of this module.
func (r *run) RunPersist(in modules.ModuleReader, out modules.ModuleWriter) {
	alertChan = make(chan string, 64)
	logChan = make(chan string, 64)
	regChan := make(chan string, 64)
	handlerErrChan = make(chan error, 64)
	configChan = make(chan modules.ConfigParams, 1)

	go moduleMain()
	l, spec, err := modules.GetPersistListener("audit")
	if err != nil {
		handlerErrChan <- err
	} else {
		regChan <- spec
	}
	go modules.HandlePersistRequest(l, requestHandler, handlerErrChan)
	modules.DefaultPersistHandlers(in, out, logChan, handlerErrChan, regChan,
		alertChan, configChan)
}

// Run is the entry point for standard (e.g., query) based invocation of this module.
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
	runtime.GOMAXPROCS(1)
	sockspec, err := modules.ReadPersistInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}
	resStr = modules.SendPersistRequest(r.Parameters, sockspec)
	return
}

// ValidateParameters validates the parameters set in the runner for the module.
func (r *run) ValidateParameters() (err error) {
	return
}

// PrintResults returns the results of a query of this module in human readable form.
func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		elem elements
	)

	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}
	resStr := fmt.Sprintf("ok:%v", elem.Ok)
	prints = append(prints, resStr)
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
	}
	return
}

type elements struct {
	Ok bool `json:"ok"`
}

// Parameters defines any query parameters used in this module.
type Parameters struct {
}

func newParameters() *Parameters {
	return &Parameters{}
}
