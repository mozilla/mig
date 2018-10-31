// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mozilla/mig"
)

type MockListActions struct {
	ListFn func(AgentID) ([]mig.Action, error)
}

func TestList(t *testing.T) {
	testCases := []struct {
		Description    string
		Agent          float64
		ResultsLimit   uint
		ShouldError    bool
		ExpectedStatus int
		RequestBody    string
		ListFn         func(AgentID) ([]mig.Action, error)
	}{
		{
			Description:    `Should get status 200 if retrieval succeeds`,
			ShouldError:    false,
			ExpectedStatus: http.StatusOK,
			Agent:          123.456,
			ResultsLimit:   2,
			ListFn:         func(_ AgentID) ([]mig.Action, error) { return []mig.Action{}, nil },
		},
		{
			Description:    `Should get status 400 if body is missing required data`,
			ShouldError:    true,
			ExpectedStatus: http.StatusBadRequest,
			Agent:          0.0,
			ResultsLimit:   2,
			ListFn:         func(_ AgentID) ([]mig.Action, error) { return []mig.Action{}, nil },
		},
		{
			Description:    `Should get status 500 if retrieving actions fails`,
			ShouldError:    true,
			ExpectedStatus: http.StatusInternalServerError,
			Agent:          321.0,
			ResultsLimit:   100,
			ListFn:         func(_ AgentID) ([]mig.Action, error) { return []mig.Action{}, errors.New("test fail") },
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestList case #%d: %s", caseNum, testCase.Description)

		// We run the test in a function so that calling `defer server.Close()` does the right
		// thing.  If we didn't do this, we'd just queue up multiple calls that would all be
		// invoked at once when the loop ends.
		func() {
			server := httptest.NewServer(NewList(MockListActions{
				ListFn: testCase.ListFn,
			}))
			defer server.Close()

			response, err := http.Get(fmt.Sprintf(
				"%s?agent=%f",
				server.URL,
				testCase.Agent))
			if err != nil {
				t.Fatalf("Error making request: %s", err.Error())
			}

			respData := listResponse{}
			decoder := json.NewDecoder(response.Body)
			decodeErr := decoder.Decode(&respData)

			defer response.Body.Close()

			if decodeErr != nil {
				t.Fatalf("Error decoding response from server: %s", decodeErr.Error())
			}

			if response.StatusCode != testCase.ExpectedStatus {
				t.Errorf("Expected status code %d but got %d", testCase.ExpectedStatus, response.StatusCode)
			}

			gotErr := respData.Error != nil
			if gotErr && !testCase.ShouldError {
				t.Errorf("Did not expect to get an error but got '%s'", *respData.Error)
			} else if !gotErr && testCase.ShouldError {
				t.Errorf("Expected to get an error but did not")
			}
		}()
	}
}

func (mock MockListActions) ListActions(agent AgentID) ([]mig.Action, error) {
	return mock.ListFn(agent)
}
