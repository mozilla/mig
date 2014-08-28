// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"mig"
	"mig/pgp/sign"
	"os"
	"reflect"
	"time"
)

// Check the action that was processed, and if it's related to upgrading agents
// extract the command results, grab the PID of the agents that was upgraded,
// and mark the agent registration in the database as 'upgraded'
func markUpgradedAgents(cmd mig.Command, ctx Context) {
	var err error
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: fmt.Sprintf("markUpgradedAgents() failed with error '%v'", e)}.Err()
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: "leaving markUpgradedAgents()"}.Debug()
	}()
	for _, operation := range cmd.Action.Operations {
		if operation.Module == "upgrade" {
			for _, result := range cmd.Results {
				reflection := reflect.ValueOf(result)
				resultMap := reflection.Interface().(map[string]interface{})
				_, ok := resultMap["success"]
				if !ok {
					ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID,
						Desc: "Invalid operation results format. Missing 'success' key."}.Err()
					panic(err)
				}
				success := reflect.ValueOf(resultMap["success"])
				if !success.Bool() {
					ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID,
						Desc: "Upgrade operation failed. Agent not marked."}.Err()
					panic(err)
				}

				_, ok = resultMap["oldpid"]
				if !ok {
					ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID,
						Desc: "Invalid operation results format. Missing 'oldpid' key."}.Err()
					panic(err)
				}
				oldpid := reflect.ValueOf(resultMap["oldpid"])
				if oldpid.Float() < 2 || oldpid.Float() > 65535 {
					desc := fmt.Sprintf("Successfully found upgraded action on agent '%s', but with PID '%s'. That's not right...",
						cmd.Agent.Name, oldpid)
					ctx.Channels.Log <- mig.Log{Desc: desc}.Err()
					panic(desc)
				}
				// update the agent's registration to mark it as upgraded
				agent, err := ctx.DB.AgentByQueueAndPID(cmd.Agent.QueueLoc, int(oldpid.Float()))
				if err != nil {
					panic(err)
				}
				err = ctx.DB.MarkAgentUpgraded(agent)
				if err != nil {
					panic(err)
				}
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID,
					Desc: fmt.Sprintf("Agent '%s' marked as upgraded", cmd.Agent.Name)}.Info()
			}
		}
	}
	return
}

// inspectMultiAgents takes a number of actions when several agents are found
// to be listening on the same queue. It will trigger an agentdestroy action
// for agents that are flagged as upgraded, and log alerts for agents that
// are not, such that an investigator can look at them.
func inspectMultiAgents(queueLoc string, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("inspectMultiAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving inspectMultiAgents()"}.Debug()
	}()
	agentsCount, agents, err := findDupAgents(queueLoc, ctx)
	if agentsCount < 2 {
		return
	}
	destroyedAgents := 0
	leftAloneAgents := 0
	for _, agent := range agents {
		switch agent.Status {
		case "upgraded":
			// upgraded agents must die
			err = destroyAgent(agent, ctx)
			if err != nil {
				panic(err)
			}
			destroyedAgents++
			desc := fmt.Sprintf("Agent '%s' with PID '%d' has been upgraded and will be destroyed.", agent.Name, agent.PID)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
		case "destroyed":
			// if the agent has already been marked as destroyed, check if
			// that was done longer than 3 heartbeats ago. If it did, the
			// destruction failed, and we need to reissue a destruction order
			hbFreq, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
			if err != nil {
				panic(err)
			}
			pointInTime := time.Now().Add(-hbFreq * 3)
			if agent.DestructionTime.Before(pointInTime) {
				err = destroyAgent(agent, ctx)
				if err != nil {
					panic(err)
				}
				destroyedAgents++
				desc := fmt.Sprintf("Re-issuing destruction action for agent '%s' with PID '%d'.", agent.Name, agent.PID)
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
			} else {
				leftAloneAgents++
			}
		}
	}

	remainingAgents := agentsCount - destroyedAgents - leftAloneAgents
	if remainingAgents > 1 {
		// there's still some agents left, raise errors for these
		desc := fmt.Sprintf("Found '%d' agents running on '%s'. Require manual inspection.", remainingAgents, queueLoc)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Warning()
	}
	return
}

// destroyAgent issues an `agentdestroy` action targetted to a specific agent
// and updates the status of the agent in the database
func destroyAgent(agent mig.Agent, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("destroyAgent() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving destroyAgent()"}.Debug()
	}()
	// generate an `agentdestroy` action for this agent
	killAction := mig.Action{
		ID:            mig.GenID(),
		Name:          fmt.Sprintf("Destroy agent %s", agent.Name),
		Target:        agent.QueueLoc,
		ValidFrom:     time.Now().Add(-60 * time.Second).UTC(),
		ExpireAfter:   time.Now().Add(30 * time.Minute).UTC(),
		SyntaxVersion: 1,
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
	str, err := killAction.String()
	if err != nil {
		panic(err)
	}
	secringFile, err := os.Open(ctx.PGP.Home + "/secring.gpg")
	defer secringFile.Close()
	pgpsig, err := sign.Sign(str, ctx.PGP.KeyID, secringFile)
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
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Requested destruction of agent '%s' with PID '%d'", agent.Name, agent.PID)}.Info()
	return
}
