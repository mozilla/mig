// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly <zmullaly@mozilla.com>

package auth

import (
	"bytes"
	"encoding/base64"
	"testing"
)

type testSecring struct {
	Email       string
	Fingerprint string
	SecringB64  string
}

func TestPGPAuth(t *testing.T) {
	testCases := []struct {
		Description               string
		Message                   string
		KeyData                   testSecring
		Passphrase                string
		ShouldCachePassphrase     bool
		ExpectSigningToSucceed    bool
		ExpectSignatureToValidate bool
	}{
		{
			Description: `
If a key doesn't use a passphrase, not caching one should not cause
verification to fail.
			`,
			Message:                   "test data 1",
			KeyData:                   testRingNoPassphrase,
			Passphrase:                "",
			ShouldCachePassphrase:     false,
			ExpectSigningToSucceed:    true,
			ExpectSignatureToValidate: true,
		},
		{
			Description: `
If a key doesn't use a passphrase, caching one should still allow verification
to succeed.
			`,
			Message:                   "test data 2",
			KeyData:                   testRingNoPassphrase,
			Passphrase:                "notN3cess4ry",
			ShouldCachePassphrase:     true,
			ExpectSigningToSucceed:    true,
			ExpectSignatureToValidate: true,
		},
		{
			Description: `
If a key does use a passphrase, caching the right one should allow verification
to succeed.
			`,
			Message:                   "test data 3",
			KeyData:                   testRingWithPassphrase,
			Passphrase:                "testing123",
			ShouldCachePassphrase:     true,
			ExpectSigningToSucceed:    true,
			ExpectSignatureToValidate: true,
		},
		{
			Description: `
If a key does use a passphrase, not caching it should cause signing to fail.
			`,
			Message:                   "test data 4",
			KeyData:                   testRingWithPassphrase,
			Passphrase:                "wrong",
			ShouldCachePassphrase:     false,
			ExpectSigningToSucceed:    false,
			ExpectSignatureToValidate: false,
		},
	}

	for testCase, caseNum := range testCases {
		t.Logf("Running TestPGPAuth case #%d: %s", caseNum, testCase.Description)

		decodedSecring, _ := base64.StdEncoding.DecodeString(testCase.KeyData.Secringb64)
		secring := bytes.NewReader(decodedSecring)
		passphrase := ""
		if testCase.ShouldCachePassphrase {
			passphrase = testCase.Passphrase
		}
		authenticator := NewPGPAuth(secring, passphrase)

		keyFingerprint := Fingerprint(testCase.KeyData.Fingerprint)
		message := []byte(testCase.Message)

		signature, err := authenticator.Sign(keyFingerprint, message)
		if err != nil && testCase.ExpectSigningToSucceed {
			t.Errorf("Expected signing to succeed, but got error '%s'", err.Error())
		} else if err == nil && !testCase.ExpectSigningToSucceed {
			t.Errorf("Expected signing to fail, but it succeeded.")
		}

		err = authenticator.Verify(keyFingerprint, signature, message)
		if err != nil && testCase.ExpectSignatureToValidate {
			t.Errorf("Expected verification to suceed, but got error '%s'", err.Error())
		} else if err == nil && !testCase.ExpectSignatureToValidate {
			t.Errorf("Expected verification to fail, but it succeeded.")
		}
	}
}

// Generated without a passphrase
var testRingNoPassphrase testSecring = testSecring{
	Email:       "mig@test.com",
	Fingerprint: "998A3BAEF09BF48E137E749419977578AC858169",
	SecringB64: `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
Dx4ORSR0+rE8oKwGNPIXIGo7YsawKhhLyg==`,
}

