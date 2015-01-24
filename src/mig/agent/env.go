// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"io/ioutil"
	"mig"
	"net"
	"net/http"
)

func findLocalIPs(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	// grab the local ip addresses
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}
	for _, addr := range addresses {
		if addr.String() == "127.0.0.1/8" || addr.String() == "::1/128" {
			continue
		}
		ctx.Agent.Env.Addresses = append(ctx.Agent.Env.Addresses, addr.String())
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Found local address %s", addr.String())}.Debug()
	}
	return
}

// findPublicIP queries the ip endpoint of the mig api to discover the
// public ip of the agent
func findPublicIP(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	resp, err := http.Get(APIURL + "/" + "ip")
	if err != nil {
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Failed to retrieve public ip from api: %v", err)}.Err()
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	ip := net.ParseIP(string(body))
	if ip == nil {
		err = fmt.Errorf("Public IP API returned bad results")
		ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
		return
	}
	ctx.Agent.Env.PublicIP = ip.String()
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("Found public ip %s", ctx.Agent.Env.PublicIP)}.Debug()
	return
}
