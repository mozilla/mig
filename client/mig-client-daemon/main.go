// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package main

import (
	"fmt"
	"net/http"
	"os/user"
	"path"

	"mig.ninja/mig/client/mig-client-daemon/config"
)

const configFileName string = ".mig.conf.json"

func main() {
	// Load the user's configuration file
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	configPath := path.Join(currentUser.HomeDir, configFileName)

	clientConfig := config.MustLoad(configPath)

	bindAddr := fmt.Sprintf("127.0.0.1:%d", clientConfig.ListenPort)
	fmt.Printf("Client daemon listening on %s\n", bindAddr)
	http.ListenAndServe(bindAddr, nil)
}
