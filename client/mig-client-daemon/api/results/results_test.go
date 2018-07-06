// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package results

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
	"mig.ninja/mig/client/mig-client-daemon/migapi/search"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
	moduletypes "mig.ninja/mig/modules"
)

type mockAggregator struct{}

type mockAuthenticator struct {
	ShouldSucceed bool
}

func TestSearchResultsHandler(t *testing.T) {
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
		Aggregator     search.ResultAggregator
		Authenticator  authentication.Authenticator
		ExpectError    bool
		ExpectedStatus int
	}{
		{
			Description: `
We should be able to search for results for a valid action.
			`,
			ActionID:       validID,
			Aggregator:     mockAggregator{},
			Authenticator:  mockAuthenticator{ShouldSucceed: true},
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		{
			Description: `
We should get an error if an invalid action ID is submitted.
			`,
			ActionID:       ident.Identifier("invalid"),
			Aggregator:     mockAggregator{},
			Authenticator:  mockAuthenticator{ShouldSucceed: true},
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Description: `
We should get an error if authentication fails.
			`,
			ActionID:       validID,
			Aggregator:     mockAggregator{},
			Authenticator:  mockAuthenticator{ShouldSucceed: false},
			ExpectError:    true,
			ExpectedStatus: http.StatusInternalServerError,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestSearchResultsHandler case #%d.\n\t%s\n", caseNum, testCase.Description)

		handler := NewSearchResultsHandler(&catalog, testCase.Aggregator, testCase.Authenticator)
		router := mux.NewRouter()
		router.Handle("/v1/results", handler).Methods("GET")
		server := httptest.NewServer(router)

		reqURL := fmt.Sprintf("%s/v1/results?action=%s", server.URL, testCase.ActionID)
		response, err := http.Get(reqURL)
		if err != nil {
			t.Fatal(err)
		}

		respData := searchResultsResponse{}
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
			t.Errorf(
				"Expected to get status %d but got %d",
				testCase.ExpectedStatus,
				response.StatusCode)
		}
	}
}

func (aggregator mockAggregator) Search(
	_ actions.InternalActionID,
	auth authentication.Authenticator,
) ([]moduletypes.Result, error) {
	const numResults = 2

	req, _ := http.NewRequest("GET", "www.mozilla.org", nil)
	err := auth.Authenticate(req)
	if err != nil {
		return []moduletypes.Result{}, err
	}

	results := make([]moduletypes.Result, numResults)
	for i := 0; i < numResults; i++ {
		results[i] = moduletypes.Result{
			FoundAnything: true,
			Success:       true,
			Elements:      []interface{}{},
			Statistics:    []interface{}{},
			Errors:        []string{},
		}
	}
	return results, nil
}

func (auth mockAuthenticator) Authenticate(_ *http.Request) error {
	if auth.ShouldSucceed {
		return nil
	}
	return errors.New("Auth was instructed to fail for test")
}
