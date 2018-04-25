// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actionmanager

import (
	"crypto/rand"
	"encoding/hex"
	//"strings"
	"time"

	"mig.ninja/mig"
	//migclient "mig.ninja/mig/client"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

// ActionCatalog maintains information about actions that have been
// created, dispatched, etc.
type ActionCatalog struct {
	actions map[string]mig.Action
}

// NewActionCatalog creates a new `ActionCatalog`.
func NewActionCatalog() ActionCatalog {
	return ActionCatalog{
		actions: make(map[string]mig.Action),
	}
}

// CreateAction attempts to create a new action with the supplied information
// and register it to the catalog, returning the identifier of the
// newly-created action as a string.
func (catalog *ActionCatalog) CreateAction(
	module modules.Module,
	agentTargetSpecifiers []targeting.Query,
	expireAfter time.Duration,
) (string, error) {
	id := catalog.generateActionID()

	queryStrings := []string{}
	for _, query := range agentTargetSpecifiers {
		whereClause, err := query.ToSQLWhereClause()
		if err != nil {
			return "", err
		}
		queryStrings = append(queryStrings, whereClause)
	}
	// target := strings.Join(queryStrings, " AND ")

	return id, nil
}

// generateActionID creates an identifier that can be used by the
// `ActionCatalog` to track an action being managed internally.
func (catalog ActionCatalog) generateActionID() string {
	bytesToGenerate := 3
	sleepBetweenReadAttempts := 250 * time.Millisecond

	randBytes := make([]byte, bytesToGenerate)

	for {
		// We don't necessarily need cryptographically secure random bytes for IDs
		// but they're reliable and easy to deal with.
		bytesRead, err := rand.Read(randBytes)
		if err != nil || bytesRead < bytesToGenerate {
			// If we encountered an error, it's probablt because the OS' pool of
			// entropy has been exhausted.  So we will just wait a little bit.
			<-time.After(sleepBetweenReadAttempts)
			continue
		}

		stringID := hex.EncodeToString(randBytes)

		_, alreadyTaken := catalog.actions[stringID]
		if alreadyTaken {
			continue
		}

		// We have generated a random ID that is not already in use.
		return stringID
	}
}
