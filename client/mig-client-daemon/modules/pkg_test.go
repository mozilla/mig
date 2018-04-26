// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"testing"

	"mig.ninja/mig/modules/pkg"
)

func TestPkgToParameters(t *testing.T) {
	version := "1.2.3"

	testCases := []struct {
		Description      string
		PackageName      string
		PackageVersion   *string
		ExpectVersionSet bool
		ExpectError      bool
	}{
		{
			Description: `
			The package name and version should be present in the output
			parameters when both are set.
			`,
			PackageName:      "*libssl*",
			PackageVersion:   &version,
			ExpectVersionSet: true,
			ExpectError:      false,
		},
		{
			Description: `
			The package version should not be present when it is left out.
			`,
			PackageName:      "*libssl*",
			PackageVersion:   nil,
			ExpectVersionSet: false,
			ExpectError:      false,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestPkgToParameters case #%d.\n\t%s\n", caseNum, testCase.Description)

		mod := Pkg{
			PackageName: testCase.PackageName,
			Version:     testCase.PackageVersion,
		}

		params, err := mod.ToParameters()

		gotErr := err != nil
		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		} else if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", err.Error())
		}

		verMatch := params.(pkg.Parameters).VerMatch
		versionSet := verMatch != ""
		if testCase.ExpectVersionSet && !versionSet {
			t.Errorf("Expected a version to match against to be present, but none is")
		} else if !testCase.ExpectVersionSet && versionSet {
			t.Errorf("Did not expect a version to match against to be present, but it's %s", verMatch)
		}
	}
}
