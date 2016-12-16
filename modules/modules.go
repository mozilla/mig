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
	"net"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

var ModuleRunDir string

// Message defines the input messages received by modules.
//
// All messages will have Class and Parameters set. PersistSock is used in a case
// where a parameters message is being sent for a persistent module. In this case,
// PersistSock will contain the socket specification the module has registered as
// listening on.
type Message struct {
	Class       MessageClass `json:"class"`                 // represent the type of message being passed to the module
	Parameters  interface{}  `json:"parameters,omitempty"`  // for `parameters` class, this interface contains the module parameters
	PersistSock string       `json:"persistsock,omitempty"` // Persistent module socket path if required
}

type MessageClass string

const (
	MsgClassParameters MessageClass = "parameters"
	MsgClassStop       MessageClass = "stop"
	MsgClassPing       MessageClass = "ping"
	MsgClassLog        MessageClass = "log"
	MsgClassRegister   MessageClass = "register"
)

// Parameter format expected for a log message
type LogParams struct {
	Message string `json:"message"`
}

// Parameter format expected for a register message
type RegParams struct {
	SockPath string `json:"sockpath"`
}

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
	Run(ModuleReader) string
	ValidateParameters() error
}

// PersistRunner provides the interface to execution of a persistent module. All
// modules will satisfy Runner. Persistent modules will satisfy both Runner and
// PersistRunner.
type PersistRunner interface {
	RunPersist(ModuleReader, ModuleWriter)
}

// ModuleReader is used to read module communications. It's intent is to
// wrap the initial reader (e.g., stdin) with a buffered reader that will exist for
// the lifetime of execution of the module. When the module reads input, it will
// read from BufferReader inside ModuleReader since our communication is line delimited,
// so we want to make sure we allocate this buffer only once.
type ModuleReader struct {
	Reader       io.Reader
	BufferReader *bufio.Reader
}

// Create a new ModuleReader wrapping reader r.
func NewModuleReader(r io.Reader) (ret ModuleReader) {
	ret.Reader = r
	ret.BufferReader = bufio.NewReader(ret.Reader)
	return
}

// ModuleWriter is used to write module communications. We don't require bufio on
// writes, but this type exists just to provide consistency with ModuleReader.
type ModuleWriter struct {
	Writer io.Writer
}

// Create a new ModuleWriter wrapping writer w.
func NewModuleWriter(w io.Writer) (ret ModuleWriter) {
	ret.Writer = w
	return
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

// MakeMessageLog creates a new message of class log.
func MakeMessageLog(f string, args ...interface{}) (rawMsg []byte, err error) {
	param := LogParams{Message: fmt.Sprintf(f, args...)}
	msg := Message{Class: MsgClassLog, Parameters: param}
	rawMsg, err = json.Marshal(&msg)
	if err != nil {
		err = fmt.Errorf("Failed to make module log message: %v", err)
		return
	}
	return
}

// Creates a new message of class register.
func MakeMessageRegister(spec string) (rawMsg []byte, err error) {
	param := RegParams{SockPath: spec}
	msg := Message{Class: MsgClassRegister, Parameters: param}
	rawMsg, err = json.Marshal(&msg)
	if err != nil {
		err = fmt.Errorf("Failed to make module register message: %v", err)
		return
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

// ReadInput reads one line of input from ModuleReader r, unmarshal it into a modules.Message
// and returns the message to the caller
func ReadInput(r ModuleReader) (msg Message, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ReadInput() -> %v", e)
		}
	}()
	linebuffer, err := readInputLine(r.BufferReader)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(linebuffer, &msg)
	if err != nil {
		panic(err)
	}
	return
}

// ReadInputParameters reads the first line from ModuleReader r and expects to find a
// modules.Message of class `parameters`. This function uses ReadInput and will
// block waiting for data on stdin
func ReadInputParameters(r ModuleReader, p interface{}) (err error) {
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

// ReadPersistInputParameters performs the same function as ReadInputParameters
// however it also validates a socket path has been specified to query the
// persistent module, returning an error if this is not present. Populates
// p and also returns the socket path.
func ReadPersistInputParameters(r ModuleReader, p interface{}) (spath string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ReadPersistInputParameters() -> %v", e)
		}
	}()
	msg, err := ReadInput(r)
	if err != nil {
		panic(err)
	}
	if msg.Class != MsgClassParameters {
		panic("unexpected input is not module parameters")
	}
	if msg.PersistSock == "" {
		panic("no persistsock set in message")
	}
	spath = msg.PersistSock
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

// Write output in buf to writer w. buf is expected to contain a single line
// of data, and a line feed is appended to terminate the line as module IO is
// line delimited.
func WriteOutput(buf []byte, w ModuleWriter) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("WriteOutput() -> %v", e)
		}
	}()
	// Since our output is line delimited, add a newline
	buf = append(buf, '\n')
	left := len(buf)
	for left > 0 {
		nb, err := w.Writer.Write(buf)
		if err != nil {
			return err
		}
		left -= nb
		buf = buf[nb:]
	}
	return
}

