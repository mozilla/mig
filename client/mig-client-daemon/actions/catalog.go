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

// Catalog maintains information about actions that have been
// created, dispatched, etc.
type Catalog struct {
	actions map[ident.Identifier]mig.Action
	lock    *sync.Mutex
}

// NewCatalog creates a new `Catalog`.
func NewCatalog() Catalog {
	return Catalog{
		actions: make(map[ident.Identifier]mig.Action),
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
	}

	err = catalog.update(id, action)
	if err != nil {
		return ident.EmptyID, err
	}

	return id, nil
}

// Lookup checks for an action with a given internal identifier in the action catalog.
func (catalog Catalog) Lookup(actionID ident.Identifier) (mig.Action, bool) {
	action, found := catalog.actions[actionID]
	return action, found
}

// AddSignature appends a new signature to an action in the catalog.
func (catalog *Catalog) AddSignature(actionID ident.Identifier, signature string) error {
	action, found := catalog.Lookup(actionID)
	if !found {
		return errors.New("The requested action was not found.")
	}

	action.PGPSignatures = append(action.PGPSignatures, signature)
	return catalog.update(actionID, action)
}

// update replaces or inserts an action in the catalog.
func (catalog *Catalog) update(actionID ident.Identifier, newAction mig.Action) error {
	catalog.lock.Lock()
	defer catalog.lock.Unlock()

	catalog.actions[actionID] = newAction
	return nil
}
