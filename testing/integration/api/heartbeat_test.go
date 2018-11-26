// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"gopkg.in/gcfg.v1"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
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

type Config struct {
	Postgres struct {
		Host, User, Password, DBName, SSLMode string
		Port, MaxConn                         int
	}
}

func TestPostHeartbeatWithValidRequest(t *testing.T) {
	agentName := "testagent123"
	statusCode, response, err := postHeartbeat(migAPIPort, fmt.Sprintf(`{
    "name": "%s",
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
  }`, agentName, time.Now().Format(time.RFC3339)))

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

	path, err := filepath.Abs("../../api.cfg")
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{}
	err = gcfg.FatalOnly(gcfg.ReadFileInto(&cfg, path))
	if err != nil {
		t.Fatal(err)
	}

	user := cfg.Postgres.User
	password := cfg.Postgres.Password
	host := cfg.Postgres.Host
	port := cfg.Postgres.Port
	dbname := cfg.Postgres.DBName
	sslmode := cfg.Postgres.SSLMode

	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
	db_conn, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("Failed to connect to postgresl DB: %s", err.Error())
	}

	defer db_conn.Close()

	rows, err := db_conn.Query("SELECT name FROM agents WHERE mode='checkin'")
	if err != nil {
		t.Fatalf("Received error when querying for heartbeats: %v", err)
	}

	if rows != nil {
		defer rows.Close()
	}

	numRows := 0
	for rows.Next() {
		numRows++
		var name string
		err = rows.Scan(&name)
		if err != nil {
			err = fmt.Errorf("Error while parsing name: '%v'", err)
			return
		}
		if name != agentName {
			t.Fatalf("Expected heartbeat with agent name: %s got %s", agentName, name)
		}
	}
	if numRows < 1 {
		t.Fatalf("Expected at least 1 row")
	}
}
