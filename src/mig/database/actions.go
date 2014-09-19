// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"mig"
	"time"

	_ "github.com/lib/pq"
)

// ActionByID retrieves an action from the database using its ID
func (db *DB) Last10Actions() (actions []mig.Action, err error) {
	rows, err := db.c.Query(`SELECT id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion
		FROM actions ORDER BY starttime DESC LIMIT 10`)
	if err != nil && err != sql.ErrNoRows {
		rows.Close()
		err = fmt.Errorf("Error while listing actions: '%v'", err)
		return
	}
	for rows.Next() {
		var jDesc, jThreat, jOps, jSig []byte
		var a mig.Action
		err = rows.Scan(&a.ID, &a.Name, &a.Target,
			&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
			&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &a.Counters.Sent,
			&a.Counters.Returned, &a.Counters.Done, &a.Counters.Cancelled,
			&a.Counters.Failed, &a.Counters.TimeOut, &jSig, &a.SyntaxVersion)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Error while retrieving action: '%v'", err)
			return
		}
		err = json.Unmarshal(jDesc, &a.Description)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
			return
		}
		err = json.Unmarshal(jThreat, &a.Threat)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
			return
		}
		err = json.Unmarshal(jOps, &a.Operations)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
			return
		}
		err = json.Unmarshal(jSig, &a.PGPSignatures)
		if err != nil {
			rows.Close()
			err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
			return
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// ActionByID retrieves an action from the database using its ID
// If the query fails, the returned action will have ID -1
func (db *DB) ActionByID(id float64) (a mig.Action, err error) {
	a.ID = -1
	var jDesc, jThreat, jOps, jSig []byte
	err = db.c.QueryRow(`SELECT id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion
		FROM actions WHERE id=$1`, id).Scan(&a.ID, &a.Name, &a.Target,
		&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
		&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &a.Counters.Sent,
		&a.Counters.Returned, &a.Counters.Done, &a.Counters.Cancelled,
		&a.Counters.Failed, &a.Counters.TimeOut, &jSig, &a.SyntaxVersion)
	if err != nil {
		err = fmt.Errorf("Error while retrieving action: '%v'", err)
		return
	}
	err = json.Unmarshal(jDesc, &a.Description)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
		return
	}
	err = json.Unmarshal(jThreat, &a.Threat)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
		return
	}
	err = json.Unmarshal(jOps, &a.Operations)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
		return
	}
	err = json.Unmarshal(jSig, &a.PGPSignatures)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
		return
	}
	return
}

