// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package database

import (
	"database/sql"
	"fmt"
	"mig"

	_ "github.com/lib/pq"
)

// FindInvestigatorByFingerprint searches the database for an investigator that
// has a given fingerprint
func (db *DB) InvestigatorByFingerprint(fp string) (iid float64, err error) {
	err = db.c.QueryRow("SELECT id FROM investigators WHERE LOWER(pgpfingerprint)=LOWER($1)", fp).Scan(&iid)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while finding investigator: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		err = fmt.Errorf("InvestigatorByFingerprint: no investigator found for fingerprint '%s'", fp)
		return
	}
	return
}

//InvestigatorByActionID returns the list of investigators that signed a given action
func (db *DB) InvestigatorByActionID(aid float64) (ivgts []mig.Investigator, err error) {
	rows, err := db.c.Query(`SELECT investigators.id, investigators.name, investigators.pgpfingerprint
		FROM investigators, signatures
		WHERE signatures.actionid=$1
		AND signatures.investigatorid=investigators.id`, aid)
	if err != nil && err != sql.ErrNoRows {
		rows.Close()
		err = fmt.Errorf("Error while finding investigator: '%v'", err)
		return
	}
	for rows.Next() {
		var ivgt mig.Investigator
		err = rows.Scan(&ivgt.ID, &ivgt.Name, &ivgt.PGPFingerprint)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to retrieve investigator data: '%v'", err)
			return
		}
		ivgts = append(ivgts, ivgt)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}
