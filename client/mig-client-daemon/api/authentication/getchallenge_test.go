// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package authentication

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChallengeHandler(t *testing.T) {
	t.Logf("The GetChallengeHandler should respond with a unique token.")

	testCases := []struct{
		ChallengeReturned *string
	}{
		{
			ChallengeReturned: nil,
		},
		{
			ChallengeReturned: nil,
		},
		{
			ChallengeReturned: nil,
		},
	}

	handler := NewGetChallengeHandler()
	server := httptest.NewServer(handler)

	for caseNum := 0; caseNum < len(testCases); caseNum++ {
		reqURL := fmt.Sprintf("%s/v1/authentication/pgp", server.URL)

		response, err := http.Get(reqURL)
		if err != nil {
			t.Fatal(err)
		}
		respData := getChallengeResponse{}
		decoder := json.NewDecoder(response.Body)
		defer response.Body.Close()
		err = decoder.Decode(&respData)
		if err != nil {
			t.Fatal(err)
		}

		testCases[caseNum].ChallengeReturned = new(string)
		*testCases[caseNum].ChallengeReturned = respData.Challenge
	}

	foundDuplicate := false
test:
	for firstCase := 0; firstCase < len(testCases); firstCase++ {
		for secondCase := firstCase + 1; secondCase < len(testCases); secondCase++ {
			if testCases[firstCase].ChallengeReturned == nil || testCases[secondCase].ChallengeReturned == nil {
				t.Fatalf("Didn't get a valid challenge")
			}
			if *testCases[firstCase].ChallengeReturned == *testCases[secondCase].ChallengeReturned {
				foundDuplicate = true
				break test
			}
		}
	}

	if foundDuplicate {
		t.Errorf("Expected all of the challenges to be unique, but a duplicate was found.")
	}
}