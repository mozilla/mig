// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

const (
	// rabbitmq exchanges and common queues
	Mq_Ex_ToAgents     = "toagents"
	Mq_Ex_ToSchedulers = "toschedulers"
	Mq_Ex_ToWorkers    = "toworkers"
	Mq_Q_Heartbeat     = "mig.agt.heartbeats"
	Mq_Q_Results       = "mig.agt.results"

	// event queues
	Ev_Q_Agt_Auth_Fail = "agent.authentication.failure"
	Ev_Q_Agt_New       = "agent.new"
	Ev_Q_Cmd_Res       = "command.results"

	// dummy queue for scheduler heartbeats to the relays
	Ev_Q_Sched_Hb = "scheduler.heartbeat"
)