// ActionMetaByID retrieves the metadata fields of an action from the database using its ID
func (db *DB) ActionMetaByID(id float64) (a mig.Action, err error) {
	err = db.c.QueryRow(`SELECT id, name, validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr FROM actions WHERE id=$1`, id).Scan(&a.ID, &a.Name, &a.ValidFrom, &a.ExpireAfter,
		&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &a.Counters.Sent,
		&a.Counters.Returned, &a.Counters.Done, &a.Counters.Cancelled,
		&a.Counters.Failed, &a.Counters.TimeOut)
	if err != nil {
		err = fmt.Errorf("Error while retrieving action: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	return
}

// InsertAction writes an action into the database.
func (db *DB) InsertAction(a mig.Action) (err error) {
	jDesc, err := json.Marshal(a.Description)
	if err != nil {
		return fmt.Errorf("Failed to marshal description: '%v'", err)
	}
	jThreat, err := json.Marshal(a.Threat)
	if err != nil {
		return fmt.Errorf("Failed to marshal threat: '%v'", err)
	}
	jOperations, err := json.Marshal(a.Operations)
	if err != nil {
		return fmt.Errorf("Failed to marshal operations: '%v'", err)
	}
	aPGPSignatures, err := json.Marshal(a.PGPSignatures)
	if err != nil {
		return fmt.Errorf("Failed to marshal pgp signatures: '%v'", err)
	}
	_, err = db.c.Exec(`INSERT INTO actions
		(id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr,
		timeoutctr, pgpsignatures, syntaxversion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
		$14, $15, $16, $17, $18, $19, $20)`,
		a.ID, a.Name, a.Target, jDesc, jThreat, jOperations,
		a.ValidFrom, a.ExpireAfter, a.StartTime, a.FinishTime, a.LastUpdateTime,
		a.Status, a.Counters.Sent, a.Counters.Returned, a.Counters.Done,
		a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut,
		aPGPSignatures, a.SyntaxVersion)
	if err != nil {
		return fmt.Errorf("Failed to store action: '%v'", err)
	}
	return
}

// UpdateAction stores updated action fields into the database.
func (db *DB) UpdateAction(a mig.Action) (err error) {
	_, err = db.c.Exec(`UPDATE actions SET (starttime, lastupdatetime,
		status, sentctr, returnedctr, donectr, cancelledctr, failedctr, timeoutctr)
		= ($2, $3, $4, $5, $6, $7, $8, $9, $10) WHERE id=$1`,
		a.ID, a.StartTime, a.LastUpdateTime, a.Status, a.Counters.Sent, a.Counters.Returned,
		a.Counters.Done, a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut)
	if err != nil {
		return fmt.Errorf("Failed to update action: '%v'", err)
	}
	return
}

// InsertOrUpdateAction looks for an existing action in DB and update it,
// or insert a new one if none is found
func (db *DB) InsertOrUpdateAction(a mig.Action) (inserted bool, err error) {
	var id float64
	err = db.c.QueryRow(`SELECT id FROM actions WHERE id=$1`, a.ID).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return inserted, fmt.Errorf("Error while retrieving action: '%v'", err)
	}
	if err == sql.ErrNoRows {
		inserted = true
		return inserted, db.InsertAction(a)
	} else {
		return inserted, db.UpdateAction(a)
	}
}

// UpdateActionStatus updates the status of an action
func (db *DB) UpdateActionStatus(a mig.Action) (err error) {
	_, err = db.c.Exec(`UPDATE actions SET (status) = ($2) WHERE id=$1`,
		a.ID, a.Status)
	if err != nil {
		return fmt.Errorf("Failed to update action status: '%v'", err)
	}
	return
}

// UpdateRunningAction stores updated time and counters on a running action
func (db *DB) UpdateRunningAction(a mig.Action) (err error) {
	_, err = db.c.Exec(`UPDATE actions SET (lastupdatetime, returnedctr,
		donectr, cancelledctr, failedctr, timeoutctr)
		= ($2, $3, $4, $5, $6, $7) WHERE id=$1`,
		a.ID, a.LastUpdateTime, a.Counters.Returned, a.Counters.Done,
		a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut)
	if err != nil {
		return fmt.Errorf("Failed to update action: '%v'", err)
	}
	return
}

// FinishAction updates the action fields to mark it as done
func (db *DB) FinishAction(a mig.Action) (err error) {
	a.FinishTime = time.Now()
	a.Status = "completed"
	_, err = db.c.Exec(`UPDATE actions SET (finishtime, lastupdatetime, status,
		returnedctr, donectr, cancelledctr, failedctr, timeoutctr)
		= ($1, $2, $3, $4, $5, $6, $7, $8) WHERE id=$9`,
		a.FinishTime, a.LastUpdateTime, a.Status, a.Counters.Returned,
		a.Counters.Done, a.Counters.Cancelled, a.Counters.Failed, a.Counters.TimeOut, a.ID)
	if err != nil {
		return fmt.Errorf("Failed to update action: '%v'", err)
	}
	return
}

// InsertSignature create an entry in the signatures tables that map an investigator
// to an action and a signature
func (db *DB) InsertSignature(aid, iid float64, sig string) (err error) {
	_, err = db.c.Exec(`INSERT INTO signatures(actionid, investigatorid, pgpsignature)
		VALUES($1, $2, $3)`, aid, iid, sig)
	if err != nil {
		return fmt.Errorf("Failed to store signature: '%v'", err)
	}
	return
}
