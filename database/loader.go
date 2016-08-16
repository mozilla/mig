// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package database /* import "mig.ninja/mig/database" */

import (
	"database/sql"
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
	err = db.c.QueryRow(`SELECT id FROM loaders WHERE
		loaderkey=$1 AND enabled=TRUE`, key).Scan(&ret)
	if err != nil {
		err = fmt.Errorf("No matching loader entry found for key")
		return
	}
	return
}

// Return a loader ID and hashed key given a prefix string
func (db *DB) GetLoaderAuthDetails(prefix string) (lad mig.LoaderAuthDetails, err error) {
	err = db.c.QueryRow(`SELECT id, salt, loaderkey FROM loaders WHERE
		keyprefix=$1 AND enabled=TRUE`, prefix).Scan(&lad.ID, &lad.Salt, &lad.Hash)
	if err != nil {
		err = fmt.Errorf("Unable to locate loader from prefix")
		return
	}
	err = lad.Validate()
	if err != nil {
		return
	}
	return
}

// Return a loader name given an ID
func (db *DB) GetLoaderName(id float64) (ret string, err error) {
	err = db.c.QueryRow(`SELECT loadername FROM loaders 
		WHERE id=$1 AND enabled=TRUE`, id).Scan(&ret)
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
		lastseen=now()
		WHERE id=$4`,
		agt.Name, jEnv, jTags, lid)
	if err != nil {
		return err
	}
	return
}

// If any expected environment has been set on a loader entry, this function
// validates the environment submitted by the loader matches that expected
// query string; returns an error if not
func (db *DB) CompareLoaderExpectEnv(lid float64) error {
	var (
		expectenv sql.NullString
		result    bool
	)
	rerr := fmt.Errorf("loader environment verification failed")
	txn, err := db.c.Begin()
	if err != nil {
		return rerr
	}
	_, err = txn.Exec("SET LOCAL ROLE migreadonly")
	if err != nil {
		txn.Rollback()
		return rerr
	}
	err = txn.QueryRow(`SELECT expectenv FROM loaders WHERE
		id=$1`, lid).Scan(&expectenv)
	if err != nil {
		txn.Rollback()
		return rerr
	}
	// If no expected environment is set, we are done here
	if !expectenv.Valid || expectenv.String == "" {
		err = txn.Commit()
		if err != nil {
			txn.Rollback()
			return err
		}
		return nil
	}
	qfmt := fmt.Sprintf("SELECT TRUE FROM loaders WHERE id=$1 AND %v", expectenv.String)
	err = txn.QueryRow(qfmt, lid).Scan(&result)
	if err != nil {
		txn.Rollback()
		return rerr
	}
	err = txn.Commit()
	if err != nil {
		txn.Rollback()
		return rerr
	}
	return nil
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
	qs := fmt.Sprintf(`SELECT id, loadername, name, lastseen, enabled
		FROM loaders WHERE enabled=TRUE AND %v`, mtarg)
	rows, err := db.c.Query(qs)
	if err != nil {
		return
	}
	if rows != nil {
		defer rows.Close()
	}
	for rows.Next() {
		var agtname sql.NullString
		nle := mig.LoaderEntry{}
		err = rows.Scan(&nle.ID, &nle.Name, &agtname, &nle.LastSeen, &nle.Enabled)
		if err != nil {
			return ret, err
		}
		// This should always be valid, if it is not that means we have a loader
		// entry updated with a valid env, but a NULL agent name. In that case we
		// just don't set the agent name in the loader entry.
		if agtname.Valid {
			nle.AgentName = agtname.String
		}
		ret = append(ret, nle)
	}
	return
}

// Return a loader entry given an ID
func (db *DB) GetLoaderFromID(lid float64) (ret mig.LoaderEntry, err error) {
	var name, expectenv sql.NullString
	err = db.c.QueryRow(`SELECT id, loadername, keyprefix, name, lastseen, enabled,
		expectenv
		FROM loaders WHERE id=$1`, lid).Scan(&ret.ID, &ret.Name,
		&ret.Prefix, &name, &ret.LastSeen, &ret.Enabled,
		&expectenv)
	if err != nil {
		err = fmt.Errorf("Error while retrieving loader: '%v'", err)
		return
	}
	if name.Valid {
		ret.AgentName = name.String
	}
	if expectenv.Valid {
		ret.ExpectEnv = expectenv.String
	}
	return
}

// Enable or disable a loader entry in the database
func (db *DB) LoaderUpdateStatus(lid float64, status bool) (err error) {
	_, err = db.c.Exec(`UPDATE loaders SET enabled=$1 WHERE
		id=$2`, status, lid)
	if err != nil {
		return err
	}
	return
}

// Update loader expect fields
func (db *DB) LoaderUpdateExpect(lid float64, eenv string) (err error) {
	var eset sql.NullString
	if eenv != "" {
		eset.String = eenv
		eset.Valid = true
	}
	_, err = db.c.Exec(`UPDATE loaders SET expectenv=$1 WHERE
		id=$2`, eset, lid)
	if err != nil {
		return err
	}
	return
}

// Change loader key, hashkey should be the hashed version of the key component
func (db *DB) LoaderUpdateKey(lid float64, hashkey []byte, salt []byte) (err error) {
	_, err = db.c.Exec(`UPDATE loaders SET loaderkey=$1, salt=$2 WHERE
		id=$3`, hashkey, salt, lid)
	if err != nil {
		return err
	}
	return
}

// Add a new loader entry to the database; the hashed loader key should
// be provided as hashkey
func (db *DB) LoaderAdd(le mig.LoaderEntry, hashkey []byte, salt []byte) (err error) {
	var eval sql.NullString
	if le.ExpectEnv != "" {
		eval.String = le.ExpectEnv
		eval.Valid = true
	}
	_, err = db.c.Exec(`INSERT INTO loaders 
		(loadername, keyprefix, loaderkey, salt, lastseen, enabled,
		expectenv)
		VALUES
		($1, $2, $3, $4, now(), FALSE, $5)`, le.Name,
		le.Prefix, hashkey, salt, eval)
	return
}
