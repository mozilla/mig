// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package mig

import (
	"encoding/json"
	"testing"
	"time"

	"mig.ninja/mig/pgp"
)

func TestSignAndVerifyACL(t *testing.T) {
	var uacl = ACL{
		Name:        "testacl",
		Target:      "os->environment='linux' AND mode='daemon'",
		ValidFrom:   time.Now(),
		ExpireAfter: time.Now(),
		Permissions: []Permission{
			Permission{
				Module:        "testmodule",
				MinimumWeight: 1,
				Investigators: []InvestigatorPerm{
					InvestigatorPerm{
						ID:             1,
						Name:           "Test Investigator",
						PGPFingerprint: "abcdef0123456789",
						Weight:         1,
					},
				},
			},
		},
	}
	data, _ := json.Marshal(uacl)
	t.Logf("%s", data)
	for _, fp := range []string{TESTACLFP1, TESTACLFP2} {
		secring, _, err := pgp.ArmoredKeysToKeyring(TESTACLPRIVKEYS)
		if err != nil {
			t.Error("failed to make secring from private key")
		}
		sig, err := uacl.Sign(fp, secring)
		if err != nil {
			t.Error("failed to sign ACL with test private key")
		}
		uacl.PGPSignatures = append(uacl.PGPSignatures, sig)
		pubring, _, err := pgp.ArmoredKeysToKeyring(TESTACLPUBKEYS)
		if err != nil {
			t.Error("failed to make pubring from private key")
		}
		err = uacl.VerifySignatures(1, pubring)
		if err != nil {
			t.Error(err)
		}
		t.Logf("Signed and verified!\n%s\n%s\n", uacl.String(), sig)
	}
}

type testACL struct {
	expect bool
	reqsig int
	acl    string
}

