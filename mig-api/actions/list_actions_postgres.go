// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"fmt"
	"time"

	"github.com/mozilla/mig"
	migdb "github.com/mozilla/mig/database"
)

type ListActionsPostgres struct {
	db       *migdb.DB
	queueLoc string
}

func NewListActionsPostgres(db *migdb.DB, queue string) ListActionsPostgres {
	return ListActionsPostgres{
		db:       db,
		queueLoc: queue,
	}
}

func (list ListActionsPostgres) ListActions(limit uint) ([]mig.Action, error) {
	now := time.Now().Add(-15 * time.Minute)
	agents, err := list.db.ActiveAgentsByQueue(list.queueLoc, now)
	if err != nil {
		return []mig.Action{}, err
	}
	if len(agents) == 0 {
		err := fmt.Errorf("No agents listening to queue %s", list.queueLoc)
		return []mig.Action{}, err
	}
	actions, err := list.db.SetupRunnableActionsForAgent(agents[0])
	if err != nil {
		return []mig.Action{}, err
	}
	return actions, nil
}
