// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mozilla/mig"
)

// ListActions abstracts over operations that allow the MIG API to retrieve
// a list of actions that an agent should run. It allows for a limit to be set
// on the number of actions to return.
type ListActions interface {
	ListActions(uint) ([]mig.Action, error)
}

// List is an HTTP request handler that serves GET requests intended by agents
// to retrieve a list of actions that can be run.
//
// This request handler must be able to construct a means of retrieving actions
// given a queue location string and integer limit.
type List struct {
	actions func(string) ListActions
}

type operation struct {
	Module     string      `json:"module"`
	Parameters interface{} `json:"parameters"`
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

type listRequest struct {
	Queue string `json:"queue"`
	Limit uint   `json:"limit"`
}

type listResponse struct {
	Error   *string  `json:"error"`
	Actions []action `json:"actions"`
}

// NewList constructs a new List handler.
func NewList(listActionsConstructor func(string) ListActions) List {
	return List{
		actions: listActionsConstructor,
	}
}

// validate ensures that a request to list actions contains all of the data
// required to satisfy the request.
func (req listRequest) validate() error {
	if req.Queue == "" {
		return fmt.Errorf("missing queue field")
	}

	return nil
}

// fromMigAction converts a mig.Action loaded from the database into our
// limited representation for use by the API.
func (a *action) fromMigAction(act mig.Action) {
	*a = action{
		Name:          act.Name,
		Target:        act.Target,
		ValidFrom:     act.ValidFrom,
		ExpireAfter:   act.ExpireAfter,
		Operations:    []operation{},
		Signatures:    act.PGPSignatures,
		Status:        act.Status,
		SyntaxVersion: uint(act.SyntaxVersion),
	}

	for _, op := range act.Operations {
		a.Operations = append(a.Operations, operation{
			Module:     op.Module,
			Parameters: op.Parameters,
		})
	}
}

func (handler List) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	resEncoder := json.NewEncoder(response)

	response.Header().Set("Content-Type", "application/json")

	queryStringValues := request.URL.Query()

	var paramsErr error
	queue := queryStringValues.Get("queue")
	limit, parseErr := strconv.Atoi(queryStringValues.Get("limit"))
	if queue == "" {
		paramsErr = fmt.Errorf("missing parameter 'queue'")
	}
	if parseErr != nil {
		paramsErr = parseErr
	}
	if paramsErr != nil {
		errMsg := fmt.Sprintf("Missing or invalid request parameters: %s", paramsErr.Error())
		response.WriteHeader(http.StatusBadRequest)
		resEncoder.Encode(&listResponse{&errMsg, []action{}})
		return
	}

	reqData := listRequest{
		Queue: queue,
		Limit: uint(limit),
	}

	validateErr := reqData.validate()
	if validateErr != nil {
		errMsg := fmt.Sprintf("Missing or invalid data in request: %s", validateErr.Error())
		response.WriteHeader(http.StatusBadRequest)
		resEncoder.Encode(&listResponse{&errMsg, []action{}})
		return
	}

	list := handler.actions(reqData.Queue)
	actions, err := list.ListActions(reqData.Limit)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to retrieve actions: %s", err.Error())
		response.WriteHeader(http.StatusInternalServerError)
		resEncoder.Encode(&listResponse{&errMsg, []action{}})
		return
	}

	respActions := make([]action, len(actions))
	for index, act := range actions {
		a := action{}
		a.fromMigAction(act)
		respActions[index] = a
	}
	resEncoder.Encode(&listResponse{nil, respActions})
}
