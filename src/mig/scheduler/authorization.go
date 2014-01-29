/* Mozilla InvestiGator Scheduler

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package main

import (
	"bufio"
	"fmt"
	"mig"
	"os"
	"regexp"
)

// If a whitelist is defined, lookup the agent in it, and return nil if found, or error if not
func isAgentAuthorized(agentName string, ctx Context) (ok bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("isAgentAuthorized() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "leaving isAgentAuthorized()"}.Debug()
	}()

	ok = false

	// bypass mode if there's no whitelist in the conf
	if ctx.Agent.Whitelist == "" {
		ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: "Agent authorization checking is disabled"}.Debug()
		return
	}

	agtRe := regexp.MustCompile("^" + agentName + "$")
	wfd, err := os.Open(ctx.Agent.Whitelist)
	if err != nil {
		panic(err)
	}
	defer wfd.Close()

	scanner := bufio.NewScanner(wfd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		if agtRe.MatchString(scanner.Text()) {
			ctx.Channels.Log <- mig.Log{OpID: ctx.OpID, Desc: fmt.Sprintf("Agent '%s' is authorized", agentName)}.Debug()
			ok = true
			return
		}
	}
	// whitelist check failed, agent isn't authorized
	return
}
