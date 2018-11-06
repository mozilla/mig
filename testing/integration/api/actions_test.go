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
	"net/http"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"gopkg.in/gcfg.v1"

	"github.com/mozilla/mig"
	migdb "github.com/mozilla/mig/database"
)

const testActionName string = "testaction"

type config struct {
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

type testState struct {
	AgentID  float64
	ActionID float64
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

func listActions(port uint16, agent float64) (int, listActionsResponse, error) {
	response, err := http.Get(fmt.Sprintf(
		"http://127.0.0.1:%d/api/v1/actions?agent=%f",
		port,
		agent))
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
	cfg := config{}
	path, err := filepath.Abs("../../api.cfg")
	if err != nil {
		t.Fatal(err)
	}
	if err := gcfg.FatalOnly(gcfg.ReadFileInto(&cfg, path)); err != nil {
		t.Fatal(err)
	}
	url := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode)
	db, err := sql.Open("postgres", url)
	migDB := migdb.NewDB(db)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	state, err := setup(migDB)
	if err != nil {
		t.Fatal(err)
	}
	defer teardown(migDB, state)

	t.Logf("Finished setup and got action %f ; agent %f", state.ActionID, state.AgentID)

	statusCode, respData, err := listActions(migAPIPort, state.AgentID)
	if err != nil {
		t.Fatal(err)
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected to get status code %d but got %d", http.StatusOK, statusCode)
	}

	if respData.Error != nil {
		t.Errorf("Got error from API: %s", *respData.Error)
	}

	if len(respData.Actions) != 1 {
		t.Fatalf("Expected to get one action, but got %d", len(respData.Actions))
	}

	if respData.Actions[0].Name != testActionName {
		t.Errorf("Expected action retrieved to have name '%s' but it is '%s'", testActionName, respData.Actions[0].Name)
	}
}

func setup(db migdb.DB) (testState, error) {
	testAgent := mig.Agent{
		ID:       mig.GenID(),
		Name:     "listactionstestagent",
		QueueLoc: "doesntmatter",
		Mode:     "daemon",
	}

	testAction := mig.Action{
		ID:          mig.GenID(),
		Name:        testActionName,
		Target:      "name = 'listactionstestagent'",
		ValidFrom:   time.Now(),
		ExpireAfter: time.Now().Add(5 * time.Minute),
		Status:      "pending",
	}

	err := db.InsertAgent(testAgent, nil)
	if err != nil {
		return testState{}, err
	}

	err = db.InsertAction(testAction)
	if err != nil {
		return testState{}, err
	}

	state := testState{
		AgentID:  testAgent.ID,
		ActionID: testAction.ID,
	}
	return state, nil
}

func teardown(db migdb.DB, state testState) error {
	return nil
}
