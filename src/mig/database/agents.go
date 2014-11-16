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
	"time"

	_ "github.com/lib/pq"
)

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
	rows, err := db.c.Query(`SELECT DISTINCT(agents.queueloc), agents.name FROM agents
		WHERE agents.heartbeattime >= $1 AND agents.heartbeattime <= NOW()
		GROUP BY agents.queueloc, agents.name`, pointInTime)
	if err != nil {
		err = fmt.Errorf("Error while finding agents: '%v'", err)
		return
	}
	for rows.Next() {
		var agent mig.Agent
		err = rows.Scan(&agent.QueueLoc, &agent.Name)
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
	jTags, err := json.Marshal(agt.Tags)
	if err != nil {
		err = fmt.Errorf("Failed to marshal agent tags: '%v'", err)
		return
	}
	agtid := mig.GenID()
	_, err = db.c.Exec(`INSERT INTO agents
		(id, name, queueloc, os, version, pid, starttime, destructiontime,
		heartbeattime, status, environment, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		agtid, agt.Name, agt.QueueLoc, agt.OS, agt.Version, agt.PID,
		agt.StartTime, agt.DestructionTime, agt.HeartBeatTS, agt.Status, jEnv, jTags)
	if err != nil {
		return fmt.Errorf("Failed to insert agent in database: '%v'", err)
	}
	return
}

// UpdateAgentHeartbeat updates the heartbeat timestamp of an agent in the database
func (db *DB) UpdateAgentHeartbeat(agt mig.Agent) (err error) {
	_, err = db.c.Exec(`UPDATE agents
		SET status='online', heartbeattime=$2 WHERE id=$1`, agt.ID, agt.HeartBeatTS)
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
		agt.Status = "online"
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
		WHERE agents.heartbeattime > $1 AND agents.queueloc=$2`, pointInTime, queueloc)
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

// ActiveAgentsByTarget runs a search for all agents that match a given target string.
// For safety, it does so in a transaction that runs as a readonly user.
func (db *DB) ActiveAgentsByTarget(target string) (agents []mig.Agent, err error) {
	// save current user
	var dbuser string
	err = db.c.QueryRow("SELECT CURRENT_USER").Scan(&dbuser)
	if err != nil {
		return
	}
	txn, err := db.c.Begin()
	if err != nil {
		return
	}
	_, err = txn.Exec(`SET ROLE migreadonly`)
	if err != nil {
		_ = txn.Rollback()
		return
	}
	rows, err := txn.Query(`SELECT DISTINCT ON (queueloc) id, name, queueloc, os, version, pid,
		starttime, destructiontime, heartbeattime, status
		FROM agents
		WHERE agents.status = 'online' AND (` + target + `)
		ORDER BY agents.queueloc, agents.heartbeattime DESC`)
	if err != nil {
		_ = txn.Rollback()
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
	_, err = txn.Exec(`SET ROLE ` + dbuser)
	if err != nil {
		_ = txn.Rollback()
		return
	}
	err = txn.Commit()
	if err != nil {
		_ = txn.Rollback()
		return
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
func (db *DB) SumAgentsByVersion() (sum []AgentsSum, err error) {
	rows, err := db.c.Query(`SELECT COUNT(*), version FROM agents
		WHERE agents.status='online' GROUP BY version`)
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
	err = db.c.QueryRow(`SELECT COUNT(DISTINCT(queueloc)) FROM agents
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
func (db *DB) CountDoubleAgents() (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(DISTINCT(queueloc)) FROM agents
		WHERE queueloc IN (
			SELECT queueloc FROM agents
			WHERE heartbeattime >= NOW() - INTERVAL '10 minutes'
			GROUP BY queueloc HAVING count(queueloc) > 1
		)`).Scan(&sum)
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

// MarkOfflineAgents updates the status of agents that have not sent a heartbeat since pointInTime
func (db *DB) MarkOfflineAgents(pointInTime time.Time) (err error) {
	_, err = db.c.Exec(`UPDATE agents SET status='offline'
		WHERE heartbeattime<$1 AND status!='offline'`, pointInTime)
	if err != nil {
		return fmt.Errorf("Failed to mark agents as offline in database: '%v'", err)
	}
	return
}
