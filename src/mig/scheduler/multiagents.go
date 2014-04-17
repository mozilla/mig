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
	"fmt"
	"mig"
	"time"
)

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
		case "destroyed":
			// if the agent has already been marked as destroyed, check if
			// that was done longer than 2 heartbeats ago. If it did, the
			// destruction failed, and we need to reissue a destruction order
			hbFreq, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
			if err != nil {
				panic(err)
			}
			twoHeartbeats := time.Now().Add(-hbFreq * 2)
			if agent.DestructionTime.Before(twoHeartbeats) {
				err = destroyAgent(agent, ctx)
				if err != nil {
					panic(err)
				}
				destroyedAgents++
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
