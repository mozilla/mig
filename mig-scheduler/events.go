// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
	"mig.ninja/mig"
)

// sendEvent publishes a message to the miginternal rabbitmq exchange
func sendEvent(key string, body []byte, ctx Context) error {
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		ContentType:  "text/plain",
		Expiration:   "6000000", // events expire after 100 minutes if not consumed
		Body:         body,
	}
	err := ctx.MQ.Chan.Publish(mig.Mq_Ex_ToWorkers, key, false, false, msg)
	if err != nil {
		err = fmt.Errorf("event publication failed. err='%v', key='%s', body='%s'", err, key, msg)
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
		return err
	}
	return nil
}
