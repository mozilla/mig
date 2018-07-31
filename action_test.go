// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package mig

import (
	"encoding/json"
	"testing"

	"github.com/mozilla/mig/pgp"
)

var keyValidSigner1 = `
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQENBFnwr+YBCADI7ZlSuG/KxWt1E/qZcjo/cnF/N0oNPn/z9bfzDb/smejcJ1tt
Rb1mXWINrofnbxmYKcxSdX6Q9x4rQ8aLnd3PuX84Dg5oNx4NkiKdHZGFi2hyJK6u
GO3FV1EF+3fD+cVBUGoTT6eL5p2nppMvRTky8E6RklIaOKNU6cIMuGDSnwNU7pip
tTwLILw5Brd/cbGxoov+jxu5ccIJ+07tjBu7U5UVt+97gMqBazTCKvO+Q1/DFJsq
j67sYqvpGU8OlUOPfmbhhK8rU/CIDHRKeD3OiLAn85xEUnuGGJ7H8NsFwnf+cVA2
wHOhFY3QD+hHeF0t9w98tRHUQBhbRSncge//ABEBAAG0G01JRyB1bml0IHRlc3Rz
IDx1bml0QHRlc3RzPokBOAQTAQIAIgUCWfCv5gIbAwYLCQgHAwIGFQgCCQoLBBYC
AwECHgECF4AACgkQDzgtIchMFDpcRQf+INfdZdcDhjg/6c7Gq+YAoDBnI95XQEwz
AQMlMdFbzwNq0KqzerFAQjrJ2s+tOhVEJ4loJNZsuHW9a24f8xXhQ8/yRUHw45ca
iiztR7oUjHJfdTvon2YEPYTa8aFOKsZHI0MTM+7/Z0jQdIWEoiLswzF68Aa3QaFZ
ZflXhuCZKg7BFoYr1hKEjIEPko6aDXgzRx4v+e9AhYouWjXGzUG6UNpbB7d0sUKm
2O2ovGXm9gxhPv41ULcunIwYy3LI/v+1v1NZT9hSfcUYuosZEdQ2bPAN0M1JeDXZ
YjJ0//tk01jIhU4T5Qh5mBIuE1bg1y3y/Ui30Zs7HK9E5elw/YkdArkBDQRZ8K/m
AQgAlX4htWXARk/Wt4ckH7fUGHgn9AZN6SfDpu7IP7DKwIOP5Grj/tQ9qijVPb6R
wrj2m4evepsWoBeUiWz+pGUuOllTXS9OpD0uSFySjf2UYjp2pvrga9s1yv7hRTW8
YAuW8eK4KPZQ7Lj34D1JCybb0inr23wJtOWHR/RnE/QgP6qPUwUQ9bXepduXTLUb
A1KU5Enr0z+b1xZQva/4iR6qVL5xIV87zbOxHwByD0cXFMic5rUay+Lqr+YtPTE9
tn9+qkVfbIp0uHzhUzVdyJ1BRW7LZ29EmbkL9U27TIN2msI/hjlsuvUmCdUjsKpy
3DKIWxQBHzbCn8UCkMTVkcFs7QARAQABiQEfBBgBAgAJBQJZ8K/mAhsMAAoJEA84
LSHITBQ6BsAH/34XFJz/B29OiA9rT4lnmb5GQjXFX3ZgID3xy+sMXhLGGQLUZX8B
dicQLNidkvSvpxOVqoFve5M63E3G/vDPbsNKcnYOm/ws3niHFNrgggvHRh3G4zSs
dtSWb4xS979DbURerGpvpyYMQvFXRTtTIh6u18Gdb804cOJZ5n0JgEtdzLYJgUWO
N2yBsyv6j1wL7Iw/LXAFoS0A0Pyo8/qA1OP0K9WtXaHF4XcMDHcxGK3sQkgI8Cgm
RRIPOaJVd05RwDyyy/l4hsjpLNJjE7EIze2sv74NPcb0C4vMqt+ZO8pozWUdjrQN
Za9LMnKQuJuSuvvGQX7mOhDCP59kHFx18sM=
=wuW6
-----END PGP PUBLIC KEY BLOCK-----
`

