// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"mig"
	"os"
	"time"
)

// periodic runs tasks at regular interval
func periodic(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("periodic() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving periodic()"}.Debug()
	}()
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "initiating periodic run"}
	start := time.Now()
	err = cleanDir(ctx, ctx.Directories.Action.Done)
	if err != nil {
		panic(err)
	}
	err = cleanDir(ctx, ctx.Directories.Action.Invalid)
	if err != nil {
		panic(err)
	}
	err = markOfflineAgents(ctx)
	if err != nil {
		panic(err)
	}
	err = markIdleAgents(ctx)
	if err != nil {
		panic(err)
	}
	err = cleanQueueDisappearedEndpoints(ctx)
	if err != nil {
		panic(err)
	}
	err = computeAgentsStats(ctx)
	if err != nil {
		panic(err)
	}
	d := time.Since(start)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("periodic run done in %v", d)}
	return
}

// cleanDir walks through a directory and delete the files that
// are older than the configured DeleteAfter parameter
func cleanDir(ctx Context, targetDir string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("cleanDir() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving cleanDir()"}.Debug()
	}()
	deletionPoint, err := time.ParseDuration(ctx.Periodic.DeleteAfter)
	dir, err := os.Open(targetDir)
	dirContent, err := dir.Readdir(-1)
	if err != nil {
		panic(err)
	}
	// loop over the content of the directory
	for _, DirEntry := range dirContent {
		if !DirEntry.Mode().IsRegular() {
			// ignore non file
			continue
		}
		// if the DeleteAfter value is after the time of last modification,
		// the file is due for deletion
		if time.Now().Add(-deletionPoint).After(DirEntry.ModTime()) {
			filename := targetDir + "/" + DirEntry.Name()
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("removing '%s'", filename)}
			os.Remove(filename)
		}
	}
	dir.Close()
	return
}

// markOfflineAgents updates the status of idle agents that passed the agent timeout to "offline"
func markOfflineAgents(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("markOfflineAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving markOfflineAgents()"}.Debug()
	}()
	timeOutPeriod, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	pointInTime := time.Now().Add(-timeOutPeriod)
	err = ctx.DB.MarkOfflineAgents(pointInTime)
	if err != nil {
		panic(err)
	}
	return
}

// markIdleAgents updates the status of agents that stopped sending heartbeats
func markIdleAgents(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("markIdleAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving markIdleAgents()"}.Debug()
	}()
	hbFreq, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
	if err != nil {
		panic(err)
	}
	pointInTime := time.Now().Add(-hbFreq * 5)
	err = ctx.DB.MarkIdleAgents(pointInTime)
	if err != nil {
		panic(err)
	}
	return
}

// cleanQueueDisappearedEndpoints deletes rabbitmq queues of endpoints that no
// longer have any agent running on them. Only the queues with 0 consumers and 0
// pending messages are deleted.
func cleanQueueDisappearedEndpoints(ctx Context) (err error) {
	start := time.Now()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("cleanQueueDisappearedEndpoints() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving cleanQueueDisappearedEndpoints()"}.Debug()
	}()
	agtTimeout, err := time.ParseDuration(ctx.Agent.TimeOut)
	if err != nil {
		panic(err)
	}
	oldest := time.Now().Add(-agtTimeout * 2)
	queues, err := ctx.DB.GetDisappearedEndpoints(oldest)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("cleanQueueDisappearedEndpoints(): inspecting %d queues from disappeared endpoints", len(queues))}.Debug()
	// create a new channel to do the maintenance. reusing the channel that exists in the
	// context has strange side effect, like reducing the consumption rates of heartbeats to
	// just a few messages per second. Thus, we prefer using a separate throwaway channel.
	amqpchan, err := ctx.MQ.conn.Channel()
	if err != nil {
		panic(err)
	}
	defer amqpchan.Close()
	for _, queue := range queues {
		// the call to inspect will fail if the queue doesn't exist, so we fail silently and continue
		qstat, err := amqpchan.QueueInspect("mig.agt." + queue)
		if err != nil {
			continue
		}
		if qstat.Consumers != 0 || qstat.Messages != 0 {
			desc := fmt.Sprintf("skipped deletion of agent queue %s, it has %d consumers and %d pending messages",
				queue, qstat.Consumers, qstat.Messages)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}
			continue
		}
		_, err = amqpchan.QueueDelete("mig.agt."+queue, true, true, true)
		if err != nil {
			panic(err)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("removed endpoint queue %s", queue)}
	}
	d := time.Since(start)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("cleanQueueDisappearedEndpoints(): done in %v", d)}.Debug()
	return
}

// computeAgentsStats computes and stores statistics about agents and endpoints

// computeAgentsStats computes and stores statistics about agents and endpoints
func computeAgentsStats(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("computeAgentsStats() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving computeAgentsStats()"}.Debug()
	}()
	var stats mig.AgentsStats
	done := make(chan bool)
	go func() {
		start := time.Now()
		stats.OnlineAgentsByVersion, err = ctx.DB.SumOnlineAgentsByVersion()
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("SumOnlineAgentsByVersion() took %v to run", d)}.Debug()
	}()
	go func() {
		start := time.Now()
		stats.IdleAgentsByVersion, err = ctx.DB.SumIdleAgentsByVersion()
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("SumIdleAgentsByVersion() took %v to run", d)}.Debug()
	}()
	go func() {
		start := time.Now()
		stats.OnlineEndpoints, err = ctx.DB.CountOnlineEndpoints()
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountOnlineEndpoints() took %v to run", d)}.Debug()
	}()
	go func() {
		start := time.Now()
		stats.IdleEndpoints, err = ctx.DB.CountIdleEndpoints()
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountIdleEndpoints() took %v to run", d)}.Debug()
	}()

	go func() {
		start := time.Now()
		// detect new endpoints from last 24 hours against endpoints from last 7 days
		stats.NewEndpoints, err = ctx.DB.CountNewEndpoints(time.Now().Add(-24*time.Hour), time.Now().Add(-7*24*time.Hour))
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountNewEndpoints() took %v to run", d)}.Debug()
	}()
	go func() {
		start := time.Now()
		stats.MultiAgentsEndpoints, err = ctx.DB.CountDoubleAgents()
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountDoubleAgents() took %v to run", d)}.Debug()
	}()
	go func() {
		start := time.Now()
		stats.DisappearedEndpoints, err = ctx.DB.CountDisappearedEndpoints(time.Now().Add(-7 * 24 * time.Hour))
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountDisappearedEndpoints() took %v to run", d)}.Debug()
	}()
	go func() {
		start := time.Now()
		stats.FlappingEndpoints, err = ctx.DB.CountFlappingEndpoints()
		if err != nil {
			panic(err)
		}
		done <- true
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountFlappingEndpoints() took %v to run", d)}.Debug()
	}()
	// each query is ran in parallel and return a boolean in the done channel
	// so when we have received 4 messages in the channel, all queries are done
	ctr := 0
	for <-done {
		ctr++
		if ctr == 8 {
			break
		}
	}
	for _, asum := range stats.OnlineAgentsByVersion {
		stats.OnlineAgents += asum.Count
	}
	for _, asum := range stats.IdleAgentsByVersion {
		stats.IdleAgents += asum.Count
	}
	err = ctx.DB.StoreAgentsStats(stats)
	if err != nil {
		panic(err)
	}
	return
}
