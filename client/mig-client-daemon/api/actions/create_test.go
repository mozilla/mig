// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"mig.ninja/mig/client/mig-client-daemon/actions"
)

func TestCreateHandler(t *testing.T) {
	testCases := []struct {
		Description    string
		Body           string
		ExpectError    bool
		ExpectedStatus int
	}{
		{
			Description: `
We should be able to have an action created provided we supply a valid module configuration.
			`,
			Body: `
{
	"module": "pkg",
	"expireAfter": 600,
	"target": [
		{
			"tagName": "operator",
			"value": "IT"
		}
	],
	"moduleConfig": {
		"packageName": "libssl"
	}
}`,
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		{
			Description: `
Action creation should fail if invalid data is supplied for a module configuration.
			`,
			Body: `
{
	"module": "pkg",
	"expireAfter": "bad",
	"target": [
		{
			"tagName": "operator",
			"value": "IT"
		}
	],
	"moduleConfig": {
		"packageName": "libssl"
	}
}`,
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Description: `
Action creation should fail if invalid targeting data is supplied.
			`,
			Body: `
{
	"module": "pkg",
	"expireAfter": "bad",
	"target": [
		{
			"invalid": "operator",
			"value": "IT"
		}
	],
	"moduleConfig": {
		"name": "libssl"
	}
}`,
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestCreateHandler case #%d.\n%s\n", caseNum, testCase.Description)

		catalog := actions.NewCatalog()
		handler := NewCreateHandler(&catalog)
		router := mux.NewRouter()
		router.Handle("/v1/actions", handler).Methods("POST")
		server := httptest.NewServer(router)

		response, err := http.Post(server.URL, "application/json", strings.NewReader(testCase.Body))
		if err != nil {
			t.Fatal(err)
		}
		respData := createResponse{}
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
		if response.StatusCode != testCase.ExpectedStatus {
			t.Errorf("Expected status code %d. Got %d", testCase.ExpectedStatus, response.StatusCode)
		}
	}
}
