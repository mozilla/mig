// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkg

import (
	"encoding/json"
	"fmt"
	"mig/testutil"
	"testing"
)

type testParams struct {
	expect bool
	params string
}

func TestRegistration(t *testing.T) {
	testutil.CheckModuleRegistration(t, "pkg")
}

func TestParameters(t *testing.T) {
	var r Runner

	validPayload := "H4sIAPiTeFUAA7PJL0vMiU9JTcvMyyzJzM8rVqjIzckrtlXKKCkpsNLXB0nr5WaWFKXq5Rel60f4+gQnZ6TmJoIldJH06Zoq2XHZ6KMbZ8cFAGdvQOxiAAAA"

	var parameters = []testParams{
		{false, `{}`},
		{true, `{"pkgmatch":{"matches":["test"]}}`},
		{false, `{"pkgmatch":{"matches":["))))))"]}}`},
		{false, `{"pkgmatch":{"matches":[]}}`},
		{false, `{"pkgmatch":{"matches":["test"]},"ovaldef":"XXXX"}`},
		{true, fmt.Sprintf(`{"ovaldef":"%v"}`, validPayload)},
		{false, `{"ovaldef":"XXXX"}`},
		{true, fmt.Sprintf(`{"ovaldef":"%v","maxconneval":2}`, validPayload)},
		{false, fmt.Sprintf(`{"ovaldef":"%v","maxconneval":50}`, validPayload)},
	}

	for _, x := range parameters {
		r.Parameters = *newParameters()
		err := json.Unmarshal([]byte(x.params), &r.Parameters)
		if err != nil && x.expect {
			t.Fatal(err)
		}
		err = r.ValidateParameters()
		if err == nil && !x.expect {
			t.Fatalf("invalid parameters '%s' considered valid", x.params)
		} else if err != nil && x.expect {
			t.Fatalf("valid parameters '%s' considered invalid: %v", x.params, err)
		}
	}
}
