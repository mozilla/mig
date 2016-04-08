// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import "time"

const (
	AgtStatusOnline    string = "online"
	AgtStatusUpgraded  string = "upgraded"
	AgtStatusDestroyed string = "destroyed"
	AgtStatusOffline   string = "offline"
	AgtStatusIdle      string = "idle"
)

// Agent stores the description of an agent and serves as a canvas
// for heartbeat messages
type Agent struct {
	ID              float64     `json:"id,omitempty"`
	Name            string      `json:"name"`
	QueueLoc        string      `json:"queueloc"`
	Mode            string      `json:"mode"`
	Version         string      `json:"version,omitempty"`
	PID             int         `json:"pid,omitempty"`
	StartTime       time.Time   `json:"starttime,omitempty"`
	DestructionTime time.Time   `json:"destructiontime,omitempty"`
	HeartBeatTS     time.Time   `json:"heartbeatts,omitempty"`
	RefreshTS       time.Time   `json:"refreshts,omitempty"`
	Status          string      `json:"status,omitempty"`
	Authorized      bool        `json:"authorized,omitempty"`
	Env             AgentEnv    `json:"environment,omitempty"`
	Tags            interface{} `json:"tags,omitempty"`
}

// AgentEnv stores basic information of the endpoint
type AgentEnv struct {
	Init      string      `json:"init,omitempty"`
	Ident     string      `json:"ident,omitempty"`
	OS        string      `json:"os,omitempty"`
	Arch      string      `json:"arch,omitempty"`
	IsProxied bool        `json:"isproxied"`
	Proxy     string      `json:"proxy,omitempty"`
	Addresses []string    `json:"addresses,omitempty"`
	PublicIP  string      `json:"publicip,omitempty"`
	AWS       AgentEnvAWS `json:"aws,omitempty"`
}

// Stores AWS specific agent environment values
type AgentEnvAWS struct {
	InstanceID   string `json:"instanceid,omitempty"`
	LocalIPV4    string `json:"localipv4,omitempty"`
	AMIID        string `json:"amiid,omitempty"`
	InstanceType string `json:"instancetype,omitempty"`
}

type AgentsStats struct {
	Timestamp             time.Time           `json:"timestamp"`
	OnlineAgents          float64             `json:"onlineagents"`
	OnlineAgentsByVersion []AgentsVersionsSum `json:"onlineagentsbyversion"`
	OnlineEndpoints       float64             `json:"onlineendpoints"`
	IdleAgents            float64             `json:"idleagents"`
	IdleAgentsByVersion   []AgentsVersionsSum `json:"idleagentsbyversion"`
	IdleEndpoints         float64             `json:"idleendpoints"`
	NewEndpoints          float64             `json:"newendpoints"`
	MultiAgentsEndpoints  float64             `json:"multiagentsendpoints"`
	DisappearedEndpoints  float64             `json:"disappearedendpoints"`
	FlappingEndpoints     float64             `json:"flappingendpoints"`
}

type AgentsVersionsSum struct {
	Version string  `json:"version"`
	Count   float64 `json:"count"`
}
