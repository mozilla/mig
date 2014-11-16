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
	ActionID         string    `json:"actionid"`
	ActionName       string    `json:"actionname"`
	After            time.Time `json:"after"`
	AgentID          string    `json:"agentid"`
	AgentName        string    `json:"agentname"`
	Before           time.Time `json:"before"`
	CommandID        string    `json:"commandid"`
	FoundAnything    bool      `json:"foundanything"`
	InvestigatorID   string    `json:"investigatorid"`
	InvestigatorName string    `json:"investigatorname"`
	Limit            float64   `json:"limit"`
	Report           string    `json:"report"`
	Status           string    `json:"status"`
	Target           string    `json:"target"`
	ThreatFamily     string    `json:"threatfamily"`
	Type             string    `json:"type"`
}

// NewSearchParameters initializes search parameters
func NewSearchParameters() (p SearchParameters) {
	p.Before = time.Now().UTC()
	p.After = time.Now().Add(-168 * time.Hour).UTC()
	p.AgentName = "%"
	p.AgentID = "∞"
	p.ActionName = "%"
	p.ActionID = "∞"
	p.CommandID = "∞"
	p.ThreatFamily = "%"
	p.Status = "%"
	p.Limit = 10000
	p.InvestigatorID = "∞"
	p.InvestigatorName = "%"
	p.Type = "action"
	return
}

type IDs struct {
	minActionID, maxActionID, minCommandID, maxCommandID, minAgentID, maxAgentID, minInvID, maxInvID float64
}

