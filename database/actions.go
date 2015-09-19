// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package database /* import "mig.ninja/mig/database" */

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mig.ninja/mig"

	_ "github.com/lib/pq"
)

// LastActions retrieves the last X actions by time from the database
func (db *DB) LastActions(limit int) (actions []mig.Action, err error) {
	rows, err := db.c.Query(`SELECT id, name, target, description, threat, operations,
		validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status, pgpsignatures, syntaxversion
		FROM actions ORDER BY starttime DESC LIMIT $1`, limit)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while listing actions: '%v'", err)
		return
	}
	for rows.Next() {
		var jDesc, jThreat, jOps, jSig []byte
		var a mig.Action
		err = rows.Scan(&a.ID, &a.Name, &a.Target,
			&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
			&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &jSig, &a.SyntaxVersion)
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
		a.Counters, err = db.GetActionCounters(a.ID)
		if err != nil {
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
		status, pgpsignatures, syntaxversion
		FROM actions WHERE id=$1`, id).Scan(&a.ID, &a.Name, &a.Target,
		&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
		&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status, &jSig, &a.SyntaxVersion)
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
	a.Counters, err = db.GetActionCounters(a.ID)
	if err != nil {
		return
	}
	return
}

// ActionMetaByID retrieves the metadata fields of an action from the database using its ID
func (db *DB) ActionMetaByID(id float64) (a mig.Action, err error) {
	err = db.c.QueryRow(`SELECT id, name, validfrom, expireafter, starttime, finishtime, lastupdatetime,
		status FROM actions WHERE id=$1`, id).Scan(&a.ID, &a.Name, &a.ValidFrom, &a.ExpireAfter,
		&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status)
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
		status, pgpsignatures, syntaxversion)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		a.ID, a.Name, a.Target, jDesc, jThreat, jOperations,
		a.ValidFrom, a.ExpireAfter, a.StartTime, a.FinishTime, a.LastUpdateTime,
		a.Status, aPGPSignatures, a.SyntaxVersion)
	if err != nil {
		return fmt.Errorf("Failed to store action: '%v'", err)
	}
	return
}

// UpdateAction stores updated action fields into the database.
func (db *DB) UpdateAction(a mig.Action) (err error) {
	_, err = db.c.Exec(`UPDATE actions SET (starttime, lastupdatetime, status) = ($2, $3, $4) WHERE id=$1`,
		a.ID, a.StartTime, a.LastUpdateTime, a.Status)
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
	_, err = db.c.Exec(`UPDATE actions SET (lastupdatetime) = ($2) WHERE id=$1`,
		a.ID, a.LastUpdateTime)
	if err != nil {
		return fmt.Errorf("Failed to update action: '%v'", err)
	}
	return
}

// FinishAction updates the action fields to mark it as done
func (db *DB) FinishAction(a mig.Action) (err error) {
	a.FinishTime = time.Now()
	a.Status = "completed"
	_, err = db.c.Exec(`UPDATE actions SET (finishtime, lastupdatetime, status) = ($1, $2, $3) WHERE id=$4`,
		a.FinishTime, a.LastUpdateTime, a.Status, a.ID)
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

func (db *DB) GetActionCounters(aid float64) (counters mig.ActionCounters, err error) {
	rows, err := db.c.Query(`SELECT DISTINCT(status), COUNT(id) FROM commands
		WHERE actionid = $1 GROUP BY status`, aid)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while retrieving counters: '%v'", err)
		return
	}
	for rows.Next() {
		var count int
		var status string
		err = rows.Scan(&status, &count)
		if err != nil {
			err = fmt.Errorf("Error while retrieving counter: '%v'", err)
		}
		switch status {
		case mig.StatusSent:
			counters.InFlight = count
			counters.Sent += count
		case mig.StatusSuccess:
			counters.Success = count
			counters.Done += count
			counters.Sent += count
		case mig.StatusCancelled:
			counters.Cancelled = count
			counters.Done += count
			counters.Sent += count
		case mig.StatusExpired:
			counters.Expired = count
			counters.Done += count
			counters.Sent += count
		case mig.StatusFailed:
			counters.Failed = count
			counters.Done += count
			counters.Sent += count
		case mig.StatusTimeout:
			counters.TimeOut = count
			counters.Done += count
			counters.Sent += count
		}
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// SetupRunnableActions retrieves actions that are ready to run. This function is designed
// to run concurrently across multiple schedulers, by update the status of the action at
// the same time as retrieving it. It returns an array of actions rady to be run.
func (db *DB) SetupRunnableActions() (actions []mig.Action, err error) {
	rows, err := db.c.Query(`UPDATE actions SET status='scheduled'
		WHERE status='pending' AND validfrom < NOW() AND expireafter > NOW()
		RETURNING id, name, target, description, threat, operations,
		validfrom, expireafter, status, pgpsignatures, syntaxversion`)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("Error while setting up runnable actions: '%v'", err)
		return
	}
	for rows.Next() {
		var jDesc, jThreat, jOps, jSig []byte
		var a mig.Action
		err = rows.Scan(&a.ID, &a.Name, &a.Target, &jDesc, &jThreat, &jOps,
			&a.ValidFrom, &a.ExpireAfter, &a.Status, &jSig, &a.SyntaxVersion)
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
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}
