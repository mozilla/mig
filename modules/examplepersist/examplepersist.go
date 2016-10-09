// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package examplepersist /* import "mig.ninja/mig/modules/examplepersist" */

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"time"

	"mig.ninja/mig/modules"
)

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("examplepersist", new(module))
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

func runSomeTasks() {
	for {
		time.Sleep(time.Second * 30)
		// Send a log message up to the agent
		logChan <- fmt.Sprintf("running, current time is %v", time.Now())
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
	// Marshal and unmarshal the parameters into the type we want
	param := Parameters{}
	buf, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, &param)
	if err != nil {
		panic(err)
	}
	// Create the response
	e := elements{String: param.String}
	resp, err := buildResults(e, &results)
	if err != nil {
		panic(err)
	}
	return string(resp)
}

func (r *run) RunPersist(in io.ReadCloser, out io.WriteCloser) {
	// Create a string channel, used to send log messages up to the agent
	// from the module tasks
	logChan = make(chan string, 64)
	// Start up an example background task
	go runSomeTasks()
	_ = os.Remove(modules.PersistSockPath("examplepersist"))
	l, err := net.Listen("unix", modules.PersistSockPath("examplepersist"))
	if err != nil {
		panic(err)
	}
	go modules.HandlePersistRequest(l, requestHandler)
	modules.DefaultPersistHandlers(in, out, logChan)
}

func (r *run) Run(in io.Reader) (resStr string) {
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
	resStr = modules.SendPersistRequest(r.Parameters, "examplepersist")
	return
}

func (r *run) ValidateParameters() (err error) {
	if r.Parameters.String == "" {
		return fmt.Errorf("must set a string to echo")
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

	resStr := fmt.Sprintf("echo string was %q", elem.String)
	prints = append(prints, resStr)

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
	String string `json:"string"` // String to echo back
}

func newParameters() *Parameters {
	return &Parameters{}
}