// SearchCommands returns an array of commands that match search parameters
func (db *DB) SearchCommands(p SearchParameters, doFoundAnything bool) (commands []mig.Command, err error) {
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	var rows *sql.Rows
	if doFoundAnything {
		rows, err = db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
					actions.id, actions.name, actions.target, actions.description, actions.threat,
					actions.operations, actions.validfrom, actions.expireafter,
					actions.pgpsignatures, actions.syntaxversion,
					agents.id, agents.name, agents.queueloc, agents.os, agents.version
					FROM commands, actions, agents, investigators, signatures
					WHERE commands.actionid=actions.id AND commands.agentid=agents.id
					AND actions.id=signatures.actionid AND signatures.investigatorid=investigators.id
					AND commands.id IN (SELECT commands.id FROM commands, actions, json_array_elements(commands.results) as r
					                    WHERE commands.actionid=actions.id
					                    AND actions.id >= $1 AND actions.id <= $2
					                    AND r#>>'{foundanything}' = $3)
					AND commands.starttime <= $4 AND commands.starttime >= $5
					AND commands.id >= $6 AND commands.id <= $7
					AND actions.name ILIKE $8
					AND agents.name ILIKE $9
					AND agents.id >= $10 AND agents.id <= $11
					AND investigators.id >= $12 AND investigators.id <= $13
					AND investigators.name ILIKE $14
					AND commands.status ILIKE $15
					GROUP BY commands.id, actions.id, agents.id
					ORDER BY commands.id ASC LIMIT $16;`, ids.minActionID, ids.maxActionID,
			p.FoundAnything, p.Before, p.After, ids.minCommandID, ids.maxCommandID, p.ActionName,
			p.AgentName, ids.minAgentID, ids.maxAgentID, ids.minInvID, ids.maxInvID,
			p.InvestigatorName, p.Status, uint64(p.Limit))
	} else {
		rows, err = db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
			actions.id, actions.name, actions.target, actions.description, actions.threat,
			actions.operations, actions.validfrom, actions.expireafter,
			actions.pgpsignatures, actions.syntaxversion,
			agents.id, agents.name, agents.queueloc, agents.os, agents.version
			FROM commands, actions, agents, investigators, signatures
			WHERE commands.actionid=actions.id AND commands.agentid=agents.id
			AND actions.id=signatures.actionid AND signatures.investigatorid=investigators.id
			AND commands.starttime <= $1 AND commands.starttime >= $2
			AND commands.id >= $3 AND commands.id <= $4
			AND actions.name ILIKE $5
			AND actions.id >= $6 AND actions.id <= $7
			AND agents.name ILIKE $8
			AND agents.id >= $9 AND agents.id <= $10
			AND investigators.id >= $11 AND investigators.id <= $12
			AND investigators.name ILIKE $13
			AND commands.status ILIKE $14
			GROUP BY commands.id, actions.id, agents.id
			ORDER BY commands.id DESC LIMIT $15`,
			p.Before, p.After, ids.minCommandID, ids.maxCommandID, p.ActionName, ids.minActionID, ids.maxActionID,
			p.AgentName, ids.minAgentID, ids.maxAgentID, ids.minInvID, ids.maxInvID, p.InvestigatorName,
			p.Status, uint64(p.Limit))
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
		err = json.Unmarshal(jThreat, &cmd.Action.Threat)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
			return
		}
		// Check for threatfamily, if asked
		if p.ThreatFamily != "%" {
			if p.ThreatFamily != cmd.Action.Threat.Family {
				// skip this record
				continue
			}
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
		cmd.Action.Counters, err = db.GetActionCounters(cmd.Action.ID)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve action counters: '%v'", err)
			return
		}
		cmd.Action.Investigators, err = db.InvestigatorByActionID(cmd.Action.ID)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve action investigators: '%v'", err)
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
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	rows, err := db.c.Query(`SELECT actions.id, actions.name, actions.target,
		actions.description, actions.threat, actions.operations, actions.validfrom,
		actions.expireafter, actions.starttime, actions.finishtime, actions.lastupdatetime,
		actions.status, actions.pgpsignatures, actions.syntaxversion
		FROM commands, actions, agents, investigators, signatures
		WHERE commands.actionid=actions.id AND commands.agentid=agents.id
		AND actions.id=signatures.actionid AND signatures.investigatorid=investigators.id
		AND actions.expireafter <= $1 AND actions.validfrom >= $2
		AND commands.id >= $3 AND commands.id <= $4
		AND actions.name ILIKE $5
		AND actions.id >= $6 AND actions.id <= $7
		AND agents.name ILIKE $8
		AND agents.id >= $9 AND agents.id <= $10
		AND investigators.id >= $11 AND investigators.id <= $12
		AND investigators.name ILIKE $13
		AND actions.status ILIKE $14
		GROUP BY actions.id
		ORDER BY actions.id DESC LIMIT $15`,
		p.Before, p.After, ids.minCommandID, ids.maxCommandID, p.ActionName, ids.minActionID, ids.maxActionID,
		p.AgentName, ids.minAgentID, ids.maxAgentID, ids.minInvID, ids.maxInvID, p.InvestigatorName,
		p.Status, uint64(p.Limit))
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
		err = json.Unmarshal(jThreat, &a.Threat)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
			return
		}
		// Check for threatfamily, if asked
		if p.ThreatFamily != "%" {
			if p.ThreatFamily != a.Threat.Family {
				// skip this record
				continue
			}
		}
		err = json.Unmarshal(jDesc, &a.Description)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
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
		a.Investigators, err = db.InvestigatorByActionID(a.ID)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve action investigators: '%v'", err)
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
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	rows, err := db.c.Query(`SELECT agents.id, agents.name, agents.queueloc, agents.os,
		agents.version, agents.pid, agents.starttime, agents.destructiontime,
		agents.heartbeattime, agents.status
		FROM commands, actions, agents, investigators, signatures
		WHERE commands.actionid=actions.id AND commands.agentid=agents.id
		AND actions.id=signatures.actionid AND signatures.investigatorid=investigators.id
		AND agents.heartbeattime <= $1 AND agents.heartbeattime >= $2
		AND commands.id >= $3 AND commands.id <= $4
		AND actions.name ILIKE $5
		AND actions.id >= $6 AND actions.id <= $7
		AND agents.name ILIKE $8
		AND agents.id >= $9 AND agents.id <= $10
		AND investigators.id >= $11 AND investigators.id <= $12
		AND investigators.name ILIKE $13
		AND agents.status ILIKE $14
		GROUP BY agents.id
		ORDER BY agents.id DESC LIMIT $15`,
		p.Before, p.After, ids.minCommandID, ids.maxCommandID, p.ActionName, ids.minActionID, ids.maxActionID,
		p.AgentName, ids.minAgentID, ids.maxAgentID, ids.minInvID, ids.maxInvID, p.InvestigatorName,
		p.Status, uint64(p.Limit))
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

