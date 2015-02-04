package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"mig"
	"time"
)

// QueuesCleanup deletes rabbitmq queues of endpoints that no
// longer have any agent running on them. Only the queues with 0 consumers and 0
// pending messages are deleted.
func QueuesCleanup(ctx Context) (err error) {
	// temporary context for amqp operations
	tmpctx := ctx
	tmpctxinitiliazed := false
	start := time.Now()
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("QueuesCleanup() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving QueuesCleanup()"}.Debug()
		if tmpctxinitiliazed {
			tmpctx.MQ.conn.Close()
		}
	}()
	// cleanup runs every QueuesCleanupFreq, so we run a query that lists all offline agents
	// between now and up to "QueuesCleanupFreq" ago. For a typical value of QueuesCleanupFreq
	// set to 24 hours, each iteration of this job will list all agents that went offline over
	// the last 24 hours, plus two hours for good measure...
	pointInTime, err := time.ParseDuration(ctx.Periodic.QueuesCleanupFreq)
	if err != nil {
		panic(err)
	}
	oldest := time.Now().Add(-(pointInTime + 2*time.Hour))
	queues, err := ctx.DB.GetDisappearedEndpoints(oldest)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("QueuesCleanup(): found %d offline endpoints between %s and now", len(queues), oldest.String())}.Debug()
	makeamqpchan := true
	for _, queue := range queues {
		if makeamqpchan {
			// keep the existing context, but create a new rabbitmq connection to prevent breaking
			// the main one if something goes wrong.
			tmpctx, err = initRelay(tmpctx)
			if err != nil {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("QueuesCleanup(): %v. Continuing!", err)}.Err()
				continue
			}
			makeamqpchan = false
			tmpctxinitiliazed = true
		}
		// the call to inspect will fail if the queue doesn't exist, so we fail silently and continue
		_, err = tmpctx.MQ.Chan.QueueInspect("mig.agt." + queue)
		if err != nil {
			// If a queue by this name does not exist, an error will be returned and the channel will be closed.
			// Reopen the channel and continue
			if amqp.ErrClosed == err || err.(*amqp.Error).Recover {
				tmpctx.MQ.Chan, err = tmpctx.MQ.conn.Channel()
				if err != nil {
					ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("QueuesCleanup(): QueueInspect failed with error '%v'. Continuing.", err)}.Warning()
					makeamqpchan = true
				}
			} else {
				ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("QueuesCleanup(): QueueInspect failed with error '%v'. Continuing.", err)}.Warning()
				makeamqpchan = true
			}
			continue
		}
		_, err = tmpctx.MQ.Chan.QueueDelete("mig.agt."+queue, false, false, false)
		if err != nil {
			desc := fmt.Sprintf("error while deleting queue mig.agt.%s: %v", queue, err)
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: desc}.Err()
			makeamqpchan = true
		} else {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("removed endpoint queue %s", queue)}
			// throttling. looks like iterating too fast on queuedelete eventually locks the connection
			time.Sleep(100 * time.Millisecond)
		}
	}
	d := time.Since(start)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("QueuesCleanup(): done in %v", d)}
	return
}
