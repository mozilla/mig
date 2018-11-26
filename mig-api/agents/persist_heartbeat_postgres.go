// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

import (
	"time"

	"github.com/mozilla/mig"
	migdb "github.com/mozilla/mig/database"
)

type PersistHeartbeatPostgres struct {
	db *migdb.DB
}

func NewPersistHeartbeatPostgres(db *migdb.DB) PersistHeartbeatPostgres {
	return PersistHeartbeatPostgres{
		db: db,
	}
}

func (persist PersistHeartbeatPostgres) PersistHeartbeat(heartbeat Heartbeat) error {
	agent := heartbeat.ToMigAgent()
	foundAgent, err := persist.db.AgentByQueueAndPID(
		heartbeat.QueueLoc,
		int(heartbeat.PID))

	// If the agent doesn't exist, we want to record it as a new row.
	// This is basically the case where an agent reports itself as operating
	// for the first time.
	if err != nil {
		agent.DestructionTime = time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
		agent.Status = mig.AgtStatusOnline
		agent.StartTime = time.Now()
		return persist.db.InsertAgent(agent, nil)
	}

	agent.Status = mig.AgtStatusOnline
	agent.HeartBeatTS = time.Now()
	agent.RefreshTS = foundAgent.RefreshTS
	agent.Authorized = foundAgent.Authorized
	agent.ID = foundAgent.ID

	cutoff := foundAgent.RefreshTS.Add(15 * time.Second)
	if !foundAgent.RefreshTS.IsZero() && foundAgent.RefreshTS.After(cutoff) {
		return persist.db.ReplaceRefreshedAgent(agent)
	}

	return persist.db.UpdateAgentHeartbeat(agent)
}
