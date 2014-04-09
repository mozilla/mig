/* Mozilla InvestiGator

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

package mig

import (
	"fmt"
	"strings"
)

type ACL []Permission

type Permission map[string]struct {
	RequiredSignatures           int
	RequiredAuthoritativeSigners int
	AuthoritativeSigners         []string
	NonAuthoritativeSigners      []string
}

// verifyPermission controls that the PGP keys, identified by their fingerprints, that
// signed an operation are sufficient to allow this operation to run
func verifyPermission(operation operation, permName string, perm Permission, fingerprints []string) (err error) {
	if perm[permName].RequiredSignatures < 1 {
		return fmt.Errorf("Invalid permission '%s'. Must require at least 1 signature, has %d",
			permName, perm[permName].RequiredSignatures)
	}
	countSignatures := 0
	countAuthoritativeSignatures := 0
	for _, fp := range fingerprints {
		for _, vfp := range perm[permName].AuthoritativeSigners {
			if strings.ToUpper(fp) == strings.ToUpper(vfp) {
				countSignatures++
				countAuthoritativeSignatures++
			}
		}
		for _, vfp := range perm[permName].NonAuthoritativeSigners {
			if strings.ToUpper(fp) == strings.ToUpper(vfp) {
				countSignatures++
			}
		}
	}
	if countAuthoritativeSignatures < perm[permName].RequiredAuthoritativeSigners {
		return fmt.Errorf("Permission denied for operation '%s'. Insufficient number of Authoritative signatures. Need %d, got %d",
			operation.Module, perm[permName].RequiredAuthoritativeSigners, countAuthoritativeSignatures)
	}
	if countSignatures < perm[permName].RequiredSignatures {
		return fmt.Errorf("Permission denied for operation '%s'. Insufficient number of signatures. Need %d, got %d",
			operation.Module, perm[permName].RequiredSignatures, countSignatures)
	}
	return
}
