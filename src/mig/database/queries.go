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

type DB struct {
	c *sql.DB
}

// Connect opens a connection to the database and returns a handler
func Open(dbname, user, password, host string, port int, sslmode string) (db DB, err error) {
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)
	db.c, err = sql.Open("postgres", url)
	return
}

func (db *DB) Close() {
	db.c.Close()
}

func (db *DB) SetMaxOpenConns(n int) {
	db.c.SetMaxOpenConns(n)
}

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
func (db *DB) SearchCommands(p SearchParameters) (commands []mig.Command, err error) {
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
	rows, err := db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
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
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion
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
			&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &a.Counters.Sent,
			&a.Counters.Returned, &a.Counters.Done, &a.Counters.Cancelled,
			&a.Counters.Failed, &a.Counters.TimeOut, &jSig, &a.SyntaxVersion)
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

// ActionByID retrieves an action from the database using its ID
func (db *DB) Last10Actions() (actions []mig.Action, err error) {
	rows, err := db.c.Query(`SELECT id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion
		FROM actions ORDER BY starttime DESC LIMIT 10`)
	if err != nil && err != sql.ErrNoRows {
		rows.Close()
		err = fmt.Errorf("Error while listing actions: '%v'", err)
		return
	}
	for rows.Next() {
		var jDesc, jThreat, jOps, jSig []byte
		var a mig.Action
		err = rows.Scan(&a.ID, &a.Name, &a.Target,
			&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
			&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &a.Counters.Sent,
			&a.Counters.Returned, &a.Counters.Done, &a.Counters.Cancelled,
			&a.Counters.Failed, &a.Counters.TimeOut, &jSig, &a.SyntaxVersion)
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
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// ActionByID retrieves an action from the database using its ID
// If the query fails, the returned action will have ID -1
func (db *DB) ActionByID(id float64) (a mig.Action, err error) {
	a.ID = -1
	var jDesc, jThreat, jOps, jSig []byte
	err = db.c.QueryRow(`SELECT id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion
		FROM actions WHERE id=$1`, id).Scan(&a.ID, &a.Name, &a.Target,
		&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
		&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &a.Counters.Sent,
		&a.Counters.Returned, &a.Counters.Done, &a.Counters.Cancelled,
		&a.Counters.Failed, &a.Counters.TimeOut, &jSig, &a.SyntaxVersion)
	if err != nil {
		err = fmt.Errorf("Error while retrieving action: '%v'", err)
		return
	}
	err = json.Unmarshal(jDesc, &a.Description)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
		return
	}
	err = json.Unmarshal(jThreat, &a.Threat)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
		return
	}
	err = json.Unmarshal(jOps, &a.Operations)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
		return
	}
	err = json.Unmarshal(jSig, &a.PGPSignatures)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
		return
	}
	return
}

