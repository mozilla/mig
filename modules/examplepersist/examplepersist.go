// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package examplepersist /* import "mig.ninja/mig/modules/examplepersist" */

// This is an example persistent module that provides a basic template that
// can be used to create one.
//
// A persistent module will run continuously and be supervised by the main
// agent process. When an investigator queries the persistent module, the
// query is sent to the persistent module over the socket it has registered
// as listening on.
//
// A lot of the concepts found in standard modules apply here, the primary
// differences between a standard module and a persistent module are documented
// in this code.

import (
	"encoding/json"
	"fmt"
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

// A persistent module is still queryable by an investigator and can return results,
// we have similar results creation functions here.
func buildResults(e elements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	r.FoundAnything = true
	buf, err = json.Marshal(r)
	return
}

// The log channel can be used by functions in the persistent module to send a log
// message up to the agent, where it is logged in the agent's log.
var logChan chan string

// The error channel can be used to indicate something went wrong in the module by
// writing an error to it. This will result in the modules default handler function
// returning and the module exiting.
var handlerErrChan chan error

// When the agent sends the persistent module it's configuration, it will come in via
// the config channel as a JSON byte slice so we can unmarshal it into our configuration
var configChan chan []byte

// An example background task the module will execute while it is being supervised by
// the agent. This example just logs the current time up to the agent every 30
// seconds by default, or if a configuration file existed for the module it will use
// the interval value set there.
func runSomeTasks() {
	var cfg config

	// After the agent starts this module, it will send any module configuration
	// which we can read immediately here. The configuration will come in via
	// configChan as a JSON document, which we unmarshal into our config struct.
	incfg := <-configChan
	err := json.Unmarshal(incfg, &cfg)
	if err != nil {
		handlerErrChan <- err
		return
	}

	logChan <- "module received configuration"
	if cfg.ExamplePersist.Interval <= 0 {
		logChan <- "config interval was <= 0, defaulting to 30 seconds"
		cfg.ExamplePersist.Interval = 30
	}

	for {
		time.Sleep(time.Duration(cfg.ExamplePersist.Interval) * time.Second)
		// Send a log message up to the agent
		logChan <- fmt.Sprintf("running, current time is %v", time.Now())
	}
}

// The request handler here would essentially be what you would find in the Run
// function of a standard module. Parameters enter this routine, where they are processed
// by the module and a result string is returned.
//
// The request handler is set as part of module initialization in the RunPersist function.
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
	// Marshal and unmarshal the parameters into the type we want; p is our
	// incoming request parameters.
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

// The configuration for the persistent module; in the case of a persistent module
// that does not require configuration, our config struct would just be empty, but
// we need to define something to return so we can satisfy the PersistRunner
// interface.
//
// We need to make sure we have JSON tags here as this structure will be marshalled
// and sent to the running module by the agent.
type config struct {
	ExamplePersist struct {
		Interval int `json:"interval"`
	}
}

// PersistModConfig must be implemented in persistent modules so we can satisfy
// the PersistRunner interface. Typically here we will just return a new config
// structure that will get used to load our configuration.
func (r *run) PersistModConfig() interface{} {
	return &config{}
}

// RunPersist is the function used to initialize the persistent component
// of the module. It should not return. In this example, we do our initialization
// and call modules.DefaultPersistHandlers, which looks after handling all
// persistent module management processes on the module side.
func (r *run) RunPersist(in modules.ModuleReader, out modules.ModuleWriter) {
	// Create a string channel, used to send log messages up to the agent
	// from the module tasks. Functions in the persistent module can
	// log messages through the agent by writing to this channel.
	logChan = make(chan string, 64)
	// Create a string channel used to send registration messages up to the
	// agent. We will pass our persistent module query socket location
	// through this channel after we have initialized it, so the agent knows
	// where we are listening.
	//
	// This string will be "protocol:address", so for example it could be
	// "unix:/var/lib/mig/mysock.sock", or "tcp:127.0.0.1:55000" (as examples)
	regChan := make(chan string, 64)
	// Create an error channel we will pass to the handlers. Writing an
	// error to this channel will cause DefaultPersistHandlers() to return
	// and the module to exit.
	handlerErrChan = make(chan error, 64)
	// Create a config channel we will read our configuration from.
	configChan = make(chan []byte, 1)
	// Start up an example background task we want our module to run
	// continuously.
	go runSomeTasks()
	// Get our listener we will listen for queries from the agent on.
	l, spec, err := modules.GetPersistListener("examplepersist")
	if err != nil {
		handlerErrChan <- err
	} else {
		// We know our listener location, send it to the agent, this registers
		// us and allows queries from an investigator to make it to the module.
		regChan <- spec
	}
	// Spawn the request handler; this will route new requests to requstHandler.
	go modules.HandlePersistRequest(l, requestHandler, handlerErrChan)
	// Finally, enter the standard module management function. This will not return
	// unless an error occurs.
	modules.DefaultPersistHandlers(in, out, logChan, handlerErrChan, regChan, configChan)
}

// Module Run function, used to make queries using the module.
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

	// Read module parameters from stdin. Note we use ReadPersistInputParameters here
	// as the socket path is being sent as well, and the function needs this to know
	// where to query the persistent module.
	sockspec, err := modules.ReadPersistInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}
	// With a standard module, we'd process the request and return the results here.
	// Since this is a persistent module, we want to forward the parameters the
	// investigator provided to the module which has been running persistently. We
	// forward the request on to the listening socket and return the results.
	resStr = modules.SendPersistRequest(r.Parameters, sockspec)
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
