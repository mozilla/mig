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

	"github.com/gorilla/mux"

	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/api"
	"mig.ninja/mig/client/mig-client-daemon/config"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
	"mig.ninja/mig/client/mig-client-daemon/migapi/dispatch"
)

const configFileName string = ".mig.conf.json"

func main() {
	// Load the user's configuration file.
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}
	configPath := path.Join(currentUser.HomeDir, configFileName)

	clientConfig := config.MustLoad(configPath)

	// Set up dependencies for services offered by the API.
	actionsCatalog := actions.NewCatalog()
	actionDispatcher := dispatch.NewAPIDispatcher(clientConfig.APIServerAddress)
	pgpAuthenticator := authentication.NewPGPAuthorizer()

	// Set up and launch the HTTP server.
	topRouter := mux.NewRouter()
	api.RegisterRoutesV1(topRouter, api.Dependencies{
		ActionsCatalog: &actionsCatalog,
		ActionDispatch: api.ActionDispatchDependencies{
			Dispatcher:    actionDispatcher,
			Authenticator: &pgpAuthenticator,
		},
	})

	http.Handle("/", topRouter)

	bindAddr := fmt.Sprintf("127.0.0.1:%d", clientConfig.ListenPort)
	fmt.Printf("Client daemon listening on %s\n", bindAddr)
	http.ListenAndServe(bindAddr, nil)
}
