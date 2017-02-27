// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package fswatch /* import "mig.ninja/mig/modules/fswatch" */

var localFsWatchProfile = profile{
	entries: []profileEntry{
		{path: "/boot", recursive: false},
		{path: "/etc/cron.d", recursive: false},
		{path: "/var/spool/cron", recursive: false},
		{path: "/bin", recursive: false},
		{path: "/sbin", recursive: false},
		{path: "/usr/bin", recursive: false},
		{path: "/usr/sbin", recursive: false},
		{path: "/etc", recursive: true},
		{path: "/etc/init.d", recursive: false},
		{path: "/etc/systemd", recursive: true},
	},
}
