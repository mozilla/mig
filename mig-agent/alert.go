// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"fmt"

	"mig.ninja/mig"
)

// alertHandler is called inside the agent when it recieves an alert message from a
// persistent module.
func alertHandler(ctx *Context, alert string) {
	ctx.Channels.Alert <- alert
}

// alertProcessor processes incoming alert messages from the alert channel and handles
// them. This results in either the alert being written to the dispatch module (if the
// dispatch module is active) or the alert being written to the agent log.
func alertProcessor(ctx *Context) {
	ctx.Channels.Log <- mig.Log{Desc: "starting alert processor"}.Info()

	var alert string
	for {
		alert = <-ctx.Channels.Alert

		// If the dispatch module is active, write the message to the dispatch
		// channel
		dispatchChanLock.Lock()
		if dispatchChan != nil {
			dispatchChan <- alert
			dispatchChanLock.Unlock()
			continue
		}
		dispatchChanLock.Unlock()

		// Lastly, if no dispatch module is available, just write the alert to the
		// agent log.
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("alert: %v", alert)}.Info()
	}
}
