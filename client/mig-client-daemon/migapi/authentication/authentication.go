// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"net/http"
)

// Autnenticator provides services for manipulating HTTP requests to perform
// authentication to the MIG API.
type Authenticator interface {
	Authenticate(*http.Request) error
}
