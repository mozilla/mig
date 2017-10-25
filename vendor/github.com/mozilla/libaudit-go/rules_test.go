// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package libaudit

import (
	"io/ioutil"
	"os"
	"testing"
)

var expectedRules = []string{
	"-w /etc/libaudit.conf -p wa -k audit",
	"-w /etc/rsyslog.conf -p wa -k syslog",
	"-a always,exit-F arch=b64 -S personality -F key=bypass",
	"-a never,exit -F path=/bin/ls -F perm=x",
	"-a always,exit-F arch=b64 -S execve -F key=exec",
	"-a always,exit -S clone,fork,vfork",
	"-a always,exit -S adjtimex,settimeofday -F key=time-change",
	"-a always,exit-F arch=b64 -S rename,renameat -F auid>=1000 -F key=rename",
}

func TestSetRules(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("skipping test, not root user")
	}

	jsonRules, err := ioutil.ReadFile("./testdata/rules.json")
	if err != nil {
		t.Fatalf("ioutil.ReadFile: %v", err)
	}

	s, err := NewNetlinkConnection()
	if err != nil {
		t.Fatalf("NewNetlinkConnection: %v", err)
	}
	err = DeleteAllRules(s)
	if err != nil {
		t.Fatalf("DeleteAllRules: %v", err)
	}

	_, err = SetRules(s, jsonRules)
	if err != nil {
		t.Fatalf("SetRules: %v", err)
	}
	s.Close()

	// Open up a new connection before we list the rules
	x, err := NewNetlinkConnection()
	if err != nil {
		t.Fatalf("NewNetlinkConnection: %v", err)
	}

	setRules, err := ListAllRules(x)
	if err != nil {
		t.Fatalf("ListAllRules: %v", err)
	}
	if len(setRules) != len(expectedRules) {
		t.Fatalf("number of set rules unexpected, wanted %v got %v", len(expectedRules),
			len(setRules))
	}
	for i := range setRules {
		if setRules[i] != expectedRules[i] {
			t.Fatalf("expected rule %q, got rule %q", expectedRules[i], setRules[i])
		}
	}
	x.Close()
}

// Try to load a rule set with strict path checking enabled, where the path in a watch rule is
// nonexistent; should fail.
func TestSetRulesStrictPathFail(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("skipping test, not root user")
	}

	jsonRules, err := ioutil.ReadFile("./testdata/strictpathfail.json")
	if err != nil {
		t.Fatalf("ioutil.ReadFile: %v", err)
	}

	s, err := NewNetlinkConnection()
	if err != nil {
		t.Fatalf("NewNetlinkConnection: %v", err)
	}
	err = DeleteAllRules(s)
	if err != nil {
		t.Fatalf("DeleteAllRules: %v", err)
	}

	_, err = SetRules(s, jsonRules)
	if err == nil {
		t.Fatalf("SetRules should have failed on nonexistent path")
	}
	s.Close()
}

// Try to load a rule set with strict path checking disabled, where the path in a watch rule is
// nonexistent; should succeed but ignore the invalid rule.
func TestSetRulesNoStrictPath(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skipf("skipping test, not root user")
	}

	jsonRules, err := ioutil.ReadFile("./testdata/badpathnostrictpath.json")
	if err != nil {
		t.Fatalf("ioutil.ReadFile: %v", err)
	}

	s, err := NewNetlinkConnection()
	if err != nil {
		t.Fatalf("NewNetlinkConnection: %v", err)
	}
	err = DeleteAllRules(s)
	if err != nil {
		t.Fatalf("DeleteAllRules: %v", err)
	}

	warnings, err := SetRules(s, jsonRules)
	if err != nil {
		t.Fatalf("SetRules: %v", err)
	}

	// We should have gotten one warning back
	if len(warnings) != 1 {
		t.Fatalf("SetRules: should have returned 1 warning")
	}

	// Open up a new connection before we list the rules
	x, err := NewNetlinkConnection()
	if err != nil {
		t.Fatalf("NewNetlinkConnection: %v", err)
	}

	setRules, err := ListAllRules(x)
	if err != nil {
		t.Fatalf("ListAllRules: %v", err)
	}
	// We should have 1 rule here, since the test data set contains 2 rules, one of which
	// should have been ignored.
	expectedRules := 1
	if len(setRules) != expectedRules {
		t.Fatalf("number of set rules unexpected, wanted %v got %v", expectedRules,
			len(setRules))
	}
	s.Close()
}
