// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package api

// Identifier contains a unique identifier for a resource managed by
// the client daemon.
type Identifier string

// EmptyID can be used to indicate that an ID was not created.
const EmptyID Identifier = Identifier("")
