// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// agentdestroy is a module used in the upgrade protocol to kill an agent
// that has been upgraded. This module will refuse to suicide, meaning that
// an agent will not run this module against itself
package agentdestroy

import (
	"encoding/json"
	"fmt"
	"github.com/kardianos/osext"
	"io"
	"mig/modules"
	"os"
	"os/exec"
	"runtime"
)

type module struct {
}

func (m *module) NewRun() interface{} {
	return new(run)
}

func init() {
	modules.Register("agentdestroy", new(module))
}

type run struct {
	Parameters Parameters
	Results    modules.Result
}

// JSON sample:
//        {
//            "module": "agentdestroy",
//            "parameters": {
//                "pid": 12345,
//                "version": "b9536d2-201403031435"
//            }
//        }
type Parameters struct {
	PID     int    `json:"pid"`
	Version string `json:"version"`
}

type results struct {
	Success bool     `json:"success"`
	Errors  []string `json:"errors,omitempty"`
}

func (r *run) ValidateParameters() (err error) {
	if r.Parameters.PID < 2 || r.Parameters.PID > 65535 {
		return fmt.Errorf("PID '%s' is not in the range [2:65535]", r.Parameters.PID)
	}
	if r.Parameters.Version == "" {
		return fmt.Errorf("parameter 'version' is empty. Expecting version.")
	}
	return
}

func (r *run) Run(in io.Reader) (out string) {
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			buf, _ := json.Marshal(r.Results)
			out = string(buf[:])
		}
	}()
	// read module parameters from stdin
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	// verify that the parameters we received are valid
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	// Refuse to suicide
	if r.Parameters.PID == os.Getppid() {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("PID '%d' is mine. Refusing to suicide.", r.Parameters.PID))
		return r.buildResults()
	}

	// get the path of the agent's executable
	var targetExecutable string
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		targetExecutable, err = os.Readlink(fmt.Sprintf("/proc/%d/exe", r.Parameters.PID))
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("Executable path of PID '%d' not found: '%v'", r.Parameters.PID, err))
			return r.buildResults()
		}
	case "windows":
		targetExecutable = fmt.Sprintf("C:\\Program Files\\mig\\mig-agent-%s.exe", r.Parameters.Version)
	default:
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("'%s' isn't a supported OS", runtime.GOOS))
		return r.buildResults()
	}
	// verify that the executable we're removing isn't in use by the current agent
	// this can happen when two agents are running of the same executable
	// in which case, do not remove the executable, and only kill the process
	myExecutable, err := osext.Executable()
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("Failed to retrieve my executable location: '%v'", err))
		return r.buildResults()
	}
	removeExecutable := true
	if myExecutable == targetExecutable {
		r.Results.Errors = append(r.Results.Errors, "Executable not removed because current agent uses it as well")
		removeExecutable = false
	}

	if removeExecutable {
		// check that the binary we're removing has the right version
		version, err := getAgentVersion(targetExecutable)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("Failed to check agent version: '%v'", err))
			return r.buildResults()
		}
		if version != r.Parameters.Version {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("Version mismatch. Expected '%s', found '%s'", r.Parameters.Version, version))
			return r.buildResults()
		}
		err = os.Remove(targetExecutable)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("Failed to remove executable '%s': '%v'", targetExecutable, err))
			return r.buildResults()
		}
	}

	// Then kill the PID
	process, err := os.FindProcess(r.Parameters.PID)
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("PID '%d' not found. Returned error '%v'", r.Parameters.PID, err))
		return r.buildResults()
	} else {
		err = process.Kill()
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("PID '%d' not killed. Returned error '%v'", r.Parameters.PID, err))
			return r.buildResults()
		}
	}
	r.Results.Success = true
	return r.buildResults()
}

// Run the agent binary to obtain its version
func getAgentVersion(binary string) (cversion string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getAgentVersion() -> %v", e)
		}
	}()
	out, err := exec.Command(binary, "-V").Output()
	if err != nil {
		panic(err)
	}
	if len(out) < 2 {
		panic("Failed to retrieve agent version.")
	}
	cversion = string(out[:len(out)-1])
	return
}

func (r *run) buildResults() (jsonResults string) {
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
