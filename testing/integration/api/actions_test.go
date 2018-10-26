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

type testState struct {
	AgentID       float64
	QueueLocation string
	ActionID      float64
}

type operation struct {
	Module     string                 `json:"module"`
	Parameters map[string]interface{} `json:"parameters"`
}

type TestConfig struct {
	Postgres struct {
		Host     string
		User     string
		Password string
		DBName   string
		SSLMode  string
		Port     int
		MaxConn  int
	}
}

type action struct {
	Name          string      `json:"name"`
	Target        string      `json:"target"`
	ValidFrom     time.Time   `json:"validFrom"`
	ExpireAfter   time.Time   `json:"expireAfter"`
	Operations    []operation `json:"operations"`
	Signatures    []string    `json:"signatures"`
	Status        string      `json:"status"`
	SyntaxVersion uint        `json:"syntaxVersion"`
}

type listActionsResponse struct {
	Error   *string `json:"error"`
	Actions []action
}

func listActions(port uint16, queue string, limit uint) (int, listActionsResponse, error) {
	response, err := http.Get(fmt.Sprintf(
		"http://127.0.0.1:%d/api/v1/actions?queue=%s&limit=%d",
		port,
		queue,
		limit))
	if err != nil {
		return 0, listActionsResponse{}, err
	}

	respData := listActionsResponse{}
	decoder := json.NewDecoder(response.Body)
	decodeErr := decoder.Decode(&respData)

	response.Body.Close()

	if decodeErr != nil {
		return 0, listActionsResponse{}, decodeErr
	}

	return response.StatusCode, respData, nil
}

func TestListActionsWithValidRequest(t *testing.T) {
}

func setup() (testState, error) {
	return testState{}, nil
}

func teardown(state testState) error {
	return nil
}
