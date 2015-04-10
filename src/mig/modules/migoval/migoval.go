// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package migoval

import (
	"github.com/ameihm0912/mozoval/go/src/oval"
	"mig"
)

func init() {
	mig.RegisterModule("migoval", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters Parameters
}

func (r Runner) ValidateParameters() (err error) {
	return
}

type Results struct {
}

type Parameters struct {
	ModePkgList bool `json:"pkglistmode"`
}

func newParameters() *Parameters {
	return &Parameters{}
}

func migovalInitialize() {
	oval.Init()
}