// WatchForStop continuously reads stdin for a stop message. When one is received,
// `true` is sent into the stop channel.
func WatchForStop(r ModuleReader, stopChan *chan bool) error {
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

// A general management function that can be called by persistent modules from the
// RunPersist function. Looks after replying to ping messages, writing logs, and other
// communication between the agent and the running persistent module.
func DefaultPersistHandlers(in ModuleReader, out ModuleWriter, logch chan string,
	errch chan error, regch chan string) {
	inChan := make(chan Message, 0)
	go func() {
		for {
			msg, err := ReadInput(in)
			if err != nil {
				close(inChan)
				break
			}
			inChan <- msg
		}
	}()
	for {
		failed := false

		select {
		case em := <-errch:
			failed = true
			// An error occurred somewhere in the persistent module and
			// we want to exit. Try to write the log message, and also
			// schedule a hard exit to ensure we do not run into a blocking
			// scenario during the write.
			go func() {
				time.Sleep(time.Second * 5)
				os.Exit(1)
			}()
			logmsg, err := MakeMessageLog("%v", em)
			if err == nil {
				WriteOutput(logmsg, out)
			}
			os.Exit(1)
		case s := <-logch:
			logmsg, err := MakeMessageLog("%v", s)
			if err != nil {
				failed = true
				break
			}
			err = WriteOutput(logmsg, out)
			if err != nil {
				failed = true
				break
			}
		case r := <-regch:
			regmsg, err := MakeMessageRegister(r)
			if err != nil {
				failed = true
				break
			}
			err = WriteOutput(regmsg, out)
			if err != nil {
				failed = true
				break
			}
		case msg, ok := <-inChan:
			if !ok {
				failed = true
				break
			}
			switch msg.Class {
			case "ping":
				buf, err := json.Marshal(&msg)
				if err != nil {
					failed = true
					break
				}
				err = WriteOutput(buf, out)
				if err != nil {
					failed = true
					break
				}
			}
		}

		if failed {
			break
		}
	}
}

// Request handler that can be called from RunPersist in a persistent module. Looks
// after accepting incoming requests to listener l, and routing the requests to the
// registered incoming request handler f.
func HandlePersistRequest(l net.Listener, f func(interface{}) string, errch chan error) {
	for {
		conn, err := l.Accept()
		if err != nil {
			errch <- err
			return
		}
		go func() {
			var p interface{}
			mw := NewModuleWriter(conn)
			mr := NewModuleReader(conn)
			err = ReadInputParameters(mr, &p)
			if err != nil {
				errch <- err
				return
			}
			resp := f(p)
			WriteOutput([]byte(resp), mw)
			err = conn.Close()
			if err != nil {
				errch <- err
				return
			}
		}()
	}
}

// Sends the parameters in p as a request to a persistent module listening at socket
// specification sockspec; would typically be used in the Run() function of a
// persistent module.
func SendPersistRequest(p interface{}, sockspec string) (res string) {
	defer func() {
		// If something goes wrong here we will want to format and
		// return the result an a Result type, with the error
		// message set.
		if e := recover(); e != nil {
			var r Result
			r.Errors = append(r.Errors, fmt.Sprintf("%v", e))
			r.Success = false
			resbuf, _ := json.Marshal(&r)
			res = string(resbuf)
		}
	}()
	var (
		fam     string
		address string
	)
	args := strings.Split(sockspec, ":")
	if len(args) < 1 || len(args) > 3 {
		panic("invalid socket specification for request")
	}
	switch args[0] {
	case "unix":
		if len(args) != 2 {
			panic("invalid socket specification for unix connection")
		}
		fam = "unix"
		address = args[1]
	case "tcp":
		if len(args) != 3 {
			panic("invalid socket specification for tcp connection")
		}
		fam = "tcp"
		address = strings.Join(args[1:], ":")
	default:
		panic("socket specification had invalid address family")
	}
	conn, err := net.Dial(fam, address)
	if err != nil {
		panic(err)
	}
	nw := NewModuleWriter(conn)
	buf, err := MakeMessage(MsgClassParameters, p, false)
	if err != nil {
		panic(err)
	}
	err = WriteOutput(buf, nw)
	if err != nil {
		panic(err)
	}
	rb, err := ioutil.ReadAll(conn)
	if err != nil {
		panic(err)
	}
	res = string(rb)
	return
}

// Get a listener for a persistent module that is appropriate for the platform type, returns
// the listener itself in addition to the socket specification that should be registered.
func GetPersistListener(modname string) (l net.Listener, specname string, err error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		sname := fmt.Sprintf("persist-%v.sock", modname)
		spath := path.Join(ModuleRunDir, sname)
		specname = "unix:" + spath
		_ = os.Remove(spath)
		l, err = net.Listen("unix", spath)
		return
	}
	err = fmt.Errorf("persistent listener not available for this platform")
	return
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

type HasEnhancedPrivacy interface {
	EnhancePrivacy(Result) (Result, error)
}

// HasParamsCreator implements a function that creates module parameters
type HasParamsCreator interface {
	ParamsCreator() (interface{}, error)
}

// HasParamsParser implements a function that parses command line parameters
type HasParamsParser interface {
	ParamsParser([]string) (interface{}, error)
}
