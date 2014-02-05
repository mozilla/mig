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
Portions created by the Initial Developer are Copyright (C) 2014
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
	"github.com/streadway/amqp"
	"labix.org/v2/mgo/bson"
	"mig"
	"time"
)

// pickUpAliveAgents lists agents that have recent keepalive in the
// database, and start listening queues for them
func pickUpAliveAgents(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("pickUpAliveAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving pickUpAliveAgents()"}.Debug()
	}()

	agents := []mig.KeepAlive{}
	// get a list of all agents that have a keepalive between ctx.Agent.TimeOut and now
	period, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}

	since := time.Now().Add(-period)
	iter := ctx.DB.Col.Reg.Find(bson.M{"heartbeatts": bson.M{"$gte": since}}).Iter()
	err = iter.All(&agents)
	if err != nil {
		panic(err)
	}

	desc := fmt.Sprintf("pickUpAliveAgents(): found %d agents to listen to", len(agents))
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()

	for _, agt := range agents {
		err = startAgentListener(agt, ctx)
		if err != nil {
			panic(err)
		}
	}
	return
}

// startKeepAliveChannel initializes the keepalive AMQP queue
func startKeepAliveChannel(ctx Context) (keepaliveChan <-chan amqp.Delivery, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startKeepAliveChannel() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startKeepAliveChannel()"}.Debug()
	}()

	_, err = ctx.MQ.Chan.QueueDeclare("mig.keepalive", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind("mig.keepalive", "mig.keepalive", "mig", false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.Qos(3, 0, false)
	if err != nil {
		panic(err)
	}

	keepaliveChan, err = ctx.MQ.Chan.Consume("mig.keepalive", "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "KeepAlive channel initialized"}

	return
}

// getKeepAlives processes the registration messages sent by agents that just
// came online. Such messages are stored in MongoDB and used to locate agents.
func getKeepAlives(msg amqp.Delivery, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getKeepAlives() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving getKeepAlives()"}.Debug()
	}()

	var ka mig.KeepAlive
	err = json.Unmarshal(msg.Body, &ka)
	if err != nil {
		panic(err)
	}

	// log new keepalive message
	desc := fmt.Sprintf("getKeepAlives() Agent '%s' OS '%s' QueueLoc '%s'", ka.Name, ka.OS, ka.QueueLoc)
	ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()

	// discard old keepalives that stay stuck in mongodb. This can happen
	// if the scheduler is down for a period of time, and agents keep sending keepalives
	agtTimeOut, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}

	// check that the keepalive isn't an old one. If that's the case, exit the function
	// without saving the keepalive
	heartbeatTimeWindow := time.Now().Add(-agtTimeOut)
	if ka.HeartBeatTS.Before(heartbeatTimeWindow) {
		desc = fmt.Sprintf("getKeepAlives() Expired keepalive received from Agent '%s'", ka.Name)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Notice()
		return
	}

	// if agent is not authorized to keepAlive, ack the message and skip the registration
	// nothing is returned to the agent. it's simply ignored.
	ok, err := isAgentAuthorized(ka.Name, ctx)
	if err != nil {
		panic(err)
	}
	if !ok {
		desc = fmt.Sprintf("getKeepAlives(): Agent '%s' is not authorized", ka.Name)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Warning()
		return
	}

	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("getKeepAlives() received valid keepalive from agent '%s'", ka.Name)}.Debug()

	// start a listener for this agent, if needed
	err = startAgentListener(ka, ctx)
	if err != nil {
		panic(err)
	}

	// try to find an existing entry to update, or create a new one
	// and save registration in database
	_, err = ctx.DB.Col.Reg.Upsert(
		// search string
		bson.M{"name": ka.Name, "os": ka.OS, "queueloc": ka.QueueLoc},
		// update string
		bson.M{"name": ka.Name, "os": ka.OS, "queueloc": ka.QueueLoc,
			"heartbeatts": ka.HeartBeatTS, "starttime": ka.StartTime})
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("getKeepAlives() KeepAlive for Agent '%s' updated in DB", ka.Name)}.Debug()

	return
}

// startAgentsListener will create an AMQP consumer for this agent if none exist
func startAgentListener(reg mig.KeepAlive, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startAgentListener() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startAgentListener()"}.Debug()
	}()

	// If a listener already exists for this agent, exit
	for _, q := range activeAgentsList {
		if q == reg.QueueLoc {
			desc := fmt.Sprintf("startAgentListener() already has a listener for '%s'", reg.QueueLoc)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
			return
		}
	}

	//create a queue for agent
	queue := fmt.Sprintf("mig.sched.%s", reg.QueueLoc)
	_, err = ctx.MQ.Chan.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind(queue, queue, "mig", false, nil)
	if err != nil {
		panic(err)
	}

	agentChan, err := ctx.MQ.Chan.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// start a goroutine for this queue
	go func() {
		for msg := range agentChan {
			ctx.OpID = mig.GenID()
			err = recvAgentResults(msg, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
				// TODO: agent is sending bogus results, do something about it
			}
		}
	}()

	desc := fmt.Sprintf("startAgentactiveAgentsListener: started recvAgentResults goroutine for agent '%s'", reg.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()

	// add the new active queue to the activeAgentsList
	activeAgentsList = append(activeAgentsList, reg.QueueLoc)

	return
}