// ActionMetaByID retrieves the metadata fields of an action from the database using its ID
func (db *DB) ActionMetaByID(id float64) (a mig.Action, err error) {
	err = db.c.QueryRow(`SELECT id, name, validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr FROM actions WHERE id=$1`, id).Scan(&a.ID, &a.Name, &a.ValidFrom, &a.ExpireAfter,
		&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &a.Counters.Sent,
		&a.Counters.Returned, &a.Counters.Done, &a.Counters.Cancelled,
		&a.Counters.Failed, &a.Counters.TimeOut)
	if err != nil {
		err = fmt.Errorf("Error while retrieving action: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// InsertAction writes an action into the database.
func (db *DB) InsertAction(a mig.Action) (err error) {
	jDesc, err := json.Marshal(a.Description)
	if err != nil {
		return fmt.Errorf("Failed to marshal description: '%v'", err)
	}
	jThreat, err := json.Marshal(a.Threat)
	if err != nil {
		return fmt.Errorf("Failed to marshal threat: '%v'", err)
	}
	jOperations, err := json.Marshal(a.Operations)
	if err != nil {
		return fmt.Errorf("Failed to marshal operations: '%v'", err)
	}
	aPGPSignatures, err := json.Marshal(a.PGPSignatures)
	if err != nil {
		return fmt.Errorf("Failed to marshal pgp signatures: '%v'", err)
	}
	_, err = db.c.Exec(`INSERT INTO actions
		(id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
		$14, $15, $16, $17, $18, $19, $20)`,
		a.ID, a.Name, a.Target, jDesc, jThreat, jOperations,
		a.ValidFrom, a.ExpireAfter, a.StartTime, a.FinishTime, a.LastUpdateTime,
		a.Status, a.Counters.Sent, a.Counters.Returned, a.Counters.Done,
		a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut,
		aPGPSignatures, a.SyntaxVersion)
	if err != nil {
		return fmt.Errorf("Failed to store action: '%v'", err)
	}
	return
}

// UpdateAction stores updated action fields into the database.
func (db *DB) UpdateAction(a mig.Action) (err error) {
	_, err = db.c.Exec(`UPDATE actions SET (starttime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr, timeoutctr)
		= ($2, $3, $4, $5, $6, $7, $8, $9, $10) WHERE id=$1`,
		a.ID, a.StartTime, a.LastUpdateTime, a.Status, a.Counters.Sent, a.Counters.Returned,
		a.Counters.Done, a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut)
	if err != nil {
		return fmt.Errorf("Failed to update action: '%v'", err)
	}
	return
}

// InsertOrUpdateAction looks for an existing action in DB and update it,
// or insert a new one if none is found
func (db *DB) InsertOrUpdateAction(a mig.Action) (inserted bool, err error) {
	var id float64
	err = db.c.QueryRow(`SELECT id FROM actions WHERE id=$1`, a.ID).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return inserted, fmt.Errorf("Error while retrieving action: '%v'", err)
	}
	if err == sql.ErrNoRows {
		inserted = true
		return inserted, db.InsertAction(a)
	} else {
		return inserted, db.UpdateAction(a)
	}
}

// UpdateActionStatus updates the status of an action
func (db *DB) UpdateActionStatus(a mig.Action) (err error) {
	_, err = db.c.Exec(`UPDATE actions SET (status) = ($2) WHERE id=$1`,
		a.ID, a.Status)
	if err != nil {
		return fmt.Errorf("Failed to update action status: '%v'", err)
	}
	return
}

// UpdateRunningAction stores updated time and counters on a running action
func (db *DB) UpdateRunningAction(a mig.Action) (err error) {
	_, err = db.c.Exec(`UPDATE actions SET (lastupdatetime, returnedctr,
		donectr, cancelledctr, failedctr, timeoutctr)
		= ($2, $3, $4, $5, $6, $7) WHERE id=$1`,
		a.ID, a.LastUpdateTime, a.Counters.Returned, a.Counters.Done,
		a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut)
	if err != nil {
		return fmt.Errorf("Failed to update action: '%v'", err)
	}
	return
}

// FinishAction updates the action fields to mark it as done
func (db *DB) FinishAction(a mig.Action) (err error) {
	a.FinishTime = time.Now()
	a.Status = "completed"
	_, err = db.c.Exec(`UPDATE actions SET (finishtime, lastupdatetime, status,
		returnedctr, donectr, cancelledctr, failedctr, timeoutctr)
		= ($1, $2, $3, $4, $5, $6, $7, $8) WHERE id=$9`,
		a.FinishTime, a.LastUpdateTime, a.Status, a.Counters.Returned,
		a.Counters.Done, a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut, a.ID)
	if err != nil {
		return fmt.Errorf("Failed to update action: '%v'", err)
	}
	return
}

// InsertSignature create an entry in the signatures tables that map an investigator
// to an action and a signature
func (db *DB) InsertSignature(aid, iid float64, sig string) (err error) {
	_, err = db.c.Exec(`INSERT INTO signatures(actionid, investigatorid, pgpsignature)
		VALUES($1, $2, $3)`, aid, iid, sig)
	if err != nil {
		return fmt.Errorf("Failed to store signature: '%v'", err)
	}
	return
}

// FindInvestigatorByFingerprint searches the database for an investigator that
// has a given fingerprint
func (db *DB) InvestigatorByFingerprint(fp string) (iid float64, err error) {
	err = db.c.QueryRow("SELECT id FROM investigators WHERE LOWER(pgpfingerprint)=LOWER($1)", fp).Scan(&iid)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while finding investigator: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		err = fmt.Errorf("InvestigatorByFingerprint: no investigator found for fingerprint '%s'", fp)
		return
	}
	return
}

//InvestigatorByActionID returns the list of investigators that signed a given action
func (db *DB) InvestigatorByActionID(aid float64) (ivgts []mig.Investigator, err error) {
	rows, err := db.c.Query(`SELECT investigators.id, investigators.name, investigators.pgpfingerprint
		FROM investigators, signatures
		WHERE signatures.actionid=$1
		AND signatures.investigatorid=investigators.id`, aid)
	if err != nil && err != sql.ErrNoRows {
		rows.Close()
		err = fmt.Errorf("Error while finding investigator: '%v'", err)
		return
	}
	for rows.Next() {
		var ivgt mig.Investigator
		err = rows.Scan(&ivgt.ID, &ivgt.Name, &ivgt.PGPFingerprint)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to retrieve investigator data: '%v'", err)
			return
		}
		ivgts = append(ivgts, ivgt)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// CommandByID retrieves a command from the database using its ID
func (db *DB) CommandByID(id float64) (cmd mig.Command, err error) {
	var jRes, jDesc, jThreat, jOps, jSig []byte
	err = db.c.QueryRow(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
		actions.id, actions.name, actions.target, actions.description, actions.threat,
		actions.operations, actions.validfrom, actions.expireafter,
		actions.pgpsignatures, actions.syntaxversion,
		agents.id, agents.name, agents.queueloc, agents.os, agents.version
		FROM commands, actions, agents
		WHERE commands.id=$1
		AND commands.actionid = actions.id AND commands.agentid = agents.id`, id).Scan(
		&cmd.ID, &cmd.Status, &jRes, &cmd.StartTime, &cmd.FinishTime,
		&cmd.Action.ID, &cmd.Action.Name, &cmd.Action.Target, &jDesc, &jThreat, &jOps,
		&cmd.Action.ValidFrom, &cmd.Action.ExpireAfter, &jSig, &cmd.Action.SyntaxVersion,
		&cmd.Agent.ID, &cmd.Agent.Name, &cmd.Agent.QueueLoc, &cmd.Agent.OS, &cmd.Agent.Version)
	if err != nil {
		err = fmt.Errorf("Error while retrieving command: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	err = json.Unmarshal(jRes, &cmd.Results)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal command results: '%v'", err)
		return
	}
	err = json.Unmarshal(jDesc, &cmd.Action.Description)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
		return
	}
	err = json.Unmarshal(jThreat, &cmd.Action.Threat)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
		return
	}
	err = json.Unmarshal(jOps, &cmd.Action.Operations)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
		return
	}
	err = json.Unmarshal(jSig, &cmd.Action.PGPSignatures)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
		return
	}
	return
}

func (db *DB) CommandsByActionID(actionid float64) (commands []mig.Command, err error) {
	rows, err := db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
		actions.id, actions.name, actions.target, actions.description, actions.threat,
		actions.operations, actions.validfrom, actions.expireafter,
		actions.pgpsignatures, actions.syntaxversion,
		agents.id, agents.name, agents.queueloc, agents.os, agents.version
		FROM commands, actions, agents
		WHERE commands.actionid=actions.id AND commands.agentid=agents.id AND actions.id=$1`, actionid)
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

// InsertCommand writes a command into the database
func (db *DB) InsertCommand(cmd mig.Command, agt mig.Agent) (err error) {
	jResults, err := json.Marshal(cmd.Results)
	if err != nil {
		return fmt.Errorf("Failed to marshal results: '%v'", err)
	}
	_, err = db.c.Exec(`INSERT INTO commands
		(id, actionid, agentid, status, results, starttime, finishtime)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, cmd.ID, cmd.Action.ID,
		agt.ID, cmd.Status, jResults, cmd.StartTime, cmd.FinishTime)
	if err != nil {
		return fmt.Errorf("Error while inserting command: '%v'", err)
	}
	return
}

// InsertCommands writes an array of commands into the database
func (db *DB) InsertCommands(cmds []mig.Command) (insertCount int64, err error) {
	emptyCmdResults, _ := json.Marshal(cmds[0].Results)
	futureDate := time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
	sql := "INSERT INTO commands (id, actionid, agentid, status, starttime, finishtime, results) VALUES "
	vals := []interface{}{}
	step := 0
	for i, cmd := range cmds {
		if i > 0 {
			sql += ", "
		}
		sql += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i+1+step, i+2+step, i+3+step, i+4+step, i+5+step, i+6+step, i+7+step)
		vals = append(vals, cmd.ID, cmd.Action.ID, cmd.Agent.ID, cmd.Status, cmd.StartTime, futureDate, emptyCmdResults)
		step += 6
	}
	stmt, err := db.c.Prepare(sql)
	if err != nil {
		err = fmt.Errorf("Error while preparing insertion statement: '%v' in '%s'", err, sql)
		return
	}
	res, err := stmt.Exec(vals...)
	if err != nil {
		err = fmt.Errorf("Error while inserting commands: '%v'", err)
		return
	}
	insertCount, err = res.RowsAffected()
	if err != nil {
		err = fmt.Errorf("Error while counting inserted commands: '%v'", err)
		return
	}
	return
}

// UpdateSentCommand updates a command into the database
func (db *DB) UpdateSentCommand(cmd mig.Command) (err error) {
	_, err = db.c.Exec(`UPDATE commands SET status=$1 WHERE id=$2`, cmd.Status, cmd.ID)
	if err != nil {
		return fmt.Errorf("Error while updating command: '%v'", err)
	}
	return
}

// FinishCommand updates a command into the database
func (db *DB) FinishCommand(cmd mig.Command) (err error) {
	jResults, err := json.Marshal(cmd.Results)
	if err != nil {
		return fmt.Errorf("Failed to marshal results: '%v'", err)
	}
	_, err = db.c.Exec(`UPDATE commands SET status=$1, results=$2,
		finishtime=$3 WHERE id=$4`, cmd.Status, jResults,
		cmd.FinishTime, cmd.ID)
	if err != nil {
		return fmt.Errorf("Error while updating command: '%v'", err)
	}
	return
}

// AgentByQueueAndPID returns a single agent that is located at a given queueloc and has a given PID
func (db *DB) AgentByQueueAndPID(queueloc string, pid int) (agent mig.Agent, err error) {
	err = db.c.QueryRow(`SELECT id, name, queueloc, os, version, pid, starttime, heartbeattime,
		status FROM agents WHERE queueloc=$1 AND pid=$2`, queueloc, pid).Scan(
		&agent.ID, &agent.Name, &agent.QueueLoc, &agent.OS, &agent.Version, &agent.PID,
		&agent.StartTime, &agent.HeartBeatTS, &agent.Status)
	if err != nil {
		err = fmt.Errorf("Error while retrieving agent: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// AgentByID returns a single agent identified by its ID
func (db *DB) AgentByID(id float64) (agent mig.Agent, err error) {
	err = db.c.QueryRow(`SELECT id, name, queueloc, os, version, pid, starttime, heartbeattime,
		status FROM agents WHERE id=$1`, id).Scan(
		&agent.ID, &agent.Name, &agent.QueueLoc, &agent.OS, &agent.Version, &agent.PID,
		&agent.StartTime, &agent.HeartBeatTS, &agent.Status)
	if err != nil {
		err = fmt.Errorf("Error while retrieving agent: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// AgentsActiveSince returns an array of Agents that have sent a heartbeat between
// a point in time and now
func (db *DB) AgentsActiveSince(pointInTime time.Time) (agents []mig.Agent, err error) {
	rows, err := db.c.Query(`SELECT agents.id, agents.name, agents.queueloc, agents.os,
		agents.version, agents.pid, agents.starttime, agents.destructiontime,
		agents.heartbeattime, agents.status
		FROM agents
		WHERE agents.heartbeattime >= $1 AND agents.heartbeattime <= NOW()`,
		pointInTime)
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

// InsertAgent creates a new agent in the database
func (db *DB) InsertAgent(agt mig.Agent) (err error) {
	jEnv, err := json.Marshal(agt.Env)
	if err != nil {
		err = fmt.Errorf("Failed to marshal agent environment: '%v'", err)
		return
	}
	agtid := mig.GenID()
	_, err = db.c.Exec(`INSERT INTO agents
		(id, name, queueloc, os, version, pid, starttime, destructiontime, heartbeattime, status, environment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`, agtid, agt.Name, agt.QueueLoc,
		agt.OS, agt.Version, agt.PID, agt.StartTime, agt.DestructionTime, agt.HeartBeatTS, agt.Status, jEnv)
	if err != nil {
		return fmt.Errorf("Failed to insert agent in database: '%v'", err)
	}
	return
}

// UpdateAgentHeartbeat updates the heartbeat timestamp of an agent in the database
func (db *DB) UpdateAgentHeartbeat(agt mig.Agent) (err error) {
	_, err = db.c.Exec(`UPDATE agents
		SET heartbeattime=$2 WHERE id=$1`, agt.ID, agt.HeartBeatTS)
	if err != nil {
		return fmt.Errorf("Failed to update agent in database: '%v'", err)
	}
	return
}

// InsertOrUpdateAgent will first search for a given agent in database and update it
// if it exists, or insert it if it doesn't
func (db *DB) InsertOrUpdateAgent(agt mig.Agent) (err error) {
	agent, err := db.AgentByQueueAndPID(agt.QueueLoc, agt.PID)
	if err != nil {
		agt.DestructionTime = time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
		agt.Status = "heartbeating"
		// create a new agent
		return db.InsertAgent(agt)
	} else {
		agt.ID = agent.ID
		// agent exists in DB, update it
		return db.UpdateAgentHeartbeat(agt)
	}
}

// ActiveAgentsByQueue retrieves an array of agents identified by their QueueLoc value
func (db *DB) ActiveAgentsByQueue(queueloc string, pointInTime time.Time) (agents []mig.Agent, err error) {
	rows, err := db.c.Query(`SELECT agents.id, agents.name, agents.queueloc, agents.os,
		agents.version, agents.pid, agents.starttime, agents.heartbeattime, agents.status
		FROM agents
		WHERE agents.heartbeattime >= $1 AND agents.heartbeattime <= NOW()
		AND agents.queueloc=$2`, pointInTime, queueloc)
	if err != nil {
		err = fmt.Errorf("Error while finding agents: '%v'", err)
		return
	}
	for rows.Next() {
		var agent mig.Agent
		err = rows.Scan(&agent.ID, &agent.Name, &agent.QueueLoc, &agent.OS, &agent.Version,
			&agent.PID, &agent.StartTime, &agent.HeartBeatTS, &agent.Status)
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

// ActiveAgentsByTarget runs a search for all agents that match a given target string
func (db *DB) ActiveAgentsByTarget(target string, pointInTime time.Time) (agents []mig.Agent, err error) {
	rows, err := db.c.Query(`SELECT DISTINCT ON (queueloc) id, name, queueloc, os, version, pid,
		starttime, destructiontime, heartbeattime, status
		FROM agents
		WHERE agents.heartbeattime >= $1 AND agents.heartbeattime <= NOW()
		AND (`+target+`)
		ORDER BY agents.queueloc, agents.heartbeattime DESC`, pointInTime)
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

// MarkAgentUpgraded updated the status of an agent in the database
func (db *DB) MarkAgentUpgraded(agent mig.Agent) (err error) {
	_, err = db.c.Exec(`UPDATE agents SET status='upgraded' WHERE id=$1`,
		agent.ID)
	if err != nil {
		return fmt.Errorf("Failed to mark agent as upgraded in database: '%v'", err)
	}
	return
}

// MarkAgentDestroyed updated the status and destructiontime of an agent in the database
func (db *DB) MarkAgentDestroyed(agent mig.Agent) (err error) {
	agent.Status = "destroyed"
	agent.DestructionTime = time.Now()
	_, err = db.c.Exec(`UPDATE agents
		SET destructiontime=$1, status=$2 WHERE id=$3`,
		agent.DestructionTime, agent.Status, agent.ID)
	if err != nil {
		return fmt.Errorf("Failed to mark agent as destroyed in database: '%v'", err)
	}
	return
}

type AgentsSum struct {
	Version string  `json:"version"`
	Count   float64 `json:"count"`
}

// SumAgentsByVersion retrieves a sum of agents grouped by version
func (db *DB) SumAgentsByVersion(pointInTime time.Time) (sum []AgentsSum, err error) {
	rows, err := db.c.Query(`SELECT COUNT(*), version FROM agents
		WHERE agents.heartbeattime >= $1 AND agents.heartbeattime <= NOW()
		GROUP BY version`, pointInTime)
	if err != nil {
		err = fmt.Errorf("Error while counting agents: '%v'", err)
		return
	}
	for rows.Next() {
		var asum AgentsSum
		err = rows.Scan(&asum.Count, &asum.Version)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to retrieve summary data: '%v'", err)
			return
		}
		sum = append(sum, asum)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// CountNewAgents retrieves a count of agents that started after `pointInTime`
func (db *DB) CountNewAgents(pointInTime time.Time) (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(name) FROM agents
		WHERE starttime >= $1 AND starttime <= NOW()`, pointInTime).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting agents: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// CountDoubleAgents counts the number of endpoints that run more than one agent
func (db *DB) CountDoubleAgents(pointInTime time.Time) (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(DISTINCT(queueloc)) FROM agents
		WHERE queueloc IN (
			SELECT queueloc FROM agents
			WHERE heartbeattime >= $1
			GROUP BY queueloc HAVING count(queueloc) > 1
		)`, pointInTime).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting double agents: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// CountDisappearedAgents counts the number of endpoints that have disappeared recently
func (db *DB) CountDisappearedAgents(seenSince, activeSince time.Time) (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(DISTINCT(name)) FROM agents
		WHERE name IN (
		        SELECT DISTINCT(name) FROM agents
		        WHERE heartbeattime >= $1
		) AND name NOT IN (
		        SELECT DISTINCT(name) FROM agents
		        WHERE heartbeattime >= $2
		)`, seenSince, activeSince).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting agents: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}
