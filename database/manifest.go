// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package database /* import "mig.ninja/mig/database" */

import (
	"fmt"
	_ "github.com/lib/pq"
	"mig.ninja/mig"
)

// Add a new manifest record to the database
func (db *DB) ManifestAdd(mr mig.ManifestRecord) (err error) {
	_, err = db.c.Exec(`INSERT INTO manifests VALUES
		(DEFAULT, $1, $2, now(), 'staged', $3)`, mr.Name,
		mr.Content, mr.Target)
	return
}

// Add a signature to the database for an existing manifest
func (db *DB) ManifestAddSignature(mid float64, sig string, invid float64, reqsig int) (err error) {
	res, err := db.c.Exec(`INSERT INTO manifestsig
		(manifestid, pgpsignature, investigatorid)
		SELECT $1, $2, $3
		WHERE EXISTS (SELECT id FROM manifests
		WHERE id=$4 AND status!='disabled')`, mid, sig, invid, mid)
	if err != nil {
		return
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return
	}
	if ra != 1 {
		return fmt.Errorf("Manifest signing operation failed")
	}

	err = db.ManifestUpdateStatus(mid, reqsig)
	return
}

// Disable a manifest record
func (db *DB) ManifestDisable(mid float64) (err error) {
	_, err = db.c.Exec(`UPDATE manifests SET status='disabled' WHERE
		id=$1`, mid)
	if err != nil {
		return
	}
	return
}

// Update the status of a manifest based on the number of signatures it has,
// reqsig is passed as an argument that indicates the number of signatures
// a manifest must have to be considered active
func (db *DB) ManifestUpdateStatus(mid float64, reqsig int) (err error) {
	var cnt int
	err = db.c.QueryRow(`SELECT COUNT(*) FROM manifestsig
		WHERE manifestid=$1`, mid).Scan(&cnt)
	if err != nil {
		return err
	}
	status := "staged"
	if cnt >= reqsig {
		status = "active"
	}
	_, err = db.c.Exec(`UPDATE manifests SET status=$1 WHERE
		id=$2 AND status!='disabled'`, status, mid)
	if err != nil {
		return err
	}
	return
}

// Clear existing signatures for a manifest record
func (db *DB) ManifestClearSignatures(mid float64) (err error) {
	_, err = db.c.Exec(`DELETE FROM manifestsig WHERE manifestid=$1`, mid)
	return err
}

// Return the entire contents of manifest ID mid from the database
func (db *DB) GetManifestFromID(mid float64) (ret mig.ManifestRecord, err error) {
	row := db.c.QueryRow(`SELECT id, name, content, timestamp, status, target
		FROM manifests WHERE id=$1`, mid)
	err = row.Scan(&ret.ID, &ret.Name, &ret.Content, &ret.Timestamp, &ret.Status, &ret.Target)
	if err != nil {
		err = fmt.Errorf("Error while retrieving manifest: '%v'", err)
		return
	}
	// Also add any signatures that exist for this manifest record
	rows, err := db.c.Query(`SELECT pgpsignature FROM manifestsig
		WHERE manifestid=$1`, mid)
	if err != nil {
		return
	}
	if rows != nil {
		defer rows.Close()
	}
	for rows.Next() {
		var s string
		err = rows.Scan(&s)
		if err != nil {
			return ret, err
		}
		ret.Signatures = append(ret.Signatures, s)
	}
	return
}
