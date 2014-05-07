/* Mozilla InvestiGator Scheduler

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
Portions created by the Initial Developer are Copyright (C) 2013
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

package main

import (
	"encoding/json"
	"fmt"
	"mig"
	"mig/pgp/sign"
	"reflect"
	"time"
)

// Check the action that was processed, and if it's related to upgrading agents
// extract the command results, grab the PID of the agents that was upgraded,
// and mark the agent registration in the database as 'upgraded'
func markUpgradedAgents(cmd mig.Command, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("markUpgradedAgents() -> %v", e)
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
				if oldpid.Int() < 2 {
					desc := fmt.Sprintf("Successfully found upgraded action on agent '%s', but with PID '%s'. That's not right...",
						cmd.Agent.Name, oldpid)
					ctx.Channels.Log <- mig.Log{Desc: desc}.Err()
					panic(desc)
				}

				// update the agent's registration to mark it as upgraded
				agent, err := ctx.DB.AgentByQueueAndPID(cmd.Agent.QueueLoc, int(oldpid.Int()))
				if err != nil {
					panic(err)
				}
				agent.Status = "upgraded"
				err = ctx.DB.InsertOrUpdateAgent(agent)
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
	pgpsig, err := sign.Sign(str, ctx.PGP.KeyID)
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
	dest := fmt.Sprintf("%s/%s", ctx.Directories.Action.New, mig.GenID())
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
