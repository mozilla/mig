// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actionsAPI

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

func TestReadForSigningHandler(t *testing.T) {
	catalog := actions.NewCatalog()
	module := modules.Pkg{
		PackageName: "*libssl*",
	}
	target := []targeting.Query{
		targeting.ByTag{
			TagName:  "operator",
			TagValue: "IT",
		},
	}
	validID, _ := catalog.Create(module, target, time.Hour)

	testCases := []struct {
		Description    string
		ID             ident.Identifier
		ExpectError    bool
		ExpectedStatus int
	}{
		{
			Description: `
We can retrieve actions that are being maintained by the client daemon.
			`,
			ID:             validID,
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		{
			Description: `
We can not retrieve actions that are not being maintained by the client daemon.
			`,
			ID:             ident.Identifier("invalidid"),
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	handler := NewReadForSigningHandler(catalog)
	server := httptest.NewServer(handler)

	for caseNum, testCase := range testCases {
		t.Logf("Running TestReadForSigningHandler case #%d.\n%s\n", caseNum, testCase.Description)

		reqURL := fmt.Sprintf("%s/v1/actions/%s/signing", server.URL, testCase.ID)

		response, err := http.Get(reqURL)
		if err != nil {
			t.Fatal(err)
		}
		respData := readForSigningResponse{}
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

		if !testCase.ExpectError && !gotErr && len(respData.Action) == 0 {
			t.Errorf("Did not get an error, but also did not get an action")
		}

		if response.StatusCode != testCase.ExpectedStatus {
			t.Errorf("Expected status code %d. Got %d", testCase.ExpectedStatus, response.StatusCode)
		}
	}
}
