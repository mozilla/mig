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
	"strconv"
	"time"

	"mig.ninja/mig"
	"mig.ninja/mig/database/search"

	_ "github.com/lib/pq"
)

type IDs struct {
	minActionID, maxActionID   float64
	minCommandID, maxCommandID float64
	minAgentID, maxAgentID     float64
	minInvID, maxInvID         float64
	minManID, maxManID         float64
	minLdrID, maxLdrID         float64
}

const MAXFLOAT64 float64 = 9007199254740991 // 2^53-1

func makeIDsFromParams(p search.Parameters) (ids IDs, err error) {
	ids.minActionID = 0
	ids.maxActionID = MAXFLOAT64
	if p.ActionID != "∞" {
		ids.minActionID, err = strconv.ParseFloat(p.ActionID, 64)
		if err != nil {
			return
		}
		ids.maxActionID = ids.minActionID
	}
	ids.minCommandID = 0
	ids.maxCommandID = MAXFLOAT64
	if p.CommandID != "∞" {
		ids.minCommandID, err = strconv.ParseFloat(p.CommandID, 64)
		if err != nil {
			return
		}
		ids.maxCommandID = ids.minCommandID
	}
	ids.minAgentID = 0
	ids.maxAgentID = MAXFLOAT64
	if p.AgentID != "∞" {
		ids.minAgentID, err = strconv.ParseFloat(p.AgentID, 64)
		if err != nil {
			return
		}
		ids.maxAgentID = ids.minAgentID
	}
	ids.minInvID = 0
	ids.maxInvID = MAXFLOAT64
	if p.InvestigatorID != "∞" {
		ids.minInvID, err = strconv.ParseFloat(p.InvestigatorID, 64)
		if err != nil {
			return
		}
		ids.maxInvID = ids.minInvID
	}
	ids.minManID = 0
	ids.maxManID = MAXFLOAT64
	if p.ManifestID != "∞" {
		ids.minManID, err = strconv.ParseFloat(p.ManifestID, 64)
		if err != nil {
			return
		}
		ids.maxManID = ids.minManID
	}
	ids.minLdrID = 0
	ids.maxLdrID = MAXFLOAT64
	if p.LoaderID != "∞" {
		ids.minLdrID, err = strconv.ParseFloat(p.LoaderID, 64)
		if err != nil {
			return
		}
		ids.maxLdrID = ids.minLdrID
	}
	return
}

