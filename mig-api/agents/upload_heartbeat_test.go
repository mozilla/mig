// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type MockPersistHeartbeat struct {
	PersistFn func(Heartbeat) error
}

type MockAuthenticator struct {
	AuthFn func(Heartbeat) error
}

func TestUploadHeartbeat(t *testing.T) {
	testCases := []struct {
		Description    string
		ShouldError    bool
		ExpectedStatus int
		RequestBody    string
		PersistFn      func(Heartbeat) error
		AuthFn         func(Heartbeat) error
	}{
		{
			Description:    `Should get status 200 if persisting succeeds`,
			ShouldError:    false,
			ExpectedStatus: http.StatusOK,
			RequestBody: fmt.Sprintf(`{
        "name": "name",
        "mode": "checkin",
        "version": "version",
        "pid": 3210,
        "queueLoc": "loc",
        "startTime": "%s",
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
      }`, time.Now().Format(time.RFC3339)),
			PersistFn: func(_ Heartbeat) error { return nil },
			AuthFn:    func(_ Heartbeat) error { return nil },
		},
		{
			Description:    `Should get status 400 if body is missing required data`,
			ShouldError:    true,
			ExpectedStatus: http.StatusBadRequest,
			RequestBody: fmt.Sprintf(`
      {
        "name": "name",
        "queueLoc": "loc",
        "startTime": "%s",
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
      }`, time.Now().Format(time.RFC3339)),
			PersistFn: func(_ Heartbeat) error { return nil },
			AuthFn:    func(_ Heartbeat) error { return nil },
		},
		{
			Description:    `Should get status 500 if persisting fails`,
			ShouldError:    true,
			ExpectedStatus: http.StatusInternalServerError,
			RequestBody: fmt.Sprintf(`
      {
        "name": "name",
        "mode": "checkin",
        "version": "version",
        "pid": 3210,
        "queueLoc": "loc",
        "startTime": "%s",
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
      }`, time.Now().Format(time.RFC3339)),
			PersistFn: func(_ Heartbeat) error { return errors.New("test fail") },
			AuthFn:    func(_ Heartbeat) error { return nil },
		},
		{
			Description:    `Should get status 401 if authentication fails`,
			ShouldError:    true,
			ExpectedStatus: http.StatusUnauthorized,
			RequestBody: fmt.Sprintf(`
      {
        "name": "name",
        "mode": "checkin",
        "version": "version",
        "pid": 3210,
        "queueLoc": "loc",
        "startTime": "%s",
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
      }`, time.Now().Format(time.RFC3339)),
			PersistFn: func(_ Heartbeat) error { return nil },
			AuthFn:    func(_ Heartbeat) error { return errors.New("test fail") },
		},
		{
			Description:    `Should get status 400 if an invalid mode is provided`,
			ShouldError:    true,
			ExpectedStatus: http.StatusBadRequest,
			RequestBody: fmt.Sprintf(`
      {
        "name": "name",
        "mode": "invalid",
        "version": "version",
        "pid": 3210,
        "queueLoc": "loc",
        "startTime": "%s",
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
      }`, time.Now().Format(time.RFC3339)),
			PersistFn: func(_ Heartbeat) error { return nil },
			AuthFn:    func(_ Heartbeat) error { return nil },
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestUploadHeartbeat case #%d: %s", caseNum, testCase.Description)

		func() {
			server := httptest.NewServer(NewUploadHeartbeat(
				MockPersistHeartbeat{PersistFn: testCase.PersistFn},
				MockAuthenticator{AuthFn: testCase.AuthFn}))
			defer server.Close()

			response, err := http.Post(server.URL, "application/json", strings.NewReader(testCase.RequestBody))
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}

			respData := uploadHeartbeatResponse{}
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

func (mock MockPersistHeartbeat) PersistHeartbeat(hb Heartbeat) error {
	return mock.PersistFn(hb)
}

func (mock MockAuthenticator) Authenticate(hb Heartbeat) error {
	return mock.AuthFn(hb)
}