func TestVerifyACL(t *testing.T) {
	var testacls = []testACL{
		// good acl
		{true, 1, `{"name":"testacl","target":"mode='daemon'","validfrom":"2015-08-03T12:05:23.008403417-04:00","expireafter":"2015-08-03T12:05:23.008403445-04:00","permissions":[{"module":"testmodule","minimumweight":1,"investigators":[{"id":1,"name":"Test Investigator","pgpfingerprint":"abcdef0123456789","weight":1}]}],"pgpsignatures":["wpwEAAEIABAFAlW/kUMJEF4AASq0LBwdAACR2gQAAJQBXBWSpNc+Hl3nSxLHhbibTel4K6BHzw+yFGV/EbI4MewszXAWjI+94W0v/Qsm4ZJvtcDM+cw5J8AEy22oZ69RfirF+ZKmP0lC+CS0cEW/E4JtowhGNYlhtedWN2eYMoBdpf6IUgad7PQA8MDQ52pJN1Y2dTPMWcGYilgWCNI==mZZw"]}`},
		// bad acl: broken signature
		{false, 1, `{"name":"testacl","target":"mode='daemon'","validfrom":"2015-08-03T12:05:23.008403417-04:00","expireafter":"2015-08-03T12:05:23.008403445-04:00","permissions":[{"module":"testmodule","minimumweight":1,"investigators":[{"id":1,"name":"Test Investigator","pgpfingerprint":"abcdef0123456789","weight":1}]}],"pgpsignatures":["wpwEAAEIABAFAlW/kUMJEF4AASq0LBwdAACR2gQAAJQBXBWSpNc+Hl3nSxLHhbibTel4K6B"]}`},
		// good acl: fields in random order
		{true, 1, `{"pgpsignatures":["wpwEAAEIABAFAlW/kUMJEF4AASq0LBwdAACR2gQAAJQBXBWSpNc+Hl3nSxLHhbibTel4K6BHzw+yFGV/EbI4MewszXAWjI+94W0v/Qsm4ZJvtcDM+cw5J8AEy22oZ69RfirF+ZKmP0lC+CS0cEW/E4JtowhGNYlhtedWN2eYMoBdpf6IUgad7PQA8MDQ52pJN1Y2dTPMWcGYilgWCNI==mZZw"],"permissions":[{"module":"testmodule","minimumweight":1,"investigators":[{"name":"Test Investigator","weight":1,"pgpfingerprint":"abcdef0123456789","id":1}]}],"validfrom":"2015-08-03T12:05:23.008403417-04:00","expireafter":"2015-08-03T12:05:23.008403445-04:00","name":"testacl"}`},
		// bad acl: not enough signatures
		{false, 2, `{"name":"testacl","validfrom":"2015-08-03T12:05:23.008403417-04:00","expireafter":"2015-08-03T12:05:23.008403445-04:00","permissions":[{"module":"testmodule","minimumweight":1,"investigators":[{"id":1,"name":"Test Investigator","pgpfingerprint":"abcdef0123456789","weight":1}]}],"pgpsignatures":["wpwEAAEIABAFAlW/kUMJEF4AASq0LBwdAACR2gQAAJQBXBWSpNc+Hl3nSxLHhbibTel4K6BHzw+yFGV/EbI4MewszXAWjI+94W0v/Qsm4ZJvtcDM+cw5J8AEy22oZ69RfirF+ZKmP0lC+CS0cEW/E4JtowhGNYlhtedWN2eYMoBdpf6IUgad7PQA8MDQ52pJN1Y2dTPMWcGYilgWCNI==mZZw"]}`},
		// bad acl: two valid signatures, but from the same keyid
		{false, 2, `{"name":"testacl","validfrom":"2015-08-03T12:05:23.008403417-04:00","expireafter":"2015-08-03T12:05:23.008403445-04:00","permissions":[{"module":"testmodule","minimumweight":1,"investigators":[{"id":1,"name":"Test Investigator","pgpfingerprint":"abcdef0123456789","weight":1}]}],"pgpsignatures":["wpwEAAEIABAFAlW/kUMJEF4AASq0LBwdAACR2gQAAJQBXBWSpNc+Hl3nSxLHhbibTel4K6BHzw+yFGV/EbI4MewszXAWjI+94W0v/Qsm4ZJvtcDM+cw5J8AEy22oZ69RfirF+ZKmP0lC+CS0cEW/E4JtowhGNYlhtedWN2eYMoBdpf6IUgad7PQA8MDQ52pJN1Y2dTPMWcGYilgWCNI==mZZw","wpwEAAEIABAFAlW/kUMJEF4AASq0LBwdAACR2gQAAJQBXBWSpNc+Hl3nSxLHhbibTel4K6BHzw+yFGV/EbI4MewszXAWjI+94W0v/Qsm4ZJvtcDM+cw5J8AEy22oZ69RfirF+ZKmP0lC+CS0cEW/E4JtowhGNYlhtedWN2eYMoBdpf6IUgad7PQA8MDQ52pJN1Y2dTPMWcGYilgWCNI==mZZw"]}`},
		// good acl: two valid signatures from different keyid
		{true, 2, `{"name":"testacl","validfrom":"2015-08-03T13:51:34.696992143-04:00","expireafter":"2015-08-03T13:51:34.696992168-04:00","permissions":[{"module":"testmodule","minimumweight":1,"investigators":[{"id":1,"name":"Test Investigator","pgpfingerprint":"abcdef0123456789","weight":1}]}],"pgpsignatures":["wpwEAAEIABAFAlW/qiYJEF4AASq0LBwdAAD3PAQAA3DHS0UQKV5HoZtcMXuDr/KYVoJ6QDfivCbVj47dle0lTGclENWsYBh/ss17vFyUevN3r5NUyUMvHCfOaW1KDZiJCAMEcGyZH2qeTuzh2dhY+psEtzD9T0YPfYmfRu04qD+qYi2OPCCVUFv2qPXTclvdt0w187HMWa5nqd2qv/8==x9IK","wpwEAAEIABAFAlW/qiYJEJrzjm0et9PIAAD2YwQAEiCyBFPsqE/jEm7qyhYZG8WV0alPr67FBM4yN/DW5Q4s2omY8DZZHL3LlFEioFb1JqkQP1N8DrU6ppGNm0A2ySpTrBNnS0fBwJgEL3R2o92q5EG+IeLiGFmJst+6IbQ9bW5kgEnUnzF5U/dYc2TUVDvex0xAL2xlu2OX57XDLvw==l2Cm"]}`},
	}
	pubring, _, err := pgp.ArmoredKeysToKeyring(TESTACLPUBKEYS)
	if err != nil {
		t.Errorf("failed to make pubring from private key: %v", err)
	}
	for _, testacl := range testacls {
		var acl ACL
		err = json.Unmarshal([]byte(testacl.acl), &acl)
		if err != nil {
			t.Error(err)
		}
		err = acl.VerifySignatures(testacl.reqsig, pubring)
		if err == nil && !testacl.expect {
			t.Fatalf("invalid ACL considered valid. acl='%s'", testacl.acl)
		} else if err != nil && testacl.expect {
			t.Fatalf("valid ACL considered invalid. acl='%s'. err='%v'", testacl.acl, err)
		}
		t.Logf("Testing ACL '%s' with %d required signatures returned expected result '%t'", testacl.acl, testacl.reqsig, testacl.expect)
	}
}

