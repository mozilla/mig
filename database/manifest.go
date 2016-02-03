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

// The number of signatures required for a manifest to be marked as active.
// XXX This should probably be somewhere else like in the configuration file.
const REQUIRED_SIGNATURES int = 1

// Add a signature to the database for an existing manifest
func (db *DB) ManifestAddSignature(mid float64, sig string, invid float64) (err error) {
	_, err = db.c.Exec(`INSERT INTO manifestsig
		(manifestid, pgpsignature, investigatorid)
		VALUES ($1, $2, $3)`, mid, sig, invid)
	if err != nil {
		return
	}

	err = db.ManifestUpdateStatus(mid)
	return
}

// Update the status of a manifest based on the number of signatures it has
func (db *DB) ManifestUpdateStatus(mid float64) (err error) {
	var cnt int
	row := db.c.QueryRow(`SELECT COUNT(*) FROM manifestsig
		WHERE manifestid=$1`, mid)
	err = row.Scan(&cnt)
	if err != nil {
		return err
	}
	status := "staged"
	if cnt >= REQUIRED_SIGNATURES {
		status = "active"
	}
	_, err = db.c.Exec(`UPDATE manifests SET status=$1 WHERE
		id=$2`, status, mid)
	if err != nil {
		panic(err)
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
