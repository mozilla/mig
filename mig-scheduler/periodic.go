// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"mig.ninja/mig"
	"os"
	"sync"
	"time"
)

// periodic runs tasks at regular interval
func periodic(ctx Context) (err error) {
	start := time.Now()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("periodic() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving periodic()"}.Debug()
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("periodic run done in %v", d)}
	}()
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "initiating periodic run"}
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
	err = computeAgentsStats(ctx)
	if err != nil {
		panic(err)
	}
	err = detectMultiAgents(ctx)
	if err != nil {
		panic(err)
	}
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
	if err != nil {
		panic(err)
	}
	dir, err := os.Open(targetDir)
	if err != nil {
		panic(err)
	}
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

// save time of last hourly run
var countNewEndpointsHourly time.Time

// computeAgentsStats computes and stores statistics about agents and endpoints
func computeAgentsStats(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("computeAgentsStats() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving computeAgentsStats()"}.Debug()
	}()
	var (
		stats mig.AgentsStats
		wg    sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		stats.OnlineAgentsByVersion, err = ctx.DB.SumOnlineAgentsByVersion()
		if err != nil {
			panic(err)
		}
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("SumOnlineAgentsByVersion() took %v to run", d)}.Debug()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		stats.IdleAgentsByVersion, err = ctx.DB.SumIdleAgentsByVersion()
		if err != nil {
			panic(err)
		}
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("SumIdleAgentsByVersion() took %v to run", d)}.Debug()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		stats.OnlineEndpoints, err = ctx.DB.CountOnlineEndpoints()
		if err != nil {
			panic(err)
		}
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountOnlineEndpoints() took %v to run", d)}.Debug()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		stats.IdleEndpoints, err = ctx.DB.CountIdleEndpoints()
		if err != nil {
			panic(err)
		}
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountIdleEndpoints() took %v to run", d)}.Debug()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		// only run that one once every hour
		if time.Now().Add(-time.Hour).After(countNewEndpointsHourly) {
			start := time.Now()
			// detect new endpoints from last 24 hours against endpoints from last 7 days
			stats.NewEndpoints, err = ctx.DB.CountNewEndpoints(time.Now().Add(-24*time.Hour), time.Now().Add(-7*24*time.Hour))
			if err != nil {
				panic(err)
			}
			d := time.Since(start)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountNewEndpoints() took %v to run", d)}.Debug()
			countNewEndpointsHourly = time.Now()
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		stats.MultiAgentsEndpoints, err = ctx.DB.CountDoubleAgents()
		if err != nil {
			panic(err)
		}
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountDoubleAgents() took %v to run", d)}.Debug()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		stats.DisappearedEndpoints, err = ctx.DB.CountDisappearedEndpoints(time.Now().Add(-7 * 24 * time.Hour))
		if err != nil {
			panic(err)
		}
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountDisappearedEndpoints() took %v to run", d)}.Debug()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		stats.FlappingEndpoints, err = ctx.DB.CountFlappingEndpoints()
		if err != nil {
			panic(err)
		}
		d := time.Since(start)
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("CountFlappingEndpoints() took %v to run", d)}.Debug()
	}()
	wg.Wait()
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

// detectMultiAgents lists endpoint queues that are running more than one agent, and sends
// the queue names to a channel where destruction orders can be emitted to shut down
// duplicate agents
func detectMultiAgents(ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("detectMultiAgents() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving detectMultiAgents()"}.Debug()
	}()
	// if detectmultiagents is not set in the scheduler configuration, do nothing
	if !ctx.Agent.DetectMultiAgents {
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "detectMultiAgents is not activated. skipping"}.Debug()
		return
	}
	hbfreq, err := time.ParseDuration(ctx.Agent.HeartbeatFreq)
	if err != nil {
		return err
	}
	pointInTime := time.Now().Add(-hbfreq)
	queues, err := ctx.DB.ListMultiAgentsQueues(pointInTime)
	if err != nil {
		return err
	}
	for _, q := range queues {
		ctx.Channels.DetectDupAgents <- q
	}
	return
}
