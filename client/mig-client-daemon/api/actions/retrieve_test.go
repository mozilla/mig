// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

func TestRetrieveHandler(t *testing.T) {
	catalog := actions.NewCatalog()
	module := &modules.Pkg{
		PackageName: "*libssl*",
	}
	target := []targeting.Query{
		&targeting.ByTag{
			TagName:  "operator",
			TagValue: "IT",
		},
	}
	validID, _ := catalog.Create(module, target, time.Hour)

	testCases := []struct {
		Description    string
		ActionID       ident.Identifier
		ExpectError    bool
		ExpectedStatus int
	}{
		{
			Description: `
We should be able to retrieve actions being managed by the client daemon.
			`,
			ActionID:       validID,
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		{
			Description: `
Requesting an action that does not exist should result in an error.
			`,
			ActionID:       ident.Identifier("invalid"),
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	handler := NewRetrieveHandler(&catalog)
	router := mux.NewRouter()
	router.Handle("/v1/actions/{id}", handler).Methods("GET")
	server := httptest.NewServer(router)

	for caseNum, testCase := range testCases {
		t.Logf("Running TestRetrieveHandler case #%d.\n%s\n", caseNum, testCase.Description)

		reqURL := fmt.Sprintf("%s/v1/actions/%s", server.URL, testCase.ActionID)

		response, err := http.Get(reqURL)
		if err != nil {
			t.Fatal(err)
		}

		respData := retrieveResponse{}
		decoder := json.NewDecoder(response.Body)
		defer response.Body.Close()
		err = decoder.Decode(&respData)
		if err != nil {
			t.Fatal(err)
		}

		gotErr := respData.Error != nil
		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not")
		} else if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", *respData.Error)
		}

		if testCase.ExpectedStatus != response.StatusCode {
			t.Errorf("Expected to get status %d but got %d", testCase.ExpectedStatus, response.StatusCode)
		}
	}
}
