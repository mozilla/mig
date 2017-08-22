// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mozilla/scribe"
)

func Example() {
	docstr := `---
objects:
  - object: raw
    raw:
      identifiers:
        - identifier: test
          value: Example
tests:
  - test: example
    name: an example test
    object: raw
    regexp:
      value: Example
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

var resultsFormattingDoc = `
{
	"objects": [
	{
		"object": "raw",
		"raw": {
			"identifiers": [
			{
				"identifier": "test",
				"value": "value"
			}
			]
		}
	}
	],

	"tests": [
	{
		"test": "test1",
		"name": "a test",
		"object": "raw",
		"regexp": {
			"value": "^va.*e$"
		}
	}
	]
}
`

func genericTestExec(t *testing.T, documentStr string) *scribe.Document {
	rdr := strings.NewReader(documentStr)
	scribe.Bootstrap()
	scribe.TestHooks(true)
	doc, err := scribe.LoadDocument(rdr)
	if err != nil {
		t.Fatalf("scribe.LoadDocument: %v", err)
	}
	err = scribe.AnalyzeDocument(doc)
	if err != nil {
		t.Fatalf("scribe.AnalyzeDocument: %v", err)
	}
	// Get results for each test and make sure the result matches what
	// expectedresult is set to
	testids := doc.GetTestIdentifiers()
	for _, x := range testids {
		stest, err := doc.GetTest(x)
		if err != nil {
			t.Fatalf("Document.GetTest: %v", err)
		}
		sres, err := scribe.GetResults(&doc, x)
		if err != nil {
			t.Fatalf("scribe.GetResults: %v", err)
		}
		if stest.ExpectError {
			if !sres.IsError {
				t.Fatalf("test %v should have been an error", x)
			}
		} else {
			if sres.IsError {
				t.Fatalf("test %v resulted in an error", x)
			}
			if sres.MasterResult != stest.ExpectedResult {
				t.Fatalf("result incorrect for test %v", x)
			}
		}
	}
	return &doc
}

func TestResultsFormatting(t *testing.T) {
	rdr := strings.NewReader(resultsFormattingDoc)
	scribe.Bootstrap()
	scribe.TestHooks(true)
	doc, err := scribe.LoadDocument(rdr)
	if err != nil {
		t.Fatalf("scribe.LoadDocument: %v", err)
	}
	err = scribe.AnalyzeDocument(doc)
	if err != nil {
		t.Fatalf("scribe.AnalyzeDocument: %v", err)
	}
	res, err := scribe.GetResults(&doc, "test1")
	if err != nil {
		t.Fatalf("scribe.GetResults: %v", err)
	}

	slr := res.SingleLineResults()
	if len(slr) != 2 {
		t.Fatalf("single line results incorrect line count")
	}
	if slr[0] != "master [true] name:\"a test\" id:\"test1\" hastrue:true error:\"\"" {
		t.Fatalf("single line result master has incorrect format")
	}
	if slr[1] != "sub [true] name:\"a test\" id:\"test1\" identifier:\"test\"" {
		t.Fatalf("single line result sub has incorrect format")
	}

	hrr_compare := `result for "a test" (test1)
	master result: true
	[true] identifier: "test"`
	if res.String() != hrr_compare {
		t.Fatalf("human readable result has incorrect format")
	}

	json_compare := `{"testid":"test1","name":"a test","description":"","iserror":false,"error":"","masterresult":true,"hastrueresults":true,"results":[{"result":true,"identifier":"test"}]}`
	if res.JSON() != json_compare {
		t.Fatalf("json result has incorrect format")
	}
}
