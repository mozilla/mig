// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"mig"
	"time"
)

// ActiveInvestigators returns a slice of investigators keys marked as active
func (db *DB) ActiveInvestigatorsKeys() (keys []string, err error) {
	rows, err := db.c.Query("SELECT publickey FROM investigators WHERE status='active'")
	if err != nil && err != sql.ErrNoRows {
		rows.Close()
		err = fmt.Errorf("Error while listing active investigators keys: '%v'", err)
		return
	}
	if err == sql.ErrNoRows { // having an empty DB is not an issue
		return
	}
	for rows.Next() {
		var key []byte
		err = rows.Scan(&key)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Error while retrieving investigator key: '%v'", err)
			return
		}
		keys = append(keys, string(key))
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete active investigators query: '%v'", err)
	}
	return
}

// InvestigatorByID searches the database for an investigator with a given ID
func (db *DB) InvestigatorByID(iid float64) (inv mig.Investigator, err error) {
	err = db.c.QueryRow("SELECT id, name, pgpfingerprint, publickey, status, createdat, lastmodified FROM investigators WHERE id=$1",
		iid).Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.PublicKey, &inv.Status, &inv.CreatedAt, &inv.LastModified)
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
func (db *DB) InvestigatorByActionID(aid float64) (invs []mig.Investigator, err error) {
	rows, err := db.c.Query(`SELECT investigators.id, investigators.name, investigators.pgpfingerprint,
		investigators.status, investigators.createdat, investigators.lastmodified
		FROM investigators, signatures
		WHERE signatures.actionid=$1
		AND signatures.investigatorid=investigators.id`, aid)
	if err != nil && err != sql.ErrNoRows {
		rows.Close()
		err = fmt.Errorf("Error while finding investigator: '%v'", err)
		return
	}
	for rows.Next() {
		var inv mig.Investigator
		err = rows.Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.Status, &inv.CreatedAt, &inv.LastModified)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to retrieve investigator data: '%v'", err)
			return
		}
		invs = append(invs, inv)
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
		(name, pgpfingerprint, publickey, status, createdat, lastmodified)
		VALUES ($1, $2, $3, 'active', $4, $5 )`,
		inv.Name, inv.PGPFingerprint, inv.PublicKey, time.Now().UTC(), time.Now().UTC())
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

// InsertSchedulerInvestigator creates a new migscheduler investigator in the database
// and returns its ID, or an error if the insertion failed, or if the investigator already exists
func (db *DB) InsertSchedulerInvestigator(inv mig.Investigator) (iid float64, err error) {
	_, err = db.c.Exec(`INSERT INTO investigators
		(name, pgpfingerprint, publickey, privatekey, status, createdat, lastmodified))
		VALUES ($1, $2, $3, $4, 'active')`,
		inv.Name, inv.PGPFingerprint, inv.PublicKey, inv.PrivateKey, time.Now().UTC(), time.Now().UTC())
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

// GetSchedulerPrivKey returns the first active private key found for user migscheduler
func (db *DB) GetSchedulerPrivKey() (key []byte, err error) {
	err = db.c.QueryRow(`SELECT privatekey FROM investigators
		WHERE name ='migscheduler' AND status='active'
		ORDER BY id ASC LIMIT 1`).Scan(&key)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while retrieving scheduler private key: '%v'", err)
		return
	}
	if err == sql.ErrNoRows { // having an empty DB is not an issue
		err = fmt.Errorf("no private key found for migscheduler")
		return
	}
	return
}
