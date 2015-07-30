// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com [:alm]

// The agentcontext package provides functionality to obtain information
// about the system a given agent or loader is running on. This includes
// information unrelated to MIG itself, such as the hostname of the system,
// IP addresses, and so on.
package agentcontext /* import "mig.ninja/mig/mig-agent/agentcontext" */

import (
	"fmt"
	"github.com/kardianos/osext"
	"mig.ninja/mig"
	"os"
	"runtime"
	"strings"
)

// Information from the system the agent is running on
type AgentContext struct {
	Hostname     string   // Hostname
	BinPath      string   // Path to invoked binary
	RunDir       string   // Agent runtime directory
	OS           string   // Operating System
	OSIdent      string   // OS release identifier
	Init         string   // OS Init
	Architecture string   // System architecture
	Addresses    []string // IP addresses
	PublicIP     string   // Systems public IP from perspective of API

	AWS AWSContext // AWS specific information
}

func (ctx *AgentContext) ToAgent() (ret mig.Agent) {
	ret.Name = ctx.Hostname
	ret.PID = os.Getpid()
	ret.Env.OS = ctx.OS
	ret.Env.Arch = ctx.Architecture
	ret.Env.Ident = ctx.OSIdent
	ret.Env.Init = ctx.Init
	ret.Env.Addresses = ctx.Addresses
	ret.Env.PublicIP = ctx.PublicIP
	ret.Env.AWS.InstanceID = ctx.AWS.InstanceID
	ret.Env.AWS.LocalIPV4 = ctx.AWS.LocalIPV4
	ret.Env.AWS.AMIID = ctx.AWS.AMIID
	ret.Env.AWS.InstanceType = ctx.AWS.InstanceType
	return
}

// Passed to NewAgentContext() to inform environment discovery
type AgentContextHints struct {
	APIUrl           string   // MIG API URL
	Proxies          []string // Proxies avialable for use in discovery
	DiscoverPublicIP bool     // Attempt to discover public IP
	DiscoverAWSMeta  bool     // Attempt to discover AWS metadata
}

// Information used for agents running in AWS environments
type AWSContext struct {
	InstanceID   string // AWS instance ID
	LocalIPV4    string // AWS Local IPV4 address
	AMIID        string // AWS AMI ID
	InstanceType string // AWS instance type
}

var logChan chan mig.Log

func NewAgentContext(lch chan mig.Log, hints AgentContextHints) (ret AgentContext, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("NewAgentContext() -> %v", e)
		}
	}()

	logChan = lch

	ret.BinPath, err = osext.Executable()
	if err != nil {
		panic(err)
	}

	ret, err = findHostname(ret)
	if err != nil {
		panic(err)
	}

	ret.OS = runtime.GOOS
	ret.Architecture = runtime.GOARCH
	ret.RunDir = GetRunDir()
	ret, err = findOSInfo(ret)
	if err != nil {
		panic(err)
	}
	ret, err = findLocalIPs(ret)
	if err != nil {
		panic(err)
	}

	if hints.DiscoverPublicIP {
		ret, err = findPublicIP(ret, hints)
		if err != nil {
			panic(err)
		}
	}

	if hints.DiscoverAWSMeta {
		ret, err = addAWSMetadata(ret)
		if err != nil {
			panic(err)
		}
	}

	return
}

// cleanString removes spaces, quotes and newlines
func cleanString(str string) string {
	if len(str) < 1 {
		return str
	}
	if str[len(str)-1] == '\n' {
		str = str[0 : len(str)-1]
	}
	// remove heading whitespaces and quotes
	for {
		if len(str) < 2 {
			break
		}
		switch str[0] {
		case ' ', '"', '\'':
			str = str[1:len(str)]
		default:
			goto trailing
		}
	}
trailing:
	// remove trailing whitespaces, quotes and linebreaks
	for {
		if len(str) < 2 {
			break
		}
		switch str[len(str)-1] {
		case ' ', '"', '\'', '\r', '\n':
			str = str[0 : len(str)-1]
		default:
			goto exit
		}
	}
exit:
	// remove in-string linebreaks
	str = strings.Replace(str, "\n", " ", -1)
	str = strings.Replace(str, "\r", " ", -1)
	return str
}
