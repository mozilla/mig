/* Mozilla InvestiGator PGP

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package sign

import (
	"fmt"
	"strings"
	"unsafe"
)

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -lgpgme -lgpg-error libmig_gpgme.a
#include <libmig_gpgme.h>
*/
import "C"

// Sign() signs a string with a key. The function uses a C library that
// calls gpgme, for compatibility with gpg-agent.
func Sign(data string, key string) (sig string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("pgpSign(): %v", e)
		}
	}()

	// convert to C variable
	ckey := C.CString(key)
	if ckey == nil {
		panic("Error converting key to CString type")
	}
	cdata := C.CString(data)
	if cdata == nil {
		panic("Error converting data to CString type")
	}

	// calculate signature
	csig := C.GPGME_Sign(cdata, ckey)
	if csig == nil {
		panic("Error computing signature")
	}

	// convert signature back to Go string
	sig = deArmor(C.GoString(csig))
	if sig == "" {
		panic("Error converting signature to GOString type")
	}

	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(cdata))

	return
}

// deArmor takes a multi line armored GPG signature, and turns it back
// into a single line signature (thank you, camlistore folks)
func deArmor(sig string) string {
	index1 := strings.Index(sig, "\n\n")
	index2 := strings.Index(sig, "\n-----")
	if index1 == -1 || index2 == -1 {
		panic("Failed to parse signature from gpg.")
	}
	inner := sig[index1+2 : index2]
	return strings.Replace(inner, "\n", "", -1)
}
