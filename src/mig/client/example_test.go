// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package client_test

import (
	"fmt"
	"mig/client"
)

func ExampleGetAction() {
	conf := client.Configuration{
		API: client.ApiConf{
			URL: "http://localhost:12345/api/v1/",
		},
		GPG: client.GpgConf{
			Home:  "/home/ulfr/.gnupg/",
			KeyID: "E60892BB9BD89A69F759A1A0A3D652173B763E8F",
		},
	}
	conf.Homedir = client.FindHomedir()
	cli := client.NewClient(conf)
	a, _, err := cli.GetAction(1234567890)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(a.ID, a.Name, a.Target)
	}
}
