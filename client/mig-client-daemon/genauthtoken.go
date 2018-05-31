// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os/user"
	"path"

	"mig.ninja/mig/client"
	"mig.ninja/mig/pgp"
)

// CONFIGURATION
const (
	DANGEROUS_PASSPHRASE = "SECRET KEY PASSPHRASE"
	keyID                = "SECRET KEY FINGERPRINT"
)

func main() {
	pgp.CachePassphrase(DANGEROUS_PASSPHRASE)

	curUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	cli := client.Client{
		Conf: client.Configuration{
			Homedir: curUser.HomeDir,
			GPG: client.GpgConf{
				Home:  path.Join(curUser.HomeDir, ".gnupg"),
				KeyID: keyID,
			},
		},
		API: &http.Client{
			Transport: &http.Transport{
				DisableCompression: false,
				DisableKeepAlives:  false,
				Proxy:              http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: false,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
				},
			},
		},
	}

	signedToken, err := cli.MakeSignedToken()
	if err != nil {
		panic(err)
	}

	fmt.Println(signedToken)
}
