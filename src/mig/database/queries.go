/* Mozilla InvestiGator Database Queries

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"mig"
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

// SearchParameters contains fields used to perform database searches
type SearchParameters struct {
	Before       time.Time `json:"before"`
	After        time.Time `json:"after"`
	Type         string    `json:"type"`
	Report       string    `json:"report"`
	AgentName    string    `json:"agentname"`
	ActionName   string    `json:"actionname"`
	ThreatFamily string    `json:"threatfamily"`
	Status       string    `json:"status"`
	Limit        float64   `json:"limit"`
}

// NewSearchParameters initializes search parameters
func NewSearchParameters() (p SearchParameters) {
	p.Before = time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
	p.After = time.Date(11, time.January, 11, 11, 11, 11, 11, time.UTC)
	p.AgentName = "%"
	p.ActionName = "%"
	p.ThreatFamily = "%"
	p.Status = "%"
	p.Limit = 10
	return
}

// SearchCommands returns an array of commands that match search parameters
func (db *DB) SearchCommands(p SearchParameters) (commands []mig.Command, err error) {
	rows, err := db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
		actions.id, actions.name, actions.target, actions.description, actions.threat,
		actions.operations, actions.validfrom, actions.expireafter,
		actions.pgpsignatures, actions.syntaxversion,
		agents.id, agents.name, agents.queueloc, agents.os, agents.version
		FROM commands, actions, agents
		WHERE commands.actionid=actions.id AND commands.agentid=agents.id
		AND commands.starttime <= $1 AND commands.starttime >= $2
		AND actions.name LIKE $3
		AND agents.name LIKE $4
		AND actions.threat->>'family' LIKE $5
		AND commands.status LIKE $6
		ORDER BY commands.id DESC LIMIT $7`,
		p.Before, p.After, p.ActionName, p.AgentName, p.ThreatFamily, p.Status, uint64(p.Limit))
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
			err = fmt.Errorf("Failed to retrieve command: '%v'", err)
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
		commands = append(commands, cmd)
	}
	rows.Close()
	return
}

// SearchActions returns an array of actions that match search parameters
func (db *DB) SearchActions(p SearchParameters) (actions []mig.Action, err error) {
	rows, err := db.c.Query(`SELECT id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion
		FROM actions
		WHERE actions.starttime <= $1 AND actions.starttime >= $2
		AND actions.name LIKE $3
		AND actions.threat->>'family' LIKE $4
		ORDER BY actions.id DESC LIMIT $5`,
		p.Before, p.After, p.ActionName, p.ThreatFamily, uint64(p.Limit))
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
		actions = append(actions, a)
	}
	rows.Close()
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
		ORDER BY agents.heartbeattime LIMIT $5`,
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
			err = fmt.Errorf("Failed to retrieve agent data: '%v'", err)
			return
		}
		agents = append(agents, agent)
	}
	rows.Close()
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
	if err == sql.ErrNoRows {
		rows.Close()
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
		actions = append(actions, a)
	}
	return
}

// ActionByID retrieves an action from the database using its ID
func (db *DB) ActionByID(id float64) (a mig.Action, err error) {
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
	if err == sql.ErrNoRows {
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

// FinishAction updates the action fields to mark it as done
func (db *DB) FinishAction(a mig.Action) (err error) {
	a.FinishTime = time.Now()
	a.Status = "completed"
	_, err = db.c.Exec(`UPDATE actions SET (finishtime, lastupdatetime, status,
		sentctr, returnedctr, donectr, cancelledctr, failedctr, timeoutctr)
		= ($1, $2, $3, $4, $5, $6, $7, $8, $9) WHERE id=$10`,
		a.FinishTime, a.LastUpdateTime, a.Status, a.Counters.Sent, a.Counters.Returned,
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
	if err == sql.ErrNoRows {
		rows.Close()
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
	rows.Close()
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
			err = fmt.Errorf("Failed to retrieve command: '%v'", err)
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
		commands = append(commands, cmd)
	}
	rows.Close()
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
			err = fmt.Errorf("Failed to retrieve agent data: '%v'", err)
			return
		}
		agents = append(agents, agent)
	}
	rows.Close()
	return
}

// InsertAgent creates a new agent in the database
func (db *DB) InsertAgent(agt mig.Agent) (err error) {
	agtid := mig.GenID()
	_, err = db.c.Exec(`INSERT INTO agents
		(id, name, queueloc, os, version, pid, starttime, destructiontime, heartbeattime, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, agtid, agt.Name, agt.QueueLoc,
		agt.OS, agt.Version, agt.PID, agt.StartTime, agt.DestructionTime, agt.HeartBeatTS, agt.Status)
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
		rows.Close()
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
	rows.Close()
	return
}

// ActiveAgentsByTarget finds all agents that have a name, os, queue location or version
// that match a given target string
func (db *DB) ActiveAgentsByTarget(target string, pointInTime time.Time) (agents []mig.Agent, err error) {
	search := fmt.Sprintf("%%%s%%", target)
	rows, err := db.c.Query(`SELECT id, name, queueloc, os, version, pid,
		starttime, destructiontime, heartbeattime, status
		FROM agents
		WHERE agents.heartbeattime >= $1 AND agents.heartbeattime <= NOW()
		AND name ILIKE $2 OR queueloc ILIKE $2
		OR os ILIKE $2 OR version ILIKE $2`, pointInTime, search)
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
	rows.Close()
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
	rows.Close()
	return
}

// NewAgents retrieves a count of agents that started after `pointInTime`
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