var keyInvalidSigner1 = `
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mQENBFnwscsBCAD5xthKnQRosuf8AhBuoPzd8jBeLQDun5aMigxKk2LYhQDr5rtP
7u9mINtKA+keLq4ofuxybFxGax0K5/4ogOxBjfUDcMvVXElhqeersIG6hHftrFvc
vmJHYtLM5PCDaWM04I66LeEJirfBL3OJJfziC9zOGuW2mIPW2fQiXAF96A5iJsvy
ijnwmvGzpBmlrbdPKdEGgu1qe3fp6xo3x+exENIpT3eBx/dtj4b1/Z6+Vp2fE+aL
9nFnsSULeDPZ+9LDco5MgUTrBBpqRCeAvxdhLTUZmlb37XUY2PqW/KS03rYD52yq
aoQ3b8G3+MWdnJ+V+b/pn2JhY/3zDNujx0eFABEBAAG0Kk1JRyB1bml0IHRlc3Rz
IGludmFsaWQgPHVuaXRpbnZhbGlkQHRlc3RzPokBOAQTAQIAIgUCWfCxywIbAwYL
CQgHAwIGFQgCCQoLBBYCAwECHgECF4AACgkQz973Z+ggHRvy6Af/Rh7xW844/GnI
g789NRzXef2bPaGCLI1msrggQG7cKspdjQgXWsBWwyPRF/6UkvsueI2kuwabQHBA
fITjqBQ4KYdDoZzmTr5Jly/9dj38g5LFvFnwr0gntmyOonIdcwncnlvOe+Qu7Tcm
Lda4Im2bi6EW+tNwQADhbyHScUZaJpMQuEA3RaSE5DceoHm7K8iM8VxAglNXt114
xCe2tuquLRU/RJEUOEEPhGsH0rU3VHI5WJMBATPUDZZoLZ1pWsv8kDnk1T/ZiZoI
E7ng/YHRvRqin7LavTMcGhSBDv56IDUIoAABP4wfFIEyjPxgViwri++s1vdU+iFj
jJCEDNsBKLkBDQRZ8LHLAQgAyNE2YoKWkISoun1oVKR36xm12hqn5cIVXAjg3yUy
AdCrJSx0N5TmbfDlWIuHq0OGiXEdayZ17Pg5b2RtbMV3BJjyJHeZADN2u881MqQp
ZNEHBpFPZuVXIGIoVYiQFvZQOeTKpHLrkN3PGVc/xNDRer8JqdiR6Bgg69z9Ay2/
ag2SEXHthrA8x8dO9YVkSjnpG2yXVqSQzbu1AVejQH8caUTpnjl+VoipE7byF10S
gGMowCF8DQ9+LnflhPyA43QLg7Riea7iG6wSgDs142pqyHrg7ADzrXI177oWgBK2
q9EW1R42bXQ3SD5fWqQbSmfTTdXP1m2E4h72MtyAxFa4JQARAQABiQEfBBgBAgAJ
BQJZ8LHLAhsMAAoJEM/e92foIB0b7lcH/iso+ZKiZtuI0J6BM0lAZZXHgHxeFoAd
TP5nMpiP96FbhjjsiV67PJZlcjUxxWReFScGGpR6qoTsXzvoRliAdcZgCSXxxC1P
3WRoG0nJr65WMi3ieAd5W9XfukrKMND3XuukQaW4NQO+7F12x6F2QXu1kwrRlmyo
lFLtiuEEAlL2qarS6ItVxAEx4PGLUmNkSJ2RXK/uyd0Do6qTmjqV0Yl1eiZcRGXn
1uUgQcqcoBEeqrzK5KvDWqzLHQiwhDW02LaeyUCgFyHJzmX63rQCcnIf2jbr9MpD
2MZRp85fHFOy5QoQENMFMcS4+nz04ePh6Zj1TXPx0wX4hn164xqUJ1k=
=jIoT
-----END PGP PUBLIC KEY BLOCK-----
`

