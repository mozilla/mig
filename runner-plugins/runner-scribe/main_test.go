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
      source = "mozdef"
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
      source = "mozdef"
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
