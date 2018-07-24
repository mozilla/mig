// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package sshkey /* import "github.com/mozilla/mig/modules/sshkey" */

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mozilla/mig/modules"
	"github.com/mozilla/mig/testutil"
)

func TestRegistration(t *testing.T) {
	testutil.CheckModuleRegistration(t, "sshkey")
}

var testAction = `
{
	"class": "parameters",
	"parameters": {
		"paths": [
			"./testdata"
		],
		"maxdepth": 3
	}
}
`

func TestRun(t *testing.T) {
	action := strings.Replace(testAction, "\n", "", -1)
	reader := modules.NewModuleReader(bytes.NewReader([]byte(action)))
	runner := run{}
	res := runner.Run(reader)
	result := modules.Result{}
	err := json.Unmarshal([]byte(res), &result)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	e := Elements{}
	err = result.GetElements(&e)
	if err != nil {
		t.Fatalf("GetElements: %v", err)
	}
	if len(e.Keys) != 8 {
		t.Fatalf("incorrect number of keys returned from module run")
	}
}

func TestFindCandidates(t *testing.T) {
	cands, err := findCandidates([]string{"./testdata"}, 2)
	if err != nil {
		t.Fatalf("findCandidates: %v", err)
	}
	if len(cands) != 7 {
		t.Fatalf("findCandidates: incorrect number of candidates returned")
	}

	cands, err = findCandidates([]string{"/nonexist"}, 2)
	if err != nil {
		t.Fatalf("findCandidates: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("findCandidates: incorrect number of candidates returned")
	}
}

func TestProcessCandidate(t *testing.T) {
	e := &Elements{}
	processCandidate("./testdata/home/testkey1", e)
	if len(e.Keys) != 1 {
		t.Fatalf("processCandidate: number of keys in elements not correct")
	}
	if e.Keys[0].Type != KeyTypePrivate {
		t.Fatalf("processCandidate: invalid key type returned")
	}
	if e.Keys[0].FingerprintMD5 != "4c:57:cb:9d:3b:8e:eb:80:3f:d2:97:2e:ad:64:67:eb" {
		t.Fatalf("processCandidate: invalid fingerprint")
	}

	e = &Elements{}
	processCandidate("./testdata/testkey3.pub", e)
	if len(e.Keys) != 1 {
		t.Fatalf("processCandidate: number of keys in elements not correct")
	}
	if e.Keys[0].Type != KeyTypePublic {
		t.Fatalf("processCandidate: invalid key type returned")
	}
	if e.Keys[0].FingerprintMD5 != "14:0f:ed:33:5d:27:57:58:2c:03:c8:3c:25:84:b0:ef" {
		t.Fatalf("processCandidate: invalid fingerprint")
	}

	e = &Elements{}
	processCandidate("./testdata/home/testkey2", e)
	if len(e.Keys) != 1 {
		t.Fatalf("processCandidate: number of keys in elements not correct")
	}
	if e.Keys[0].Type != KeyTypePrivate {
		t.Fatalf("processCandidate: invalid key type returned")
	}
	if e.Keys[0].FingerprintMD5 != "" {
		t.Fatalf("processCandidate: fingerprint was not unset as expected")
	}

	e = &Elements{}
	processCandidate("./testdata/home/authorized_keys", e)
	if len(e.Keys) != 2 {
		t.Fatalf("processCandidate: number of keys in elements not correct")
	}
	if e.Keys[1].Type != KeyTypeAuthorizedKeys {
		t.Fatalf("processCandidate: invalid key type returned")
	}
	if e.Keys[1].FingerprintMD5 != "c0:bd:f1:e2:42:1c:36:cc:f8:f0:8a:cb:9f:12:b8:ad" {
		t.Fatalf("processCandidate: invalid fingerprint")
	}

	e = &Elements{}
	processCandidate("./testdata/home/testkey2.pub", e)
	if len(e.Keys) != 1 {
		t.Fatalf("processCandidate: number of keys in elements not correct")
	}
	if e.Keys[0].Type != KeyTypePublic {
		t.Fatalf("processCandidate: invalid key type returned")
	}
	if e.Keys[0].FingerprintMD5 != "c0:bd:f1:e2:42:1c:36:cc:f8:f0:8a:cb:9f:12:b8:ad" {
		t.Fatalf("processCandidate: invalid fingerprint")
	}

	e = &Elements{}
	processCandidate("./testdata/home/notakey", e)
	if len(e.Keys) != 0 {
		t.Fatalf("processCandidate: number of keys in elements not correct")
	}
}
