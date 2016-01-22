// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package memory /* import "mig.ninja/mig/modules/memory" */

import (
	"bytes"
	"encoding/json"
	"mig.ninja/mig/modules"
	"mig.ninja/mig/testutil"
	"testing"
)

func TestRegistration(t *testing.T) {
	testutil.CheckModuleRegistration(t, "memory")
}

type testParams struct {
	expect bool
	params string
}

func TestParameters(t *testing.T) {
	var (
		r   run
		err error
	)
	var parameters = []testParams{
		{true, `{"searches":{"s1":{"names":["foo"],"libraries":["bar"],"bytes":["abcd"]}}}`},
		{false, `{"searches":{"*&^!*@&#^*!":{"names":["foo"]}}}`},
		{false, `{"searches":{"":{"names":["foo"]}}}`},
		{false, `{"searches":{"s1":{"names":["["]}}}`},
		{false, `{"searches":{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":{"names":["foo"]}}}`},
		{true, `{"searches":{"s1":{"libraries":["^[a-z]{10,50}$"]}}}`},
		{false, `{"searches":{"s1":{"libraries":["["]}}}`},
		{true, `{"searches":{"s1":{"bytes":["dead","beef"]}}}`},
		{false, `{"searches":{"s1":{"bytes":["zzzzzzzz"]}}}`},
		{true, `{"searches":{"s1":{"contents":["^(.+)[a-zA-Z0-9]{10.50}$"]}}}`},
		{false, `{"searches":{"s1":{"contents":["^$", "["]}}}`},
	}
	for _, tp := range parameters {
		r.Parameters = *newParameters()
		err = json.Unmarshal([]byte(tp.params), &r.Parameters)
		if err != nil && tp.expect {
			t.Fatal(err)
		}
		err = r.ValidateParameters()
		if err == nil && !tp.expect {
			t.Fatalf("invalid parameters '%s' considered valid", tp.params)
		} else if err != nil && tp.expect {
			t.Fatalf("valid parameters '%s' considered invalid: %v", tp.params, err)
		}
	}
}

func TestFindGoTestProcess(t *testing.T) {
	var (
		r run
		s search
	)
	r.Parameters = *newParameters()
	s.Names = append(s.Names, "go-build")
	s.Bytes = append(s.Bytes, "7465737420736561726368206c6f6f6b696e6720666f722073656c66")
	s.Contents = append(s.Contents, "test search looking for self")
	s.Options.MatchAll = true
	s.Options.Offset = 0.0
	s.Options.MaxLength = 10000000
	s.Options.LogFailures = true
	r.Parameters.Searches["testsearch"] = s
	msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters, false)
	if err != nil {
		t.Fatal(err)
	}
	out := r.Run(bytes.NewBuffer(msg))
	if len(out) == 0 {
		t.Fatal("run failed")
	}
	t.Log(out)
	err = json.Unmarshal([]byte(out), &r.Results)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Results.Success {
		t.Fatal("failed to run memory search")
	}
	if !r.Results.FoundAnything {
		t.Fatal("should have found own go test process but didn't")
	}
	prints, err := r.PrintResults(r.Results, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(prints) < 2 {
		t.Fatal("not enough results printed")
	}
	prints, err = r.PrintResults(r.Results, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(prints) != 1 {
		t.Fatal("wrong number of results, should be one")
	}
}

func TestSearches(t *testing.T) {
	var parameters = []testParams{
		{true, `{"searches":{"s1":{"names":["go"]}}}`},
		{false, `{"searches":{"s1":{"libraries":["^caribou.so$"]}}}`},
		{true, `{"searches":{"s1":{"contents":["memory_test"], "names": ["go"]}}}`},
		{false, `{"searches":{"s1":{"names":["1983yrotewdshhhoiufhes7fd29"],"bytes":["ffffffffaaaabbbbcccceeee"],"options":{"matchall": true}}}}`},
	}
	for _, tp := range parameters {
		var r run
		r.Parameters = *newParameters()
		err := json.Unmarshal([]byte(tp.params), &r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters, false)
		if err != nil {
			t.Fatal(err)
		}
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = json.Unmarshal([]byte(out), &r.Results)
		if err != nil {
			t.Fatal(err)
		}
		if !r.Results.Success {
			t.Fatal("failed to run memory search")
		}
		if r.Results.FoundAnything && !tp.expect {
			t.Fatalf("found something for search '%s' and shouldn't have", tp.params)
		} else if !r.Results.FoundAnything && tp.expect {
			t.Fatalf("found nothing for search '%s' and should have", tp.params)
		}
	}
}
