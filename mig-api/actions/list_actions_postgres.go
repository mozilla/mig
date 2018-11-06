// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"github.com/mozilla/mig"
	migdb "github.com/mozilla/mig/database"
)

type ListActionsPostgres struct {
	db *migdb.DB
}

func NewListActionsPostgres(db *migdb.DB) ListActionsPostgres {
	return ListActionsPostgres{
		db: db,
	}
}

func (list ListActionsPostgres) ListActions(aid AgentID) ([]mig.Action, error) {
	agent := mig.Agent{
		ID: float64(aid),
	}
	actions, err := list.db.SetupRunnableActionsForAgent(agent)
	return actions, err
}
