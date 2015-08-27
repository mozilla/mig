// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package database /* import "mig.ninja/mig/database" */

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DB struct {
	c *sql.DB
}

// Connect opens a connection to the database and returns a handler
func Open(dbname, user, password, host string, port int, sslmode string) (db DB, err error) {
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)
	db.c, err = sql.Open("postgres", url)
	return
}

func (db *DB) Close() {
	db.c.Close()
}

func (db *DB) SetMaxOpenConns(n int) {
	db.c.SetMaxOpenConns(n)
}
