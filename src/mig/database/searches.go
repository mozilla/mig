// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"mig"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// SearchParameters contains fields used to perform database searches
type SearchParameters struct {
	Before        time.Time `json:"before"`
	After         time.Time `json:"after"`
	Type          string    `json:"type"`
	Report        string    `json:"report"`
	AgentID       string    `json:"agentid"`
	AgentName     string    `json:"agentname"`
	ActionName    string    `json:"actionname"`
	ActionID      string    `json:"actionid"`
	CommandID     string    `json:"commandid"`
	ThreatFamily  string    `json:"threatfamily"`
	Status        string    `json:"status"`
	Limit         float64   `json:"limit"`
	FoundAnything bool      `json:"foundanything"`
}

// NewSearchParameters initializes search parameters
func NewSearchParameters() (p SearchParameters) {
	p.Before = time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
	p.After = time.Date(11, time.January, 11, 11, 11, 11, 11, time.UTC)
	p.AgentName = "%"
	p.AgentID = "∞"
	p.ActionName = "%"
	p.ActionID = "∞"
	p.CommandID = "∞"
	p.ThreatFamily = "%"
	p.Status = "%"
	p.Limit = 10
	return
}

// SearchCommands returns an array of commands that match search parameters
func (db *DB) SearchCommands(p SearchParameters, doFoundAnything bool) (commands []mig.Command, err error) {
	var minActionID float64 = 0
	var maxActionID float64 = 18446744073709551616 //2^64
	if p.ActionID != "∞" {
		minActionID, err = strconv.ParseFloat(p.ActionID, 64)
		if err != nil {
			return
		}
		maxActionID, err = strconv.ParseFloat(p.ActionID, 64)
		if err != nil {
			return
		}
	}
	var minCommandID float64 = 0
	var maxCommandID float64 = 18446744073709551616 //2^64
	if p.CommandID != "∞" {
		minCommandID, err = strconv.ParseFloat(p.CommandID, 64)
		if err != nil {
			return
		}
		maxCommandID, err = strconv.ParseFloat(p.CommandID, 64)
		if err != nil {
			return
		}
	}
	var minAgentID float64 = 0
	var maxAgentID float64 = 18446744073709551616 //2^64
	if p.AgentID != "∞" {
		minAgentID, err = strconv.ParseFloat(p.AgentID, 64)
		if err != nil {
			return
		}
		maxAgentID, err = strconv.ParseFloat(p.AgentID, 64)
		if err != nil {
			return
		}
	}
	var rows *sql.Rows
	if doFoundAnything {
		rows, err = db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
					actions.id, actions.name, actions.target, actions.description, actions.threat,
					actions.operations, actions.validfrom, actions.expireafter,
					actions.pgpsignatures, actions.syntaxversion,
					agents.id, agents.name, agents.queueloc, agents.os, agents.version
					FROM commands, actions, agents
					WHERE commands.actionid=actions.id AND commands.agentid=agents.id
					AND commands.id IN (SELECT commands.id FROM commands, actions, json_array_elements(commands.results) as r
					                    WHERE commands.actionid=actions.id
					                    AND actions.id >= $1 AND actions.id <= $2
					                    AND r#>>'{foundanything}' = $3)
					AND commands.starttime <= $4 AND commands.starttime >= $5
					AND commands.id >= $6 AND commands.id <= $7
					AND actions.name LIKE $8
					AND agents.name LIKE $9
					AND agents.id >= $10 AND agents.id <= $11
					AND commands.status LIKE $12
					ORDER BY agents.name ASC LIMIT $13;`, minActionID, maxActionID, p.FoundAnything, p.Before, p.After,
			minCommandID, maxCommandID, p.ActionName, p.AgentName, minAgentID, maxAgentID, p.Status, uint64(p.Limit))
	} else {
		rows, err = db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
			actions.id, actions.name, actions.target, actions.description, actions.threat,
			actions.operations, actions.validfrom, actions.expireafter,
			actions.pgpsignatures, actions.syntaxversion,
			agents.id, agents.name, agents.queueloc, agents.os, agents.version
			FROM commands, actions, agents
			WHERE commands.actionid=actions.id AND commands.agentid=agents.id
			AND commands.starttime <= $1 AND commands.starttime >= $2
			AND commands.id >= $3 AND commands.id <= $4
			AND actions.name LIKE $5
			AND actions.id >= $6 AND actions.id <= $7
			AND agents.name LIKE $8
			AND agents.id >= $9 AND agents.id <= $10
			AND actions.threat->>'family' LIKE $11
			AND commands.status LIKE $12
			ORDER BY commands.id DESC LIMIT $13`,
			p.Before, p.After, minCommandID, maxCommandID, p.ActionName, minActionID, maxActionID,
			p.AgentName, minAgentID, maxAgentID, p.ThreatFamily, p.Status, uint64(p.Limit))
	}
	if err != nil {
		err = fmt.Errorf("Error while finding commands: '%v'", err)
		return
	}
	for rows.Next() {
		var jRes, jDesc, jThreat, jOps, jSig []byte
		var cmd mig.Command
		err = rows.Scan(&cmd.ID, &cmd.Status, &jRes, &cmd.StartTime, &cmd.FinishTime,
			&cmd.Action.ID, &cmd.Action.Name, &cmd.Action.Target, &jDesc, &jThreat, &jOps,
			&cmd.Action.ValidFrom, &cmd.Action.ExpireAfter, &jSig, &cmd.Action.SyntaxVersion,
			&cmd.Agent.ID, &cmd.Agent.Name, &cmd.Agent.QueueLoc, &cmd.Agent.OS, &cmd.Agent.Version)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to retrieve command: '%v'", err)
			return
		}
		err = json.Unmarshal(jRes, &cmd.Results)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal command results: '%v'", err)
			return
		}
		err = json.Unmarshal(jDesc, &cmd.Action.Description)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
			return
		}
		err = json.Unmarshal(jThreat, &cmd.Action.Threat)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
			return
		}
		err = json.Unmarshal(jOps, &cmd.Action.Operations)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
			return
		}
		err = json.Unmarshal(jSig, &cmd.Action.PGPSignatures)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
			return
		}
		commands = append(commands, cmd)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// SearchActions returns an array of actions that match search parameters
