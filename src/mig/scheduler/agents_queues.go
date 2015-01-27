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

// startHeartbeatsListener initializes the routine that receives heartbeats from agents
func startHeartbeatsListener(ctx Context) (heartbeatChan <-chan amqp.Delivery, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startHeartbeatsListener() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startHeartbeatsListener()"}.Debug()
	}()

	_, err = ctx.MQ.Chan.QueueDeclare("mig.agt.heartbeats", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind("mig.agt.heartbeats", "mig.agt.heartbeats", "mig", false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.Qos(0, 0, false)
	if err != nil {
		panic(err)
	}

	heartbeatChan, err = ctx.MQ.Chan.Consume("mig.agt.heartbeats", "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "agents heartbeats listener initialized"}

	return
}

// getHeartbeats processes the heartbeat messages sent by agents
func getHeartbeats(msg amqp.Delivery, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getHeartbeats() -> %v", e)
		}
		if ctx.Debug.Heartbeats {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving getHeartbeats()"}.Debug()
		}
	}()

	var agt mig.Agent
	err = json.Unmarshal(msg.Body, &agt)
	if err != nil {
		panic(err)
	}
	if ctx.Debug.Heartbeats {
		desc := fmt.Sprintf("Received heartbeat for Agent '%s' QueueLoc '%s'", agt.Name, agt.QueueLoc)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Debug()
	}
	// discard expired heartbeats
	agtTimeOut, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	expirationDate := time.Now().Add(-agtTimeOut)
	if agt.HeartBeatTS.Before(expirationDate) {
		desc := fmt.Sprintf("Expired heartbeat received from Agent '%s'", agt.Name)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Notice()
		return
	}
	// if agent is not authorized, ack the message and skip the registration
	// nothing is returned to the agent. it's simply ignored.
	ok, err := isAgentAuthorized(agt.QueueLoc, ctx)
	if err != nil {
		panic(err)
	}
	if !ok {
		desc := fmt.Sprintf("getHeartbeats(): Agent '%s' is not authorized", agt.QueueLoc)
		ctx.Channels.Log <- mig.Log{Desc: desc}.Warning()
		return
	}

	// write to database in a goroutine to avoid blocking
	go func() {
		err = ctx.DB.InsertOrUpdateAgent(agt)
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB insertion failed with error '%v' for agent '%s'", err, agt.Name)}.Err()
		}
	}()

	// If multiple agents are active at the same time, alert the cleanup routine
	if ctx.Agent.DetectMultiAgents {
		go func() {
			agtCnt, _, err := findDupAgents(agt.QueueLoc, ctx)
			if err != nil {
				panic(err)
			}
			if agtCnt > 1 {
				ctx.Channels.DetectDupAgents <- agt.QueueLoc
			}
		}()
	}
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
		if ctx.Debug.Heartbeats {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving findDupAgents()"}.Debug()
		}
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

// startResultsListener initializes the routine that receives heartbeats from agents
func startResultsListener(ctx Context) (resultsChan <-chan amqp.Delivery, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startResultsListener() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startResultsListener()"}.Debug()
	}()

	_, err = ctx.MQ.Chan.QueueDeclare("mig.agt.results", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind("mig.agt.results", "mig.agt.results", "mig", false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.Qos(0, 0, false)
	if err != nil {
		panic(err)
	}

	resultsChan, err = ctx.MQ.Chan.Consume("mig.agt.results", "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "agents results listener initialized"}

	return
}
