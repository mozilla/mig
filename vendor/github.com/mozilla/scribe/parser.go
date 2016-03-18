// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

// Load a scribe JSON document from the reader specified by r. Returns
// a Document type that can be passed to AnalyzeDocument(). On error,
// LoadDocument() returns the error that occurred.
func LoadDocument(r io.Reader) (Document, error) {
	var ret Document

	debugPrint("loading new document\n")
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return ret, err
	}
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return ret, err
	}

	debugPrint("new document has %v test(s)\n", len(ret.Tests))
	debugPrint("new document has %v object(s)\n", len(ret.Objects))
	debugPrint("new document has %v variable(s)\n", len(ret.Variables))

	debugPrint("validating document...\n")
	err = ret.Validate()
	if err != nil {
		return ret, err
	}

	return ret, nil
}

// Analyze a scribe document on the host system. The will prepare and
// execute all tests specified in the scribe document. Returns an error
// if a fatal error occurs.
//
// Note that an error in an individual test does not neccessarily represent
// a fatal error condition. In these cases, the test itself will be marked
// as having an error condition (stored in the Err field of the Test).
func AnalyzeDocument(d Document) error {
	debugPrint("preparing objects...\n")
	err := d.prepareObjects()
	if err != nil {
		return err
	}
	debugPrint("analyzing document...\n")
	return d.runTests()
}
