// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe_test

import (
	"github.com/mozilla/scribe"
	"testing"
)

type evrTestTable struct {
	verA string
	op   string
	verB string
}

var evrTests = []evrTestTable{
	{"2:3.14.5-1+deb7u1", "=", "2:3.14.5-1+deb7u1"},
	{"0:2.6.6-6+deb7u2", "=", "0:2.6.6-6+deb7u2"},
	{"0:2.2.22-13+deb7u3", "=", "0:2.2.22-13+deb7u3"},
	{"0:0.140-5+deb7u1", "=", "0:0.140-5+deb7u1"},
	{"0:3.2.60-1+deb7u3", "=", "0:3.2.60-1+deb7u3"},
	{"0:1.5.3-5+deb7u4", "=", "0:1.5.3-5+deb7u4"},
	{"2:3.14.5-1+deb7u1", "=", "2:3.14.5-1+deb7u1"},
	{"0:5.5.38-0+wheezy1", "=", "0:5.5.38-0+wheezy1"},
	{"0:0.2.4.23-1~deb7u1", "=", "0:0.2.4.23-1~deb7u1"},
	{"0:2.06-1+deb7u1", "=", "0:2.06-1+deb7u1"},
	{"0:24.7.0esr-1~deb7u1", "=", "0:24.7.0esr-1~deb7u1"},
	{"0:2.52-3+nmu2", "=", "0:2.52-3+nmu2"},
	{"0:7u65-2.5.1-2~deb7u1", "=", "0:7u65-2.5.1-2~deb7u1"},
	{"0:24.7.0-1~deb7u1", "=", "0:24.7.0-1~deb7u1"},
	{"0:1.0.1e-2+deb7u9", "=", "0:1.0.1e-2+deb7u9"},
	{"0:6b31-1.13.3-1~deb7u1", "=", "0:6b31-1.13.3-1~deb7u1"},
	{"2:7.3.547-7", "=", "2:7.3.547-7"},
	{"1:7.3.547-7", "<", "2:7.3.547-7"},
	{"2:7.2.547-7", "<", "2:7.3.547-7"},
	{"2:7.3.540-7", "<", "2:7.3.547-7"},
	{"2:7.3.547-6", "<", "2:7.3.547-7"},
	{"1.0.1e-2+deb7u14", "=", "1.0.1e-2+deb7u14"},
	{"1.0.1d-2+deb7u14", "<", "1.0.1e-2+deb7u14"},
	{"1.0.1", "<", "1.0.1e"},
	{"1.0.1", "=", "1.0.1"},
	{"1.0.1d", "<", "1.0.1e"},
	{"0:0.2.4.23-1~deb7u1", "=", "0:0.2.4.23-1~deb7u1"},
	{"0:0.2.4-1~deb7u1", "<", "0:0.2.4.23-1~deb7u1"},
	{"0:0.2.4.23-0~deb7u1", "<", "0:0.2.4.23-1~deb7u1"},
	{"2.6.32-504.el6", "<", "0:2.6.32-504.8.1.el6"},
	{"1.5.9", "<", "1.5.10"},
	{"1.5.9", "<", "1.5.10z"},
	{"1.5.9z", "<", "1.5.10"},
	{"1.5.9b", "<", "1.5.9f"},
	{"1.5.9g", "=", "1.5.9g"},
	{"1.5.10", ">", "1.5.9"},
	{"f", ">", "a"},
	{"10", ">", "9"},
	{"123", "=", "123"},
	{"1241", "<", "14444"},
	{"12412", ">", "50"},
	{"51", ">", "50"},
}

func TestEvrops(t *testing.T) {
	scribe.Bootstrap()
	scribe.TestHooks(true)

	for _, x := range evrTests {
		var opmode int

		switch x.op {
		case "=":
			opmode = scribe.EvropEquals
		case "<":
			opmode = scribe.EvropLessThan
		case ">":
			opmode = scribe.EvropGreaterThan
		default:
			t.Fatalf("evr test has invalid operation %v", x.op)
		}
		t.Logf("%v %v %v", x.verA, x.op, x.verB)
		result, err := scribe.TestEvrCompare(opmode, x.verA, x.verB)
		if err != nil {
			t.Fatalf("scribe.TestEvrCompare: %v", err)
		}
		if !result {
			t.Fatalf("scribe.TestEvrCompare: failed %v %v %v", x.verA, x.op, x.verB)
		}
	}
}
