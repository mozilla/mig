// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe_test

import (
	"fmt"
	"scribe"
	"strings"
)

func Example() {
	docstr := `
	{
		"objects": [
		{
			"object": "raw",
			"raw": {
				"identifiers": [
				{
					"identifier": "test",
					"value": "Example"
				}
				]
			}
		}
		],
		"tests": [
		{
			"test": "example",
			"object": "raw",
			"regexp": {
				"value": "Example"
			}
		}
		]
	}
	`
	rdr := strings.NewReader(docstr)
	// Initialize the library.
	scribe.Bootstrap()
	// Load the document from the reader, returning a Document type.
	// LoadDocument() will also call ValidateDocument() to check for any
	// consistency issues.
	doc, err := scribe.LoadDocument(rdr)
	if err != nil {
		fmt.Println("LoadDocument:", err)
		return
	}
	// Analyze the document.
	err = scribe.AnalyzeDocument(doc)
	if err != nil {
		fmt.Println("AnalyzeDocument:", err)
		return
	}
	// Grab the results for the test, most of the time you would loop
	// through the results of GetTestIdentifiers() rather then call a result
	// directly.
	result, err := scribe.GetResults(&doc, "example")
	if err != nil {
		fmt.Println("GetResults:", err)
		return
	}
	if result.MasterResult {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}

	// Output: true
}
