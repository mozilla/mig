// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package client /* import "mig.ninja/mig/client" */

import (
	"fmt"
	"strings"
)

// Parse macros specified in the client configuration for use in the client
func addTargetMacros(conf *Configuration) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("addTargetMacros() -> %v", e)
		}
	}()

	for _, x := range conf.Targets.Macro {
		iv := strings.Index(x, ":")
		if iv < 1 {
			es := fmt.Sprintf("Invalid macro format: %q", x)
			panic(es)
		}
		name := x[:iv]
		tgt := x[iv+1:]
		conf.Targets.addMacro(name, tgt)
	}

	return nil
}