// SearchCommands returns an array of commands that match search parameters
func (db *DB) SearchCommands(p search.Parameters, doFoundAnything bool) (commands []mig.Command, err error) {
	var (
		rows *sql.Rows
	)
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	query := `SELECT commands.id, commands.status, commands.results, commands.starttime, commands.finishtime,
			actions.id, actions.name, actions.target, actions.description, actions.threat,
			actions.operations, actions.validfrom, actions.expireafter, actions.pgpsignatures,
			actions.syntaxversion, agents.id, agents.name, agents.version, agents.tags, agents.environment
		FROM	commands
			INNER JOIN actions ON ( commands.actionid = actions.id)
			INNER JOIN signatures ON ( actions.id = signatures.actionid )
			INNER JOIN investigators ON ( signatures.investigatorid = investigators.id )
			INNER JOIN agents ON ( commands.agentid = agents.id )
		WHERE `
	vals := []interface{}{}
	valctr := 0
	if p.Before.Before(time.Now().Add(search.DefaultWindow - time.Hour)) {
		query += fmt.Sprintf(`commands.starttime <= $%d `, valctr+1)
		vals = append(vals, p.Before)
		valctr += 1
	}
	if p.After.After(time.Now().Add(-(search.DefaultWindow - time.Hour))) {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`commands.starttime >= $%d `, valctr+1)
		vals = append(vals, p.After)
		valctr += 1
	}
	if p.CommandID != "∞" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`commands.id >= $%d AND commands.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minCommandID, ids.maxCommandID)
		valctr += 2
	}
	if p.Status != "%" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`commands.status ILIKE $%d`, valctr+1)
		vals = append(vals, p.Status)
		valctr += 1
	}
	if p.ActionID != "∞" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`actions.id >= $%d AND actions.id <= $%d`, valctr+1, valctr+2)
		vals = append(vals, ids.minActionID, ids.maxActionID)
		valctr += 2
	}
	if p.ActionName != "%" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`actions.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.ActionName)
		valctr += 1
	}
	if p.InvestigatorID != "∞" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`investigators.id >= $%d AND investigators.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minInvID, ids.maxInvID)
		valctr += 2
	}
	if p.InvestigatorName != "%" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`investigators.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.InvestigatorName)
		valctr += 1
	}
	if p.AgentID != "∞" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`agents.id >= $%d AND agents.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minAgentID, ids.maxAgentID)
		valctr += 2
	}
	if p.AgentName != "%" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`agents.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentName)
		valctr += 1
	}
	if p.AgentVersion != "%" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`agents.version ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentVersion)
		valctr += 1
	}
	if doFoundAnything {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`commands.status = $%d
			AND commands.id IN (	SELECT commands.id FROM commands, actions, json_array_elements(commands.results) as r
						WHERE commands.actionid=actions.id
						AND actions.id >= $%d AND actions.id <= $%d
						AND r#>>'{foundanything}' = $%d) `,
			valctr+1, valctr+2, valctr+3, valctr+4)
		vals = append(vals, mig.StatusSuccess, ids.minActionID, ids.maxActionID, p.FoundAnything)
		valctr += 4
	}
	if p.ThreatFamily != "%" {
		if valctr > 0 {
			query += " AND "
		}
		query += fmt.Sprintf(`actions.threat#>>'{family}' ILIKE $%d `, valctr+1)
		vals = append(vals, p.ThreatFamily)
		valctr += 1
	}
	query += fmt.Sprintf(` GROUP BY commands.id, actions.id, agents.id
		ORDER BY commands.starttime DESC LIMIT $%d OFFSET $%d;`, valctr+1, valctr+2)
	vals = append(vals, uint64(p.Limit), uint64(p.Offset))

	stmt, err := db.c.Prepare(query)
	if stmt != nil {
		defer stmt.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while preparing search statement: '%v' in '%s'", err, query)
		return
	}
	rows, err = stmt.Query(vals...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while finding commands: '%v'", err)
		return
	}
	for rows.Next() {
		var jRes, jDesc, jThreat, jOps, jSig, jAgtTags, jAgtEnv []byte
		var cmd mig.Command
		err = rows.Scan(&cmd.ID, &cmd.Status, &jRes, &cmd.StartTime, &cmd.FinishTime,
			&cmd.Action.ID, &cmd.Action.Name, &cmd.Action.Target, &jDesc, &jThreat, &jOps,
			&cmd.Action.ValidFrom, &cmd.Action.ExpireAfter, &jSig, &cmd.Action.SyntaxVersion,
			&cmd.Agent.ID, &cmd.Agent.Name, &cmd.Agent.Version, &jAgtTags, &jAgtEnv)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve command: '%v'", err)
			return
		}
		err = json.Unmarshal(jThreat, &cmd.Action.Threat)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
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
		err = json.Unmarshal(jAgtTags, &cmd.Agent.Tags)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal agent tags: '%v'", err)
			return
		}
		err = json.Unmarshal(jAgtEnv, &cmd.Agent.Env)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal agent environment: '%v'", err)
			return
		}
		cmd.Action.Counters, err = db.GetActionCounters(cmd.Action.ID)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve action counters: '%v'", err)
			return
		}
		cmd.Action.Investigators, err = db.InvestigatorByActionID(cmd.Action.ID)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve action investigators: '%v'", err)
			return
		}
		commands = append(commands, cmd)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// SearchActions returns an array of actions that match search parameters
