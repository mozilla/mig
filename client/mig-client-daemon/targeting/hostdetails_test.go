// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"testing"
)

func TestByHostDetailsToSQLWhereClause(t *testing.T) {
	linux := "linux"
	amd64 := "amd64"

	testCases := []struct {
		Description string
		Query       ByHostDetails
		ExpectError bool
		ExpectedSQL string
	}{
		{
			Description: `
			If no fields are provided to query, we should get an error.
			`,
			Query:       ByHostDetails{},
			ExpectError: true,
			ExpectedSQL: "",
		},
		{
			Description: `
			If at least one field is provided to query, we should not get an error.
			`,
			Query: ByHostDetails{
				OS: &linux,
			},
			ExpectError: false,
			ExpectedSQL: "environment->>'os' = 'linux'",
		},
		{
			Description: `
			If multiple fields are provided to query, each should be present in the resulting SQL.
			`,
			Query: ByHostDetails{
				OS:   &linux,
				Arch: &amd64,
			},
			ExpectError: false,
			ExpectedSQL: "environment->>'os' = 'linux' AND environment->>'arch' = 'amd64'",
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestByHostDetailsToSQLWhereClause case #%d;\n\t%s\n", caseNum, testCase.Description)

		whereClause, err := testCase.Query.ToSQLWhereClause()
		gotErr := err != nil

		if !testCase.ExpectError && gotErr {
			t.Errorf("Did not expect an error, but got %s", err.Error())
		} else if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error, but did not.")
		}

		if testCase.ExpectedSQL != whereClause {
			t.Errorf("Expected to get WHERE clause containing\n\t%s\nbut got\n\t%s", testCase.ExpectedSQL, whereClause)
		}
	}
}
