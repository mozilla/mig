// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package modules /* import "mig.ninja/mig/modules" */

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type testModule struct {
}

func (m *testModule) NewRun() Runner {
	return new(testRunner)
}

type testRunner struct {
	Parameters params
	Results    Result
}

func (r *testRunner) ValidateParameters() (err error) {
	return nil
}

func (r *testRunner) Run(in io.Reader) (out string) {
	return ""
}

type params struct {
	SomeParam string `json:"someparam"`
}

func TestRegister(t *testing.T) {
	// test simple registration
	Register("testing", new(testModule))
	if _, ok := Available["testing"]; !ok {
		t.Fatalf("testing module registration failed")
	}
	// test availability of unregistered module
	if _, ok := Available["shouldnotberegistered"]; ok {
		t.Fatalf("testing module availability failed")
	}
	// test registration of already registered module
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("failed to panic on double registration of testing module")
		}
	}()
	Register("testing", new(testModule))
}

func TestMakeMessage(t *testing.T) {
	var p params
	p.SomeParam = "foo"
	raw, err := MakeMessage(MsgClassParameters, p, false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if string(raw) != `{"class":"parameters","parameters":{"someparam":"foo"}}` {
		t.Fatalf("Invalid module message class `parameters`")
	}

	// Test parameter decompression
	jb, err := json.Marshal(p)
	if err != nil {
		t.Fatalf(err.Error())
	}
	var b bytes.Buffer
	wb64 := base64.NewEncoder(base64.StdEncoding, &b)
	w := gzip.NewWriter(wb64)
	_, err = w.Write(jb)
	if err != nil {
		t.Fatalf(err.Error())
	}
	w.Close()
	wb64.Close()
	raw, err = MakeMessage(MsgClassParameters, string(b.Bytes()), true)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if string(raw) != `{"class":"parameters","parameters":{"someparam":"foo"}}` {
		t.Fatalf("Invalid module message class `parameters`")
	}

	raw, err = MakeMessage(MsgClassStop, nil, false)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if string(raw) != `{"class":"stop"}` {
		t.Fatalf("Invalid module message class `stop`")
	}
}

type element struct {
	SomeElement string `json:"someelement"`
}

func TestGetElements(t *testing.T) {
	var r Result
	r.Elements = struct {
		SomeElement string `json:"someelement"`
	}{
		SomeElement: "foo",
	}
	var el element
	err := r.GetElements(&el)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if el.SomeElement != "foo" {
		t.Fatalf("failed to get element from module results")
	}

}

type statistics struct {
	SomeCounter float64 `json:"somecounter"`
}

func TestGetStatistics(t *testing.T) {
	var r Result
	r.Statistics = struct {
		SomeCounter float64 `json:"somecounter"`
	}{
		SomeCounter: 16.64,
	}
	var stats statistics
	err := r.GetStatistics(&stats)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if stats.SomeCounter != 16.64 {
		t.Fatalf("failed to get statistics from module results")
	}
}

func TestReadInputParameters(t *testing.T) {
	var p params
	w := strings.NewReader(`{"class":"parameters","parameters":{"someparam":"foo"}}`)
	err := ReadInputParameters(w, &p)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if p.SomeParam != "foo" {
		t.Fatalf("failed to read input parameters from stdin")
	}
	// test delayed write. use a pipe so that reader doesn't reach EOF on the first
	// read of the empty buffer.
	r2, w2, err := os.Pipe()
	block := make(chan bool)
	go func() {
		err = ReadInputParameters(r2, &p)
		block <- true
	}()
	time.Sleep(100 * time.Millisecond)
	w2.WriteString(`{"class":"parameters","parameters":{"someparam":"bar"}}`)
	w2.Close() // close the pipe to trigger EOF on the reader
	select {
	case <-block:
	case <-time.After(2 * time.Second):
		t.Fatalf("input parameters read timed out")
	}
	if err != nil {
		t.Fatalf(err.Error())
	}
	if p.SomeParam != "bar" {
		t.Fatalf("failed to read input parameters")
	}
}

func TestWatchForStop(t *testing.T) {
	stopChan := make(chan bool)
	w := strings.NewReader(`{"class":"stop"}`)
	var err error
	go func() {
		err = WatchForStop(w, &stopChan)
	}()
	select {
	case <-stopChan:
		break
	case <-time.After(1 * time.Second):
		t.Fatalf("failed to catch stop message")
	}
}
