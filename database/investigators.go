// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package database /* import "github.com/mozilla/mig/database" */

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/mozilla/mig"
)

// ActiveInvestigators returns a slice of investigators keys marked as active
func (db *DB) ActiveInvestigatorsPubKeys() (keys [][]byte, err error) {
	rows, err := db.c.Query(`SELECT publickey FROM investigators WHERE status='active'
		AND publickey IS NOT NULL`)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil && err != sql.ErrNoRows {
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
			err = fmt.Errorf("Error while retrieving investigator key: '%v'", err)
			return
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete active investigators query: '%v'", err)
	}
	return
}

// InvestigatorByID searches the database for an investigator with a given ID
func (db *DB) InvestigatorByID(iid float64) (inv mig.Investigator, err error) {
	var perm int64
	err = db.c.QueryRow(`SELECT id, name, COALESCE(pgpfingerprint, ''),
		COALESCE(publickey, ''), status, createdat, lastmodified, permissions,
		CASE WHEN apikey IS NOT NULL THEN 'set' ELSE '' END
		FROM investigators WHERE id=$1`,
		iid).Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.PublicKey,
		&inv.Status, &inv.CreatedAt, &inv.LastModified, &perm, &inv.APIKey)
	if err != nil {
		err = fmt.Errorf("Error while retrieving investigator: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	inv.Permissions.FromMask(perm)
	return
}

// InvestigatorByFingerprint searches the database for an investigator that
// has a given fingerprint
func (db *DB) InvestigatorByFingerprint(fp string) (inv mig.Investigator, err error) {
	var perm int64
	err = db.c.QueryRow(`SELECT investigators.id, investigators.name, investigators.pgpfingerprint,
		investigators.publickey, investigators.status, investigators.createdat,
		investigators.lastmodified, investigators.permissions
		FROM investigators WHERE pgpfingerprint IS NOT NULL AND
		LOWER(pgpfingerprint)=LOWER($1)`,
		fp).Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.PublicKey, &inv.Status,
		&inv.CreatedAt, &inv.LastModified, &perm)
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while finding investigator: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		err = fmt.Errorf("InvestigatorByFingerprint: no investigator found for fingerprint '%s'", fp)
		return
	}
	inv.Permissions.FromMask(perm)
	return
}

// Returns a set of InvestigatorAPIAuthHelper structs that the API can utilize to
// authorize requests containing the X-MIGAPIKEY header
func (db *DB) InvestigatorAPIKeyAuthHelpers() (ret []mig.InvestigatorAPIAuthHelper, err error) {
	rows, err := db.c.Query(`SELECT id, apikey, apisalt FROM investigators
		WHERE apikey IS NOT NULL AND apisalt IS NOT NULL AND status='active'`)
	if err != nil {
		return
	}
	for rows.Next() {
		var (
			nh    mig.InvestigatorAPIAuthHelper
			invid int
		)
		err = rows.Scan(&invid, &nh.APIKey, &nh.Salt)
		if err != nil {
			rows.Close()
			return
		}
		nh.ID = float64(invid)
		ret = append(ret, nh)
	}
	return
}

