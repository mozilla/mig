// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package database /* import "mig.ninja/mig/database" */

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/modules"

	_ "github.com/lib/pq"
)

// CommandByID retrieves a command from the database using its ID
func (db *DB) CommandByID(id float64) (cmd mig.Command, err error) {
	var jRes, jDesc, jThreat, jOps, jSig []byte
	err = db.c.QueryRow(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
		actions.id, actions.name, actions.target, actions.description, actions.threat,
		actions.operations, actions.validfrom, actions.expireafter,
		actions.pgpsignatures, actions.syntaxversion,
		agents.id, agents.name, agents.queueloc, agents.mode, agents.version
		FROM commands, actions, agents
		WHERE commands.id=$1
		AND commands.actionid = actions.id AND commands.agentid = agents.id`, id).Scan(
		&cmd.ID, &cmd.Status, &jRes, &cmd.StartTime, &cmd.FinishTime,
		&cmd.Action.ID, &cmd.Action.Name, &cmd.Action.Target, &jDesc, &jThreat, &jOps,
		&cmd.Action.ValidFrom, &cmd.Action.ExpireAfter, &jSig, &cmd.Action.SyntaxVersion,
		&cmd.Agent.ID, &cmd.Agent.Name, &cmd.Agent.QueueLoc, &cmd.Agent.Mode, &cmd.Agent.Version)
	if err != nil {
		err = fmt.Errorf("Error while retrieving command: '%v'", err)
		return
	}
	if err == sql.ErrNoRows {
		return
	}
	err = json.Unmarshal(jRes, &cmd.Results)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal command results: '%v'", err)
		return
	}
	err = json.Unmarshal(jDesc, &cmd.Action.Description)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
		return
	}
	err = json.Unmarshal(jThreat, &cmd.Action.Threat)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
		return
	}
	err = json.Unmarshal(jOps, &cmd.Action.Operations)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
		return
	}
	err = json.Unmarshal(jSig, &cmd.Action.PGPSignatures)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
		return
	}
	return
}

func (db *DB) CommandsByActionID(actionid float64) (commands []mig.Command, err error) {
	rows, err := db.c.Query(`SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
		actions.id, actions.name, actions.target, actions.description, actions.threat,
		actions.operations, actions.validfrom, actions.expireafter,
		actions.pgpsignatures, actions.syntaxversion,
		agents.id, agents.name, agents.version
		FROM commands, actions, agents
		WHERE commands.actionid=actions.id AND commands.agentid=agents.id AND actions.id=$1`, actionid)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while finding commands: '%v'", err)
		return
	}
	for rows.Next() {
		var jRes, jDesc, jThreat, jOps, jSig []byte
		var cmd mig.Command
		err = rows.Scan(&cmd.ID, &cmd.Status, &jRes, &cmd.StartTime, &cmd.FinishTime,
			&cmd.Action.ID, &cmd.Action.Name, &cmd.Action.Target, &jDesc, &jThreat, &jOps,
			&cmd.Action.ValidFrom, &cmd.Action.ExpireAfter, &jSig, &cmd.Action.SyntaxVersion,
			&cmd.Agent.ID, &cmd.Agent.Name, &cmd.Agent.Version)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve command: '%v'", err)
			return
		}
		err = json.Unmarshal(jRes, &cmd.Results)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal command results: '%v'", err)
			return
		}
		err = json.Unmarshal(jDesc, &cmd.Action.Description)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
			return
		}
		err = json.Unmarshal(jThreat, &cmd.Action.Threat)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
			return
		}
		err = json.Unmarshal(jOps, &cmd.Action.Operations)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal action operations: '%v'", err)
			return
		}
		err = json.Unmarshal(jSig, &cmd.Action.PGPSignatures)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal action signatures: '%v'", err)
			return
		}
		commands = append(commands, cmd)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// InsertCommand writes a command into the database
func (db *DB) InsertCommand(cmd mig.Command, agt mig.Agent) (err error) {
	jResults, err := json.Marshal(cmd.Results)
	if err != nil {
		return fmt.Errorf("Failed to marshal results: '%v'", err)
	}
	_, err = db.c.Exec(`INSERT INTO commands
		(id, actionid, agentid, status, results, starttime, finishtime)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, cmd.ID, cmd.Action.ID,
		agt.ID, cmd.Status, jResults, cmd.StartTime, cmd.FinishTime)
	if err != nil {
		return fmt.Errorf("Error while inserting command: '%v'", err)
	}
	return
}

