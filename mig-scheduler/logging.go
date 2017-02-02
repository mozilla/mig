// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"fmt"
	"mig.ninja/mig"
)

func logAgentAction(ctx Context, cmd mig.Command) (err error) {
	var logmsg string
	logmsg = fmt.Sprintf("Agent action: %q %q %q", cmd.Agent.Name, cmd.Agent.LoaderName, cmd.Action.Name)
	ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, ActionID: cmd.Action.ID, CommandID: cmd.ID, Desc: logmsg}.Info()
	return
}
