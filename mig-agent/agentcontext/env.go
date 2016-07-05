// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package agentcontext

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"mig.ninja/mig"
)

// findLocalIPs updates the given context with the IP Addresses found in the machine.
func findLocalIPs(orig_ctx AgentContext) (ctx AgentContext, err error) {
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
		ctx.Addresses = append(ctx.Addresses, addr.String())
		logChan <- mig.Log{Desc: fmt.Sprintf("Found local address %s", addr.String())}.Debug()
	}
	return
}

// findPublicIP queries the ip endpoint of the mig api to discover the
// public ip of the agent
func findPublicIP(orig_ctx AgentContext, hints AgentContextHints) (ctx AgentContext, err error) {
	ctx = orig_ctx

	tr := &http.Transport{
		Dial: (&net.Dialer{Timeout: 10 * time.Second}).Dial,
	}
	client := &http.Client{Transport: tr}
	var resp *http.Response

	// If any proxies have been configured, try to use those first, fall back to a
	// direct connection.
	for _, proxy := range hints.Proxies {
		logChan <- mig.Log{Desc: fmt.Sprintf("Trying proxy %v for public IP retrieval", proxy)}.Debug()
		pu, err := url.Parse("http://" + proxy)
		if err != nil {
			logChan <- mig.Log{Desc: fmt.Sprintf("Failed to parse proxy url http://%s - %v", proxy, err)}.Info()
			continue
		}
		tr.Proxy = http.ProxyURL(pu)
		resp, err = client.Get(hints.APIUrl + "/ip")
		if err != nil {
			logChan <- mig.Log{Desc: fmt.Sprintf("Public IP retrieval failed through proxy http://%s - %v", proxy, err)}.Info()
			continue
		} else {
			goto parseBody
		}
	}

	// Try a direct connection, but also take into consideration any proxies that may
	// have been configured in the proxy related environment variables.
	logChan <- mig.Log{Desc: "Trying proxy from environment otherwise direct connection for public IP retrieval"}.Debug()
	tr.Proxy = http.ProxyFromEnvironment
	resp, err = client.Get(hints.APIUrl + "/ip")
	if err != nil {
		logChan <- mig.Log{Desc: fmt.Sprintf("Public IP retrieval from API failed. Error was: %v", err)}.Info()
	} else {
		goto parseBody
	}

	// exit here if no connection succeeded
	logChan <- mig.Log{Desc: fmt.Sprintf("Failed to retrieve public ip from api: %v", err)}.Err()
	return

parseBody:
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	ip := net.ParseIP(string(body))
	if ip == nil {
		err = fmt.Errorf("Public IP API returned bad results")
		logChan <- mig.Log{Desc: fmt.Sprintf("%v", err)}.Err()
		return
	}
	ctx.PublicIP = ip.String()
	logChan <- mig.Log{Desc: fmt.Sprintf("Found public ip %s", ctx.PublicIP)}.Debug()
	return
}
