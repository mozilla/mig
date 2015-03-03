// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Sushant Dinesh sushant.dinesh94@gmail.com [:sushant94]

// ping module is used to check the connection between an endpoint
// and a destination host.

package ping

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"mig"
	"net"
	"os"
	"strings"
	"time"
)

func init() {
	mig.RegisterModule("ping", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters params
	Results    mig.ModuleResult
}

type params struct {
	Destination     string  `json:"destination"`               // ipv4, ipv6 or fqdn.
	DestinationPort float64 `json:"destinationport,omitempty"` // 16 bits integer. Throws an error when used with icmp. Defaults to 80 otherwise.
	Protocol        string  `json:"protocol"`                  // icmp, tcp, udp
	Count           float64 `json:"count,omitempty"`           // Number of tests
	Timeout         float64 `json:"timeout,omitempty"`         // Timeout for individual test. defaults to 5s.
}

type elements struct {
	MsLatencies  []float64 `json:"ms_latencies"` // response latency in milliseconds: 9999999 indicates timeout, -1 indicates unreachable, 0 general error.
	Reachable    bool      `json:"reachable"`
	ResolvedHost string    `json:"resolvedhost"` // Information about the ip:port being pinged
}

// Global variable latencies.
var latencies elements

func (r *Runner) ValidateParameters() (err error) {
	// Check if Protocol is a valid one that we support with this module
	if r.Parameters.Protocol != "icmp" &&
		r.Parameters.Protocol != "udp" &&
		r.Parameters.Protocol != "tcp" {
		return fmt.Errorf("Unsupported protocol requested")
	}

	// Check if protocol is icmp and a destinationport is specified
	if r.Parameters.Protocol == "icmp" && r.Parameters.DestinationPort != 0 {
		return fmt.Errorf("icmp does not take a destinationport parameter")
	} else if r.Parameters.Protocol != "icmp" && r.Parameters.DestinationPort == 0 {
		r.Parameters.DestinationPort = 80
	}

	// Check if port number is in valid range
	if !(r.Parameters.DestinationPort >= 0 && r.Parameters.DestinationPort <= 65535) {
		return fmt.Errorf("Invalid DestinationPort: %v", r.Parameters.DestinationPort)
	}

	ips, err := net.LookupHost(r.Parameters.Destination)
	ip := ""
	// Get ip based on destination.
	// if ip == nil, destination may not be a hostname.
	if err != nil {
		ip = r.Parameters.Destination
	} else {
		if len(ips) == 0 {
			return fmt.Errorf("FQDN does not resolve to any known ip")
		}
		ip = ips[0]
	}

	ip_parsed := net.ParseIP(ip)
	// If ParseIP failed, then ip is not a valid IP.
	if ip_parsed == nil {
		return fmt.Errorf("Invalid Destination: %v", ip)
	}

	if r.Parameters.Timeout == 0.0 {
		r.Parameters.Timeout = 5.0
	}

	// Default count to 3 if it is 0
	if r.Parameters.Count == 0.0 {
		r.Parameters.Count = 3
	}

	r.Parameters.Destination = ip
	return
}

func (r Runner) Ping() (err error) {
	var (
		icmpType icmp.Type
		network  string
	)

	if strings.Contains(r.Parameters.Destination, ":") {
		network = "ip6:ipv6-icmp"
		icmpType = ipv6.ICMPTypeEchoRequest
	} else {
		network = "ip4:icmp"
		icmpType = ipv4.ICMPTypeEcho
	}

	c, err := net.Dial(network, r.Parameters.Destination)
	if err != nil {
		return fmt.Errorf("Dial failed: %v", err)
	}
	c.SetDeadline(time.Now().Add(time.Duration(r.Parameters.Timeout) * time.Second))
	defer c.Close()

	// xid is the process ID.
	// Get process ID and make sure it fits in 16bits.
	xid := os.Getpid() & 0xffff
	// Sequence number of the packet.
	xseq := 0
	packet := icmp.Message{
		Type: icmpType, // Type of icmp message
		Code: 0,        // icmp query messages use code 0
		Body: &icmp.Echo{
			ID:   xid,  // Packet id
			Seq:  xseq, // Sequence number of the packet
			Data: bytes.Repeat([]byte("Ping!Ping!Ping!"), 3),
		},
	}

	wb, err := packet.Marshal(nil)

	if err != nil {
		return fmt.Errorf("Connection failed: %v", err)
	}

	if _, err := c.Write(wb); err != nil {
		return fmt.Errorf("Conn.Write Error: %v", err)
	}

	rb := make([]byte, 1500)

	if _, err := c.Read(rb); err != nil {
		// If Timeout, we return "Timeout"
		if e := err.(*net.OpError).Timeout(); e {
			return fmt.Errorf("Timeout")
		}
		return fmt.Errorf("Conn.Read failed: %v", err)
	}

	_, err = icmp.ParseMessage(icmpType.Protocol(), rb)
	if err != nil {
		return fmt.Errorf("ParseICMPMessage failed: %v", err)
	}

	return
}

func (r Runner) tcpPing() (err error) {
	// Make it ip:port format
	destination := r.Parameters.Destination + ":" + fmt.Sprintf("%d", int(r.Parameters.DestinationPort))

	// If dial Timeout it means that the handshake could not be completed and the port may not be open.
	if _, err := net.DialTimeout("tcp", destination, time.Duration(r.Parameters.Timeout)*time.Second); err != nil {
		// If Timeout, we return "Timeout"
		if e := err.(*net.OpError).Timeout(); e {
			return fmt.Errorf("Timeout")
		}
		return fmt.Errorf("Dial Error: %v", err)
	}
	return
}

// Open UDP ports are hard to determine.
// If there is a Timeout on Read it means that the port *maybe* open,
// however a connection refused either means that the port is filtered or closed.
func (r Runner) udpPing() (err error) {
	// Make it ip:port format
	destination := r.Parameters.Destination + ":" + fmt.Sprintf("%d", int(r.Parameters.DestinationPort))

	c, err := net.Dial("udp", destination)

	if err != nil {
		return fmt.Errorf("Dial Error: %v", err)
	}

	c.Write([]byte("Ping!Ping!Ping!"))
	c.SetReadDeadline(time.Now().Add(time.Duration(r.Parameters.Timeout) * time.Second))
	defer c.Close()

	rb := make([]byte, 1500)

	if _, err := c.Read(rb); err != nil {
		// If Timeout, we return "Timeout"
		if e := err.(*net.OpError).Timeout(); e {
			return fmt.Errorf("Timeout")
		}

		return fmt.Errorf("Read Error: %v", err.Error())
	}

	return nil
}

func (r Runner) Run(Args []byte) string {

	err := json.Unmarshal(Args, &r.Parameters)
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}

	err = r.ValidateParameters()
	if err != nil {
		r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		return r.buildResults()
	}

	latencies.Reachable = false
	latencies.ResolvedHost = fmt.Sprintf("%s:%v", r.Parameters.Destination, r.Parameters.DestinationPort)

	for i := 0; i < int(r.Parameters.Count); i += 1 {
		var err error
		// startTime for calculating the latency/RTT
		startTime := time.Now()

		switch r.Parameters.Protocol {
		case "icmp":
			err = r.Ping()
		case "tcp":
			err = r.tcpPing()
		case "udp":
			err = r.udpPing()
		}

		latency := time.Since(startTime).Seconds() * 1000

		if err != nil {
			if err.Error() == "Timeout" {
				latency = 9999999
			} else if strings.Contains(err.Error(), "connection refused") {
				latency = -1
			} else {
				r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("Fail on test#%d (%v)", i+1, err))
				latency = 0
			}
		}

		switch latency {
		case -1, 0:
			// do nothing
		case 9999999:
			// For udp, a timeout indicates that the port *maybe* open.
			if r.Parameters.Protocol == "udp" {
				latencies.Reachable = true
			}
		default:
			latencies.Reachable = true
		}
		latencies.MsLatencies = append(latencies.MsLatencies, latency)
	}

	r.Results.Success = true
	return r.buildResults()
}

func (r Runner) buildResults() string {
	r.Results.FoundAnything = latencies.Reachable
	r.Results.Elements = latencies
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}

func (r Runner) PrintResults(rawResults []byte, foundOnly bool) (prints []string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Print Error: %v", e)
		}
	}()

	var modres mig.ModuleResult
	err = json.Unmarshal(rawResults, modres)
	if err != nil {
		panic(err)
	}
	buf, err := json.Marshal(modres)
	if err != nil {
		panic(err)
	}

	var elms elements
	err = json.Unmarshal(buf, elms)
	if err != nil {
		panic(err)
	}

	resStr := fmt.Sprintf("Pinged host %s\nResults:\n", elms.ResolvedHost)

	if elms.Reachable == false {
		resStr = resStr + "  Host Unreachable\n"
		prints = append(prints, resStr)
		return
	}

	resStr = " Latencies: "
	for i := range elms.MsLatencies {
		resStr += fmt.Sprintf("%v ", elms.MsLatencies[i])
	}
	resStr += "\n"

	resStr = resStr + "  Host Reachable\n"
	prints = append(prints, resStr)

	return
}
