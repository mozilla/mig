// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"errors"
)

// Module is implemented by types that contain parameters for a module
// supported by the MIG agent.  The `ToParameters` method is expected to
// validate the module configuration data and return a data type that
// can be set as the `Parameters` field in an `Action`'s
// [Operation.Parameters](https://github.com/mozilla/mig/blob/master/action.go#L84)
// field.
type Module interface {
	Name() string
	ToParameters() (interface{}, error)
	InitFromMap(map[string]interface{}) error
}

// FromMap attempts to populate a `Module` with data from a `map` containing
// configuration data for a module specified by `moduleName`.
func FromMap(moduleName string, jsonMap map[string]interface{}) (Module, error) {
	pkg := new(Pkg)
	ping := new(Ping)
	file := new(File)
	scribe := new(Scribe)
	memory := new(Memory)
	netstat := new(NetStat)
	sshkey := new(SSHKey)
	fswatch := new(FSWatch)
	audit := new(Audit)
	agentdestroy := new(AgentDestroy)
	dispatch := new(Dispatch)
	timedrift := new(TimeDrift)
	//yara := new(Yara)

	moduleContainers := map[string]Module{
		pkg.Name():          pkg,
		ping.Name():         ping,
		file.Name():         file,
		scribe.Name():       scribe,
		memory.Name():       memory,
		netstat.Name():      netstat,
		sshkey.Name():       sshkey,
		fswatch.Name():      fswatch,
		audit.Name():        audit,
		agentdestroy.Name(): agentdestroy,
		dispatch.Name():     dispatch,
		timedrift.Name():    timedrift,
		//yara.Name():    yara,
	}

	module, found := moduleContainers[moduleName]
	if !found {
		return new(InvalidModule), errors.New("not a recognized module")
	}

	err := module.InitFromMap(jsonMap)
	if err != nil {
		return new(InvalidModule), err
	}

	return module, nil
}
