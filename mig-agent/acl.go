// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
// - Guillaume Destuynder <kang@mozilla.com>
package main

import (
	"fmt"
	"mig.ninja/mig"
	"mig.ninja/mig/pgp"
	"time"
)

// checkActionAuthorization verifies the PGP signatures of a given action
// against the Access Control List of the agent.
func checkActionAuthorization(a mig.Action, ctx *Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("checkActionAuthorization() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "leaving checkActionAuthorization()"}.Debug()
	}()
	var keys [][]byte
	for _, pk := range PUBLICPGPKEYS {
		keys = append(keys, []byte(pk))
	}
	// get an io.Reader from the public pgp key
	keyring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: fmt.Sprintf("loaded %d keys", keycount)}.Debug()

	// Check the action syntax and signature
	err = a.Validate()
	if err != nil {
		desc := fmt.Sprintf("action validation failed: %v", err)
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: desc}.Err()
		panic(desc)
	}
	// Validate() checks that the action hasn't expired, but we need to
	// check the start time ourselves
	if time.Now().Before(a.ValidFrom) {
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "action is scheduled for later"}.Err()
		panic("Action ValidFrom date is in the future")
	}

	// check ACLs, includes verifying signatures
	err = a.VerifyACL(ctx.ACL, keyring)
	if err != nil {
		desc := fmt.Sprintf("action ACL verification failed: %v", err)
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: desc}.Err()
		panic(desc)
	}

	ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "ACL verification succeeded."}.Debug()
	return
}
