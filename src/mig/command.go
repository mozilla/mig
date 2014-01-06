package mig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"
)

type Command struct {
	ID uint64
	Action Action
	AgentName, AgentQueueLoc, Status string
	Results interface{}
	StartTime, FinishTime time.Time
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
