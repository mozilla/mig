// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"time"

	"mig.ninja/mig"

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

	_, err = ctx.MQ.Chan.QueueDeclare(mig.Mq_Q_Heartbeat, true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind(mig.Mq_Q_Heartbeat, mig.Mq_Q_Heartbeat, mig.Mq_Ex_ToSchedulers, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.Qos(0, 0, false)
	if err != nil {
		panic(err)
	}

	heartbeatChan, err = ctx.MQ.Chan.Consume(mig.Mq_Q_Heartbeat, "", true, false, false, false, nil)
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
	// a normal heartbeat should be between 500 and 5000 characters, some may be larger if
	// large environments are collected, so we fix the upper limit to 100kB
	if len(msg.Body) > 102400 {
		panic("discarded heartbeat larger than 100kB")
	}
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
	// replace the heartbeat with current time
	agt.HeartBeatTS = time.Now()
	// do some sanity checking
	if agt.Mode != "" && agt.Mode != "daemon" && agt.Mode != "checkin" {
		panic(fmt.Sprintf("invalid mode '%s' received from agent '%s'", agt.Mode, agt.QueueLoc))
	}
	if len(agt.Name) > 1024 {
		panic(fmt.Sprintf("agent name longer than 1024 characters: name '%s' from '%s'", agt.Name, agt.QueueLoc))
	}
	if len(agt.Version) > 128 {
		panic(fmt.Sprintf("agent version longer than 128 characters: version '%s' from '%s'", agt.Version, agt.QueueLoc))
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
		// send an event to notify workers of the failed agent auth
		err = sendEvent(mig.Ev_Q_Agt_Auth_Fail, msg.Body, ctx)
		if err != nil {
			panic(err)
		}
		// agent authorization failed so we drop this heartbeat and return
		return
	}

	// write to database in a goroutine to avoid blocking
	go func() {
		// if an agent already exists in database, we update it, otherwise we insert it
		agent, err := ctx.DB.AgentByQueueAndPID(agt.QueueLoc, agt.PID)
		if err != nil {
			agt.DestructionTime = time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
			agt.Status = mig.AgtStatusOnline
			// create a new agent, set starttime to now
			agt.StartTime = time.Now()
			err = ctx.DB.InsertAgent(agt, nil)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB insertion failed with error '%v' for agent '%s'", err, agt.Name)}.Err()
			}
			// notify the agt.new event queue
			err = sendEvent(mig.Ev_Q_Agt_New, msg.Body, ctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to send migevent to %s: %v", err, mig.Ev_Q_Agt_New)}.Err()
			}
		} else {
			// the agent exists in database. reuse the existing ID, and keep the status if it was
			// previously set to destroyed, otherwise set status to online
			agt.ID = agent.ID
			if agt.Status == mig.AgtStatusDestroyed {
				agt.Status = agent.Status
			} else {
				agt.Status = mig.AgtStatusOnline
			}
			// If the refresh time is newer than what we know for the agent, replace
			// the agent in the database with the newer information. We want to keep
			// history here, so don't want to just update the information in the
			// existing row.
			//
			// Note: with older agents which might not send a refresh time, the refresh
			// time will be interpreted as the zero value, and the agents should just
			// update using UpdateAgentHeartbeat()
			if agt.RefreshTS.IsZero() {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("agent '%v' not sending refresh time, perhaps an older version?", agt.Name)}.Warning()
			}
			cutoff := agent.RefreshTS.Add(15 * time.Second)
			if !agt.RefreshTS.IsZero() && agt.RefreshTS.After(cutoff) {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("replacing refreshed agent for agent '%v'", agt.Name)}.Info()
				err = ctx.DB.ReplaceRefreshedAgent(agt)
				if err != nil {
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB update failed (refresh) with error '%v' for agent '%s'", err, agt.Name)}.Err()
				}
			} else {
				err = ctx.DB.UpdateAgentHeartbeat(agt)
				if err != nil {
					ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Heartbeat DB update failed with error '%v' for agent '%s'", err, agt.Name)}.Err()
				}
			}
			// if the agent that exists in the database has a status of 'destroyed'
			// we should not be received a heartbeat from it. so, if detectmultiagents
			// is set in the scheduler configuration, we pass the agent queue over to the
			// routine than handles the destruction of agents
			if agent.Status == mig.AgtStatusDestroyed && ctx.Agent.DetectMultiAgents {
				ctx.Channels.DetectDupAgents <- agent.QueueLoc
			}
		}
	}()

	return
}

// startResultsListener initializes the routine that receives heartbeats from agents
func startResultsListener(ctx Context) (resultsChan <-chan amqp.Delivery, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startResultsListener() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startResultsListener()"}.Debug()
	}()

	_, err = ctx.MQ.Chan.QueueDeclare(mig.Mq_Q_Results, true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind(mig.Mq_Q_Results, mig.Mq_Q_Results, mig.Mq_Ex_ToSchedulers, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.Qos(0, 0, false)
	if err != nil {
		panic(err)
	}

	resultsChan, err = ctx.MQ.Chan.Consume(mig.Mq_Q_Results, "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "agents results listener initialized"}

	return
}
