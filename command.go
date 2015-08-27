// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mig.ninja/mig/modules"
	"time"
)

type Command struct {
	ID     float64 `json:"id"`
	Action Action  `json:"action"`
	Agent  Agent   `json:"agent"`

	// Status can be one of:
	// sent: the command has been sent by the scheduler to the agent
	// success: the command has successfully ran on the agent and been returned to the scheduler
	// cancelled: the command has been cancelled by the investigator
	// expired: the command has been expired by the scheduler
	// failed: the command has failed on the agent and been returned to the scheduler
	// timeout: module execution has timed out, and the agent returned the command to the scheduler
	Status string `json:"status"`

	Results    []modules.Result `json:"results"`
	StartTime  time.Time        `json:"starttime"`
	FinishTime time.Time        `json:"finishtime"`
}

const (
	StatusSent      string = "sent"
	StatusSuccess   string = "success"
	StatusCancelled string = "cancelled"
	StatusExpired   string = "expired"
	StatusFailed    string = "failed"
	StatusTimeout   string = "timeout"
)

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
	if cmd.Agent.Name == "" {
		return errors.New("cmd.Agent.Name is empty. Expecting string.")
	}
	if cmd.Agent.QueueLoc == "" {
		return errors.New("cmd.Agent.QueueLoc is empty. Expecting string.")
	}
	if cmd.Status == "" {
		return errors.New("cmd.Status is empty. Expecting string.")
	}
	return nil
}