func (db *DB) SearchActions(p search.Parameters) (actions []mig.Action, err error) {
	var (
		rows                                     *sql.Rows
		joinAgent, joinInvestigator, joinCommand bool = false, false, false
	)
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	columns := `actions.id, actions.name, actions.target,  actions.description, actions.threat, actions.operations,
		actions.validfrom, actions.expireafter, actions.starttime, actions.finishtime, actions.lastupdatetime,
		actions.status, actions.pgpsignatures, actions.syntaxversion `
	join := ""
	where := ""
	vals := []interface{}{}
	valctr := 0
	if p.Before.Before(time.Now().Add(search.DefaultWindow - time.Hour)) {
		where += fmt.Sprintf(`actions.expireafter <= $%d `, valctr+1)
		vals = append(vals, p.Before)
		valctr += 1
	}
	if p.After.After(time.Now().Add(-(search.DefaultWindow - time.Hour))) {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.validfrom >= $%d `, valctr+1)
		vals = append(vals, p.After)
		valctr += 1
	}
	if p.Status != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`action.status ILIKE $%d`, valctr+1)
		vals = append(vals, p.Status)
		valctr += 1
	}
	if p.ActionID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.id >= $%d AND actions.id <= $%d`, valctr+1, valctr+2)
		vals = append(vals, ids.minActionID, ids.maxActionID)
		valctr += 2
	}
	if p.ActionName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.ActionName)
		valctr += 1
	}
	if p.InvestigatorID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.id >= $%d AND investigators.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minInvID, ids.maxInvID)
		valctr += 2
		joinInvestigator = true
	}
	if p.InvestigatorName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.InvestigatorName)
		valctr += 1
		joinInvestigator = true
	}
	if p.AgentID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.id >= $%d AND agents.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minAgentID, ids.maxAgentID)
		valctr += 2
		joinAgent = true
		joinCommand = true
	}
	if p.AgentName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentName)
		valctr += 1
		joinAgent = true
		joinCommand = true
	}
	if p.AgentVersion != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.version ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentVersion)
		valctr += 1
		joinAgent = true
		joinCommand = true
	}
	if p.CommandID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`commands.id >= $%d AND commands.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minCommandID, ids.maxCommandID)
		valctr += 2
		joinCommand = true
	}
	if joinCommand {
		join += "INNER JOIN commands ON ( commands.actionid = actions.id) "
	}
	if joinAgent {
		join += " INNER JOIN agents ON ( commands.agentid = agents.id ) "
	}
	if joinInvestigator {
		join += ` INNER JOIN signatures ON ( actions.id = signatures.actionid )
			INNER JOIN investigators ON ( signatures.investigatorid = investigators.id ) `
	}
	if p.ThreatFamily != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.threat#>>'{family}' ILIKE $%d `, valctr+1)
		vals = append(vals, p.ThreatFamily)
		valctr += 1
	}
	query := fmt.Sprintf(`SELECT %s FROM actions %s WHERE %s GROUP BY actions.id
		ORDER BY actions.validfrom DESC LIMIT $%d OFFSET $%d;`,
		columns, join, where, valctr+1, valctr+2)
	vals = append(vals, uint64(p.Limit), uint64(p.Offset))

	stmt, err := db.c.Prepare(query)
	if stmt != nil {
		defer stmt.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while preparing search statement: '%v' in '%s'", err, query)
		return
	}
	rows, err = stmt.Query(vals...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while finding actions: '%v'", err)
		return
	}
	for rows.Next() {
		var jDesc, jThreat, jOps, jSig []byte
		var a mig.Action
		err = rows.Scan(&a.ID, &a.Name, &a.Target,
			&jDesc, &jThreat, &jOps, &a.ValidFrom, &a.ExpireAfter,
			&a.StartTime, &a.FinishTime, &a.LastUpdateTime, &a.Status,
			&jSig, &a.SyntaxVersion)
		if err != nil {
			err = fmt.Errorf("Error while retrieving action: '%v'", err)
			return
		}
		err = json.Unmarshal(jThreat, &a.Threat)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal action threat: '%v'", err)
			return
		}
		err = json.Unmarshal(jDesc, &a.Description)
		if err != nil {
			err = fmt.Errorf("Failed to unmarshal action description: '%v'", err)
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
			err = fmt.Errorf("Failed to retrieve action counters: '%v'", err)
			return
		}
		a.Investigators, err = db.InvestigatorByActionID(a.ID)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve action investigators: '%v'", err)
			return
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

