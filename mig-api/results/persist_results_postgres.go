// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package results

import (
	migdb "github.com/mozilla/mig/database"
	"github.com/mozilla/mig/modules"
)

type PersistResultsPostgres struct {
	db *migdb.DB
}

func NewPersistResultsPostgres(db *migdb.DB) PersistResultsPostgres {
	return PersistResultsPostgres{
		db: db,
	}
}

func (persist PersistResultsPostgres) PersistResults(
	actionID float64,
	results []modules.Result,
) PersistError {
	return PersistErrorNil
}