// InsertCommands writes an array of commands into the database
func (db *DB) InsertCommands(cmds []mig.Command) (insertCount int64, err error) {
	futureDate := time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
	sql := "INSERT INTO commands (id, actionid, agentid, status, starttime, finishtime, results) VALUES "
	vals := []interface{}{}
	step := 0
	for i, cmd := range cmds {
		jRes, err := json.Marshal(cmd.Results)
		if err != nil {
			return int64(i), err
		}
		if i > 0 {
			sql += ", "
		}
		sql += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i+1+step, i+2+step, i+3+step, i+4+step, i+5+step, i+6+step, i+7+step)
		vals = append(vals, cmd.ID, cmd.Action.ID, cmd.Agent.ID, cmd.Status, cmd.StartTime, futureDate, jRes)
		step += 6
	}
	stmt, err := db.c.Prepare(sql)
	defer stmt.Close()
	if err != nil {
		err = fmt.Errorf("Error while preparing insertion statement: '%v' in '%s'", err, sql)
		return
	}
	res, err := stmt.Exec(vals...)
	if err != nil {
		err = fmt.Errorf("Error while inserting commands: '%v'", err)
		return
	}
	insertCount, err = res.RowsAffected()
	if err != nil {
		err = fmt.Errorf("Error while counting inserted commands: '%v'", err)
		return
	}
	return
}

// UpdateSentCommand updates a command into the database, unless its status is already
// set to 'success'
func (db *DB) UpdateSentCommand(cmd mig.Command) (err error) {
	res, err := db.c.Exec(`UPDATE commands SET status=$1 WHERE id=$2 and status!=$3`,
		cmd.Status, cmd.ID, mig.StatusSuccess)
	if err != nil {
		return fmt.Errorf("Error while updating command: '%v'", err)
	}
	ctr, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("Error while evaluating query results: '%v'", err)
	}
	if ctr != 1 {
		return fmt.Errorf("Failed to update command status correctly, %d rows affected", ctr)
	}
	return
}

// FinishCommand updates a command into the database unless its status is already set
// to 'success'. If the status has already been set to "success" (maybe by a concurrent
// scheduler), do not update further. this prevents scheduler A from expiring a command
// that has already succeeded and been returned to scheduler B.
func (db *DB) FinishCommand(cmd mig.Command) (err error) {
	jResults, err := json.Marshal(cmd.Results)
	if err != nil {
		return fmt.Errorf("Failed to marshal results: '%v'", err)
	}

	// XXX Filter any unicode NULL escape sequences present in the command
	// results before we insert. Postgres disallows this value to be present;
	// if an entry containing this value is present and JSON processing is done
	// on the entry by the database it will result in an error.
	//
	// See Postgres 9.4.1 release notes for details:
	// http://www.postgresql.org/docs/9.4/static/release-9-4-1.html
	jResults = bytes.Replace(jResults, []byte("\\u0000"), []byte("NULL"), -1)
	// Validate the result is still valid JSON before the insert
	var tmpres []modules.Result
	err = json.Unmarshal(jResults, &tmpres)
	if err != nil {
		return err
	}

	res, err := db.c.Exec(`UPDATE commands SET status=$1, results=$2, finishtime=$3
		WHERE id=$4 AND status!=$5 AND agentid IN (
			SELECT id FROM agents
			WHERE agents.queueloc=$6 AND agents.pid=$7 AND status IN ('online','idle')
		)`, cmd.Status, jResults, cmd.FinishTime, cmd.ID, mig.StatusSuccess,
		cmd.Agent.QueueLoc, cmd.Agent.PID)
	if err != nil {
		return fmt.Errorf("Error while updating command: '%v'", err)
	}
	ctr, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("Error while evaluating query results: '%v'", err)
	}
	if ctr != 1 {
		return fmt.Errorf("Failed to finish command status correctly, %d rows affected", ctr)
	}
	return
}