// SearchAgents returns an array of agents that match search parameters
func (db *DB) SearchAgents(p search.Parameters) (agents []mig.Agent, err error) {
	var (
		rows                                      *sql.Rows
		joinAction, joinInvestigator, joinCommand bool = false, false, false
	)
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	columns := `agents.id, agents.name, agents.queueloc, agents.mode,
		agents.version, agents.pid, agents.starttime, agents.destructiontime,
		agents.heartbeattime, agents.status, agents.tags, agents.environment`
	join := ""
	where := ""
	vals := []interface{}{}
	valctr := 0
	if p.Before.Before(time.Now().Add(search.DefaultWindow - time.Hour)) {
		where += fmt.Sprintf(`agents.heartbeattime <= $%d `, valctr+1)
		vals = append(vals, p.Before)
		valctr += 1
	}
	if p.After.After(time.Now().Add(-(search.DefaultWindow - time.Hour))) {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.heartbeattime >= $%d `, valctr+1)
		vals = append(vals, p.After)
		valctr += 1
	}
	if p.AgentID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.id >= $%d AND agents.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minAgentID, ids.maxAgentID)
		valctr += 2
	}
	if p.AgentName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentName)
		valctr += 1
	}
	if p.AgentVersion != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.version ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentVersion)
		valctr += 1
	}
	if p.Status != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.status ILIKE $%d`, valctr+1)
		vals = append(vals, p.Status)
		valctr += 1
	}
	if p.ActionID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.id >= $%d AND actions.id <= $%d`, valctr+1, valctr+2)
		vals = append(vals, ids.minActionID, ids.maxActionID)
		valctr += 2
		joinAction = true
		joinCommand = true
	}
	if p.ActionName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.ActionName)
		valctr += 1
		joinAction = true
		joinCommand = true
	}
	if p.ThreatFamily != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.threat#>>'{family}' ILIKE $%d `, valctr+1)
		vals = append(vals, p.ThreatFamily)
		valctr += 1
		joinAction = true
		joinCommand = true
	}
	if p.InvestigatorID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.id >= $%d AND investigators.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minInvID, ids.maxInvID)
		valctr += 2
		joinInvestigator = true
		joinCommand = true
		joinAction = true
	}
	if p.InvestigatorName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.InvestigatorName)
		valctr += 1
		joinInvestigator = true
		joinCommand = true
		joinAction = true
	}
	if p.CommandID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`commands.id >= $%d AND commands.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minCommandID, ids.maxCommandID)
		valctr += 2
		joinCommand = true
	}
	if joinCommand {
		join += "INNER JOIN commands ON ( commands.agentid = agents.id) "
	}
	if joinAction {
		join += " INNER JOIN actions ON ( commands.actionid = actions.id ) "
	}
	if joinInvestigator {
		join += ` INNER JOIN signatures ON ( actions.id = signatures.actionid )
			INNER JOIN investigators ON ( signatures.investigatorid = investigators.id ) `
	}
	query := fmt.Sprintf(`SELECT %s FROM agents %s WHERE %s GROUP BY agents.id
		ORDER BY agents.heartbeattime DESC LIMIT $%d OFFSET $%d;`,
		columns, join, where, valctr+1, valctr+2)
	vals = append(vals, uint64(p.Limit), uint64(p.Offset))

	stmt, err := db.c.Prepare(query)
	if stmt != nil {
		defer stmt.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while preparing search statement: '%v' in '%s'", err, query)
		return
	}
	rows, err = stmt.Query(vals...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while finding agents: '%v'", err)
		return
	}
	for rows.Next() {
		var agent mig.Agent
		var jTags, jEnv []byte
		err = rows.Scan(&agent.ID, &agent.Name, &agent.QueueLoc, &agent.Mode, &agent.Version,
			&agent.PID, &agent.StartTime, &agent.DestructionTime, &agent.HeartBeatTS,
			&agent.Status, &jTags, &jEnv)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve agent data: '%v'", err)
			return
		}
		err = json.Unmarshal(jTags, &agent.Tags)
		if err != nil {
			return
		}
		err = json.Unmarshal(jEnv, &agent.Env)
		if err != nil {
			return
		}
		agents = append(agents, agent)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}

	return
}

// SearchInvestigators returns an array of investigators that match search parameters
func (db *DB) SearchInvestigators(p search.Parameters) (investigators []mig.Investigator, err error) {
	var (
		rows                               *sql.Rows
		joinAction, joinAgent, joinCommand bool = false, false, false
	)
	ids, err := makeIDsFromParams(p)
	if err != nil {
		return
	}
	columns := `investigators.id, investigators.name, investigators.pgpfingerprint,
		investigators.status, investigators.createdat, investigators.lastmodified,
		investigators.permissions`
	join := ""
	where := ""
	vals := []interface{}{}
	valctr := 0
	if p.Before.Before(time.Now().Add(search.DefaultWindow - time.Hour)) {
		where += fmt.Sprintf(`investigators.lastmodified <= $%d `, valctr+1)
		vals = append(vals, p.Before)
		valctr += 1
	}
	if p.After.After(time.Now().Add(-(search.DefaultWindow - time.Hour))) {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.lastmodified >= $%d `, valctr+1)
		vals = append(vals, p.After)
		valctr += 1
	}
	if p.InvestigatorID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.id >= $%d AND investigators.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minInvID, ids.maxInvID)
		valctr += 2
	}
	if p.InvestigatorName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.InvestigatorName)
		valctr += 1
	}
	if p.Status != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`investigators.status ILIKE $%d`, valctr+1)
		vals = append(vals, p.Status)
		valctr += 1
	}
	if p.ActionID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.id >= $%d AND actions.id <= $%d`, valctr+1, valctr+2)
		vals = append(vals, ids.minActionID, ids.maxActionID)
		valctr += 2
		joinAction = true
	}
	if p.ActionName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.ActionName)
		valctr += 1
		joinAction = true
	}
	if p.ThreatFamily != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`actions.threat#>>'{family}' ILIKE $%d `, valctr+1)
		vals = append(vals, p.ThreatFamily)
		valctr += 1
		joinAction = true
	}
	if p.CommandID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`commands.id >= $%d AND commands.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minCommandID, ids.maxCommandID)
		valctr += 2
		joinCommand = true
		joinAction = true
	}
	if p.AgentID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.id >= $%d AND agents.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minAgentID, ids.maxAgentID)
		valctr += 2
		joinCommand = true
		joinAction = true
		joinAgent = true
	}
	if p.AgentName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentName)
		valctr += 1
		joinCommand = true
		joinAction = true
		joinAgent = true
	}
	if p.AgentVersion != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`agents.version ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentVersion)
		valctr += 1
		joinCommand = true
		joinAction = true
		joinAgent = true
	}
	if joinAction {
		join += ` INNER JOIN signatures ON ( signatures.investigatorid = investigators.id ) 
			INNER JOIN actions ON ( actions.id = signatures.actionid ) `
	}
	if joinCommand {
		join += "INNER JOIN commands ON ( commands.actionid = actions.id) "
	}
	if joinAgent {
		join += " INNER JOIN agents ON ( commands.agentid = agents.id ) "
	}
	query := fmt.Sprintf(`SELECT %s FROM investigators %s WHERE %s GROUP BY investigators.id
		ORDER BY investigators.id ASC LIMIT $%d OFFSET $%d;`,
		columns, join, where, valctr+1, valctr+2)
	vals = append(vals, uint64(p.Limit), uint64(p.Offset))

	stmt, err := db.c.Prepare(query)
	if stmt != nil {
		defer stmt.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while preparing search statement: '%v' in '%s'", err, query)
		return
	}
	rows, err = stmt.Query(vals...)
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		err = fmt.Errorf("Error while finding investigators: '%v'", err)
		return
	}
	for rows.Next() {
		var (
			inv  mig.Investigator
			perm int64
		)
		err = rows.Scan(&inv.ID, &inv.Name, &inv.PGPFingerprint, &inv.Status, &inv.CreatedAt, &inv.LastModified, &perm)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve investigator data: '%v'", err)
			return
		}
		inv.Permissions.FromMask(perm)
		investigators = append(investigators, inv)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

