// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe_test

import (
	"testing"
)

// Used in TestHasLinePolicy
var hasLinePolicyDoc = `
{
        "variables": [
        { "key": "root", "value": "./test/hasline" }
        ],

        "objects": [
        {
                "object": "file-hasline",
                "hasline": {
                        "path": "${root}",
                        "file": ".*\\.txt",
                        "expression": ".*test.*"
                }
        }
        ],

        "tests": [
        {
                "test": "files-without-line",
                "expectedresult": true,
                "object": "file-hasline",
                "exactmatch": {
                        "value": "true"
                }
        }
        ]
}
`

func TestHasLinePolicy(t *testing.T) {
	genericTestExec(t, hasLinePolicyDoc)
}

// Used in TestFileContentPolicy
var fileContentPolicyDoc = `
{
	"variables": [
	{ "key": "root", "value": "./test/filecontent" }
	],

	"objects": [
	{
		"object": "nosuchfile",
		"filecontent": {
			"path": "${root}",
			"file": "nosuchfile",
			"expression": ".*"
		}
	},

	{
		"object": "testfile0-test",
		"filecontent": {
			"path": "${root}",
			"file": "testfile0",
			"expression": ".*(Test).*"
		}
	},

	{
		"object": "testfile0-all",
		"filecontent": {
			"path": "${root}",
			"file": "testfile0",
			"expression": "(.*)"
		}
	},

	{
		"object": "testfile0-regex",
		"filecontent": {
			"path": "${root}",
			"file": "testfile0",
			"expression": "^(T\\S+).*"
		}
	},

	{
		"object": "testfile1-version",
		"filecontent": {
			"path": "${root}",
			"file": ".*file1",
			"expression": "^Version = (\\S+)"
		}
	},

	{
		"object": "testfile2-version",
		"filecontent": {
			"path": "${root}",
			"file": ".*file2",
			"expression": "^Version = (\\S+)"
		}
	},

	{
		"object": "anyfile",
		"filecontent": {
			"path": "${root}/",
			"file": ".*",
			"expression": "(.*)"
		}
	}
	],

	"tests": [
	{
		"test": "simplecontent0",
		"description": "test with nonexistent file",
		"expectedresult": false,
		"object": "nosuchfile"
	},

	{
		"test": "simplecontent1",
		"expectedresult": true,
		"object": "testfile0-test"
	},

	{
		"test": "simplecontent2",
		"expectedresult": true,
		"object": "testfile0-all",
		"regexp": {
			"value": "Test"
		}
	},

	{
		"test": "simplecontent3",
		"expectedresult": true,
		"object": "testfile0-regex",
		"regexp": {
			"value": "Test"
		}
	},

	{
		"test": "filecontent0",
		"expectedresult": true,
		"object": "testfile1-version",
		"evr": {
			"operation": "<",
			"value": "0.6"
		}
	},

	{
		"test": "filecontent1",
		"expectedresult": false,
		"object": "testfile1-version",
		"evr": {
			"operation": "<",
			"value": "0.6"
		},
		"if": [ "simplecontent0" ]
	},

	{
		"test": "filecontent2",
		"expectedresult": true,
		"object": "testfile1-version",
		"evr": {
			"operation": "<",
			"value": "0.6"
		},
		"if": [ "simplecontent1" ]
	},

	{
		"test": "filecontent3",
		"description": "version is ok",
		"expectedresult": false,
		"object": "testfile1-version",
		"evr": {
			"operation": "<",
			"value": "0.4z"
		}
	},

	{
		"test": "filecontent4",
		"description": "version is ok",
		"expectedresult": false,
		"object": "testfile2-version",
		"evr": {
			"operation": "<",
			"value": "0.4z"
		}
	},

	{
		"test": "anyfile0",
		"expectedresult": true,
		"object": "anyfile"
	}

	]
}
`

func TestFileContentPolicy(t *testing.T) {
	genericTestExec(t, fileContentPolicyDoc)
}

var fileNamePolicyDoc = `
{
	"variables": [
	{ "key": "root", "value": "./test/filename" }
	],

	"objects": [
	{
		"object": "anyfile",
		"filename": {
			"path": "${root}",
			"file": "(.*)"
		}
	},

	{
		"object": "nosuchfile",
		"filename": {
			"path": "${root}",
			"file": ".*(nosuchfile).*"
		}
	},

	{
		"object": "fileversion",
		"filename": {
			"path": "${root}",
			"file": "file-(\\S+).txt"
		}
	},

	{
		"object": "testfile0",
		"filename": {
			"path": "${root}",
			"file": "^(testfile0)$"
		}
	}
	],

	"tests": [
	{
		"test": "filename0",
		"expectedresult": true,
		"object": "anyfile"
	},

	{
		"test": "filename1",
		"expectedresult": false,
		"object": "nosuchfile"
	},

	{
		"test": "filename2",
		"expectedresult": true,
		"object": "fileversion",
		"evr": {
			"operation": "<",
			"value": "1.6.0"
		}
	},

	{
		"test": "filename3",
		"expectedresult": false,
		"object": "fileversion",
		"evr": {
			"operation": "<",
			"value": "1.1.0"
		}
	},

	{
		"test": "filename4",
		"expectedresult": true,
		"object": "fileversion",
		"regexp": {
			"value": "1.2.3"
		}
	},

	{
		"test": "filename5",
		"expectedresult": true,
		"object": "testfile0"
	}

	]
}
`

func TestFileNamePolicy(t *testing.T) {
	genericTestExec(t, fileNamePolicyDoc)
}