// SearchInvestigators returns an array of investigators that match search parameters
func (db *DB) SearchInvestigators(p SearchParameters) (investigators []mig.Investigator, err error) {
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	var rows *sql.Rows
	if p.ActionID != "∞" || p.ActionName != "%" || p.CommandID != "∞" || p.AgentID != "∞" || p.AgentName != "%" {
		rows, err = db.c.Query(`SELECT investigators.id, investigators.name, investigators.pgpfingerprint,
			investigators.status, investigators.createdat, investigators.lastmodified
			FROM commands, actions, agents, investigators, signatures
			WHERE commands.actionid=actions.id AND commands.agentid=agents.id
			AND actions.id=signatures.actionid AND signatures.investigatorid=investigators.id
			AND investigators.lastmodified <= $1 AND investigators.createdat >= $2
			AND commands.id >= $3 AND commands.id <= $4
			AND actions.name ILIKE $5
			AND actions.id >= $6 AND actions.id <= $7
			AND agents.name ILIKE $8
			AND agents.id >= $9 AND agents.id <= $10
			AND investigators.id >= $11 AND investigators.id <= $12
			AND investigators.name ILIKE $13
			AND investigators.status ILIKE $14
			GROUP BY investigators.id
			ORDER BY investigators.id DESC LIMIT $15`,
			p.Before, p.After, ids.minCommandID, ids.maxCommandID, p.ActionName, ids.minActionID, ids.maxActionID,
			p.AgentName, ids.minAgentID, ids.maxAgentID, ids.minInvID, ids.maxInvID, p.InvestigatorName,
			p.Status, uint64(p.Limit))
	} else {
		rows, err = db.c.Query(`SELECT investigators.id, investigators.name, investigators.pgpfingerprint,
			investigators.status, investigators.createdat, investigators.lastmodified
			FROM investigators
			WHERE investigators.id >= $1 AND investigators.id <= $2
			AND investigators.name ILIKE $3
			AND investigators.status ILIKE $4
			GROUP BY investigators.id
			ORDER BY investigators.id DESC LIMIT $5`,
			ids.minInvID, ids.maxInvID, p.InvestigatorName, p.Status, uint64(p.Limit))
	}
	if err != nil {
		err = fmt.Errorf("Error while finding investigators: '%v'", err)
		return
	}
	for rows.Next() {
		var inv mig.Investigator
		err = rows.Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.Status, &inv.CreatedAt, &inv.LastModified)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to retrieve investigator data: '%v'", err)
			return
		}
		investigators = append(investigators, inv)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

func makeIDsFromParams(p SearchParameters) (ids IDs, err error) {
	ids.minActionID = 0
	ids.maxActionID = 18446744073709551616 //2^64
	if p.ActionID != "∞" {
		ids.minActionID, err = strconv.ParseFloat(p.ActionID, 64)
		if err != nil {
			return
		}
		ids.maxActionID = ids.minActionID
	}
	ids.minCommandID = 0
	ids.maxCommandID = 18446744073709551616 //2^64
	if p.CommandID != "∞" {
		ids.minCommandID, err = strconv.ParseFloat(p.CommandID, 64)
		if err != nil {
			return
		}
		ids.maxCommandID = ids.minCommandID
	}
	ids.minAgentID = 0
	ids.maxAgentID = 18446744073709551616 //2^64
	if p.AgentID != "∞" {
		ids.minAgentID, err = strconv.ParseFloat(p.AgentID, 64)
		if err != nil {
			return
		}
		ids.maxAgentID = ids.minAgentID
	}
	ids.minInvID = 0
	ids.maxInvID = 18446744073709551616 //2^64
	if p.InvestigatorID != "∞" {
		ids.minInvID, err = strconv.ParseFloat(p.InvestigatorID, 64)
		if err != nil {
			return
		}
		ids.maxInvID = ids.minInvID
	}
	return
}