func (db *DB) SearchManifests(p search.Parameters) (mrecords []mig.ManifestRecord, err error) {
	var rows *sql.Rows
	ids, err := makeIDsFromParams(p)
	columns := `manifests.id, manifests.name, manifests.status, manifests.target, manifests.timestamp`
	where := ""
	vals := []interface{}{}
	valctr := 0
	if p.Before.Before(time.Now().Add(search.DefaultWindow - time.Hour)) {
		where += fmt.Sprintf(`manifests.timestamp <= $%d `, valctr+1)
		vals = append(vals, p.Before)
		valctr += 1
	}
	if p.After.After(time.Now().Add(-(search.DefaultWindow - time.Hour))) {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`manifests.timestamp >= $%d `, valctr+1)
		vals = append(vals, p.After)
		valctr += 1
	}
	if p.ManifestName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`manifests.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.ManifestName)
		valctr += 1
	}
	if p.ManifestID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`manifests.id >= $%d AND manifests.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minManID, ids.maxManID)
		valctr += 2
	}
	if p.Status != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`manifests.status ILIKE $%d`, valctr+1)
		vals = append(vals, p.Status)
		valctr += 1
	}
	query := fmt.Sprintf(`SELECT %s FROM manifests WHERE %s ORDER BY timestamp DESC;`, columns, where)
	stmt, err := db.c.Prepare(query)
	if err != nil {
		err = fmt.Errorf("Error while preparing search statement: '%v' in '%s'", err, query)
		return
	}
	if stmt != nil {
		defer stmt.Close()
	}
	rows, err = stmt.Query(vals...)
	if err != nil {
		err = fmt.Errorf("Error while finding manifests: '%v'", err)
	}
	if rows != nil {
		defer rows.Close()
	}
	for rows.Next() {
		var mr mig.ManifestRecord
		err = rows.Scan(&mr.ID, &mr.Name, &mr.Status, &mr.Target, &mr.Timestamp)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve manifest data: '%v'", err)
			return
		}
		mrecords = append(mrecords, mr)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}

