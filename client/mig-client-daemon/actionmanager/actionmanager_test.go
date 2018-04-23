// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actionmanager

import (
	"testing"
	"time"
	
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

func TestCreateAction(t *testing.T) {
	// Some data to create targeting queries with.
	queueLoc := "linux"
	online := "online"
	tagName := "operator"
	tagValue := "IT"

	testCases := []struct {
		Description string,
		Module modules.Module
		TargetQueries []targeting.Query,
		Expiration time.Duration,
		ExpectError bool,
	}{
		{
			Description: `
			Given a valid set of target queries, should be able to create
			a new action.
			`,
			Module: modules.PkgModule {
				name: "*libssl*",
				version: nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByAgentDetails{
					ID: nil,
					Name: nil,
					QueueLocation: &queueLoc,
					Version: nil,
					Pid: nil,
					Status: &online,
				},
				targeting.ByTag{
					TagName: &tagName,
					value: &tagValue,
				},
			},
			Expiration: 1 * time.Hour,
			ExpectError: false,
		},
		{
			Description: `
			Given an invalid set of target queries, creating a new action
			should fail.
			`,
			Module: modules.PkgModule {
				name: "*libssl*",
				version: nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByHostDetails{
					Ident: nil,
					OS: nil,
					Arch: nil,
					PublicIP: nil,
				},
			},
			Expiration: 1 * time.Hour,
			ExpectError: true,
		}
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestCreateAction case #%d.\n\t%s\n", caseNum, testCase.Description)

		actions := NewActionCatalog()

		newId, err := actions.CreateAction(
			testCase.Module,
			testCase.TargetQueries,
			testCase.Expiration)

		gotErr := err != nil
		if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect an error, but got %s", err.Error())
		} else if testCase.ExpectError && !gotErr {
			t.Errof("Expected to get an error, but did not.")
		}
	}
}
