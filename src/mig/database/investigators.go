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

// InvestigatorByID searches the database for an investigator with a given ID
func (db *DB) InvestigatorByID(iid float64) (inv mig.Investigator, err error) {
	err = db.c.QueryRow("SELECT id, name, pgpfingerprint, publickey FROM investigators WHERE id=$1",
		iid).Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.PublicKey)
	if err != nil {
		err = fmt.Errorf("Error while retrieving investigator: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// InvestigatorByFingerprint searches the database for an investigator that
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

// InsertInvestigator creates a new investigator in the database and returns its ID,
// or an error if the insertion failed, or if the investigator already exists
func (db *DB) InsertInvestigator(inv mig.Investigator) (iid float64, err error) {
	_, err = db.c.Exec(`INSERT INTO investigators
		(name, pgpfingerprint, publickey, status)
		VALUES ($1, $2, $3, 'active')`,
		inv.Name, inv.PGPFingerprint, inv.PublicKey)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "investigators_pgpfingerprint_idx"` {
			return iid, fmt.Errorf("Investigator's PGP Fingerprint already exists in database")
		}
		return iid, fmt.Errorf("Failed to create investigator: '%v'", err)
	}
	iid, err = db.InvestigatorByFingerprint(inv.PGPFingerprint)
	if err != nil {
		return iid, fmt.Errorf("Failed to retrieve investigator ID: '%v'", err)
	}
	return
}
