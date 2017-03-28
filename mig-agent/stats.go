// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package main

// Contains functions related to agent statistics. Currently this data is not
// persistent and will be lost of the agent process is restarted, this could be
// adapted to make use of local storage as needed.
//
// The information here can be presented by querying the agent on its status
// socket.

import (
	"mig.ninja/mig"
	"strings"
	"sync"
	"time"
)

// Defines statistics kept by the agent
type agentStats struct {
	Actions []agentStatsAction
	sync.Mutex
}

// Import information about an incoming action into agent statistics. If
// accepted is true the action was executed by the agent, if false it means
// the action was rejected (e.g., due to lack of signatures).
func (s *agentStats) importAction(a mig.Action, accepted bool) error {
	s.Lock()
	defer s.Unlock()
	if STATSMAXACTIONS == 0 {
		return nil
	}
	ns := agentStatsAction{}
	err := ns.importAction(a, accepted)
	if err != nil {
		return err
	}
	if len(s.Actions) == STATSMAXACTIONS {
		s.Actions = s.Actions[1:]
	}
	s.Actions = append(s.Actions, ns)
	return nil
}

// Stats we keep for an action processed by the agent
type agentStatsAction struct {
	Time     string
	Name     string
	Accepted string
	Modules  string
}

// Add data to an agentStatsAction based on action a
func (s *agentStatsAction) importAction(a mig.Action, accepted bool) error {
	s.Time = time.Now().UTC().Format(time.RFC3339)
	s.Name = a.Name
	fl := make([]string, 0)
	for _, x := range a.Operations {
		found := false
		for _, y := range fl {
			if y == strings.ToLower(x.Module) {
				found = true
			}
		}
		if found {
			continue
		}
		fl = append(fl, strings.ToLower(x.Module))
	}
	s.Modules = strings.Join(fl, ", ")
	if accepted {
		s.Accepted = "Accepted"
	} else {
		s.Accepted = "Rejected"
	}
	return nil
}

// Initialize agent statistics
func initAgentStats(ctx Context) (newctx Context, err error) {
	newctx = ctx
	ctx.Stats = agentStats{}
	ctx.Channels.Log <- mig.Log{Desc: "Agent statistics management initialized"}.Info()
	return
}
