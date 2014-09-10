// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package filechecker

import (
	"encoding/json"
	"strings"
)

// a helper to convert v1 syntax to v2 syntax
func ConvertParametersV1toV2(input []byte) Parameters {
	v1 := make(map[string]map[string]map[string][]string)
	v2 := newParameters()
	err := json.Unmarshal(input, &v1)
	if err != nil {
		panic(err)
	}
	for path, _ := range v1 {
		for method, _ := range v1[path] {
			for label, _ := range v1[path][method] {
				var s search
				s.Paths = append(s.Paths, path)
				slabel := strings.Replace(label, " ", "", -1)
				for _, value := range v1[path][method][label] {
					switch method {
					case "filename":
						s.Filenames = append(s.Filenames, value)
					case "regex":
						s.Regexes = append(s.Regexes, value)
					case "md5":
						s.MD5 = append(s.MD5, value)
					case "sha1":
						s.SHA1 = append(s.SHA1, value)
					case "sha256":
						s.SHA256 = append(s.SHA256, value)
					case "sha384":
						s.SHA384 = append(s.SHA384, value)
					case "sha512":
						s.SHA512 = append(s.SHA512, value)
					case "sha3_224":
						s.SHA3_224 = append(s.SHA3_224, value)
					case "sha3_256":
						s.SHA3_256 = append(s.SHA3_256, value)
					case "sha3_384":
						s.SHA3_384 = append(s.SHA3_384, value)
					case "sha3_512":
						s.SHA3_512 = append(s.SHA3_512, value)
					}
				}
				v2.Searches[slabel] = s
			}
		}
	}
	return *v2
}
