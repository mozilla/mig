// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

// Package scribe implements the Mozilla scribe host policy evaluator.
package scribe

import (
	"fmt"
	"io"
)

type runtime struct {
	debugging   bool
	debugWriter io.Writer
	excall      func(TestResult)
	testHooks   bool
	fileLocator func(string, bool, string, int) ([]string, error)
}

// Version is the scribe library version
const Version = "0.5"

var sRuntime runtime

func (r *runtime) initialize() {
}

func init() {
	sRuntime.initialize()
}

// Bootstrap the scribe library. This function is currently not used but code
// should call this function before any other functions in the library. An
// error is returned if one occurs.
//
// Applications should call this as it may be mandatory in the future to do
// more explicit initialization of the library outside of init().
func Bootstrap() (err error) {
	return err
}

// ExpectedCallback can be used to set a callback function for test results.
//
// Set an expected result callback. f should be a function that takes a TestResult
// type as an argument. When this is set, if the result of a test does not
// match the value set in "expectedresult" for the test, the function is
// immediately called with the applicable TestResult as an argument.
func ExpectedCallback(f func(TestResult)) {
	sRuntime.excall = f
}

// InstallFileLocator installs alternate file walking functions.
//
// Install an alternate file location function. This overrides use of the
// SimpleFileLocator locate() function, and allows specification of an
// alternate function to use for locating candidate files on the filesystem.
//
// This function is primarily used within the scribe mig module to make use
// of the file module traversal function.
func InstallFileLocator(f func(string, bool, string, int) ([]string, error)) {
	sRuntime.fileLocator = f
}

// TestHooks enables or disables testing hooks in the library.
//
// Enable or disable test hooks. If test hooks are enabled, certain functions
// such as requesting package data from the host system are bypassed in favor
// of test tables.
func TestHooks(f bool) {
	sRuntime.testHooks = f
}

func debugPrint(s string, args ...interface{}) {
	if !sRuntime.debugging {
		return
	}
	buf := fmt.Sprintf(s, args...)
	fmt.Fprintf(sRuntime.debugWriter, "[scribe] %v", buf)
}

// SetDebug enables or disables debugging. If debugging is enabled, output is written
// to the io.Writer specified by w.
func SetDebug(f bool, w io.Writer) {
	sRuntime.debugging = f
	sRuntime.debugWriter = w
	debugPrint("debugging enabled\n")
}