var TESTACLFP1 string = `6988709832D65C38FF5193885E00012AB42C1C1D`
var TESTACLFP2 string = `E3AF9CEFF308C9FCBF6BFE6E9AF38E6D1EB7D3C8`

var TESTACLPUBKEYS = [][]byte{
	[]byte(`-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mI0EVb+D9AEEAKzQR95gnAEK8c3u4nKM3t7fo/YvxFzGnD+YtbuQrv0PWssfocBI
rXeYZITl+CzPhHft3vruXFQqn82a1b1BOMl6n46UHd58zNp3AXAC2pWIo/v9qBef
uGDKnRv9huEVPRVhOj2uzEuICvJXHtfP7qIOcOk16ds/AecKynDAlGflABEBAAG0
N01JRyBUZXN0IEludmVzdGlnYXRvciA8bWlndGVzdGludmVzdGlnYXRvckBleGFt
cGxlLm5ldD6IvQQTAQgAJwUCVb+D9AIbLwUJD3MUAAULCQgHAwUVCgkICwUWAgMB
AAIeAQIXgAAKCRBeAAEqtCwcHY75BACdB4PBHigoobPg3PlLGu6dUmCU+3XFnSqZ
jKc/ypxAEsjRhTP5MuOd9aJ7mh0t7obYs+5sajUwtAFZYreW5NK1ibzSaCzrotDC
ATyJen75UpX4Ca0mU8nxYGGA1VZ+VuI9wL5EayZ0glA/0cOIZ8NyjbOJ2Y5uFSo2
aDB9OnokibiNBFW/g/QBBADss4N0OM0Htiyr9+9bc7lImj1xE/yvtr7XJNPhl31C
nCqADQpVJgC+UisMaf4dgjKtNarBmGiadAHvuG5sizOEBMdgm3pPksnx81rAd1m7
pp65RLlmtCVv4aEh9vJ9epbLgIyftZWGSsqB8S+ljMquUhzw36nDCuhBuJ6Piv5C
6wARAQABiQFDBBgBCAAPBQJVv4P0AhsuBQkPcxQAAKgJEF4AASq0LBwdnSAEGQEI
AAYFAlW/g/QACgkQFjYPM+4SneQ24wP8CVn0qNJhqDnW8kuIhHfASplmP9/cvh6D
VGvlwvhQx2wS2z9Y9sO9TBK6N8EcMHaQTfVe/KdOkZtWH9mNVgogtREFz9KnWN13
prbGWdibIOf6eS/iWz3Vhly3n9Y61wljgvQ2StrOocvcp3VF+Kokdhe19rYuS4yE
PItLDdmkZrHNCQP8DgMGaqFPr6oXHTmBayBYLRBBf03lnGf4DcSTlmtE7Tiv8eZn
1PsMhIm4d64PoFgbZMoZlL6OvNGTwG/8tkVvKK85YkQOjlnwIvI8NJ3cnKHwJsDA
5VVBjVWz4SvxKxxqs2epY1n1/Xd6tN67VAm+7Suvmb/EdULHDwDYUTSRYtI=
=R3QX
-----END PGP PUBLIC KEY BLOCK-----`),
	[]byte(`-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mI0EVb+nWwEEAMnKnjaSJUvMeUzZ2JHj9lzHhPQoDsfxJTnC53lK2cldGkwQtfAX
0orUi4aTIIJ2ej4vEAxwr5oPjo1mpsQp1c6KVkETRq5x0pTtcApWfMHHqrmy/Yh2
J0pAK+zj+ilXCVN+1Jp72it9640Jh6RacTs4b9Znx+3gE1qGgsa/MMM1ABEBAAG0
Ok1JRyBUZXN0IEludmVzdGlnYXRvciAyIDxtaWd0ZXN0aW52ZXN0aWdhdG9yMkBl
eGFtcGxlLm5ldD6IvQQTAQgAJwUCVb+nWwIbLwUJD3MUAAULCQgHAwUVCgkICwUW
AgMBAAIeAQIXgAAKCRCa845tHrfTyJhjBACXEd7rMd+SUb5+maSdhGeZBYNjrVLT
G7PburQ/PthMZttNRRT6sVcwB1i2ADp7ZDQu9q5kDTfUKvNvrh0CckLlH69lWJL+
DRp8xuA1ArOmSDA9dmyku4qgKZKd5KLiP4O6ERNQQMQ9n8pRxPFvjhiDSoTzJzSP
Ui4MfuEbypBTLriNBFW/p1sBBACiV02spkszgDnvqgrrtUj/3kxfo+SFNnDeYZS9
0sLvA64okO4iuCeFYVsEqb1QzHROoI8acuF6EZ+7uiiQPk2xyrJkuEe8or2pDXRw
y2leKfMxnSTTfsgXowKj0yQbblQLZrMsZIgyn3/pqTTc9q5exfscuwTrSBJXCvhP
lWv6PwARAQABiQFDBBgBCAAPBQJVv6dbAhsuBQkPcxQAAKgJEJrzjm0et9PInSAE
GQEIAAYFAlW/p1sACgkQ3mCMmglfPElG5QP/dAmiMEGsQroBRklvaZVxap1VnnoD
gi//WDLHTg7Fzrdk9X7kNA7mFHllewoGIaFXZJUp9QcfOqvQPZ2UQ/80JDwEtAT9
qfzVhE5go9R71qEw73iTATzHuQd4plbdn40B4OIsQYR2/TMVCPWHoB24uYjVybRY
a8T6tWXkVN9DrJ0AnwP/Tse820GNNcAT5XOUd1k56HlOvkBBIIheSDtgbLOG/gac
hSCxiaHDYR8gxeV6H0zgBm+55uvpKC8C067jzN0CJ5cZMlY97+40XrwEP7SpDEJV
pKh++ywOzbHUNWDuHfiRA8Dq0cZy4ooocwlSFEz/Gflu4/lMrJ+mqAT/qh7rhUE=
=XJCT
-----END PGP PUBLIC KEY BLOCK-----`)}

