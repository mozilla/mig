// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

// Various constants that indicate exchange and queue names used in RabbitMQ
const (
	ExchangeToAgents     = "toagents"
	ExchangeToSchedulers = "toschedulers"
	QueueAgentHeartbeat  = "mig.agt.heartbeats"
	QueueAgentResults    = "mig.agt.results"
)
