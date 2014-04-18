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

// agentRegId is used to retrieve the internal mongodb ID of an agen'ts registration
type agentRegId struct {
	Id bson.ObjectId `bson:"_id,omitempty"`
}

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
	var ids []agentRegId
	iter := ctx.DB.Col.Reg.Find(bson.M{"name": ka.Name, "os": ka.OS, "queueloc": ka.QueueLoc, "pid": ka.PID, "version": ka.Version}).Iter()
	err = iter.All(&ids)
	if err != nil {
		panic(err)
	}
	switch {
	case len(ids) == 0:
		// no registration for this agent in database, create one
		err = ctx.DB.Col.Reg.Insert(bson.M{"name": ka.Name, "os": ka.OS, "queueloc": ka.QueueLoc,
			"pid": ka.PID, "version": ka.Version, "heartbeatts": ka.HeartBeatTS, "starttime": ka.StartTime})
		if err != nil {
			panic(err)
		}
	case len(ids) == 1:
		// update existing registration for this agent
		mgoId := ids[0].Id
		err := ctx.DB.Col.Reg.Update(bson.M{"_id": mgoId}, bson.M{"$set": bson.M{"heartbeatts": ka.HeartBeatTS}})
		if err != nil {
			panic(err)
		}
	case len(ids) > 1:
		// more than one registration is a problem !
		desc := fmt.Sprintf("%d agents match this registration in database. That's a problem!", len(ids))
		panic(desc)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("getKeepAlives() KeepAlive for Agent '%s' updated in DB", ka.Name)}.Debug()

	// If multiple agents are listening on the same queue, alert the cleanup routine
	agtCnt, _, err := findDupAgents(ka.QueueLoc, ctx)
	if err != nil {
		panic(err)
	}
	if agtCnt > 1 {
		ctx.Channels.DetectDupAgents <- ka.QueueLoc
	}
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
		desc := fmt.Sprintf("Starting listener for agent '%s' on '%s'.", reg.Name, reg.QueueLoc)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
		for msg := range agentChan {
			ctx.OpID = mig.GenID()
			desc := fmt.Sprintf("Received message from agent '%s' on '%s'.", reg.Name, reg.QueueLoc)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
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

// findDupAgents counts agents that are listening on a given queue and
// have sent a heartbeat in recent times, to detect systems that are running
// two or more agents
func findDupAgents(queueLoc string, ctx Context) (count int, agents []mig.KeepAlive, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findDupAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving findDupAgents()"}.Debug()
	}()
	// retrieve agents that have sent in heartbeat in twice their heartbeat time
	period, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
	if err != nil {
		panic(err)
	}
	since := time.Now().Add(-period * 2)
	iter := ctx.DB.Col.Reg.Find(
		bson.M{"heartbeatts": bson.M{"$gte": since}, "queueloc": queueLoc},
	).Iter()
	agents = []mig.KeepAlive{}
	err = iter.All(&agents)
	if err != nil {
		panic(err)
	}
	return len(agents), agents, err
}
