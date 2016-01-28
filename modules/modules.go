// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

/* This package implements interfaces to write modules for MIG.
For a guide on how to write modules, head over to
http://mig.mozilla.org/doc/modules.rst.html
*/
package modules /* import "mig.ninja/mig/modules" */

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
)

// Message defines the input messages received by modules.
type Message struct {
	Class      MessageClass `json:"class"`                // represent the type of message being passed to the module
	Parameters interface{}  `json:"parameters,omitempty"` // for `parameters` class, this interface contains the module parameters
}

type MessageClass string

const (
	MsgClassParameters MessageClass = "parameters"
	MsgClassStop       MessageClass = "stop"
)

// Result implement the base type for results returned by modules.
// All modules must return this type of result. The fields are:
//
// - FoundAnything: a boolean that must be set to true if the module ran
//                  a search that returned at least one positive result
//
// - Success: a boolean that must be set to true if the module ran without
//            fatal errors. soft errors are reported in Errors
//
// - Elements: an undefined type that can be customized by the module to
//             contain the detailled results
//
// - Statistics: an undefined type that can be customized by the module to
//               contain some information about how it ran
//
// - Errors: an array of strings that contain non-fatal errors encountered
//           by the module
type Result struct {
	FoundAnything bool        `json:"foundanything"`
	Success       bool        `json:"success"`
	Elements      interface{} `json:"elements"`
	Statistics    interface{} `json:"statistics"`
	Errors        []string    `json:"errors"`
}

// Runner provides the interface to an execution of a module
type Runner interface {
	Run(io.Reader) string
	ValidateParameters() error
}

// MakeMessage creates a new modules.Message with a given class and parameters and
// return the byte slice of the json marshalled message
func MakeMessage(class MessageClass, params interface{}, comp bool) (rawMsg []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Failed to make modules.Message: %v", e)
		}
	}()

	var msg Message
	msg.Class = class
	msg.Parameters = params
	// If the compression flag is set, treat Parameters as a compressed
	// byte string.
	if comp {
		pstr, ok := msg.Parameters.(string)
		if !ok {
			panic("Compressed parameter was not a string")
		}
		b := bytes.NewBuffer([]byte(pstr))
		rb64 := base64.NewDecoder(base64.StdEncoding, b)
		r, err := gzip.NewReader(rb64)
		if err != nil {
			panic(err)
		}
		rb, err := ioutil.ReadAll(r)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(rb, &msg.Parameters)
		if err != nil {
			panic(err)
		}
	}
	rawMsg, err = json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return
}

// Keep reading until we get a full line or an error, and return
func readInputLine(rdr *bufio.Reader) ([]byte, error) {
	var ret []byte
	for {
		lb, isPrefix, err := rdr.ReadLine()
		if err != nil {
			return ret, err
		}
		ret = append(ret, lb...)
		if !isPrefix {
			break
		}
	}
	return ret, nil
}

// ReadInput reads one line of input from stdin, unmarshal it into a modules.Message
// and returns the message to the caller
func ReadInput(r io.Reader) (msg Message, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ReadInput() -> %v", e)
		}
	}()
	reader := bufio.NewReader(r)
	linebuffer, err := readInputLine(reader)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(linebuffer, &msg)
	if err != nil {
		panic(err)
	}
	return
}

// ReadInputParameters reads the first line from stdin and expects to find a
// modules.Message of class `parameters`. This function uses ReadInput and will
// block waiting for data on stdin
func ReadInputParameters(r io.Reader, p interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ReadInputParameters() -> %v", e)
		}
	}()
	msg, err := ReadInput(r)
	if err != nil {
		panic(err)
	}
	if msg.Class != MsgClassParameters {
		panic("unexpected input is not module parameters")
	}
	rawParams, err := json.Marshal(msg.Parameters)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(rawParams, p)
	if err != nil {
		panic(err)
	}
	return
}

// WatchForStop continuously reads stdin for a stop message. When one is received,
// `true` is sent into the stop channel.
func WatchForStop(r io.Reader, stopChan *chan bool) error {
	for {
		msg, err := ReadInput(r)
		if err != nil {
			return err
		}
		if msg.Class == MsgClassStop {
			*stopChan <- true
			return nil
		}
	}
}

// HasResultsPrinter implements functions used by module to print information
type HasResultsPrinter interface {
	PrintResults(Result, bool) ([]string, error)
}

// GetElements reads the elements from a struct of results into the el interface
func (r Result) GetElements(el interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetElements() -> %v", e)
		}
	}()
	buf, err := json.Marshal(r.Elements)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, el)
	if err != nil {
		panic(err)
	}
	return
}

// GetStatistics reads the statistics from a struct of results into the stats interface
func (r Result) GetStatistics(stats interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetStatistics() -> %v", e)
		}
	}()
	buf, err := json.Marshal(r.Statistics)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, stats)
	if err != nil {
		panic(err)
	}
	return
}

// HasParamsCreator implements a function that creates module parameters
type HasParamsCreator interface {
	ParamsCreator() (interface{}, error)
}

// HasParamsParser implements a function that parses command line parameters
type HasParamsParser interface {
	ParamsParser([]string) (interface{}, error)
}
