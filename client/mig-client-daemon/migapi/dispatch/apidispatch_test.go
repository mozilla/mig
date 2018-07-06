// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package dispatch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

const testPGPAuthHeader string = "testing"

type mockAuthenticator struct{}

func TestAPIDispatcherDispatch(t *testing.T) {
	testCases := []struct {
		Description string
		Handler     http.Handler
		ExpectError bool
	}{
		{
			Description: `
A POST request should be sent to the MIG API's action creation endpoint, with
authentication data added to the request
			`,
			Handler:     testVerifyFormatHandler(t),
			ExpectError: false,
		},
		{
			Description: `
An error should be returned if the MIG API responds with a status code other
than status 202
			`,
			Handler:     http.HandlerFunc(testRejectHandler),
			ExpectError: true,
		},
	}

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
	record, _ := catalog.Lookup(validID)

	for caseNum, testCase := range testCases {
		t.Logf("Running TestAPIDispatcherDispatch case #%d\n%s\n", caseNum, testCase.Description)

		server := httptest.NewServer(testCase.Handler)
		dispatcher := NewAPIDispatcher(server.URL)
		authenticator := mockAuthenticator{}

		_, err := dispatcher.Dispatch(record.Action, authenticator)
		gotErr := err != nil

		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not")
		} else if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", err.Error())
		}
	}
}

func testVerifyFormatHandler(t *testing.T) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		errorJSON := "{}"
		statusCode := http.StatusAccepted

		body, _ := ioutil.ReadAll(req.Body)
		bodyStr := string(body)
		if !strings.HasPrefix(bodyStr, "action=") {
			t.Logf("Incorreclty formatted body")

			errorJSON = `{"code": "123456789", "message": "Improperly-formatted request body. Expected 'action=' prefix."}`
			statusCode = http.StatusBadRequest
		}
		encodedAction, err := url.QueryUnescape(bodyStr[len("action="):])
		if err != nil {
			t.Logf("Failed to decode action")

			errorJSON = `{"code": "123456789", "message": "could not decode action using QueryUnescape"}`
			statusCode = http.StatusBadRequest
		}
		reqData := mig.Action{}
		err = json.Unmarshal([]byte(encodedAction), &reqData)

		if reqData.Name == "" {
			t.Logf("Did not decode an action")

			errorJSON = `{"code": "123456789", "message": "Did not decode an action"}`
			statusCode = http.StatusBadRequest
		} else if err != nil {
			t.Logf("Failed to decode request body")

			errorJSON = `{"code": "123456789", "message": "invalid request body; invalid JSON"}`
			statusCode = http.StatusBadRequest
		} else if req.Header.Get("X-PGPAUTHORIZATION") != testPGPAuthHeader {
			t.Logf("Missing X-PGPAUTHENTICATION header")

			errorJSON = `{"code": "123456789", "message": "missing or invalid auth header"}`
			statusCode = http.StatusForbidden
		} else if req.Method != "POST" {
			t.Logf("Incorrect method")

			errorJSON = `{"code": "123456789", "message": "not a POST request"}`
			statusCode = http.StatusBadRequest
		}

		resBody := ""
		if errorJSON != "{}" {
			resBody = fmt.Sprintf(`{
				"collection": {
					"error": %s
				}
			}`, errorJSON)
		} else {
			// Success case where we want to mimick the MIG API.
			resBody = `{
				"collection": {
					"error": {},
					"items": [
						{
							"data": [
								{
									"name": "action ID 123456789",
									"value": {
										"id": 123456789
									}
								}
							]
						}
					]
				}
			}`
		}
		res.WriteHeader(statusCode)
		res.Write([]byte(resBody))
	}
}

func testRejectHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusForbidden)
	res.Write([]byte(`
		{
			"collection": {
				"error": {
					"code": "6077873045059431424",
					"message": "rejected"
				}
			}
		}
	`))
}

func (auth mockAuthenticator) Authenticate(req *http.Request) error {
	req.Header.Set("X-PGPAUTHORIZATION", testPGPAuthHeader)
	return nil
}
