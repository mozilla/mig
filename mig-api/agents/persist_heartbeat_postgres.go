// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

import (
	"fmt"

	migdb "github.com/mozilla/mig/database"
)

type PersistHeartbeatPostgres struct {
	db *migdb.DB
}

func NewPersistHeartbeatPostgres(db *migdb.DB) PersistHeartbeatPostgres {
	return PersistHeartbeatPostgres{
		db: db,
	}
}

func (persist PersistHeartbeatPostgres) PersistHeartbeat(heartbeat Heartbeat) error {
	fmt.Printf("POST /heartbeat got heartbeat %v\n", heartbeat)

	agent := heartbeat.ToMigAgent()
	err := persist.db.InsertAgent(agent, nil)

	return err
}