var TESTACLPRIVKEYS = [][]byte{
	[]byte(`-----BEGIN PGP PRIVATE KEY BLOCK-----
Version: GnuPG v1

lQHYBFW/g/QBBACs0EfeYJwBCvHN7uJyjN7e36P2L8Rcxpw/mLW7kK79D1rLH6HA
SK13mGSE5fgsz4R37d767lxUKp/NmtW9QTjJep+OlB3efMzadwFwAtqViKP7/agX
n7hgyp0b/YbhFT0VYTo9rsxLiAryVx7Xz+6iDnDpNenbPwHnCspwwJRn5QARAQAB
AAP9HMinon1dQ/qH3QD0McKPSqMIYx8rVI1LhXeQ6YKx9XotvsBwMh/zg1ncKvCc
82XFztiizRE6Sqs7fm+oOINuYDE/FgRMB9oMjhj2e8Y8m8pxePytRXL+JJyzLImB
td6CAImo+0XR1CelJIigyJU8Nk+oKeUkywUUOtbNUUfN1i8CAMusAnY0X86TlLZq
uHOlctRMBh4fJcRQdz4n+0oDgNJyIJnkXJZjsJIi5ahD4PG0VONLfLGTpIVSoGfy
M38Wk48CANk2pUilDdOr1DhMZzwFg+mZ6NqNmNiQ53wRyV5xRBr5We+QNNcDheyx
yeuJx5mLh+ZWjhLd6XoUl1qUufK7g0sCAJV1Vj97p9+WlatyM4dV2+1MAmsFp8ua
UDc2+MkyjuM29smMCuJsW9n1jmUKz4yK0YD66Pr++WEJmS7DKAlgN3ulTLQ3TUlH
IFRlc3QgSW52ZXN0aWdhdG9yIDxtaWd0ZXN0aW52ZXN0aWdhdG9yQGV4YW1wbGUu
bmV0Poi9BBMBCAAnBQJVv4P0AhsvBQkPcxQABQsJCAcDBRUKCQgLBRYCAwEAAh4B
AheAAAoJEF4AASq0LBwdjvkEAJ0Hg8EeKCihs+Dc+Usa7p1SYJT7dcWdKpmMpz/K
nEASyNGFM/ky4531onuaHS3uhtiz7mxqNTC0AVlit5bk0rWJvNJoLOui0MIBPIl6
fvlSlfgJrSZTyfFgYYDVVn5W4j3AvkRrJnSCUD/Rw4hnw3KNs4nZjm4VKjZoMH06
eiSJnQHYBFW/g/QBBADss4N0OM0Htiyr9+9bc7lImj1xE/yvtr7XJNPhl31CnCqA
DQpVJgC+UisMaf4dgjKtNarBmGiadAHvuG5sizOEBMdgm3pPksnx81rAd1m7pp65
RLlmtCVv4aEh9vJ9epbLgIyftZWGSsqB8S+ljMquUhzw36nDCuhBuJ6Piv5C6wAR
AQABAAP6AguCMDsPzAHb6Veia45yMSYvL7UcTFxU0idqss4MC/5Gmh9pHnE0nAnt
S0jDJETsSsJmIxDXa0/JqhhUakwNk6soWeHwVD6jGrwV946sSaJ3GJ4DfFALsJm0
rzSZWRqm/Dcxf1yR0pnEfJAI6KFlQWx3ifdAeXkQXBa/g1IQwDkCAPNlr6qjDwy3
S4cp81+bay/O7cOHvQL1EVxodTd202LjDDkD8MZGil8ymakVYahoR90limtVQnRt
CYOrgQqoBscCAPj1EdackndbL4PWLGKY1CXR7YvRZBgYHSSybiJugX2SzQi55W6j
L3fltN0vZU2aXbhxMOzxKBZUuAW+2Vb6rr0B/jHN/b9BGoW+AboL6V2Urr4KWnWt
dZ5/zxzz7XxnOgE3GJ0k/CwSwwWeKXDrJXtfA+YGsMmSLtgtLsyXTfOjDR2U8okB
QwQYAQgADwUCVb+D9AIbLgUJD3MUAACoCRBeAAEqtCwcHZ0gBBkBCAAGBQJVv4P0
AAoJEBY2DzPuEp3kNuMD/AlZ9KjSYag51vJLiIR3wEqZZj/f3L4eg1Rr5cL4UMds
Ets/WPbDvUwSujfBHDB2kE31XvynTpGbVh/ZjVYKILURBc/Sp1jdd6a2xlnYmyDn
+nkv4ls91YZct5/WOtcJY4L0NkrazqHL3Kd1RfiqJHYXtfa2LkuMhDyLSw3ZpGax
zQkD/A4DBmqhT6+qFx05gWsgWC0QQX9N5Zxn+A3Ek5ZrRO04r/HmZ9T7DISJuHeu
D6BYG2TKGZS+jrzRk8Bv/LZFbyivOWJEDo5Z8CLyPDSd3Jyh8CbAwOVVQY1Vs+Er
8SscarNnqWNZ9f13erTeu1QJvu0rr5m/xHVCxw8A2FE0kWLS
=eXat
-----END PGP PRIVATE KEY BLOCK-----`),
	[]byte(`-----BEGIN PGP PRIVATE KEY BLOCK-----
Version: GnuPG v1

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
=9dDO
-----END PGP PRIVATE KEY BLOCK-----`)}
