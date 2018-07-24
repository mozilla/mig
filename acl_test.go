// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package mig /* import "github.com/mozilla/mig" */

import (
	"encoding/json"
	"testing"
)

func TestValidACL(t *testing.T) {
	aclstr := `{
		"file": {
			"minimumweight": 1,
			"investigators": {
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err != nil {
		t.Fatalf("verifyPermission should have returned no error: %v", err)
	}
}

func TestValidACLDefault(t *testing.T) {
	aclstr := `{
		"default": {
			"minimumweight": 1,
			"investigators": {
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err != nil {
		t.Fatalf("verifyPermission should have returned no error: %v", err)
	}
}

func TestValidACLMultiple(t *testing.T) {
	aclstr := `{
		"default": {
			"minimumweight": 1,
			"investigators": {
				"test3": {
					"fingerprint": "CCCCCCCCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				},
				"test2": {
					"fingerprint": "BBBBBBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				},
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err != nil {
		t.Fatalf("verifyPermission should have returned no error: %v", err)
	}
}

func TestValidACLMultipleSigners(t *testing.T) {
	aclstr := `{
		"default": {
			"minimumweight": 2,
			"investigators": {
				"test3": {
					"fingerprint": "CCCCCCCCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				},
				"test2": {
					"fingerprint": "BBBBBBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				},
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"BBBBBBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err != nil {
		t.Fatalf("verifyPermission should have returned no error: %v", err)
	}
}

func TestValidACLMultipleSignersOneValid(t *testing.T) {
	aclstr := `{
		"default": {
			"minimumweight": 2,
			"investigators": {
				"test3": {
					"fingerprint": "CCCCCCCCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				},
				"test2": {
					"fingerprint": "BBBBBBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				},
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"DDDDDDDDAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err == nil {
		t.Fatalf("verifyPermission should have failed. Error: %v", err)
	}
}

func TestInvalidACLNoFingerprint(t *testing.T) {
	aclstr := `{
		"file": {
			"minimumweight": 1,
			"investigators": {
				"test": {
					"fingerprint": "BBBBBBBBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err == nil {
		t.Fatalf("verifyPermission should have failed")
	}
}

func TestInvalidACLNoModuleEntry(t *testing.T) {
	aclstr := `{
		"somemodule": {
			"minimumweight": 1,
			"investigators": {
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err == nil {
		t.Fatalf("verifyPermission should have failed")
	}
}

func TestInvalidACLBadWeight(t *testing.T) {
	aclstr := `{
		"file": {
			"minimumweight": 2,
			"investigators": {
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	if err == nil {
		t.Fatalf("verifyPermission should have failed")
	}
}

func TestInvalidACLZeroFingerprints(t *testing.T) {
	aclstr := `{
		"file": {
			"minimumweight": 2,
			"investigators": {
				"test": {
					"fingerprint": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err := json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	err = verifyPermission("file", acl, []string{})
	if err == nil {
		t.Fatalf("verifyPermission should have failed")
	}
}
