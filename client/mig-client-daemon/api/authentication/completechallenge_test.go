// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	apiAuth "mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
)

func TestCompleteChallengeHandler(t *testing.T) {
	testCases := []struct {
		Description    string
		Body           string
		ExpectError    bool
		ExpectedStatus int
	}{
		{
			Description: `
Well-formed requests should always be accepted.
			`,
			Body:           `{"challenge": "abc123", "signature": "dGVzdA=="}`,
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		{
			Description: `
Requests containing a signature not encoded as base64 should be rejected.
			`,
			Body:           `{"challenge": "abc123", "signature": ";;;"}`,
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Description: `
Requests missing fields should be rejected.
			`,
			Body:           `{"signature": "def123"}`,
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	auth := apiAuth.NewPGPAuthorizer()
	handler := NewCompleteChallengeHandler(&auth)
	router := mux.NewRouter()
	router.Handle("/v1/authentication/pgp", handler).Methods("POST")
	server := httptest.NewServer(router)

	for caseNum, testCase := range testCases {
		t.Logf("Running TestCompleteChallengeHandler case #%d\n%s\n", caseNum, testCase.Description)

		reqURL := fmt.Sprintf("%s/v1/authentication/pgp", server.URL)

		response, err := http.Post(reqURL, "application/json", strings.NewReader(testCase.Body))
		if err != nil {
			t.Fatal(err)
		}

		respData := completeChallengeResponse{}
		decoder := json.NewDecoder(response.Body)
		defer response.Body.Close()
		err = decoder.Decode(&respData)
		if err != nil {
			t.Fatal(err)
		}

		gotErr := respData.Error != nil
		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		} else if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", *respData.Error)
		}

		if testCase.ExpectedStatus != response.StatusCode {
			t.Errorf("Expected to get status code %d but got %d", testCase.ExpectedStatus, response.StatusCode)
		}
	}
}
