// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

// import (
// 	"net/http"
// 	"testing"
// )

// func TestPGPAuthorizationAuthenticate(t *testing.T) {
// 	testCases := []struct {
// 		Description            string
// 		ShouldProvideSignature bool
// 		ExpectError            bool
// 	}{
// 		{
// 			Description: `
// We should be able to add authentication information for requests when a signed
// token has been provided.
// 			`,
// 			ShouldProvideSignature: true,
// 			ExpectError:            false,
// 		},
// 		{
// 			Description: `
// Adding authentication information to a request should fail if a signed token
// has not been provided.
// 			`,
// 			ShouldProvideSignature: false,
// 			ExpectError:            true,
// 		},
// 	}

// 	for caseNum, testCase := range testCases {
// 		t.Logf("Running TestPGPAuthorizationAuthenticate case #%d.\n%s\n", caseNum, testCase.Description)

// 		auth := NewPGPAuthorization()
// 		base := auth.GenerateUnsignedToken()

// 		if testCase.ShouldProvideSignature {
// 			token := base.ProvideSignature("testsignature")
// 			auth.StoreSignedToken(token)
// 		}

// 		request, _ := http.NewRequest("GET", "http://mig.ninja", nil)

// 		err := auth.Authenticate(request)
// 		gotErr := err != nil

// 		if testCase.ExpectError && !gotErr {
// 			t.Errorf("Expected to get an error, but did not.")
// 		} else if !testCase.ExpectError && gotErr {
// 			t.Errorf("Did not expect to get an error, but got %s", err.Error())
// 		}

// 		if !testCase.ExpectError && request.Header.Get("X-PGPAUTHORIZATION") == "" {
// 			t.Errorf("Expected the X-PGPAUTHORIZATION header to be set but it is not.")
// 		}
// 	}
// }
