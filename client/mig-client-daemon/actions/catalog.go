// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actions

import (
	"errors"
	"strings"
	"sync"
	"time"

	"mig.ninja/mig"
	//migclient "mig.ninja/mig/client"
	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

// The number of random bytes to generate for action identifiers.
const internalActionIDLength uint = 3

// ActionRecord contains information about an action managed by a `Catalog`.
type ActionRecord struct {
	Action     mig.Action
	Status     string
	InternalID InternalActionID
}

// Catalog maintains information about actions that have been
// created, dispatched, etc.
type Catalog struct {
	actions map[ident.Identifier]ActionRecord
	lock    *sync.Mutex
}

// NewCatalog creates a new `Catalog`.
func NewCatalog() Catalog {
	return Catalog{
		actions: make(map[ident.Identifier]ActionRecord),
		lock:    new(sync.Mutex),
	}
}

// Create attempts to create a new action with the supplied information
// and register it to the catalog, returning the identifier of the
// newly-created action as a string.
func (catalog *Catalog) Create(
	module modules.Module,
	agentTargetSpecifiers []targeting.Query,
	expireAfter time.Duration,
) (ident.Identifier, error) {
	id := ident.GenerateUniqueID(internalActionIDLength, 250*time.Millisecond, func(id ident.Identifier) bool {
		_, alreadyTaken := catalog.actions[id]
		return !alreadyTaken
	})

	queryStrings := []string{}
	for _, query := range agentTargetSpecifiers {
		whereClause, err := query.ToSQLWhereClause()
		if err != nil {
			return "", err
		}
		queryStrings = append(queryStrings, whereClause)
	}
	target := strings.Join(queryStrings, " AND ")

	moduleParams, err := module.ToParameters()
	if err != nil {
		return ident.EmptyID, err
	}

	action := mig.Action{
		Name:        string(id),
		ExpireAfter: time.Now().Add(expireAfter),
		Target:      target,
		Operations: []mig.Operation{
			{
				Module:     module.Name(),
				Parameters: moduleParams,
			},
		},
		SyntaxVersion: mig.ActionVersion,
	}

	record := ActionRecord{
		Action:     action,
		Status:     StatusPendingDispatch,
		InternalID: InvalidID,
	}
	catalog.update(id, record)

	return id, nil
}

// Lookup checks for an action with a given internal identifier in the action catalog.
func (catalog Catalog) Lookup(actionID ident.Identifier) (ActionRecord, bool) {
	record, found := catalog.actions[actionID]
	return record, found
}

// AddSignature appends a new signature to an action in the catalog.
func (catalog *Catalog) AddSignature(actionID ident.Identifier, signature string) error {
	record, found := catalog.Lookup(actionID)
	if !found {
		return errors.New("the requested action was not found")
	}

	record.Action.PGPSignatures = append(record.Action.PGPSignatures, signature)
	catalog.update(actionID, record)
	return nil
}

// MarkAsDispatched updates an action record to indicate that the action
// identified by `actionID` in the catalog has been assigned an ID by the MIG API.
func (catalog *Catalog) MarkAsDispatched(
	actionID ident.Identifier,
	internalID InternalActionID,
) error {
	record, found := catalog.actions[actionID]
	if !found {
		return errors.New("the requested action was not found")
	}

	record.Status = StatusDispatched
	record.InternalID = internalID
	catalog.update(actionID, record)
	return nil
}

// update replaces or inserts an action in the catalog.
// The update takes place in a thread-safe way, so calling `update` should be
// preferred to updating an action record directly.
func (catalog *Catalog) update(actionID ident.Identifier, newRecord ActionRecord) {
	catalog.lock.Lock()
	catalog.actions[actionID] = newRecord
	catalog.lock.Unlock()
}
