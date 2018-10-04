// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockPersistHeartbeat struct {
	PersistFn func(Heartbeat) error
}

func TestUploadHeartbeat(t *testing.T) {
	testCases := []struct {
		Description    string
		ShouldError    bool
		ExpectedStatus int
		RequestBody    string
		PersistFn      func(Heartbeat) error
	}{
		{
			Description:    `Should get status 200 if persisting succeeds`,
			ShouldError:    false,
			ExpectedStatus: http.StatusOK,
			RequestBody: `{
        "name": "name",
        "mode": "mode",
        "version": "version",
        "pid": 3210,
        "queueLoc": "loc",
        "startTime": "2018-10-03T20:45:00Z",
        "environment": {
          "init": "init",
          "ident": "ident",
          "os": "os",
          "arch": "arch",
          "isProxied": false,
          "proxy": "",
          "addresses": [],
          "publicIP": "publicip",
          "modules": []
        },
        "tags": []
      }`,
			PersistFn: func(_ Heartbeat) error { return nil },
		},
		{
			Description:    `Should get status 400 if body is missing required data`,
			ShouldError:    true,
			ExpectedStatus: http.StatusBadRequest,
			RequestBody: `
      {
        "name": "name",
        "queueLoc": "loc",
        "startTime": "2018-10-03T20:45:00Z",
        "environment": {
          "init": "init",
          "ident": "ident",
          "os": "os",
          "arch": "arch",
          "isProxied": false,
          "proxy": "",
          "addresses": [],
          "publicIP": "publicip",
          "modules": []
        },
        "tags": []
      }
      `,
			PersistFn: func(_ Heartbeat) error { return nil },
		},
		{
			Description:    `Should get status 500 if persisting fails`,
			ShouldError:    true,
			ExpectedStatus: http.StatusInternalServerError,
			RequestBody: `
      {
        "name": "name",
        "mode": "mode",
        "version": "version",
        "pid": 3210,
        "queueLoc": "loc",
        "startTime": "2018-10-03T20:45:00Z",
        "environment": {
          "init": "init",
          "ident": "ident",
          "os": "os",
          "arch": "arch",
          "isProxied": false,
          "proxy": "",
          "addresses": [],
          "publicIP": "publicip",
          "modules": []
        },
        "tags": []
      }
      `,
			PersistFn: func(_ Heartbeat) error { return errors.New("test fail") },
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestUploadHeartbeat case #%d: %s", caseNum, testCase.Description)

		server := httptest.NewServer(NewUploadHeartbeat(MockPersistHeartbeat{
			PersistFn: testCase.PersistFn,
		}))
		defer server.Close()

		response, err := http.Post(server.URL, "application/json", strings.NewReader(testCase.RequestBody))
		if err != nil {
			t.Fatal(err)
		} else {
			defer response.Body.Close()
		}

		respData := uploadHeartbeatResponse{}
		decoder := json.NewDecoder(response.Body)
		decodeErr := decoder.Decode(&respData)
		if decodeErr != nil {
			t.Fatal(decodeErr)
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
	}
}

func (mock MockPersistHeartbeat) PersistHeartbeat(hb Heartbeat) error {
	return mock.PersistFn(hb)
}
