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
	"time"

	"github.com/streadway/amqp"
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

	// get a list of all agents that have a keepalive between ctx.Agent.TimeOut and now
	timeOutPeriod, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	pointInTime := time.Now().Add(-timeOutPeriod)
	agents, err := ctx.DB.AgentsActiveSince(pointInTime)
	if err != nil {
		panic(err)
	}

	desc := fmt.Sprintf("pickUpAliveAgents(): found %d active agents", len(agents))
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()

	for _, agt := range agents {
		err = startAgentListener(agt, ctx)
		if err != nil {
			panic(err)
		}
	}
	return
}

// startActiveAgentsChannel initializes the keepalive AMQP queue
func startActiveAgentsChannel(ctx Context) (activeAgentsChan <-chan amqp.Delivery, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startActiveAgentsChannel() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startActiveAgentsChannel()"}.Debug()
	}()

	_, err = ctx.MQ.Chan.QueueDeclare("mig.heartbeat", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind("mig.heartbeat", "mig.heartbeat", "mig", false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.Qos(3, 0, false)
	if err != nil {
		panic(err)
	}

	activeAgentsChan, err = ctx.MQ.Chan.Consume("mig.heartbeat", "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "Active Agents channel initialized"}

	return
}

// getHeartbeats processes the registration messages sent by agents that just
// came online. Such messages are stored in MongoDB and used to locate agents.
func getHeartbeats(msg amqp.Delivery, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getHeartbeats() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving getHeartbeats()"}.Debug()
	}()

	var agt mig.Agent
	err = json.Unmarshal(msg.Body, &agt)
	if err != nil {
		panic(err)
	}
	desc := fmt.Sprintf("Received heartbeat for Agent '%s' OS '%s' QueueLoc '%s'", agt.Name, agt.OS, agt.QueueLoc)
	ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()

	// discard expired heartbeats
	agtTimeOut, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	considerationDate := time.Now().Add(-agtTimeOut)
	if agt.HeartBeatTS.Before(considerationDate) {
		desc = fmt.Sprintf("Expired heartbeat received from Agent '%s'", agt.Name)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Notice()
		return
	}

	// if agent is not authorized, ack the message and skip the registration
	// nothing is returned to the agent. it's simply ignored.
	ok, err := isAgentAuthorized(agt.Name, ctx)
	if err != nil {
		panic(err)
	}
	if !ok {
		desc = fmt.Sprintf("getHeartbeats(): Agent '%s' is not authorized", agt.Name)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Warning()
		return
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Received valid keepalive from agent '%s'", agt.Name)}.Debug()

	// write to database in a goroutine to avoid blocking
	go func() {
		err = ctx.DB.InsertOrUpdateAgent(agt)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB insertion failed with error '%v' for agent '%s'", err, agt.Name)}.Err()
		}
	}()

	// start a listener for this agent, if needed
	err = startAgentListener(agt, ctx)
	if err != nil {
		panic(err)
	}

	// If multiple agents are listening on the same queue, alert the cleanup routine
	if ctx.Agent.DetectMultiAgents {
		agtCnt, _, err := findDupAgents(agt.QueueLoc, ctx)
		if err != nil {
			panic(err)
		}
		if agtCnt > 1 {
			ctx.Channels.DetectDupAgents <- agt.QueueLoc
		}
	}
	return
}

// startAgentsListener will create an AMQP consumer for this agent if none exist
func startAgentListener(agt mig.Agent, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startAgentListener() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startAgentListener()"}.Debug()
	}()

	// If a listener already exists for this agent, exit
	for _, q := range activeAgentsList {
		if q == agt.QueueLoc {
			desc := fmt.Sprintf("startAgentListener() already has a listener for '%s'", agt.QueueLoc)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
			return
		}
	}

	//create a queue for agent
	queue := fmt.Sprintf("mig.sched.%s", agt.QueueLoc)
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
		desc := fmt.Sprintf("Starting listener for agent '%s' on '%s'.", agt.Name, agt.QueueLoc)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
		for msg := range agentChan {
			ctx.OpID = mig.GenID()
			desc := fmt.Sprintf("Received message from agent '%s' on '%s'.", agt.Name, agt.QueueLoc)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()
			err = recvAgentResults(msg, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("%v", err)}.Err()
				// TODO: agent is sending bogus results, do something about it
			}
		}
	}()

	desc := fmt.Sprintf("startAgentactiveAgentsListener: started recvAgentResults goroutine for agent '%s'", agt.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Debug()

	// add the new active queue to the activeAgentsList
	activeAgentsList = append(activeAgentsList, agt.QueueLoc)

	return
}

// findDupAgents counts agents that are listening on a given queue and
// have sent a heartbeat in recent times, to detect systems that are running
// two or more agents
func findDupAgents(queueloc string, ctx Context) (count int, agents []mig.Agent, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("findDupAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving findDupAgents()"}.Debug()
	}()
	// retrieve agents that have sent in heartbeat in twice their heartbeat time
	timeOutPeriod, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
	if err != nil {
		panic(err)
	}
	pointInTime := time.Now().Add(-timeOutPeriod * 2)
	agents, err = ctx.DB.ActiveAgentsByQueue(queueloc, pointInTime)
	if err != nil {
		panic(err)
	}
	return len(agents), agents, err
}
