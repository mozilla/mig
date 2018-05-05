// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"net/http"
	"strings"
	"testing"
)

func TestTokenString(t *testing.T) {
	t.Logf("Running TestTokenString")
	t.Logf(`
Tokens should be suffixed with the signature provided to the UnsignedToken it
is based on
	`)

	auth := NewPGPAuthorization()
	baseTkn := auth.GenerateUnsignedToken()
	token := baseTkn.ProvideSignature("testsignature")
	if !strings.HasSuffix(token.String(), "testsignature") {
		t.Errorf("Expected signed token to be suffixed with the signature we provided.")
	}
}

func TestPGPAuthorizationAuthenticate(t *testing.T) {
	testCases := []struct {
		Description            string
		ShouldProvideSignature bool
		ExpectError            bool
	}{
		{
			Description: `
We should be able to add authentication information for requests when a signed
token has been provided.
			`,
			ShouldProvideSignature: true,
			ExpectError:            false,
		},
		{
			Description: `
Adding authentication information to a request should fail if a signed token
has not been provided.
			`,
			ShouldProvideSignature: false,
			ExpectError:            true,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestPGPAuthorizationAuthenticate case #%d.\n%s\n", caseNum, testCase.Description)

		auth := NewPGPAuthorization()
		base := auth.GenerateUnsignedToken()
		var token Token

		if testCase.ShouldProvideSignature {
			token = base.ProvideSignature("testsignature")
		}

		request, _ := http.NewRequest("GET", "http://mig.ninja", nil)

		err := auth.Authenticate(request)
		gotErr := err != nil

		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		} else if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", err.Error())
		}

		if !testCase.ExpectError && request.Header().Get("X-PGPAUTHORIZATION") == "" {
			t.Errorf("Expected the X-PGPAUTHORIZATION header to be set but it is not.")
		}
	}
}