func (db *DB) SearchLoaders(p search.Parameters) (lrecords []mig.LoaderEntry, err error) {
	var rows *sql.Rows
	ids, err := makeIDsFromParams(p)
	columns := `loaders.id, loaders.loadername, loaders.name, loaders.lastseen, loaders.enabled`
	where := ""
	vals := []interface{}{}
	valctr := 0
	if p.Before.Before(time.Now().Add(search.DefaultWindow - time.Hour)) {
		where += fmt.Sprintf(`loaders.lastseen <= $%d `, valctr+1)
		vals = append(vals, p.Before)
		valctr += 1
	}
	if p.After.After(time.Now().Add(-(search.DefaultWindow - time.Hour))) {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`loaders.lastseen >= $%d `, valctr+1)
		vals = append(vals, p.After)
		valctr += 1
	}
	if p.LoaderName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`loaders.loadername ILIKE $%d`, valctr+1)
		vals = append(vals, p.LoaderName)
		valctr += 1
	}
	if p.AgentName != "%" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`loaders.name ILIKE $%d`, valctr+1)
		vals = append(vals, p.AgentName)
		valctr += 1
	}
	if p.LoaderID != "∞" {
		if valctr > 0 {
			where += " AND "
		}
		where += fmt.Sprintf(`loaders.id >= $%d AND loaders.id <= $%d`,
			valctr+1, valctr+2)
		vals = append(vals, ids.minLdrID, ids.maxLdrID)
		valctr += 2
	}
	query := fmt.Sprintf(`SELECT %s FROM loaders WHERE %s ORDER BY loadername;`, columns, where)
	stmt, err := db.c.Prepare(query)
	if err != nil {
		err = fmt.Errorf("Error while preparing search statement: '%v' in '%s'", err, query)
		return
	}
	if stmt != nil {
		defer stmt.Close()
	}
	rows, err = stmt.Query(vals...)
	if err != nil {
		err = fmt.Errorf("Error while finding loaders: '%v'", err)
	}
	if rows != nil {
		defer rows.Close()
	}
	for rows.Next() {
		var le mig.LoaderEntry
		var agtnameNull sql.NullString
		err = rows.Scan(&le.ID, &le.Name, &agtnameNull, &le.LastSeen, &le.Enabled)
		if err != nil {
			err = fmt.Errorf("Failed to retrieve loader data: '%v'", err)
			return
		}
		le.AgentName = "unset"
		if agtnameNull.Valid {
			le.AgentName = agtnameNull.String
		}
		lrecords = append(lrecords, le)
	}
	if err := rows.Err(); err != nil {
		err = fmt.Errorf("Failed to complete database query: '%v'", err)
	}
	return
}
