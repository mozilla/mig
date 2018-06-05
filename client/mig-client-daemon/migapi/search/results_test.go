// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package search

import (
	"net/http"
	"testing"

	"mig.ninja/mig/client/mig-client-daemon/actions"
)

func TestAPIResultAggregatorSearch(t *testing.T) {
	testCases := []struct {
		Description        string
		ActionID           actions.InternalActionID
		Handler            http.Handler
		ExpectError        bool
		ExpectedStatus     int
		NumExpectedResults uint
	}{
		{
			Description: `
We should be able to retrieve a couple of results from a single response.
			`,
			ActionID:           actions.InternalActionID(32),
			Handler:            serveResults(t),
			ExpectError:        false,
			ExpectedStatus:     http.StatusOK,
			NumExpectedResults: 2,
		},
		{
			Description: `
We should be able to retrieve a large number of results from multiple requests.
			`,
			ActionID:           actions.InternalActionID(10),
			Handler:            serveManyResults(t),
			ExpectError:        false,
			ExpectedStatus:     http.StatusOK,
			NumExpectedResults: 123,
		},
		{
			Description: `
We should get an error if one appears in a response.
			`,
			ActionID:           actions.InternalActionID(0),
			Handler:            serveError(t),
			ExpectError:        true,
			ExpectedStatus:     http.StatusInternalServerError,
			NumExpectedResults: 0,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestAPIResultAggregatorSearch case #%d.\n\t%s\n", caseNum, testCase.Description)
	}
}

func serveResults(t *testing.T) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
	})
}

func serveManyResults(t *testing.T) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
	})
}

func serveError(t *testing.T) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
	})
}