// A valid signed action, signed using keyValidSigner1
var validSignedAction1 = `
{
    "id": 0,
    "name": "pkg -s -name test ",
    "target": "status='online'",
    "description": {},
    "threat": {},
    "validfrom": "2017-10-25T15:50:05.220759533Z",
    "expireafter": "2017-10-25T15:56:05.220759533Z",
    "operations": [
        {
            "module": "pkg",
            "parameters": {
                "pkgmatch": {
                    "matches": [
                        "test"
                    ]
                },
                "vermatch": ""
            }
        }
    ],
    "pgpsignatures": [
        "wsBcBAABCAAQBQJZ8LLpCRAPOC0hyEwUOgAA9lgIAIMJwIlhb9FTtT6gcMaoj2JSEnSS921DA3JR7VFd057wCBjmu9IfYD/j47TS2fmE2kwmwBbNQ/uAjqM61FycvH2p/zdfmrmTaiwWUArpGtmRCnEloUfmeDh0d7PcwzkcKa9rmtH1dvEW8SgQq/yDNw9hvfyYJHQWeqk9WzpCLfJlqUu/NLZZbjP/hJuHla8B161q4r0jlzqQwT5gn7nk+O0/q2zw3QzbU8WGT7Q9STB18JcCQ0NyeMLjHCf5yCKqeh1PtGOQ8NTuYo8p4M/xvMe5bLbHsONpay9JYM6NdUdGeyYHdU3odiTYDYaT3JqCdceZVM2Vjsj+IKeOyaVJQgw==Jaj/"
    ],
    "starttime": "0001-01-01T00:00:00Z",
    "finishtime": "0001-01-01T00:00:00Z",
    "lastupdatetime": "0001-01-01T00:00:00Z",
    "counters": {}
}
`

// An action signed with the invalid signer key
var invalidSignedAction1 = `
{
    "id": 0,
    "name": "pkg -s -name test ",
    "target": "status='online'",
    "description": {},
    "threat": {},
    "validfrom": "2017-10-25T16:09:35.583635919Z",
    "expireafter": "2017-10-25T16:15:35.583635919Z",
    "operations": [
        {
            "module": "pkg",
            "parameters": {
                "pkgmatch": {
                    "matches": [
                        "test"
                    ]
                },
                "vermatch": ""
            }
        }
    ],
    "pgpsignatures": [
        "wsBcBAABCAAQBQJZ8Ld7CRDP3vdn6CAdGwAAXCoIAPGzPOY21S5jpr8wZzlx1+L7PP5jY8KGDAVuI75WurKC+HOG6fecx/Tf97JtJ8QqrLb7NJg/Jx6JQ+/qsoxRPMgI9Jr6lF1nw6XL4vvCLjxoBVa6H526Hj1FgDbNjy7F6HgWGEjiXFrrYnODKA37x8eozJ8Fx3+gzojTKz+EA9uBr/0a0o+VMx3Yw/HV+Rix6I/B3Tfxpeit+3W+Z40/eXOPQfbhfaT0uzIe6giWAriy4ll3DJ2LZDfvAP+49ze7mImxMxnMNhsD2BM5VvdmhdPD4CrvC3QCRkGe6P9h3ecJ8Kx++19sVplHIX3SaKRfaTgIvQ6mrwocMIs/yMFWPxE==1Uf/"
    ],
    "starttime": "0001-01-01T00:00:00Z",
    "finishtime": "0001-01-01T00:00:00Z",
    "lastupdatetime": "0001-01-01T00:00:00Z",
    "counters": {}
}
`

