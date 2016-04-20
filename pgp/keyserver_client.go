// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package pgp /* import "mig.ninja/mig/pgp" */

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

func GetArmoredKeyFromKeyServer(keyid, keyserver string) (key []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetArmoredKeyFromKeyServer() -> %v", e)
		}
	}()
	re := regexp.MustCompile(`^0x[ABCDEF0-9]{8,64}$`)
	if !re.MatchString(keyid) {
		panic("Invalid key id. Must be in format '0x[ABCDEF0-9]{8,64}")
	}
	reqstr := fmt.Sprintf("%s/pks/lookup?op=get&options=mr&search=%s", keyserver, keyid)
	resp, err := http.Get(reqstr)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic("keyserver lookup error: " + http.StatusText(resp.StatusCode))
	}
	key, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return
}
