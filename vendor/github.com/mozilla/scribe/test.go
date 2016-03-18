// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"fmt"
	"strings"
)

// Describes arbitrary key value tags that can be associated with a test
type TestTag struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Test struct {
	TestID      string `json:"test"`   // The ID for this test.
	TestName    string `json:"name"`   // An optional name for this test
	Object      string `json:"object"` // The object this test references.
	Description string `json:"description,omitempty"`

	// Evaluators
	EVR    EVRTest    `json:"evr,omitempty"`        // EVR version comparison
	Regexp Regex      `json:"regexp,omitempty"`     // Regular expression comparison
	EMatch ExactMatch `json:"exactmatch,omitempty"` // Exact string match

	Tags []TestTag `json:"tags,omitempty"` // Tags associated with the test

	If []string `json:"if,omitempty"` // Slice of test names for dependencies

	// These values are optional but can be set to use the expected result
	// callback handler. These are primarily used for testing but can also
	// be used to trigger scribecmd to return and a non-zero exit status
	// if a test does not evaluate to the desired value.
	ExpectedResult bool `json:"expectedresult,omitempty"` // Expected master result for test
	ExpectError    bool `json:"expecterror,omitempty"`    // True if test should result in error

	prepared  bool // True if test has been prepared.
	evaluated bool // True if test has been evaluated at least once.

	err error // The last error condition encountered during preparation or execution.

	// The final result for this test, a rolled up version of the results
	// of this test for any identified candidates. If at least one
	// candidate for the test evaluated to true, the master result will be
	// true.
	masterResult   bool               // The final result for the test.
	hasTrueResults bool               // True if at least one result evaluated to true.
	results        []evaluationResult // A slice of results for the test.
}

// The result of evaluation of a test. There can be more then one
// EvaluationResult present in the results of a test, if the source
// information returned more than one matching object.
type evaluationResult struct {
	criteria evaluationCriteria // Criteria used during evaluation.
	result   bool               // The result of the evaluation.
}

// Generic criteria for an evaluation. A source object should always support
// conversion from the specific type to a set of evaluation criteria.
//
// An identifier is used to track the source of an evaluation. For example,
// this may be a filename or a package name. In those examples, the testValue
// may be matched content from the file, or a package version string.
type evaluationCriteria struct {
	identifier string // The identifier used to track the source.
	testValue  string // the actual test data passed to the evaluator.
}

type genericEvaluator interface {
	evaluate(evaluationCriteria) (evaluationResult, error)
}

func (t *Test) validate(d *Document) error {
	if len(t.TestID) == 0 {
		return fmt.Errorf("a test in document has no identifier")
	}
	if t.getEvaluationInterface() == nil {
		return fmt.Errorf("%v: no valid evaluation interface", t.TestID)
	}
	for _, x := range t.If {
		ptr, err := d.getTest(x)
		if err != nil {
			return fmt.Errorf("%v: %v", t.TestID, err)
		}
		if ptr == t {
			return fmt.Errorf("%v: test cannot reference itself", t.TestID)
		}
	}
	// Ensure the tags only contain valid characters
	for _, x := range t.Tags {
		if strings.ContainsRune(x.Key, '"') {
			return fmt.Errorf("%v: test tag key cannot contain quote", t.TestID)
		}
		if strings.ContainsRune(x.Value, '"') {
			return fmt.Errorf("%v: test tag value cannot contain quote", t.TestID)
		}
	}
	return nil
}

func (t *Test) getEvaluationInterface() genericEvaluator {
	if t.EVR.Value != "" {
		return &t.EVR
	} else if t.Regexp.Value != "" {
		return &t.Regexp
	} else if t.EMatch.Value != "" {
		return &t.EMatch
	}
	// If no evaluation criteria exists, use a no op evaluator
	// which will always return true for the test if any source objects
	// are identified.
	return &noop{}
}

func (t *Test) errorHandler(d *Document) error {
	if sRuntime.excall == nil {
		return t.err
	}
	if !t.ExpectError {
		tr, err := GetResults(d, t.TestID)
		if err != nil {
			panic("GetResults() in errorHandler")
		}
		sRuntime.excall(tr)
	}
	return t.err
}

func (t *Test) runTest(d *Document) error {
	if t.evaluated {
		return nil
	}

	// If this test has failed at some point, return the error.
	if t.err != nil {
		return t.err
	}

	debugPrint("runTest(): running \"%v\"\n", t.TestID)
	t.evaluated = true
	// First, see if this test has any dependencies. If so, run those
	// before we execute this one.
	for _, x := range t.If {
		dt, err := d.getTest(x)
		if err != nil {
			t.err = err
			return t.errorHandler(d)
		}
		err = dt.runTest(d)
		if err != nil {
			t.err = fmt.Errorf("a test dependency failed (\"%v\")", x)
			return t.errorHandler(d)
		}
	}

	ev := t.getEvaluationInterface()
	if ev == nil {
		t.err = fmt.Errorf("test has no valid evaluation interface")
		return t.errorHandler(d)
	}
	// Make sure the object is prepared before we use it.
	flag, err := d.objectPrepared(t.Object)
	if err != nil {
		t.err = err
		return t.errorHandler(d)
	}
	if !flag {
		t.err = fmt.Errorf("object not prepared")
		return t.errorHandler(d)
	}
	si, _ := d.getObjectInterface(t.Object)
	if si == nil {
		t.err = fmt.Errorf("test has no valid source interface")
		return t.errorHandler(d)
	}
	for _, x := range si.getCriteria() {
		res, err := ev.evaluate(x)
		if err != nil {
			t.err = err
			return t.errorHandler(d)
		}
		t.results = append(t.results, res)
	}

	// Set the master result for the test. If any of the dependent tests
	// are false from a master result perspective, this one is also false.
	// If at least one result for this test is true, the master result for
	// the test is true.
	t.hasTrueResults = false
	for _, x := range t.results {
		if x.result {
			t.hasTrueResults = true
		}
	}
	t.masterResult = false
	if t.hasTrueResults {
		t.masterResult = true
	}
	for _, x := range t.If {
		dt, err := d.getTest(x)
		if err != nil {
			t.err = err
			t.masterResult = false
			return t.errorHandler(d)
		}
		if !dt.masterResult {
			t.masterResult = false
			break
		}
	}

	// See if there is a test expected result handler installed, if so
	// validate it and call the handler if required.
	if sRuntime.excall != nil {
		if (t.masterResult != t.ExpectedResult) ||
			t.ExpectError {
			tr, err := GetResults(d, t.TestID)
			if err != nil {
				panic("GetResults() in expected handler")
			}
			sRuntime.excall(tr)
		}
	}

	return nil
}