// A multi operation action signed with the valid signer key
var signedMultiOpAction1 = `
{
    "id": 0,
    "name": "multiop action",
    "target": "status='online'",
    "description": {},
    "threat": {},
    "validfrom": "2017-10-25T16:50:42.601631961Z",
    "expireafter": "2017-10-25T16:56:42.601631961Z",
    "operations": [
        {
            "module": "pkg",
            "parameters": {
                "pkgmatch": {
                    "matches": [
                        "test"
                    ]
                },
                "vermatch": ""
            }
        },
        {
            "module": "file",
            "parameters": {
                "searches": {
                    "s1": {
                        "names": [
                            "test"
                        ],
                        "options": {
                            "macroal": false,
                            "matchall": true,
                            "matchlimit": 1000,
                            "maxdepth": 1000,
                            "maxerrors": 30,
                            "mismatch": null
                        },
                        "paths": [
                            "/etc"
                        ]
                    }
                }
            }
        }
    ],
    "pgpsignatures": [
        "wsBcBAABCAAQBQJZ8MEeCRAPOC0hyEwUOgAAcwMIAGwU1Cv5FCQB9crkbiNrIJc99UodAtjmfIHa4u/OsHNUB+OcnEPNb1T2zI+rRYZzKQgV9WxSJfVDjdHtmVkAUzVF8ybOresBYSsgNN+FYAhpF2yvy5d8HF5IXXjBKlrJvqKSMzf1pOU1b4Sbklfu02LqR/MrjYYjvIA918NT1Nf/UCxHZdURCA0xyUYzJY2bYXGK7qNJs1ImYzsuv+YCYybIynS8XwSbwhLOgcSygLDPJjTBFdsupDMGBe8L+eoQuJF9zXalPtjO/qBF4ZYECzFrbTUu1R6SIGgeaCqyaRY8dIkNGZPmgyMOzGYcUFd3/k9GgjwezrmEHCu/J1MVo4o==pa/c"
    ],
    "starttime": "0001-01-01T00:00:00Z",
    "finishtime": "0001-01-01T00:00:00Z",
    "lastupdatetime": "0001-01-01T00:00:00Z",
    "counters": {}
}
`

func TestVerifyACLValid(t *testing.T) {
	keys := make([][]byte, 0)
	keys = append(keys, []byte(keyValidSigner1))
	keyring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		t.Fatalf("pgp.ArmoredKeysToKeyring: %v", err)
	}
	// We should have one valid key
	if keycount != 1 {
		t.Fatalf("unexpected number of keys in keyring: %v", keycount)
	}

	var a Action
	err = json.Unmarshal([]byte(validSignedAction1), &a)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// This should verify
	err = a.VerifySignatures(keyring)
	if err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}

	aclstr := `{
		"pkg": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	// Try verification again, this time include a valid ACL as well
	var acl ACL
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	// Rewind the keyring reader for a subsequent read on it
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err != nil {
		t.Fatalf("VerifyACL: %v", err)
	}

	// Try again, use a default ACL this time though which should also work
	aclstr = `{
		"default": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	// Rewind the keyring reader for a subsequent read on it
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err != nil {
		t.Fatalf("VerifyACL: %v", err)
	}
}

