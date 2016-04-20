// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"mig.ninja/mig"
	"time"
)

// Given an agent queue location queueLoc, send kill actions for duplicate
// agents. Where multiple agents exist on the same host, we will attempt
// to kill the older agents.
func killDupAgents(queueLoc string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("killDupAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving killDupAgents()"}.Debug()
	}()
	hbfreq, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
	if err != nil {
		return err
	}
	pointInTime := time.Now().Add(-hbfreq)
	agents, err := ctx.DB.ActiveAgentsByQueue(queueLoc, pointInTime)
	agentsCount := len(agents)
	if agentsCount < 2 {
		return
	}
	destroyedAgents := 0
	leftAloneAgents := 0
	for _, agent := range agents {
		if agent.Status == "destroyed" {
			// if the agent has already been marked as destroyed, check if
			// that was done longer than 3 heartbeats ago. If it did, the
			// destruction failed, and we need to reissue a destruction order
			hbFreq, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
			if err != nil {
				panic(err)
			}
			pointInTime := time.Now().Add(-hbFreq * 3)
			if agent.DestructionTime.Before(pointInTime) {
				err = issueKillAction(agent, ctx)
				if err != nil {
					panic(err)
				}
				destroyedAgents++
				desc := fmt.Sprintf("Re-issuing destruction action for "+
					"agent '%s' with PID '%d'.", agent.Name, agent.PID)
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
			} else {
				leftAloneAgents++
			}
		}
	}

	remainingAgents := agentsCount - destroyedAgents - leftAloneAgents
	if remainingAgents > 1 {
		// there's still some agents left. if killdupagents is set, issue kill orders
		if ctx.Agent.KillDupAgents {
			oldest := agents[0]
			for _, agent := range agents {
				if agent.Status != "online" {
					continue
				}
				if agent.StartTime.Before(oldest.StartTime) {
					oldest = agent
				}
			}
			desc := fmt.Sprintf("Issuing destruction action for agent '%s' "+
				"with PID '%d'.", oldest.Name, oldest.PID)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}
			err = issueKillAction(oldest, ctx)
			if err != nil {
				panic(err)
			}
			// throttling to prevent issuing too many kill orders at the same time
			time.Sleep(5 * time.Second)
		} else {
			// Build a list of relevant agent names to include in the manual inspection
			// notification
			var namelist string
			for _, agent := range agents {
				if namelist == "" {
					namelist = agent.Name
				} else {
					namelist += ", " + agent.Name
				}
			}
			desc := fmt.Sprintf("found %v agents running on %v. Require "+
				"manual inspection (%v).", remainingAgents, queueLoc, namelist)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Warning()
		}
	}
	return
}

// issueKillAction issues an `agentdestroy` action targeted to a specific agent
// and updates the status of the agent in the database
func issueKillAction(agent mig.Agent, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("issueKillAction() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving issueKillAction()"}.Debug()
	}()
	// generate an `agentdestroy` action for this agent
	killAction := mig.Action{
		ID:            mig.GenID(),
		Name:          fmt.Sprintf("Kill agent %s", agent.Name),
		Target:        fmt.Sprintf("queueloc='%s'", agent.QueueLoc),
		ValidFrom:     time.Now().Add(-60 * time.Second).UTC(),
		ExpireAfter:   time.Now().Add(30 * time.Minute).UTC(),
		SyntaxVersion: 2,
	}
	var opparams struct {
		PID     int    `json:"pid"`
		Version string `json:"version"`
	}
	opparams.PID = agent.PID
	opparams.Version = agent.Version
	killOperation := mig.Operation{
		Module:     "agentdestroy",
		Parameters: opparams,
	}
	killAction.Operations = append(killAction.Operations, killOperation)

	// sign the action with the scheduler PGP key
	secring, err := getSecring(ctx)
	if err != nil {
		panic(err)
	}
	pgpsig, err := killAction.Sign(ctx.PGP.PrivKeyID, secring)
	if err != nil {
		panic(err)
	}
	killAction.PGPSignatures = append(killAction.PGPSignatures, pgpsig)
	var jsonAction []byte
	jsonAction, err = json.Marshal(killAction)
	if err != nil {
		panic(err)
	}

	// write the action to the spool for scheduling
	dest := fmt.Sprintf("%s/%.0f.json", ctx.Directories.Action.New, killAction.ID)
	err = safeWrite(ctx, dest, jsonAction)
	if err != nil {
		panic(err)
	}

	// mark the agent as `destroyed` in the database
	err = ctx.DB.MarkAgentDestroyed(agent)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("issued kill action for agent '%s' "+
		"with PID '%d'", agent.Name, agent.PID)}.Warning()
	return
}
