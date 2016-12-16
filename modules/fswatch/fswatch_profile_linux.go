// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package fswatch /* import "mig.ninja/mig/modules/fswatch" */

var localFsWatchProfile = fsWatchProfile{
	[]fsWatchProfileEntry{
		{"/boot", []fsWatchObject{}},
		{"/etc/cron.d", []fsWatchObject{}},
		{"/var/spool/cron", []fsWatchObject{}},
		{"/bin", []fsWatchObject{}},
		{"/sbin", []fsWatchObject{}},
		{"/usr/bin", []fsWatchObject{}},
		{"/usr/sbin", []fsWatchObject{}},
		{"/etc", []fsWatchObject{}},
		{"/etc/init.d", []fsWatchObject{}},
		{"/etc/systemd/system", []fsWatchObject{}},
	},
}
