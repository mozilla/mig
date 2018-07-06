// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package search

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
)

type mockAuthenticator struct {
	ShouldSucceed bool
}

func TestAPIResultAggregatorSearch(t *testing.T) {
	testCases := []struct {
		Description        string
		ActionID           actions.InternalActionID
		Authenticator      authentication.Authenticator
		Handler            http.Handler
		ExpectError        bool
		NumExpectedResults uint
	}{
		{
			Description: `
We should be able to retrieve a couple of results from a single response.
			`,
			ActionID:           actions.InternalActionID(32),
			Authenticator:      mockAuthenticator{ShouldSucceed: true},
			Handler:            serveResults(),
			ExpectError:        false,
			NumExpectedResults: 2,
		},
		{
			Description: `
We should be able to retrieve a large number of results from multiple requests.
			`,
			ActionID:           actions.InternalActionID(10),
			Authenticator:      mockAuthenticator{ShouldSucceed: true},
			Handler:            serveManyResults(t),
			ExpectError:        false,
			NumExpectedResults: 125,
		},
		{
			Description: `
We should get an error if authentication fails.
			`,
			ActionID:           actions.InternalActionID(32),
			Authenticator:      mockAuthenticator{ShouldSucceed: false},
			Handler:            serveResults(),
			ExpectError:        true,
			NumExpectedResults: 0,
		},
		{
			Description: `
We should get an error if one appears in a response.
			`,
			ActionID:           actions.InternalActionID(0),
			Authenticator:      mockAuthenticator{ShouldSucceed: true},
			Handler:            http.HandlerFunc(serveError),
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

		foundResults, err := results.Search(testCase.ActionID, testCase.Authenticator)

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

func serveResults() http.Handler {
	hasBeenCalled := false

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if hasBeenCalled {
			res.Write([]byte(`{
				"collection": {
					"error": {
						"code": "0123123",
						"message": "no results found"
					}
				}
			}`))
		} else {
			hasBeenCalled = true
		}

		res.Header().Set("Content-Type", "application/json")
		res.Write([]byte(`{
			"collection": {
				"error": {},
				"items": [
					{
						"data": [
							{
								"name": "command",
								"value": {
									"results": [
										{
											"foundanything": true,
											"success": true,
											"elements": [],
											"statistics": [],
											"errors": []
										},
										{
											"foundanything": true,
											"success": true,
											"elements": [],
											"statistics": [],
											"errors": []
										}
									]
								}
							}
						]
					}
				]
			}
		}`))
	})
}

func serveManyResults(t *testing.T) http.HandlerFunc {
	const maxToSend = 125
	const sendEachRequest = 25

	sent := 0

	itemStr := `
			{
				"data": [
					{
						"name": "command",
						"value": {
							"results": [
								{
									"foundanything": true,
									"success": true,
									"elements": [],
									"statistics": [],
									"errors": []
								}
							]
						}
					}
				]
			}`
	items := make([]string, sendEachRequest)
	for i := 0; i < sendEachRequest; i++ {
		items[i] = itemStr
	}
	itemsJSON := strings.Join(items, ",\n")

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		if sent < maxToSend {
			responseStr := fmt.Sprintf(`
{
	"collection": {
		"error": {},
		"items": [
			%s
		]
	}
}`, itemsJSON)
			res.Write([]byte(responseStr))

			sent += 25
		} else {
			res.Write([]byte(`
{
	"collection": {
		"error": {
			"code": "0123123",
			"message": "no results found"
		}
	}
}`))
		}
	})
}

func serveError(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	res.Write([]byte(`{
		"collection": {
			"error": {
				"code": "123456789",
				"message": "serving an error for testing purposes"
			}
		}
	}`))
}

func (auth mockAuthenticator) Authenticate(_ *http.Request) error {
	if auth.ShouldSucceed {
		return nil
	}
	return errors.New("Auth was instructed to fail for test")
}
