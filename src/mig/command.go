/* Mozilla InvestiGator

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

package mig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"
)

type Command struct {
	ID                               uint64
	Action                           Action
	AgentName, AgentQueueLoc, Status string
	Results                          interface{}
	StartTime, FinishTime            time.Time
}

// FromFile reads a command from a local file on the file system
// and return the mig.Command structure
func CmdFromFile(path string) (cmd Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.CmdFromFile()-> %v", e)
		}
	}()
	jsonCmd, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(jsonCmd, &cmd)
	if err != nil {
		panic(err)
	}
	// Syntax Check
	err = checkCmd(cmd)
	if err != nil {
		panic(err)
	}
	return
}

// CheckCmd verifies that the Command received contained all the
// necessary fields, and returns an error when it doesn't.
func checkCmd(cmd Command) error {
	if cmd.AgentName == "" {
		return errors.New("cmd.AgentName is empty. Expecting string.")
	}
	if cmd.AgentQueueLoc == "" {
		return errors.New("cmd.AgentQueueLoc is empty. Expecting string.")
	}
	if cmd.Status == "" {
		return errors.New("cmd.Status is empty. Expecting string.")
	}
	return nil
}
