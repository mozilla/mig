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
	ListFn func(uint) ([]mig.Action, error)
}

func TestList(t *testing.T) {
	testCases := []struct {
		Description    string
		QueueName      string
		ResultsLimit   uint
		ShouldError    bool
		ExpectedStatus int
		RequestBody    string
		ListFn         func(uint) ([]mig.Action, error)
	}{
		{
			Description:    `Should get status 200 if retrieval succeeds`,
			ShouldError:    false,
			ExpectedStatus: http.StatusOK,
			QueueName:      "testqueue",
			ResultsLimit:   2,
			ListFn:         func(limit uint) ([]mig.Action, error) { return []mig.Action{}, nil },
		},
		{
			Description:    `Should get status 400 if body is missing required data`,
			ShouldError:    true,
			ExpectedStatus: http.StatusBadRequest,
			QueueName:      "",
			ResultsLimit:   2,
			ListFn:         func(limit uint) ([]mig.Action, error) { return []mig.Action{}, nil },
		},
		{
			Description:    `Should get status 500 if retrieving actions fails`,
			ShouldError:    true,
			ExpectedStatus: http.StatusInternalServerError,
			QueueName:      "test",
			ResultsLimit:   100,
			ListFn:         func(limit uint) ([]mig.Action, error) { return []mig.Action{}, errors.New("test fail") },
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestList case #%d: %s", caseNum, testCase.Description)

		func() {
			server := httptest.NewServer(NewList(func(_ string) ListActions {
				return MockListActions{
					ListFn: testCase.ListFn,
				}
			}))
			defer server.Close()

			response, err := http.Get(fmt.Sprintf(
				"%s?queue=%s&limit=%d",
				server.URL,
				testCase.QueueName,
				testCase.ResultsLimit))
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

func (mock MockListActions) ListActions(limit uint) ([]mig.Action, error) {
	return mock.ListFn(limit)
}
