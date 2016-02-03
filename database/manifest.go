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

func (db *DB) ManifestAddSignature(mid float64, sig string, invid float64) (err error) {
	_, err = db.c.Exec(`INSERT INTO manifestsig
		(manifestid, pgpsignature, investigatorid)
		VALUES ($1, $2, $3)`, mid, sig, invid)
	if err != nil {
		return
	}
	return
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
