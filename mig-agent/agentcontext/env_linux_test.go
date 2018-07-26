// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Zack Mullaly <zmullaly@mozilla.com>

package agentcontext

import (
	"strings"
	"testing"
)

func TestGetOSRelease(t *testing.T) {
	testCases := []struct {
		Description   string
		FileContents  string
		ExpectedIdent string
		ExpectError   bool
	}{
		{
			Description: "A valid CentOS Linux 7 (Core) file should yield CentOS 7",
			FileContents: `NAME="CentOS Linux"
VERSION="7 (Core)"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="7"
PRETTY_NAME="CentOS Linux 7 (Core)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:centos:centos:7"
HOME_URL="https://www.centos.org/"
BUG_REPORT_URL="https://bugs.centos.org/"

CENTOS_MANTISBT_PROJECT="CentOS-7"
CENTOS_MANTISBT_PROJECT_VERSION="7"
REDHAT_SUPPORT_PRODUCT="centos"
REDHAT_SUPPORT_PRODUCT_VERSION="7"`,
			ExpectedIdent: "CentOS 7",
			ExpectError:   false,
		},
		{
			Description:   "A file missing a CentOS version should yield an error",
			FileContents:  "\\S",
			ExpectedIdent: "",
			ExpectError:   true,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("TestGetOSRelease #%d: %s", caseNum, testCase.Description)

		reader := strings.NewReader(testCase.FileContents)
		ident, err := getOSRelease(reader)
		defer reader.Close()

		gotErr := err != nil
		if testCase.ExpectError && !gotErr {
			t.Errorf("Expected to get an error but did not")
		} else if !testCase.ExpectedIdent && gotErr {
			t.Errorf("Did not expect to get an error, but got %s", err.Error())
		} else if ident != testCase.ExpectedIdent {
			t.Errorf("Expected to get ident \"%s\" but got \"%s\"", testCase.ExpectedIdent, ident)
		}
	}
}
