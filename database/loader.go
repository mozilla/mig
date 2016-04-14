// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package database /* import "mig.ninja/mig/database" */

import (
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"mig.ninja/mig"
)

// Return a loader entry ID given a loader key
func (db *DB) GetLoaderEntryID(key string) (ret float64, err error) {
	if key == "" {
		return ret, fmt.Errorf("key cannot be empty")
	}
	err = db.c.QueryRow("SELECT id FROM loaders WHERE loaderkey=$1", key).Scan(&ret)
	if err != nil {
		err = fmt.Errorf("No matching loader entry found for key")
		return
	}
	return
}

// Return a loader name given an ID
func (db *DB) GetLoaderName(id float64) (ret string, err error) {
	err = db.c.QueryRow("SELECT loadername FROM loaders WHERE id=$1", id).Scan(&ret)
	if err != nil {
		err = fmt.Errorf("Unable to locate name for loader ID")
		return
	}
	return
}

// Update a given loader entry using supplied agent information (e.g., provided
// during a manifest request by a loader instance
func (db *DB) UpdateLoaderEntry(lid float64, agt mig.Agent) (err error) {
	if agt.Name == "" {
		return fmt.Errorf("will not update loader entry with no agent name")
	}
	jEnv, err := json.Marshal(agt.Env)
	if err != nil {
		return
	}
	jTags, err := json.Marshal(agt.Tags)
	if err != nil {
		return
	}
	_, err = db.c.Exec(`UPDATE loaders
		SET name=$1, env=$2, tags=$3,
		lastused=now()
		WHERE id=$4`,
		agt.Name, jEnv, jTags, lid)
	if err != nil {
		return err
	}
	return
}

// Given a loader ID, identify which manifest is applicable to return to this
// loader in a manifest request
func (db *DB) ManifestIDFromLoaderID(lid float64) (ret float64, err error) {
	rows, err := db.c.Query(`SELECT id, target FROM manifests
		WHERE status='active' ORDER BY timestamp DESC`)
	if err != nil {
		return
	}
	if rows != nil {
		defer rows.Close()
	}
	for rows.Next() {
		var mtarg string
		err = rows.Scan(&ret, &mtarg)
		if err != nil {
			return 0, err
		}
		qs := fmt.Sprintf("SELECT 1 FROM loaders WHERE id=$1 AND %v", mtarg)
		tr, err := db.c.Query(qs, lid)
		if err != nil {
			return 0, err
		}
		if tr == nil {
			continue
		}
		if tr.Next() {
			// We had a valid match
			tr.Close()
			return ret, nil
		}
		tr.Close()
	}
	err = fmt.Errorf("No matching manifest was found for loader entry")
	return
}

// Return all the loader entries that match the targeting string for manifest mid
func (db *DB) AllLoadersFromManifestID(mid float64) (ret []mig.LoaderEntry, err error) {
	var mtarg string
	err = db.c.QueryRow(`SELECT target FROM manifests
		WHERE (status='active' OR status='staged') AND id=$1`, mid).Scan(&mtarg)
	if err != nil {
		return
	}
	qs := fmt.Sprintf("SELECT id, loadername, name, lastused FROM loaders WHERE %v", mtarg)
	rows, err := db.c.Query(qs)
	if err != nil {
		return
	}
	if rows != nil {
		defer rows.Close()
	}
	for rows.Next() {
		nle := mig.LoaderEntry{}
		err = rows.Scan(&nle.ID, &nle.Name, &nle.AgentName, &nle.LastUsed)
		if err != nil {
			return ret, err
		}
		ret = append(ret, nle)
	}
	return
}
