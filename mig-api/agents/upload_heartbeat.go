// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

import (
	"encoding/json"
	"net/http"
)

// PersistHeartbeat abstracts over operations that allow the MIG API to
// save some information about agents sent in a heartbeat message.
type PersistHeartbeat interface {
	PersistHeartbeat(Heartbeat) error
}

// UploadHeartbeat is an HTTP request handler that serves POST requests
// containing a Heartbeat encoded as JSON.
type UploadHeartbeat struct {
	persist *PersistHeartbeat
}

// NewUploadHeartbeat constructs a new UploadHeartbeat.
func NewUploadHeartbeat(persist *PersistHeartbeat) UploadHeartbeat {
	return UploadHeartbeat{
		persist: persist,
	}
}

// Environment contains information about the environment an agent is running in.
type Environment struct {
	Init      string   `json:"init"`
	Ident     string   `json:"ident"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	IsProxied bool     `json:"isPrpxied"`
	Proxy     string   `json:"proxy"`
	Addresses []string `json:"addresses"`
	PublicIP  string   `json:"publicIP"`
	Modules   []string `json:"modules"`
}

// Tag is a label associated with an agent.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Heartbeat contains information describing an active agent.
type Heartbeat struct {
	Name        string      `json:"name"`
	Mode        string      `json:"mode"`
	Version     string      `json:"version"`
	PID         uint        `json:"pid"`
	QueueLoc    string      `json:"queueLoc"`
	StartTime   time.Time   `json:"startTime"`
	Environment environment `json:"environment"`
	Tags        []tag       `json:"tags"`
}

type uploadHeartbeatResponse struct {
	Error *string `json:"error"`
}

func (handler UploadHeartbeat) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	reqData := Heartbeat{}
	decoder := json.NewDecoder(request.Body)
	resEncoder := json.NewEncoder(response)

	response.Header().Set("Content-Type", "application/json")

	defer request.Body.Close()

	decodeErr := decoder.Decode(&reqData)
	if decodeErr != nil {
		errMsg := fmt.Sprintf("Failed to decode request body: %s", decodeErr.Error())
		response.WriteHeader(http.StatusBadRequest)
		resEncoder.Encode(&uploadHeartbeatResponse{&errMsg})
		return
	}

	persistErr := handler.persist.PersistHeartbeat(reqData)
	if persistErr != nil {
		errMsg := fmt.Sprintf("Failed to save heartbeat: %s", persistErr.Error())
		response.WriteHeader(http.StatusInternalServerError)
		resEncoder.Encode(&uploadHeartbeatResponse{&errMsg})
		return
	}
}
