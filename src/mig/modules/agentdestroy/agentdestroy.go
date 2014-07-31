/* Kill an agent and remove its binary

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

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
