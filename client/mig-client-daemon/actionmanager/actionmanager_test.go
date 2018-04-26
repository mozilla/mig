// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package actionmanager

import (
	"testing"
	"time"

	"mig.ninja/mig/client/mig-client-daemon/ident"
	"mig.ninja/mig/client/mig-client-daemon/modules"
	"mig.ninja/mig/client/mig-client-daemon/targeting"
)

func TestCreate(t *testing.T) {
	// Some data to create targeting queries with.
	queueLoc := "linux"
	online := "online"

	testCases := []struct {
		Description   string
		Module        modules.Module
		TargetQueries []targeting.Query
		Expiration    time.Duration
		ExpectError   bool
	}{
		{
			Description: `
			Given a valid set of target queries, should be able to create
			a new action.
			`,
			Module: modules.Pkg{
				PackageName: "*libssl*",
				Version:     nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByAgentDetails{
					ID:            nil,
					Name:          nil,
					QueueLocation: &queueLoc,
					Version:       nil,
					Pid:           nil,
					Status:        &online,
				},
				targeting.ByTag{
					TagName:  "operator",
					TagValue: "IT",
				},
			},
			Expiration:  1 * time.Hour,
			ExpectError: false,
		},
		{
			Description: `
			Given an invalid set of target queries, creating a new action
			should fail.
			`,
			Module: modules.Pkg{
				PackageName: "*libssl*",
				Version:     nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByHostDetails{
					Ident:    nil,
					OS:       nil,
					Arch:     nil,
					PublicIP: nil,
				},
			},
			Expiration:  1 * time.Hour,
			ExpectError: true,
		},
		{
			Description: `
			IDs for actions produced by the ActionCatalog should be unique.
			`,
			Module: modules.Pkg{
				PackageName: "*libssl*",
				Version:     nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByAgentDetails{
					ID:            nil,
					Name:          nil,
					QueueLocation: &queueLoc,
					Version:       nil,
					Pid:           nil,
					Status:        &online,
				},
				targeting.ByTag{
					TagName:  "operator",
					TagValue: "IT",
				},
			},
			Expiration:  1 * time.Hour,
			ExpectError: false,
		},
	}

	idsGenerated := []ident.Identifier{}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestCreate case #%d.\n\t%s\n", caseNum, testCase.Description)

		actions := NewActionCatalog()

		id, err := actions.Create(
			testCase.Module,
			testCase.TargetQueries,
			testCase.Expiration)

		gotErr := err != nil
		if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect an error, but got %s", err.Error())
		} else if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		}

		for _, idSeen := range idsGenerated {
			if id == idSeen {
				t.Errorf("Expected Create to generate unique IDs, but got %s twice.", id)
			}
		}

		idsGenerated = append(idsGenerated, id)
	}
}

func TestGetAction(t *testing.T) {
	// Some data to create targeting queries with.
	queueLoc := "linux"
	online := "online"

	testCases := []struct {
		Description       string
		Module            modules.Module
		TargetQueries     []targeting.Query
		Expiration        time.Duration
		ShouldCreateFirst bool
		ExpectError       bool
		ExpectToFind      bool
	}{
		{
			Description: `
			We should be able to find actions that we successfully create.
			`,
			Module: modules.Pkg{
				PackageName: "*libssl*",
				Version:     nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByAgentDetails{
					ID:            nil,
					Name:          nil,
					QueueLocation: &queueLoc,
					Version:       nil,
					Pid:           nil,
					Status:        &online,
				},
				targeting.ByTag{
					TagName:  "operator",
					TagValue: "IT",
				},
			},
			Expiration:        1 * time.Hour,
			ShouldCreateFirst: true,
			ExpectError:       false,
			ExpectToFind:      true,
		},
		{
			Description: `
			We should not be able to find actions that are not successfully created.
			`,
			Module: modules.Pkg{
				PackageName: "*libssl*",
				Version:     nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByHostDetails{
					Ident:    nil,
					OS:       nil,
					Arch:     nil,
					PublicIP: nil,
				},
			},
			Expiration:        1 * time.Hour,
			ShouldCreateFirst: true,
			ExpectError:       true,
			ExpectToFind:      false,
		},
		{
			Description: `
			We should not be able to find actions that we don't create.
			`,
			Module: modules.Pkg{
				PackageName: "*libssl*",
				Version:     nil,
			},
			TargetQueries: []targeting.Query{
				targeting.ByHostDetails{
					Ident:    nil,
					OS:       nil,
					Arch:     nil,
					PublicIP: nil,
				},
			},
			Expiration:        1 * time.Hour,
			ShouldCreateFirst: false,
			ExpectError:       false,
			ExpectToFind:      false,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestGetAction case #%d.\n\t%s\n", caseNum, testCase.Description)

		actions := NewActionCatalog()

		lastActionID := ident.EmptyID

		if testCase.ShouldCreateFirst {
			id, err := actions.Create(
				testCase.Module,
				testCase.TargetQueries,
				testCase.Expiration)

			gotErr := err != nil
			if !testCase.ExpectError && gotErr {
				t.Errorf("Did not expect to get an error, but got %s", err.Error())
			} else if testCase.ExpectError && !gotErr {
				t.Errorf("Expected to get an error, but did not")
			}

			lastActionID = id
		}

		action, found := actions.Lookup(lastActionID)
		if testCase.ExpectToFind && !found {
			t.Errorf("Expected to find a newly-created action in the action catalog, but did not")
		} else if !testCase.ExpectToFind && found {
			t.Errorf("Did not expect to find an action, but we did")
		} else if testCase.ExpectToFind && action.Target == "" {
			t.Errorf("Expected to find an action, but got one that hasn't been initialized")
		}
	}
}
