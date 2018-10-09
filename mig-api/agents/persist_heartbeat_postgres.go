// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

import (
	"database/sql"
	"fmt"
)

type PersistHeartbeatPostgres struct {
	db *sql.DB
}

func NewPersistHeartbeatPostgres(db *sql.DB) PersistHeartbeatPostgres {
	return PersistHeartbeatPostgres{
		db: db,
	}
}

func (persist PersistHeartbeatPostgres) PersistHeartbeat(heartbeat Heartbeat) error {
	fmt.Printf("POST /heartbeat got heartbeat %v\n", heartbeat)
	return nil
}
