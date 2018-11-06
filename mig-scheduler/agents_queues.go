// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"

	"github.com/mozilla/mig"

	"github.com/streadway/amqp"
)

// startResultsListener initializes the routine that receives heartbeats from agents
func startResultsListener(ctx Context) (resultsChan <-chan amqp.Delivery, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("startResultsListener() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving startResultsListener()"}.Debug()
	}()

	_, err = ctx.MQ.Chan.QueueDeclare(mig.QueueAgentResults, true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.QueueBind(mig.QueueAgentResults, mig.QueueAgentResults, mig.ExchangeToSchedulers, false, nil)
	if err != nil {
		panic(err)
	}

	err = ctx.MQ.Chan.Qos(0, 0, false)
	if err != nil {
		panic(err)
	}

	resultsChan, err = ctx.MQ.Chan.Consume(mig.QueueAgentResults, "", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "agents results listener initialized"}

	return
}
