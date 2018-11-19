// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

import (
	"fmt"

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
	fmt.Printf("POST /heartbeat got heartbeat %v\n", heartbeat)

	agent := heartbeat.ToMigAgent()
	//err := persist.db.InsertAgent(agent, nil)
	agent, err := persist.db.AgentByQueueAndPID(agent.QueueLoc, agent.PID)
	if err != nil {
		return err
	}

	return persist.db.UpdateAgentHeartbeat(agent)
}

// _dontrun invokes a goroutine that updates the agent table when a heartbeat
// message would have been handled by the scheduler. For now we're holding onto
// the code as a reference
func _dontrun() {
	go func() {
		// if an agent already exists in database, we update it, otherwise we insert it
		agent, err := ctx.DB.AgentByQueueAndPID(agt.QueueLoc, agt.PID)
		if err != nil {
			agt.DestructionTime = time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
			agt.Status = mig.AgtStatusOnline
			// create a new agent, set starttime to now
			agt.StartTime = time.Now()
			err = ctx.DB.InsertAgent(agt, nil)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB insertion failed with error '%v' for agent '%s'", err, agt.Name)}.Err()
			}
		} else {
			// the agent exists in database. reuse the existing ID, and keep the status if it was
			// previously set to destroyed, otherwise set status to online
			agt.ID = agent.ID
			if agt.Status == mig.AgtStatusDestroyed {
				agt.Status = agent.Status
			} else {
				agt.Status = mig.AgtStatusOnline
			}
			// If the refresh time is newer than what we know for the agent, replace
			// the agent in the database with the newer information. We want to keep
			// history here, so don't want to just update the information in the
			// existing row.
			//
			// Note: with older agents which might not send a refresh time, the refresh
			// time will be interpreted as the zero value, and the agents should just
			// update using UpdateAgentHeartbeat()
			if agt.RefreshTS.IsZero() {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("agent '%v' not sending refresh time, perhaps an older version?", agt.Name)}.Warning()
			}
			cutoff := agent.RefreshTS.Add(15 * time.Second)
			if !agt.RefreshTS.IsZero() && agt.RefreshTS.After(cutoff) {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("replacing refreshed agent for agent '%v'", agt.Name)}.Info()
				err = ctx.DB.ReplaceRefreshedAgent(agt)
				if err != nil {
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB update failed (refresh) with error '%v' for agent '%s'", err, agt.Name)}.Err()
				}
			} else {
				err = ctx.DB.UpdateAgentHeartbeat(agt)
				if err != nil {
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB update failed with error '%v' for agent '%s'", err, agt.Name)}.Err()
				}
			}
			// if the agent that exists in the database has a status of 'destroyed'
			// we should not be received a heartbeat from it. so, if detectmultiagents
			// is set in the scheduler configuration, we pass the agent queue over to the
			// routine than handles the destruction of agents
			if agent.Status == mig.AgtStatusDestroyed && ctx.Agent.DetectMultiAgents {
				ctx.Channels.DetectDupAgents <- agent.QueueLoc
			}
		}
	}()
}
