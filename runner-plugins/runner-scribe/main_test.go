// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package main

import (
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/gcfg.v1"
)

type mozdef struct {
	URL      string
	UseProxy bool
}

func TestConfigParsing(t *testing.T) {
	testCases := []struct {
		Description    string
		ConfigString   string
		ExpectedConfig config
		ExpectError    bool
	}{
		{
			Description: "A valid configuration should parse correctly",
			ConfigString: `
      [mozdef]
      url = "testurl"
      useProxy = true
      `,
			ExpectedConfig: config{
				MozDef: mozdef{
					URL:      "testurl",
					UseProxy: true,
				},
			},
			ExpectError: false,
		},
		{
			Description: "Parsing fails if UseProxy is not a boolean",
			ConfigString: `
      [mozdef]
      url = "testurl"
      useProxy = "notbool"
      `,
			ExpectedConfig: config{
				MozDef: mozdef{
					URL:      "testurl",
					UseProxy: false,
				},
			},
			ExpectError: true,
		},
	}

	for caseNum, testCase := range testCases {
		t.Logf("Running TestConfigParsing case #%d: %s", caseNum, testCase.Description)

		configFile, err := ioutil.TempFile("", "*.cfg")
		if err != nil {
			t.Fatal(err)
		}

		if _, err = configFile.Write([]byte(testCase.ConfigString)); err != nil {
			t.Fatal(err)
		}
		if err = configFile.Close(); err != nil {
			t.Fatal(err)
		}

		conf := config{}
		parseErr := gcfg.ReadFileInto(&conf, configFile.Name())

		gotErr := parseErr != nil
		if gotErr && !testCase.ExpectError {
			t.Errorf("Did not expect to get a parse error, but got '%s'", parseErr.Error())
		} else if !gotErr && testCase.ExpectError {
			t.Errorf("Expected to get a parse error, but did not")
		}

		os.Remove(configFile.Name())

		if testCase.ExpectError {
			continue
		}

		if conf.MozDef.URL != testCase.ExpectedConfig.MozDef.URL {
			t.Errorf(
				"Expected parsed URL to be %s but it's %s",
				testCase.ExpectedConfig.MozDef.URL,
				conf.MozDef.URL)
		}

		if conf.MozDef.UseProxy != testCase.ExpectedConfig.MozDef.UseProxy {
			t.Errorf(
				"Expected parsed UseProxy to be %v but it's %v",
				testCase.ExpectedConfig.MozDef.UseProxy,
				conf.MozDef.UseProxy)
		}
	}
}

func TestLookupOperatorTeam(t *testing.T) {
	var serviceApiAssets = make(map[string]ServiceApiAsset)
	testCases := [][]string{
		{ "hostname1", "team1", "operator1" },
		{ "hostname2", "", "operator2" },
		{ "hostname3", "team3", "" },
		{ "hostname4", "", ""},
	}

	// fill the map with test cases
	for _, test := range testCases {
		serviceApiAssets[test[0]] = ServiceApiAsset{AssetIdentifier: test[0], Team: test[1], Operator: test[2]}	
	}

	// run the test cases
	for _, test := range testCases {
		testOperator, testTeam := LookupOperatorTeam(test[0], serviceApiAssets)
		if testOperator != test[2] || testTeam != test[1] {
			t.Errorf(
				"Expected operator to be %v but it is %v. Expected team to be %v but it is %v",
				test[2],
				testOperator,
				test[1],
				testTeam)
		}	
	}

	// test lookup on a nonexistent hostname
	testOperator, testTeam := LookupOperatorTeam("hostnameDoesNotExist", serviceApiAssets)
	if testOperator != "" || testTeam != "" {
		t.Errorf(
			"Expected operator to be %v but it is %v. Expected team to be %v but it is %v",
			"",
			testOperator,
			"",
			testTeam)
	}	


}
