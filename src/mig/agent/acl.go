/* Mozilla InvestiGator Agent

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]
Guillaume Destuynder <kang@mozilla.com>

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package main

import (
	"fmt"
	"mig"
	"mig/pgp"
	"time"
)

// checkActionAuthorization verifies the PGP signatures of a given action
// against the Access Control List of the agent.
func checkActionAuthorization(a mig.Action, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("checkActionAuthorization() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{ActionID: a.ID, Desc: "leaving checkActionAuthorization()"}.Debug()
	}()
	// get an io.Reader from the public pgp key
	keyring, keycount, err := pgp.ArmoredPubKeysToKeyring(PUBLICPGPKEYS[0:])
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
