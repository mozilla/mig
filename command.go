// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mig.ninja/mig/modules"
	"time"
)

type Command struct {
	ID    float64 `json:"id"`
	Agent Agent   `json:"agent"`

	// The command action. If CompressedAction is non-zero, this represents
	// a GZIP compressed version of the marshaled action information, and is
	// used when sending actions between the scheduler and agents. Compressed
	// elements within the command should be manipulated using the
	// command.Compress() and command.Decompress() functions.
	Action           Action `json:"action"`
	CompressedAction []byte `json:"compressed_action,omitempty"`

	// Status can be one of:
	// sent: the command has been sent by the scheduler to the agent
	// success: the command has successfully ran on the agent and been returned to the scheduler
	// cancelled: the command has been cancelled by the investigator
	// expired: the command has been expired by the scheduler
	// failed: the command has failed on the agent and been returned to the scheduler
	// timeout: module execution has timed out, and the agent returned the command to the scheduler
	Status string `json:"status"`

	// Command results, which operate in a similar manner as described above for
	// the command action.
	Results           []modules.Result `json:"results"`
	CompressedResults [][]byte         `json:"compressed_results,omitempty"`

	StartTime  time.Time `json:"starttime"`
	FinishTime time.Time `json:"finishtime"`
}

// Decompress command information
func (cmd *Command) Decompress() error {
	err := cmd.DecompressAction()
	if err != nil {
		return err
	}
	err = cmd.DecompressResult()
	if err != nil {
		return err
	}
	return nil
}

// Given a command, decompress the action information if it is compressed
func (cmd *Command) DecompressAction() error {
	if len(cmd.CompressedAction) == 0 {
		// Action is not compressed, ignore
		return nil
	}
	r, err := gzip.NewReader(bytes.NewBuffer(cmd.CompressedAction))
	if err != nil {
		return err
	}
	defer r.Close()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &cmd.Action)
	if err != nil {
		return err
	}
	cmd.CompressedAction = nil
	return nil
}

// Given a command, decompress the result information if it is compressed
func (cmd *Command) DecompressResult() error {
	if len(cmd.CompressedResults) == 0 {
		return nil
	}
	cmd.Results = make([]modules.Result, len(cmd.CompressedResults))
	for i := range cmd.CompressedResults {
		r, err := gzip.NewReader(bytes.NewBuffer(cmd.CompressedResults[i]))
		if err != nil {
			return err
		}
		data, err := ioutil.ReadAll(r)
		if err != nil {
			r.Close()
			return err
		}
		err = json.Unmarshal(data, &cmd.Results[i])
		if err != nil {
			r.Close()
			return err
		}
		cmd.CompressedResults[i] = nil
		r.Close()
	}
	return nil
}

// Compress command information
func (cmd *Command) Compress() error {
	err := cmd.CompressAction()
	if err != nil {
		return err
	}
	err = cmd.CompressResult()
	if err != nil {
		return err
	}
	return nil
}

// Given a command, compress the action information stored in the command
func (cmd *Command) CompressAction() error {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	nb, err := json.Marshal(cmd.Action)
	if err != nil {
		return err
	}
	w.Write(nb)
	w.Close()
	cmd.CompressedAction = b.Bytes()
	cmd.Action = Action{}
	return nil
}

// Given a command, compress the result information stored in the command
func (cmd *Command) CompressResult() error {
	if len(cmd.Results) == 0 {
		return nil
	}
	cmd.CompressedResults = make([][]byte, len(cmd.Results))
	for i := range cmd.Results {
		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		nb, err := json.Marshal(cmd.Results[i])
		if err != nil {
			return err
		}
		w.Write(nb)
		w.Close()
		cmd.CompressedResults[i] = b.Bytes()
		cmd.Results[i] = modules.Result{}
	}
	return nil
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
	// Perform any required decompression on the command
	err = cmd.Decompress()
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
