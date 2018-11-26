// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package agents

// NilAuthenticator implements Authenticator in such a way that all
// attempts to upload a heartbeat will be allowed.
type NilAuthenticator struct{}

// NewNilAuthenticator constructs a new NilAuthenticator.
func NewNilAuthenticator() NilAuthenticator {
	return NilAuthenticator{}
}

// Authenticate always returns nil.
func (auth NilAuthenticator) Authenticate(_ Heartbeat) error {
	return nil
}