// Generated with passphrase "testing123"
var testRingWithPassphrase testSecring = testSecring{
	Email:       "mig@test2.com",
	Fingerprint: "A823E24C1A095176642131F605D10E57C9430F32",
	SecringB64: `lQPGBFukE1MBCAC+DQXhZiz42LANoXMppdikqSkp/X3kDBGvcjeTIXb0DPXJog3N
Csls1uUK0jEHdawzUIFN915GyoHgU0J9RlmFlg6P2pOlZUZxsAOMmToof/RK90y9
Ys9BQfhT7LGFgebqhQL5w2TNtbezomITMdN9eZAeiAOoEhhlFOsy8+3UW5CvqxzL
h6CqH4o/GNDRqGgHGqUwgyTmHTMcLsW/C6hObuwDWcN2dWcOQF9X9K1SKAS9SKKu
Qrgo5E9/G5eZEGZK8GEyLiSoUBaHykkivbpy820NE6F6yAKF7MhutRaIgfED13aK
8XbQQjnF8tv5+ot1qABKlBJB9thHyAfrKqJ1ABEBAAH+BwMCsETmybO3Q///l1FX
fhAunjN0pEjPr3c2KaB9U9yKj+iqH/4lFhNxZHOe0yy1tQUkhoTmUkB4K2bOM9YT
JykCdrCttLche7kPguIBaY5FiglUDkTJaVxpkvT9XJohFelrByePs6mtUAs0GC7N
EUX8jpG+HbKy1WOs6ek3VDqW5dTdXERo9e4x4ZaNim1jSDfRpckBkcF+tPpJh1/X
I+ffETX1s915l9P+/kyjSTxj3T92DzgL+3wNBMybB9ta8ZH8GMwsVCnSQARqZaOq
Geol6PikxhO8T7DKejz3w0bzUsJw2pRiwZpwl1uaegXO59WFgfkRv4aj4xGKI46m
KHKz8O/WHrYh8fLUhfJqVZ2sNbLOmP+/LAMPR6PESAbgsRq9R1yACz9EftUggjq7
ZXwqxgYRVK6PNu+zcCZ4Bek5xUYesK5OlBh+zlGK9vKiWQtsPvx4xMb1h4Xcp2HV
gKPk813DkPmrDNjXU/ZFwvbMsiIw/Wl+t05icI5m1uyiyX+DfHlXObfL0Ysw7bhs
awEKWGdW0s3bFE90D7naiSIujcbCWiTpvfgUoodSF1x3N8OcLgO3StLDL7s/B3bE
ikn0aY+/3QY/8fvdpkrcLoiW0rGxzZyyy8UQoySJYqK9v1x6DL6i4s+JTClfptEw
J05SuqkeDyd2P0R/ZTqKGlJKaywVWsHlkTcSsQKbIe3ezCPvLjLMecvbO30pjtvB
X0qZs3DCp7aV9uRyr78OF/ryrqsfeNgVnfJ14MJGpSZOiHPWZUu+IrmbGWLOhflD
4700q2w6/xp5fBSaR16Q6dDTVxPR5MvyNZDD+tJSBJkdxzNwxSh/kHv2WVlQomj6
I9fO6FolO0vKKBe6bc5iBkiznCpLUtaXO0AJZUU9NJaKTuZ9fX35n1yDygmGnsvu
4RGeToD/57u2tBlNaWcgVGVzdDIgPG1pZ0B0ZXN0Mi5jb20+iQFOBBMBCgA4FiEE
qCPiTBoJUXZkITH2BdEOV8lDDzIFAlukE1MCGwMFCwkIBwIGFQoJCAsCBBYCAwEC
HgECF4AACgkQBdEOV8lDDzJLDQf9Eak/h1W9H1WalmHEr/4s7/2Kx5ygG9B0cRc4
Q+E2h2sEJWTkJcdrQbiESZae/moBj0e+gxH4R2BGxk42U0derZu5Fh3PcuvaBknT
drJIlO+jiWGc2FQ3LAm9qt1Ua7thEyDFUvW6DI/Stf/TeruNti6YV+Qdk33Ri/AY
Yer4qlPGDlZoQa5fbYVXOo29Zs3Y6r0ZGuTbnoEhaoywGdv8NcND4Xmx0tm4DTti
zB0jwPaB7KZtGNeJvFqUtvaS7yA8znwH5RToivo1jdvprKc7WaYXsD3cEz7LwHUY
GJ0Y6PKf58+ITSj2TF7msT56z6LLovLOG+bNwxwhXJiNm7Zy6Z0DxgRbpBNTAQgA
zfCeuhSrzrVLXJKnCOt7dC75WJWrDnfCyBvsaYPWgxACE2LnQu4MTl7wX0dTxwYL
8/Ap/3w+Czl1JUdUQmavnEagy1N4nvaDMYLSqsVX86fFKKXVxNnX7vcQHvksRS+m
z45PUSSCf7ZqTlGGx48bjfnuqkZI8aMMYgvbRo164bmqxnqD14LOVSCtZ/WtuUn+
2z3y/fHRiph9WQkiBaG/VfGsWrZz2X6jHe2ob/9lNs3CtDvRB32f3dggz1LG/f/O
SpcgBG6OYDpA547cG3JBuM9pkwIT5KKmwgGVmsMjMsqPCPQfDqPKsnGjCKtWmNgA
MoC/QMl5eoik8TLCq5pubwARAQAB/gcDAjt0eklqZWBN/2+cQuF0zCrzLslExgI6
J+ojrKf3SBbZeltHKa5xradPkMRIZn7QAIc96gV95aLUffMygKR0qpKdY+bdxh5F
YCsFsKOF7NEpz1Zd1pMw5XPiquD9T/JuBmpFxwDvexhTD6W5wSO/7lRj+61FUtbp
+MYQEUKsmbtHMYtUHMUhx25t/xDM6vGYTVkzW65LJDZEEkYZb6ORsGPNx2+hekYb
0dI1OFzWikEC6a+keDmRrwZ7gpoKpt/thneTRgyJ0AMpH42QCgOumM8g7bGluApz
flP69a/r0yR0jf4SJRlnGxFaKbeMvVHAZGkXn/3tW99AXTDsw1TzT8xgwsNS3BRa
Z1FLNtBOGQudRAUiNGhKec10T8TkntDzf/0nJDkgLxmWjjQXsDDVfP6z7UZnerD9
uTvMDCYiKsszpKyXjcD/dauVnPWpzNoJzzaOGqwCHmeCHm6/URgLdf/79CLrA1Px
vbt6UadtVY7N4dc0vqckYRg/KYDbHrk8B4wRM0RGWLjvTF/slj+KXvEAJsa3AHSa
dZE+ek79lHyqQ1L7c9riHR3HctYgwdVXvR4oRXQSLNxNlrbD6+hd23BbV5w8DF3u
OUfutQfeVwiQNBy503MWVaUygjxBqC9KvyZQxRBQSQ0+am+ljGG1MKzT8najxu77
jI0LUR4YZIAgbvkpv/a+z/ev062mtI3Mf7dJcsLqqzK3ksuzLP81oZFpEPqHr+Ed
T1URHe6M9bV/sP7MkpHjsJsbA2CLzBj8/7TDDYvRW7hccZlokzV6oYI15in+ecwj
wWR0Vvq0AcadbonErQO8Pw1XV2hygYIGJHgWJRSmXB8UegQpCKe5C8DO8H5wvEhe
bdO0VGKpyEalX0jJxuz7KRxjyjwbQdXATb2s+Er9FYczQc3rgQFBVINyZw/DYYkB
NgQYAQoAIBYhBKgj4kwaCVF2ZCEx9gXRDlfJQw8yBQJbpBNTAhsMAAoJEAXRDlfJ
Qw8yf54H/jl7jupnxuG9Z7QcwnVyz0gZnAuWvm0EWrNo9hqXDDH2RIYCRC+Jxg/y
pvG00sMMIn53mXIhrAf/MXcrgnZ2ms+M3pTOswEcso3OhNtIKOOajQDDpXHezTfG
nVTPpRGlqzRdqxgj9lOzTP1OJYvKems5r+N7jUMZbfpbTGmvGkGl8AIhdjPrUvQ2
Y3AhOCeQou1CPB9ryLk5S50FvAldoZ1Tpun7UxX595dr+j0fcRamQfcm89D7OWRT
ypew0WaGjxEIc/3MO0vO0w4Qjhqgmlui/qMScCmPCiGO360LguYYeVuB+vCkgrBT
JrOZOm2DH0Sxtt6uzXsUGNiqdjj6oco=`,
}
