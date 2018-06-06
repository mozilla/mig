// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package search

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mig.ninja/mig/client/mig-client-daemon/actions"
)

func TestAPIResultAggregatorSearch(t *testing.T) {
	testCases := []struct {
		Description        string
		ActionID           actions.InternalActionID
		Handler            http.Handler
		ExpectError        bool
		NumExpectedResults uint
	}{
		{
			Description: `
We should be able to retrieve a couple of results from a single response.
			`,
			ActionID:           actions.InternalActionID(32),
			Handler:            serveResults(t),
			ExpectError:        false,
			NumExpectedResults: 2,
		},
		{
			Description: `
We should be able to retrieve a large number of results from multiple requests.
			`,
			ActionID:           actions.InternalActionID(10),
			Handler:            serveManyResults(t),
			ExpectError:        false,
			NumExpectedResults: 125,
		},
		{
			Description: `
We should get an error if one appears in a response.
			`,
			ActionID:           actions.InternalActionID(0),
			Handler:            serveError(t),
			ExpectError:        true,
			NumExpectedResults: 0,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf(
			"Running TestAPIResultAggregatorSearch case #%d.\n\t%s\n",
			caseNum,
			testCase.Description)

		server := httptest.NewServer(testCase.Handler)
		results := NewAPIResultAggregator(server.URL)

		foundResults, err := results.Search(testCase.ActionID)

		gotErr := err != nil
		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		} else if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", err.Error())
		}

		if uint(len(foundResults)) != testCase.NumExpectedResults {
			t.Errorf(
				"Expected to get %d results, but got %d",
				testCase.NumExpectedResults,
				len(foundResults))
		}
	}
}

func serveResults(t *testing.T) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		res.Write([]byte(`{
			"collection": {
				"error": {},
				"items": [
					{
						"data": {
							"name": "command",
							"value": {}
						}
					},
					{
						"data": {
							"name": "command",
							"value": {}
						}
					}
				]
			}
		}`))
	})
}

func serveManyResults(t *testing.T) http.HandlerFunc {
	maxToSend := 125
	sent := 0

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		if sent < maxToSend {
			res.Write([]byte(`{
				"collection": {
					"error": {},
					"items": [
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } },
						{ "data": { "name": "command", "value": {} } }
					]
				}
			}`))

			sent += 25
		} else {
			res.Write([]byte(`{
				"collection": {
					"error": {},
					"items": []
				}
			}`))
		}
	})
}

func serveError(t *testing.T) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		res.Write([]byte(`{
			"collection": {
				"error": {
					"code": "123456789",
					"message": "serving an error for testing purposes"
				}
			}
		}`))
	})
}