func (db *DB) SearchActions(p SearchParameters) (actions []mig.Action, err error) {
	var minActionID float64 = 0
	var maxActionID float64 = 18446744073709551616 //2^64
	if p.ActionID != "∞" {
		minActionID, err = strconv.ParseFloat(p.ActionID, 64)
		if err != nil {
			return
		}
		maxActionID, err = strconv.ParseFloat(p.ActionID, 64)
		if err != nil {
			return
		}
	}
	rows, err := db.c.Query(`SELECT id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, pgpsignatures, syntaxversion
		FROM actions
		WHERE actions.starttime <= $1 AND actions.starttime >= $2
		AND actions.name LIKE $3
		AND actions.id >= $4 AND actions.id <= $5
		AND actions.threat->>'family' LIKE $6
		ORDER BY actions.id DESC LIMIT $7`,
		p.Before, p.After, p.ActionName, minActionID, maxActionID, p.ThreatFamily, uint64(p.Limit))
	if err != nil {
		err = fmt.Errorf("Error while finding actions: '%v'", err)
		return
	}
	for rows.Next() {
		var jDesc, jThreat, jOps, jSig []byte
		var a mig.Action
		err = rows.Scan(&a.ID, &a.Name, &a.Target,
			&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
			&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status,
			&jSig, &a.SyntaxVersion)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Error while retrieving action: '%v'", err)
			return
		}
		err = json.Unmarshal(jDesc, &a.Description)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
			return
		}
		err = json.Unmarshal(jThreat, &a.Threat)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
			return
		}
		err = json.Unmarshal(jOps, &a.Operations)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
			return
		}
		err = json.Unmarshal(jSig, &a.PGPSignatures)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
			return
		}
		a.Counters, err = db.GetActionCounters(a.ID)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve action counters: '%v'", err)
			return
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// SearchAgents returns an array of agents that match search parameters
func (db *DB) SearchAgents(p SearchParameters) (agents []mig.Agent, err error) {
	rows, err := db.c.Query(`SELECT agents.id, agents.name, agents.queueloc, agents.os,
		agents.version, agents.pid, agents.starttime, agents.destructiontime,
		agents.heartbeattime, agents.status
		FROM agents
		WHERE agents.heartbeattime <= $1 AND agents.heartbeattime >= $2
		AND agents.name LIKE $3
		AND agents.status LIKE $4
		ORDER BY agents.heartbeattime DESC LIMIT $5`,
		p.Before, p.After, p.AgentName, p.Status, uint64(p.Limit))
	if err != nil {
		err = fmt.Errorf("Error while finding agents: '%v'", err)
		return
	}
	for rows.Next() {
		var agent mig.Agent
		err = rows.Scan(&agent.ID, &agent.Name, &agent.QueueLoc, &agent.OS, &agent.Version,
			&agent.PID, &agent.StartTime, &agent.DestructionTime, &agent.HeartBeatTS,
			&agent.Status)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to retrieve agent data: '%v'", err)
			return
		}
		agents = append(agents, agent)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}

	return
}
