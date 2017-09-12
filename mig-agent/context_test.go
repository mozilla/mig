// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"path"
	"testing"

	"mig.ninja/mig/mig-agent/agentcontext"
)

func TestInitKeyring(t *testing.T) {
	ctx := testContext

	agentcontext.EnableTestHooks(path.Join("testdata", "agentkeys"))

	PUBLICPGPKEYS = PUBLICPGPKEYS[:0]

	ctx, err := initKeyring(ctx)
	if err != nil {
		t.Errorf("initKeyring: %v", err)
	}

	if len(PUBLICPGPKEYS) != 1 {
		t.Errorf("expected 1 public key to be present in keyring")
	}
}

func TestInitBadKeyring(t *testing.T) {
	ctx := testContext

	agentcontext.EnableTestHooks(path.Join("testdata", "badagentkeys"))

	PUBLICPGPKEYS = PUBLICPGPKEYS[:0]

	ctx, err := initKeyring(ctx)
	if err != nil {
		t.Errorf("initKeyring: %v", err)
	}

	// We should have one key in the keyring since the bad key will have been
	// rejected
	if len(PUBLICPGPKEYS) != 1 {
		t.Errorf("expected 1 public key to be present in keyring")
	}
}
