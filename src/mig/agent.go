// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package mig

import "time"

type Agent struct {
	ID              float64   `json:"id,omitempty"`
	Name            string    `json:"name"`
	QueueLoc        string    `json:"queueloc"`
	OS              string    `json:"os,omitempty"`
	Version         string    `json:"version,omitempty"`
	PID             int       `json:"pid,omitempty"`
	StartTime       time.Time `json:"starttime,omitempty"`
	DestructionTime time.Time `json:"destructiontime,omitempty"`
	HeartBeatTS     time.Time `json:"heartbeatts,omitempty"`
	Status          string    `json:"status,omitempty"`
	Authorized      bool      `json:"authorized,omitempty"`
	Env             AgentEnv  `json:"environment,omitempty"`
}

type AgentEnv struct {
	Init      string `json:"init,omitempty"`
	Ident     string `json:"ident,omitempty"`
	Arch      string `json:"arch,omitempty"`
	IsProxied bool   `json:"isproxied"`
	Proxy     string `json:"proxy,omitempty"`
}

// findHostname retrieves the hostname of the node
