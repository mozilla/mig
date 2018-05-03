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
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

func TestProvideSignatureHandler(t *testing.T) {
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
		Body           string
		ExpectError    bool
		ExpectedStatus int
	}{
		{
			Description: `
We can provide signatures for actions in the client daemon.
			`,
			ActionID:       validID,
			Body:           `{"signature": "testsignature"}`,
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		{
			Description: `
Trying to provide a signature for an action that does not exist should fail.
			`,
			ActionID:       ident.Identifier("invalid"),
			Body:           `{"signature": "testsignature"}`,
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	handler := NewProvideSignatureHandler(&catalog)
	router := mux.NewRouter()
	router.Handle("/v1/actions/{id}/sign", handler)
	server := httptest.NewServer(router)

	for caseNum, testCase := range testCases {
		t.Logf("Running TestProvideSignatureHandler case #%d.\n%s\n", caseNum, testCase.Description)

		reqURL := fmt.Sprintf("%s/v1/actions/%s/sign", server.URL, testCase.ActionID)

		client := &http.Client{}
		request, _ := http.NewRequest("PUT", reqURL, strings.NewReader(testCase.Body))
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}

		respData := provideSignatureResponse{}
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
			t.Errorf("Expected to get status %d but got %d", testCase.ExpectedStatus, response.StatusCode)
		}
	}
}