func TestVerifyGoodSigBadACL(t *testing.T) {
	keys := make([][]byte, 0)
	keys = append(keys, []byte(keyValidSigner1))
	keyring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		t.Fatalf("pgp.ArmoredKeysToKeyring: %v", err)
	}
	// We should have one valid key
	if keycount != 1 {
		t.Fatalf("unexpected number of keys in keyring: %v", keycount)
	}

	var a Action
	err = json.Unmarshal([]byte(validSignedAction1), &a)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// This should verify
	err = a.VerifySignatures(keyring)
	if err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}

	// This ACL should result in a verification failure as we don't have an ACL entry
	// for the pkg module
	aclstr := `{
		"file": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err == nil {
		t.Fatalf("VerifyACL should have failed")
	}

	// Rerun the test with onlyVerifyPubkey set to true, this should be successful despite
	// the ACL
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, true)
	if err != nil {
		t.Fatalf("VerifyACL: %v", err)
	}
}

func TestVerifyBadSig(t *testing.T) {
	keys := make([][]byte, 0)
	keys = append(keys, []byte(keyValidSigner1))
	keyring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		t.Fatalf("pgp.ArmoredKeysToKeyring: %v", err)
	}
	// We should have one valid key
	if keycount != 1 {
		t.Fatalf("unexpected number of keys in keyring: %v", keycount)
	}

	var a Action
	err = json.Unmarshal([]byte(invalidSignedAction1), &a)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// This should fail
	err = a.VerifySignatures(keyring)
	if err == nil {
		t.Fatalf("VerifySignatures should have failed")
	}

	// Test verification given an ACL that is valid but an invalid signature
	aclstr := `{
		"pkg": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "474F6DCF3515C0E802519333CFDEF767E8201D1B",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err == nil {
		t.Fatalf("VerifyACL should have failed. Error: %v", err)
	}

	// Try this again, but this time add the invalid signer to the keyring and invalidate the entry
	// in the ACL
	keys = append(keys, []byte(keyInvalidSigner1))
	keyring, keycount, err = pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		t.Fatalf("pgp.ArmoredKeysToKeyring: %v", err)
	}
	// We should have 2 keys
	if keycount != 2 {
		t.Fatalf("unexpected number of keys in keyring: %v", keycount)
	}
	// This should pass now that the invalid signature is in the keyring
	err = a.VerifySignatures(keyring)
	if err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}
	// An ACL that does not include the signature of the invalid signer
	aclstr = `{
		"pkg": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err == nil {
		t.Fatalf("VerifyACL should have failed")
	}
}

func TestVerifyMultiOperation(t *testing.T) {
	keys := make([][]byte, 0)
	keys = append(keys, []byte(keyValidSigner1))
	keyring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		t.Fatalf("pgp.ArmoredKeysToKeyring: %v", err)
	}
	// We should have one valid key
	if keycount != 1 {
		t.Fatalf("unexpected number of keys in keyring: %v", keycount)
	}

	var a Action
	err = json.Unmarshal([]byte(signedMultiOpAction1), &a)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// This should succeed
	err = a.VerifySignatures(keyring)
	if err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}

	// Test verification given an ACL with just one module from the action listed
	aclstr := `{
		"file": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err == nil {
		t.Fatalf("VerifyACL should have failed")
	}

	// Retry verification with an ACL with both modules now, which should succeed
	aclstr = `{
		"pkg": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		},
		"file": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err != nil {
		t.Fatalf("VerifyACL: %v", err)
	}
}

func TestVerifyNoSignatures(t *testing.T) {
	keys := make([][]byte, 0)
	keys = append(keys, []byte(keyValidSigner1))
	keyring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		t.Fatalf("pgp.ArmoredKeysToKeyring: %v", err)
	}
	// We should have one valid key
	if keycount != 1 {
		t.Fatalf("unexpected number of keys in keyring: %v", keycount)
	}

	var a Action
	err = json.Unmarshal([]byte(validSignedAction1), &a)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	// Remove the signature from the action
	a.PGPSignatures = []string{}

	// This should fail
	err = a.VerifySignatures(keyring)
	if err == nil {
		t.Fatalf("VerifySignatures should have failed")
	}

	aclstr := `{
		"pkg": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err == nil {
		t.Fatalf("VerifyACL should have failed")
	}
}

func TestVerifyNoKeyring(t *testing.T) {
	keys := make([][]byte, 0)
	keyring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		t.Fatalf("pgp.ArmoredKeysToKeyring: %v", err)
	}
	// We should have no keys
	if keycount != 0 {
		t.Fatalf("unexpected number of keys in keyring: %v", keycount)
	}

	var a Action
	err = json.Unmarshal([]byte(validSignedAction1), &a)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	// This should fail
	err = a.VerifySignatures(keyring)
	if err == nil {
		t.Fatalf("VerifySignatures should have failed")
	}

	aclstr := `{
		"pkg": {
			"minimumweight": 1,
			"investigators": {
				"valid user": {
					"fingerprint": "397FD1F5E3DD4020BEF0E37E0F382D21C84C143A",
					"weight": 1
				}
			}
		}
	}`
	var acl ACL
	err = json.Unmarshal([]byte(aclstr), &acl)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	keyring.Seek(0, 0)
	err = a.VerifyACL(acl, keyring, false)
	if err == nil {
		t.Fatalf("VerifyACL should have failed")
	}
}
