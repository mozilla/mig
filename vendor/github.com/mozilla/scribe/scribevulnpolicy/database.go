// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

func dbGetSupportedPlatforms() (ret []supportedPlatform, err error) {
	rows, err := dbconn.Query("SELECT id, name FROM namespace")
	if err != nil {
		return
	}
	for rows.Next() {
		var (
			nsid     int
			nsname   string
			platform supportedPlatform
		)
		err = rows.Scan(&nsid, &nsname)
		if err != nil {
			rows.Close()
			return
		}
		platform, err = getPlatform(nsname)
		if err == nil {
			platform.clairNamespaceID = nsid
			ret = append(ret, platform)
		}
	}
	err = rows.Err()
	return
}

func dbVulnsForPlatform(platform supportedPlatform) (ret []vuln, err error) {
	rows, err := dbconn.Query(`SELECT f.name, vff.version, v.name,
		v.severity, v.link, v.description
		FROM feature f
		JOIN vulnerability_fixedin_feature vff ON (vff.feature_id = f.id)
		JOIN vulnerability v ON (v.id = vff.vulnerability_id)
		WHERE f.namespace_id = $1
		ORDER BY f.name`, platform.clairNamespaceID)
	if err != nil {
		return
	}
	for rows.Next() {
		var newent vuln
		err = rows.Scan(&newent.pkgName, &newent.fixedInVersion,
			&newent.name, &newent.severity, &newent.link,
			&newent.description)
		if err != nil {
			rows.Close()
			return
		}
		ret = append(ret, newent)
	}
	err = rows.Err()
	return
}
