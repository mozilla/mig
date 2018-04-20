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
)

func TestCreateHandler(t *testing.T) {
	testCases := []struct {
		Body           string
		ExpectError    bool
		ExpectedStatus int
	}{
		// Well-formed body containing valid data.
		{
			Body:           "{\"module\": \"pkg\", \"expireAfter\": 600, \"target\": \"status='online'\", \"moduleConfig\": {\"name\": \"libssl\"}}",
			ExpectError:    false,
			ExpectedStatus: http.StatusOK,
		},
		// Invalid body. `expireAfter` must be a number.
		{
			Body:           "{\"module\": \"pkg\", \"expireAfter\": \"bad\", \"target\": \"status='online'\", \"moduleConfig\": {\"name\": \"libssl\"}}",
			ExpectError:    true,
			ExpectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range testCases {
		handler := NewCreateHandler()
		server := httptest.NewServer(handler)

		response, err := http.Post(server.URL, "application/json", strings.NewReader(test.Body))
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
		if test.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		} else if !test.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", *respData.Error)
		}
		if response.StatusCode != test.ExpectedStatus {
			t.Errorf("Expected status code %d. Got %d", test.ExpectedStatus, response.StatusCode)
		}
	}
}
