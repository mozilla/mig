// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"mig"
	"net"

	"github.com/ccding/go-stun/stun"
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

var stunHosts = map[string]int{
	"stun01.sipphone.com": 3478,
	"stun.ekiga.net":      3478,
	"stun.fwdnet.net":     3478,
	"stun.ideasip.com":    3478,
	"stun.iptel.org":      3478,
	"stun.rixtelecom.se":  3478,
	"stunserver.org":      3478,
	"stun.softjoys.com":   3478,
	"stun.voiparound.com": 3478,
	"stun.voipbuster.com": 3478,
	"stun.voipstunt.com":  3478,
	"stun.voxgratia.org":  3478,
	"stun.xten.com":       3478,
}

func findNATviaStun(orig_ctx Context) (ctx Context, err error) {
	ctx = orig_ctx
	ctr := 0
	for stunSrv, stunPort := range stunHosts {
		stun.SetServerHost(stunSrv, stunPort)
		nat, host, err := stun.Discover()
		if err != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("STUN discovery failed against %s:%d with error '%v'", stunSrv, stunPort, err)}.Debug()
		}
		switch nat {
		case stun.NAT_ERROR:
			ctx.Agent.Env.NAT.Result = "Test failed"
		case stun.NAT_UNKNOWN:
			ctx.Agent.Env.NAT.Result = "Unexpected response from the STUN server"
		case stun.NAT_BLOCKED:
			ctx.Agent.Env.NAT.Result = "UDP is blocked"
		case stun.NAT_FULL:
			ctx.Agent.Env.NAT.Result = "Full cone NAT"
		case stun.NAT_SYMETRIC:
			ctx.Agent.Env.NAT.Result = "Symmetric NAT"
		case stun.NAT_RESTRICTED:
			ctx.Agent.Env.NAT.Result = "Restricted NAT"
		case stun.NAT_PORT_RESTRICTED:
			ctx.Agent.Env.NAT.Result = "Port restricted NAT"
		case stun.NAT_NONE:
			ctx.Agent.Env.NAT.Result = "Not behind a NAT"
		case stun.NAT_SYMETRIC_UDP_FIREWALL:
			ctx.Agent.Env.NAT.Result = "Symmetric UDP firewall"
		default:
			ctx.Agent.Env.NAT.Result = "Unknown"
		}
		if host != nil {
			ctx.Agent.Env.NAT.IP = host.Ip()
			ctx.Agent.Env.NAT.StunServer = fmt.Sprintf("%s:%d", stunSrv, stunPort)
			break
		}
		ctr++
		if ctr == 3 {
			ctx.Agent.Env.NAT.Result = "Attempted 3 lookups without results"
			break
		}
	}
	return
}
