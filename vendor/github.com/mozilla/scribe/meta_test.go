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

// Used in testConcatPolicy
var concatPolicyDoc = `
{
	"variables": [
	{ "key": "root", "value": "./test/concat" }
	],

	"objects": [
	{
		"object": "testfile0-content",
		"filecontent": {
			"path": "${root}",
			"file": "testfile0",
			"expression": "var = \\((\\S+), (\\S+)\\)",
			"concat": "."
		}
	}
	],

	"tests": [
	{
		"test": "testfile0-noop",
		"expectedresult": true,
		"object": "testfile0-content"
	}
	]
}
`

func TestConcatPolicy(t *testing.T) {
	genericTestExec(t, concatPolicyDoc)
}

// Used in testImportChainPolicy
var importChainPolicyDoc = `
{
        "variables": [
        { "key": "root", "value": "./test/import-chain" }
        ],

        "objects": [
        {
                "object": "testfile0-combined",
                "filecontent": {
                        "path": "${root}",
                        "file": "testfile0",
                        "expression": "var = \\((\\S+), (\\S+)\\)",
                        "concat": ".",
                        "import-chain": [ "testfile1-minor" ]
                }
        },

        {
                "object": "testfile1-minor",
                "filecontent": {
                        "path": "${chain_root}",
                        "file": "testfile1",
                        "expression": "minor = (\\S+)",
                        "import-chain": [ "rawappend" ]
                }
        },

        {
                "object": "rawappend",
                "raw": {
                        "identifiers": [
                        {
                                "identifier": "rawidentifier",
                                "value": "teststring"
                        }
                        ]
                }
        }
        ],

        "tests": [
        {
                "test": "testfile0-noop",
                "expectedresult": true,
                "object": "testfile0-combined",
                "regexp": {
                        "value": "^1.5.8.teststring$"
                }
        }
        ]
}
`

func TestImportChainPolicy(t *testing.T) {
	genericTestExec(t, importChainPolicyDoc)
}

var tagsPolicyDoc = `
{
        "variables": [
        { "key": "root", "value": "./test/tags" }
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
                "tags": [
                        {
                                "key": "testtag",
                                "value": "tag1"
                        },
                        {
                                "key": "another tag",
                                "value": "Another tag with spaces"
                        }
                ],
                "expectedresult": true,
                "object": "file-hasline",
                "exactmatch": {
                        "value": "false"
                }
        }
        ]
}
`

func TestTagsPolicy(t *testing.T) {
	genericTestExec(t, tagsPolicyDoc)
}

var rawPolicyDoc = `
{
        "objects": [
        {
                "object": "rawobject",
                "raw": {
                        "identifiers": [
                        {
                                "identifier": "an identifier",
                                "value": "VALUE"
                        },
                        {
                                "identifier": "another identifier",
                                "value": "TEST"
                        }
                        ]
                }
        }
        ],

        "tests": [
        {
                "test": "test0",
                "expectedresult": true,
                "object": "rawobject",
                "regexp": {
                        "value": "TEST"
                }
        }
        ]
}
`

func TestRawPolicy(t *testing.T) {
	genericTestExec(t, rawPolicyDoc)
}
