// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package targeting

// Query is implemented by types that can be converted into a string
// containing a condition for a SQL query's `WHERE` clause, used to
// target agents that we want to run an action.
type Query interface {
	ToSQLWhereClause() (string, error)
}