//InvestigatorByActionID returns the list of investigators that signed a given action
func (db *DB) InvestigatorByActionID(aid float64) (invs []mig.Investigator, err error) {
	var perm int64
	rows, err := db.c.Query(`SELECT investigators.id, investigators.name, investigators.pgpfingerprint,
		investigators.status, investigators.createdat, investigators.lastmodified,
		investigators.permissions
		FROM investigators, signatures
		WHERE signatures.actionid=$1
		AND signatures.investigatorid=investigators.id`, aid)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while finding investigator: '%v'", err)
		return
	}
	for rows.Next() {
		var inv mig.Investigator
		err = rows.Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.Status, &inv.CreatedAt, &inv.LastModified, &perm)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve investigator data: '%v'", err)
			return
		}
		inv.Permissions.FromMask(perm)
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
	var newid int
	err = db.c.QueryRow(`INSERT INTO investigators
		(name, pgpfingerprint, publickey, status, createdat, lastmodified, permissions)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, E'\\x'::bytea), 'active', $4, $5, $6)
		RETURNING id`,
		inv.Name, inv.PGPFingerprint, inv.PublicKey, time.Now().UTC(), time.Now().UTC(),
		inv.Permissions.ToMask()).Scan(&newid)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "investigators_pgpfingerprint_idx"` {
			return iid, fmt.Errorf("Investigator's PGP Fingerprint already exists in database")
		}
		return iid, fmt.Errorf("Failed to create investigator: '%v'", err)
	}
	iid = float64(newid)
	inv, err = db.InvestigatorByID(iid)
	if err != nil {
		return 0, fmt.Errorf("Failed to retrieve investigator ID: '%v'", err)
	}
	return
}

// InsertSchedulerInvestigator creates a new migscheduler investigator in the database
// and returns its ID, or an error if the insertion failed, or if the investigator already exists
func (db *DB) InsertSchedulerInvestigator(inv mig.Investigator) (iid float64, err error) {
	_, err = db.c.Exec(`INSERT INTO investigators
		(name, pgpfingerprint, publickey, privatekey, status, createdat, lastmodified)
		VALUES ($1, $2, $3, $4, 'active', $5, $6)`,
		inv.Name, inv.PGPFingerprint, inv.PublicKey, inv.PrivateKey, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "investigators_pgpfingerprint_idx"` {
			return iid, fmt.Errorf("Investigator's PGP Fingerprint already exists in database")
		}
		return iid, fmt.Errorf("Failed to create investigator: '%v'", err)
	}
	inv, err = db.InvestigatorByFingerprint(inv.PGPFingerprint)
	if err != nil {
		return 0, fmt.Errorf("Failed to retrieve investigator ID: '%v'", err)
	}
	iid = inv.ID
	return
}

// UpdateInvestigatorStatus updates the status of an investigator
func (db *DB) UpdateInvestigatorStatus(inv mig.Investigator) (err error) {
	if inv.Status != mig.StatusActiveInvestigator && inv.Status != mig.StatusDisabledInvestigator {
		return fmt.Errorf("Invalid investigator status '%s'", inv.Status)
	}
	_, err = db.c.Exec(`UPDATE investigators SET (status) = ($1) WHERE id=$2`,
		inv.Status, inv.ID)
	if err != nil {
		return fmt.Errorf("Failed to update investigator: '%v'", err)
	}
	return
}

// UpdateInvestigatorAPIKey enables or disabled a standard API key for an investigator
func (db *DB) UpdateInvestigatorAPIKey(inv mig.Investigator, key []byte, salt []byte) (err error) {
	if len(key) == 0 {
		_, err = db.c.Exec(`UPDATE investigators SET apikey=NULL, apisalt=NULL
			WHERE id=$1`, inv.ID)
		if err != nil {
			return err
		}
	} else {
		_, err = db.c.Exec(`UPDATE investigators SET apikey=$1, apisalt=$2
			WHERE id=$3`, key, salt, inv.ID)
		if err != nil {
			return err
		}
	}
	return
}

func (db *DB) UpdateInvestigatorPerms(inv mig.Investigator) (err error) {
	// If the desired permissions do not include an admin bit, do a check here
	// to see how many administrators remain. If this change will reduce the
	// number of admins to 0 prevent the modification.
	mask := inv.Permissions.ToMask()
	if ((mask & mig.PermInvestigator) == 0) || ((mask & mig.PermInvestigatorUpdate) == 0) {
		var ts mig.InvestigatorPerms
		ts.AdminSet()
		tmask := ts.ToMask()
		cnt := 0
		err = db.c.QueryRow(`SELECT COUNT(*) FROM investigators
			WHERE (permissions & $1) != 0 AND
			id != $2`, tmask, inv.ID).Scan(&cnt)
		if err != nil {
			return fmt.Errorf("Failed to update investigator: '%v'", err)
		}
		if cnt < 1 {
			return fmt.Errorf("Failed to update investigator: 'will not remove last admin'")
		}
	}
	_, err = db.c.Exec(`UPDATE investigators SET (permissions) = ($1) WHERE id=$2`,
		mask, inv.ID)
	if err != nil {
		return fmt.Errorf("Failed to update investigator: '%v'", err)
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

// GetSchedulerInvestigator returns the first active scheduler investigator
func (db *DB) GetSchedulerInvestigator() (inv mig.Investigator, err error) {
	err = db.c.QueryRow(`SELECT id, name, pgpfingerprint, publickey, status, createdat, lastmodified
		FROM investigators WHERE name ='migscheduler' AND status='active' ORDER BY id ASC LIMIT 1`,
	).Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.PublicKey, &inv.Status, &inv.CreatedAt, &inv.LastModified)
	if err != nil {
		err = fmt.Errorf("Error while retrieving scheduler investigator: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}
