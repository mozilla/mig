// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const migAPIPort uint16 = 12345

type heartbeatResponse struct {
	Error *string `json:"error"`
}

func postHeartbeat(port uint16, body string) (int, heartbeatResponse, error) {
	response, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%d/api/v1/heartbeat", port),
		"application/json",
		strings.NewReader(body))
	if err != nil {
		return 0, heartbeatResponse{nil}, err
	}

	respData := heartbeatResponse{}
	decoder := json.NewDecoder(response.Body)
	decodeErr := decoder.Decode(&respData)

	response.Body.Close()

	if decodeErr != nil {
		return 0, heartbeatResponse{nil}, decodeErr
	}

	return response.StatusCode, respData, nil
}

func TestPostHeartbeatWithValidRequest(t *testing.T) {
	statusCode, response, err := postHeartbeat(migAPIPort, fmt.Sprintf(`{
    "name": "testagent123",
    "mode": "checkin",
    "version": "20181012-0.abc123.dev",
    "pid": 1234,
    "queueLoc": "testagent",
    "startTime": "%s",
    "environment": {
      "init": "systemd",
      "ident": "Ubuntu 16.04",
      "os": "linux",
      "arch": "x86_64",
      "proxied": false,
      "proxy": "",
      "addresses": ["127.0.0.1"],
      "publicIP": "127.0.0.1",
      "modules": ["memory", "file"]
    },
    "tags": [
      {
        "name": "operator",
        "value": "tester"
      }
    ]
  }`, time.Now().Format(time.RFC3339)))

	errMsg := ""
	if response.Error != nil {
		errMsg = *response.Error
	}

	if err != nil {
		t.Fatalf("Failed to make request or decode response: %s", err.Error())
	}
	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}
	if errMsg != "" {
		t.Errorf("Did not expect an error from API, got '%s'", errMsg)
	}
}
