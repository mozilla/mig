// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com [:alm]

package main

import (
	"os"
	"testing"

	"mig.ninja/mig"
)

var testContext Context

var testLogBuffer []mig.Log

func initTestContext() {
	testLogBuffer = make([]mig.Log, 0)
	testContext.Channels.Log = make(chan mig.Log, 100)
	go func() {
		for event := range testContext.Channels.Log {
			testLogBuffer = append(testLogBuffer, event)
		}
	}()
}

func TestMain(m *testing.M) {
	initTestContext()
	ret := m.Run()
	os.Exit(ret)
}
