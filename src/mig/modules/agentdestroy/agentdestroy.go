// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// agentdestroy is a module used in the upgrade protocol to kill an agent
// that has been upgraded, and remove its binary from the file system.
// The only sanity check it does, aside from validating the parameters, is
// refusing to suicide. Meaning an agent will not run this module against itself.
package agentdestroy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

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

func NewParameters() (p Parameters) {
	return
}

type Results struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (p Parameters) Validate() (err error) {
	if p.PID < 2 || p.PID > 65535 {
		return fmt.Errorf("PID '%s' is not in the range [2:65535]", p.PID)
	}
	if p.Version == "" {
		return fmt.Errorf("parameter 'version' is empty. Expecting version.")
	}
	return
}

func Run(Args []byte) string {
	params := NewParameters()
	var results Results

	err := json.Unmarshal(Args, &params)
	if err != nil {
		panic(err)
	}

	err = params.Validate()
	if err != nil {
		panic(err)
	}
	// Refuse to suicide
	if params.PID == os.Getppid() {
		results.Error = fmt.Sprintf("PID '%d' is mine. Refusing to suicide.", params.PID)
		return buildResults(results)
	}

	// get the path of the agent's executable
	var binary string
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		binary, err = os.Readlink(fmt.Sprintf("/proc/%d/exe", params.PID))
		if err != nil {
			results.Error = fmt.Sprintf("Binary path of PID '%d' not found: '%v'", params.PID, err)
			return buildResults(results)
		}
	case "windows":
		binary = fmt.Sprintf("C:/Windows/mig-agent-%s.exe", params.Version)
	default:
		results.Error = fmt.Sprintf("'%s' isn't a supported OS", runtime.GOOS)
		return buildResults(results)
	}

	// check that the binary we're removing has the right version
	version, err := getAgentVersion(binary)
	if err != nil {
		results.Error = fmt.Sprintf("Failed to check agent version: '%v'", err)
		return buildResults(results)
	}
	if version != params.Version {
		results.Error = fmt.Sprintf("Version mismatch. Expected '%s', found '%s'", params.Version, version)
		return buildResults(results)
	}
	err = os.Remove(binary)
	if err != nil {
		results.Error = fmt.Sprintf("Failed to remove binary '%s': '%v'", binary, err)
		return buildResults(results)
	}

	// Then kill the PID
	process, err := os.FindProcess(params.PID)
	if err != nil {
		results.Error = fmt.Sprintf("PID '%d' not found. Returned error '%v'", params.PID, err)
		return buildResults(results)
	} else {
		err = process.Kill()
		if err != nil {
			results.Error = fmt.Sprintf("PID '%d' not killed. Returned error '%v'", params.PID, err)
			return buildResults(results)
		}
	}
	results.Success = true
	return buildResults(results)
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

func buildResults(results Results) (jsonResults string) {
	jsonOutput, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
