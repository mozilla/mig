// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package results

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mozilla/mig/modules"
)

type MockPersistResults struct {
	PersistFn func(float64, []modules.Result) PersistError
}

func TestUpload(t *testing.T) {
	testCases := []struct {
		Description    string
		ShouldError    bool
		ExpectedStatus int
		RequestBody    string
		PersistFn      func(float64, []modules.Result) PersistError
	}{
		{
			Description:    `Should get status 200 if persisting succeeds`,
			ShouldError:    false,
			ExpectedStatus: http.StatusOK,
			RequestBody: `{
        "action": 12351,
        "results": [
          {
            "foundAnything": true,
            "success": true,
            "elements": {
              "data": [
                {
                  "file": "/Users/test/.ssh/unauthorized.key"
                }
              ]
            },
            "statistics": {
              "directoriesScanned": 9001,
              "findings": 1
            },
            "errors": []
          }
        ]
      }`,
			PersistFn: func(_ float64, _ []modules.Result) PersistError { return PersistErrorNil },
		},
		{
			Description:    `Should get an error if request is missing data`,
			ShouldError:    true,
			ExpectedStatus: http.StatusBadRequest,
			RequestBody: `{
        "results": []
      }`,
			PersistFn: func(_ float64, _ []modules.Result) PersistError { return PersistErrorNil },
		},
		{
			Description:    `Should get an error if persisting fails`,
			ShouldError:    true,
			ExpectedStatus: http.StatusInternalServerError,
			RequestBody: `{
        "action": 172341,
        "results": []
      }`,
			PersistFn: func(_ float64, _ []modules.Result) PersistError { return PersistErrorMediumFailure },
		},
		{
			Description:    `Should get an error if persisting fails because the action is invalid`,
			ShouldError:    true,
			ExpectedStatus: http.StatusBadRequest,
			RequestBody: `{
        "action": 12341,
        "results": []
      }`,
			PersistFn: func(_ float64, _ []modules.Result) PersistError { return PersistErrorInvalidAction },
		},
		{
			Description:    `Should get an error if the agent is determined to not be allowed to save results`,
			ShouldError:    true,
			ExpectedStatus: http.StatusUnauthorized,
			RequestBody: `{
        "action": 184234,
        "results": []
      }`,
			PersistFn: func(_ float64, _ []modules.Result) PersistError { return PersistErrorNotAuthorized },
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestUpload case #%d: %s", caseNum, testCase.Description)

		func() {
			server := httptest.NewServer(NewUpload(MockPersistResults{
				PersistFn: testCase.PersistFn,
			}))
			defer server.Close()

			response, err := http.Post(server.URL, "application/json", strings.NewReader(testCase.RequestBody))
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}

			respData := uploadResponse{}
			decoder := json.NewDecoder(response.Body)
			decodeErr := decoder.Decode(&respData)

			defer response.Body.Close()

			if decodeErr != nil {
				t.Fatalf("Error decoding response from server: %v", decodeErr)
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

func (mock MockPersistResults) PersistResults(actionID float64, results []modules.Result) PersistError {
	return mock.PersistFn(actionID, results)
}
