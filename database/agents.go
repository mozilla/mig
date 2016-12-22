// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package database /* import "mig.ninja/mig/database" */

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mig.ninja/mig"

	_ "github.com/lib/pq"
)

// AgentByQueueAndPID returns a single agent that is located at a given queueloc and has a given PID
func (db *DB) AgentByQueueAndPID(queueloc string, pid int) (agent mig.Agent, err error) {
	err = db.c.QueryRow(`SELECT id, name, queueloc, mode, version, pid, starttime, heartbeattime,
		refreshtime, status FROM agents WHERE queueloc=$1 AND pid=$2 AND status!=$3`,
		queueloc, pid, mig.AgtStatusOffline).Scan(
		&agent.ID, &agent.Name, &agent.QueueLoc, &agent.Mode, &agent.Version, &agent.PID,
		&agent.StartTime, &agent.HeartBeatTS, &agent.RefreshTS, &agent.Status)
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
	var jTags, jEnv []byte
	err = db.c.QueryRow(`SELECT id, name, queueloc, mode, version, pid, starttime, heartbeattime,
		refreshtime, status, tags, environment FROM agents WHERE id=$1`, id).Scan(
		&agent.ID, &agent.Name, &agent.QueueLoc, &agent.Mode, &agent.Version, &agent.PID,
		&agent.StartTime, &agent.HeartBeatTS, &agent.RefreshTS, &agent.Status,
		&jTags, &jEnv)
	if err != nil {
		err = fmt.Errorf("Error while retrieving agent: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	err = json.Unmarshal(jTags, &agent.Tags)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal agent tags")
		return
	}
	err = json.Unmarshal(jEnv, &agent.Env)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal agent environment")
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
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while finding agents: '%v'", err)
		return
	}
	for rows.Next() {
		var agent mig.Agent
		err = rows.Scan(&agent.QueueLoc, &agent.Name)
		if err != nil {
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
//
// If useTx is not nil, the transaction will be used instead of the standard
// connection
func (db *DB) InsertAgent(agt mig.Agent, useTx *sql.Tx) (err error) {
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
	if useTx != nil {
		_, err = useTx.Exec(`INSERT INTO agents
		(id, name, queueloc, mode, version, pid, starttime, destructiontime,
		heartbeattime, refreshtime, status, environment, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			agtid, agt.Name, agt.QueueLoc, agt.Mode, agt.Version, agt.PID,
			agt.StartTime, agt.DestructionTime, agt.HeartBeatTS, agt.RefreshTS,
			agt.Status, jEnv, jTags)
	} else {
		_, err = db.c.Exec(`INSERT INTO agents
		(id, name, queueloc, mode, version, pid, starttime, destructiontime,
		heartbeattime, refreshtime, status, environment, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			agtid, agt.Name, agt.QueueLoc, agt.Mode, agt.Version, agt.PID,
			agt.StartTime, agt.DestructionTime, agt.HeartBeatTS, agt.RefreshTS,
			agt.Status, jEnv, jTags)
	}
	if err != nil {
		return fmt.Errorf("Failed to insert agent in database: '%v'", err)
	}
	return
}

// UpdateAgentHeartbeat updates the heartbeat timestamp of an agent in the database
// unless the agent has been marked as destroyed
func (db *DB) UpdateAgentHeartbeat(agt mig.Agent) (err error) {
	_, err = db.c.Exec(`UPDATE agents SET status=$1, heartbeattime=$2 WHERE id=$3`,
		mig.AgtStatusOnline, agt.HeartBeatTS, agt.ID)
	if err != nil {
		return fmt.Errorf("Failed to update agent in database: '%v'", err)
	}
	return
}

// Replace an existing agent in the database with newer environment information. This
// should be called when we receive a heartbeat for an active agent, but the refresh
// time indicates newer environment information exists.
func (db *DB) ReplaceRefreshedAgent(agt mig.Agent) (err error) {
	// Do this in a transaction to ensure other parts of the scheduler don't
	// pick up invalid information
	tx, err := db.c.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(`UPDATE agents SET status=$1 WHERE id=$2`,
		mig.AgtStatusOffline, agt.ID)
	if err != nil {
		_ = tx.Rollback()
		return
	}
	err = db.InsertAgent(agt, tx)
	if err != nil {
		_ = tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		return
	}
	return
}

// ListMultiAgentsQueues retrieves an array of queues that have more than one active agent
func (db *DB) ListMultiAgentsQueues(pointInTime time.Time) (queues []string, err error) {
	rows, err := db.c.Query(`SELECT queueloc FROM agents
		WHERE heartbeattime > $1 AND mode != 'checkin'
		GROUP BY queueloc HAVING COUNT(queueloc) > 1`, pointInTime)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while listing multi agents queues: '%v'", err)
		return
	}
	for rows.Next() {
		var q string
		err = rows.Scan(&q)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve agent queue: '%v'", err)
			return
		}
		queues = append(queues, q)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// ActiveAgentsByQueue retrieves an array of agents identified by their QueueLoc value
func (db *DB) ActiveAgentsByQueue(queueloc string, pointInTime time.Time) (agents []mig.Agent, err error) {
	rows, err := db.c.Query(`SELECT id, name, queueloc, mode, version, pid, starttime,
		heartbeattime, refreshtime, status
		FROM agents WHERE agents.heartbeattime > $1 AND agents.queueloc=$2
		AND agents.status!=$3`,
		pointInTime, queueloc, mig.AgtStatusOffline)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while finding agents: '%v'", err)
		return
	}
	for rows.Next() {
		var agent mig.Agent
		err = rows.Scan(&agent.ID, &agent.Name, &agent.QueueLoc, &agent.Mode, &agent.Version,
			&agent.PID, &agent.StartTime, &agent.HeartBeatTS,
			&agent.RefreshTS, &agent.Status)
		if err != nil {
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
	var jTags, jEnv []byte
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
	rows, err := txn.Query(fmt.Sprintf(`SELECT DISTINCT ON (queueloc) id, name, queueloc,
		version, pid, starttime, destructiontime, heartbeattime, refreshtime, status,
		mode, environment, tags
		FROM agents WHERE agents.status IN ('%s', '%s') AND (%s)
		ORDER BY agents.queueloc ASC`, mig.AgtStatusOnline, mig.AgtStatusIdle, target))
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		_ = txn.Rollback()
		err = fmt.Errorf("Error while finding agents: '%v'", err)
		return
	}
	for rows.Next() {
		var agent mig.Agent
		err = rows.Scan(&agent.ID, &agent.Name, &agent.QueueLoc, &agent.Version,
			&agent.PID, &agent.StartTime, &agent.DestructionTime, &agent.HeartBeatTS,
			&agent.RefreshTS, &agent.Status, &agent.Mode, &jEnv, &jTags)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve agent data: '%v'", err)
			return
		}
		err = json.Unmarshal(jTags, &agent.Tags)
		if err != nil {
			err = fmt.Errorf("failed to unmarshal agent tags")
			return
		}
		err = json.Unmarshal(jEnv, &agent.Env)
		if err != nil {
			err = fmt.Errorf("failed to unmarshal agent environment")
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

// MarkAgentDestroyed updated the status and destructiontime of an agent in the database
func (db *DB) MarkAgentDestroyed(agent mig.Agent) (err error) {
	agent.DestructionTime = time.Now()
	_, err = db.c.Exec(`UPDATE agents
		SET destructiontime=$1, status=$2 WHERE id=$3`,
		agent.DestructionTime, mig.AgtStatusDestroyed, agent.ID)
	if err != nil {
		return fmt.Errorf("Failed to mark agent as destroyed in database: '%v'", err)
	}
	return
}

// GetAgentsStats retrieves the latest agents statistics. limit controls how many rows
// of statistics are returned
func (db *DB) GetAgentsStats(limit int) (stats []mig.AgentsStats, err error) {
	rows, err := db.c.Query(`SELECT timestamp, online_agents, online_agents_by_version,
		online_endpoints, idle_agents, idle_agents_by_version, idle_endpoints, new_endpoints,
		multi_agents_endpoints, disappeared_endpoints, flapping_endpoints
		FROM agents_stats ORDER BY timestamp DESC LIMIT $1`, limit)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while retrieving agent statistics: '%v'", err)
		return
	}
	for rows.Next() {
		var jOnlAgtVer, jIdlAgtVer []byte
		var s mig.AgentsStats
		err = rows.Scan(&s.Timestamp, &s.OnlineAgents, &jOnlAgtVer, &s.OnlineEndpoints,
			&s.IdleAgents, &jIdlAgtVer, &s.IdleEndpoints, &s.NewEndpoints,
			&s.MultiAgentsEndpoints, &s.DisappearedEndpoints, &s.FlappingEndpoints)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve agent statistics data: '%v'", err)
			return
		}
		err = json.Unmarshal(jOnlAgtVer, &s.OnlineAgentsByVersion)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal online agent by version statistics: '%v'", err)
			return
		}
		err = json.Unmarshal(jIdlAgtVer, &s.IdleAgentsByVersion)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal idle agent by version statistics: '%v'", err)
			return
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// StoreAgentsStats store a new row of agents statistics and sets the timestamp to the current time
func (db *DB) StoreAgentsStats(stats mig.AgentsStats) (err error) {
	jOnlAgtVer, err := json.Marshal(stats.OnlineAgentsByVersion)
	if err != nil {
		err = fmt.Errorf("Failed to marshal online agents by version: '%v'", err)
		return
	}
	jIdlAgtVer, err := json.Marshal(stats.IdleAgentsByVersion)
	if err != nil {
		err = fmt.Errorf("Failed to marshal idle agents by version: '%v'", err)
		return
	}
	_, err = db.c.Exec(`INSERT INTO agents_stats
		(timestamp, online_agents, online_agents_by_version, online_endpoints,
		idle_agents, idle_agents_by_version, idle_endpoints, new_endpoints,
		multi_agents_endpoints, disappeared_endpoints, flapping_endpoints)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		time.Now().UTC(), stats.OnlineAgents, jOnlAgtVer, stats.OnlineEndpoints,
		stats.IdleAgents, jIdlAgtVer, stats.IdleEndpoints, stats.NewEndpoints,
		stats.MultiAgentsEndpoints, stats.DisappearedEndpoints, stats.FlappingEndpoints)
	if err != nil {
		return fmt.Errorf("Failed to insert agent statistics in database: '%v'", err)
	}
	return
}

// SumOnlineAgentsByVersion retrieves a sum of online agents grouped by version
func (db *DB) SumOnlineAgentsByVersion() (sum []mig.AgentsVersionsSum, err error) {
	rows, err := db.c.Query(`SELECT COUNT(*), version FROM agents
		WHERE agents.status=$1 GROUP BY version`, mig.AgtStatusOnline)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while counting agents: '%v'", err)
		return
	}
	for rows.Next() {
		var asum mig.AgentsVersionsSum
		err = rows.Scan(&asum.Count, &asum.Version)
		if err != nil {
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

// SumIdleAgentsByVersion retrieves a sum of idle agents grouped by version
// and excludes endpoints where an online agent is running
func (db *DB) SumIdleAgentsByVersion() (sum []mig.AgentsVersionsSum, err error) {
	rows, err := db.c.Query(`SELECT COUNT(*), version FROM agents
		WHERE agents.status=$1 AND agents.queueloc NOT IN (
			SELECT distinct(queueloc) FROM agents
			WHERE agents.status=$2)
		GROUP BY version`, mig.AgtStatusIdle, mig.AgtStatusOnline)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while counting agents: '%v'", err)
		return
	}
	for rows.Next() {
		var asum mig.AgentsVersionsSum
		err = rows.Scan(&asum.Count, &asum.Version)
		if err != nil {
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

// CountOnlineEndpoints retrieves a count of unique endpoints that have online agents
func (db *DB) CountOnlineEndpoints() (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(DISTINCT(queueloc)) FROM agents WHERE status=$1`,
		mig.AgtStatusOnline).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting online endpoints: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// CountIdleEndpoints retrieves a count of unique endpoints that have idle agents
// and do not have an online agent
func (db *DB) CountIdleEndpoints() (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(*) FROM (
			SELECT queueloc FROM agents
			WHERE status=$1
			AND queueloc NOT IN (
				SELECT queueloc FROM agents
				WHERE status=$2
				GROUP BY queueloc
			) GROUP BY queueloc
		) AS idleendpoints`, mig.AgtStatusIdle, mig.AgtStatusOnline).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting idle endpoints: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// CountNewEndpointsretrieves a count of new endpoints that started after `pointInTime`
func (db *DB) CountNewEndpoints(recent, old time.Time) (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(*) FROM (
				SELECT queueloc FROM agents
				WHERE queueloc NOT IN (
					SELECT queueloc FROM agents
					WHERE heartbeattime > $2
					AND heartbeattime < $1
					GROUP BY queueloc
				)
				AND starttime > $1
				GROUP BY queueloc
			)AS newendpoints`, recent, old).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting new endpoints: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// CountDoubleAgents counts the number of endpoints that run more than one agent
func (db *DB) CountDoubleAgents() (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(*) FROM (
			SELECT queueloc FROM agents
			WHERE queueloc IN (
				SELECT queueloc FROM agents
				WHERE status=$1
				GROUP BY queueloc
				HAVING count(queueloc) > 1
			)
			GROUP BY queueloc
		) AS doubleagents`, mig.AgtStatusOnline).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting double agents: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// CountDisappearedEndpoints a count of endpoints that have disappeared over a given period
func (db *DB) CountDisappearedEndpoints(pointInTime time.Time) (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(*) FROM (
			SELECT queueloc FROM agents
			WHERE queueloc NOT IN (
				SELECT queueloc FROM agents
				WHERE status=$1 OR status=$2
				GROUP BY queueloc
			)
			AND heartbeattime > $3
			GROUP BY queueloc) AS disappeared`,
		mig.AgtStatusIdle, mig.AgtStatusOnline, pointInTime).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting disappeared endpoints: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// GetDisappearedEndpoints retrieves a list of queues from endpoints that are no longer active
func (db *DB) GetDisappearedEndpoints(oldest time.Time) (queues []string, err error) {
	rows, err := db.c.Query(`SELECT queueloc FROM agents
		WHERE status='offline' AND heartbeattime > $1 AND queueloc NOT IN (
			SELECT queueloc FROM agents
			WHERE status='idle' or status='online'
			GROUP BY queueloc
			)
		GROUP BY queueloc`, oldest)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while retrieving disappeared endpoints: '%v'", err)
		return
	}
	for rows.Next() {
		var q string
		err = rows.Scan(&q)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve endpoint queue: '%v'", err)
			return
		}
		queues = append(queues, q)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// CountFlappingEndpoints a count of endpoints that have restarted their agent recently
func (db *DB) CountFlappingEndpoints() (sum float64, err error) {
	err = db.c.QueryRow(`SELECT COUNT(*) FROM (
				SELECT queueloc FROM agents
				WHERE status=$1 OR status=$2
				GROUP BY queueloc
				HAVING count(queueloc) > 1
			) AS flapping`, mig.AgtStatusOnline, mig.AgtStatusIdle).Scan(&sum)
	if err != nil {
		err = fmt.Errorf("Error while counting flapping endpoints: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// MarkOfflineAgents updates the status of idle agents that have not sent a heartbeat since pointInTime
func (db *DB) MarkOfflineAgents(pointInTime time.Time) (err error) {
	_, err = db.c.Exec(`UPDATE agents SET status=$1
		WHERE heartbeattime<$2 AND status=$3`,
		mig.AgtStatusOffline, pointInTime, mig.AgtStatusIdle)
	if err != nil {
		return fmt.Errorf("Failed to mark agents as offline in database: '%v'", err)
	}
	return
}

// MarkIdleAgents updates the status of online agents that have not sent a heartbeat since pointInTime
func (db *DB) MarkIdleAgents(pointInTime time.Time) (err error) {
	_, err = db.c.Exec(`UPDATE agents SET status=$1
		WHERE heartbeattime<$2 AND status=$3`,
		mig.AgtStatusIdle, pointInTime, mig.AgtStatusOnline)
	if err != nil {
		return fmt.Errorf("Failed to mark agents as idle in database: '%v'", err)
	}
	return
}
