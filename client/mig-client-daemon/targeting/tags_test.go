// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

import (
	"testing"
)

func TestByTagToSQLWhereClause(t *testing.T) {
	expectedSQL := "tags->>'test' = 'testvalue'"

	query := ByTag{
		TagName:  "test",
		TagValue: "testvalue",
	}

	whereClause, err := query.ToSQLWhereClause()
	if err != nil {
		t.Fatal(err)
	}
	if whereClause != expectedSQL {
		t.Errorf("Expected to get WHERE clause\n\t%s\nbut got\n\t%s", expectedSQL, whereClause)
	}
}
