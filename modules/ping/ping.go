// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributors: Sushant Dinesh sushant.dinesh94@gmail.com [:sushant94]
//               Julien Vehent jvehent@mozilla.com [:ulfr]

// ping module is used to check the connection between an endpoint
// and a destination host.

package ping /* import "mig.ninja/mig/modules/ping" */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"io"
	"mig.ninja/mig/modules"
	"net"
	"os"
	"strings"
	"time"
)

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("ping", new(module))
}

type run struct {
	Parameters params
	Results    modules.Result
}

type params struct {
	Destination     string  `json:"destination"`               // ipv4, ipv6 or fqdn.
	DestinationPort float64 `json:"destinationport,omitempty"` // 16 bits integer. Throws an error when used with icmp. Defaults to 80 otherwise.
	Protocol        string  `json:"protocol"`                  // icmp, tcp, udp
	Count           float64 `json:"count,omitempty"`           // Number of tests
	Timeout         float64 `json:"timeout,omitempty"`         // Timeout for individual test. defaults to 5s.
	ipDest          string
}

type elements struct {
	Latencies    []float64 `json:"latencies"`    // response latency in milliseconds: 9999999 indicates timeout, -1 indicates unreachable, 0 general error.
	Protocol     string    `json:"protocol"`     // icmp, tcp, udp
	ResolvedHost string    `json:"resolvedhost"` // Information about the ip:port being pinged
	Failures     []string  `json:"failures"`     // ping failures, soft errors
}

const (
	E_ConnRefused = "connection refused"
	E_Timeout     = "connection timed out"
)

func (r *run) Run(in io.Reader) (out string) {
	var (
		err error
		el  elements
	)
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			buf, _ := json.Marshal(r.Results)
			out = string(buf[:])
		}
	}()
	err = modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	el.ResolvedHost = r.Parameters.Destination
	if r.Parameters.Protocol == "udp" || r.Parameters.Protocol == "tcp" {
		el.ResolvedHost += fmt.Sprintf(":%.0f", r.Parameters.DestinationPort)
	}
	el.ResolvedHost += " (" + r.Parameters.ipDest + ")"
	el.Protocol = r.Parameters.Protocol
	for i := 0; i < int(r.Parameters.Count); i += 1 {
		var err error
		// startTime for calculating the latency/RTT
		startTime := time.Now()

		switch r.Parameters.Protocol {
		case "icmp":
			err = r.pingIcmp()
		case "tcp":
			err = r.pingTcp()
		case "udp":
			err = r.pingUdp()
		}
		// store the time elapsed before processing potential errors
		latency := time.Since(startTime).Seconds() * 1000

		// evaluate potential ping failures
		if err != nil {
			switch err.Error() {
			case E_Timeout:
				latency = 9999999
			case E_ConnRefused:
				latency = -1
			default:
				el.Failures = append(el.Failures, fmt.Sprintf("ping #%d failed with error: %v", i+1, err))
				latency = 0
			}
		}

		switch latency {
		case -1, 0:
			// do nothing
		case 9999999:
			// For udp, a timeout indicates that the port *maybe* open.
			if r.Parameters.Protocol == "udp" {
				r.Results.FoundAnything = true
			}
		default:
			r.Results.FoundAnything = true
		}
		el.Latencies = append(el.Latencies, latency)

		// sleep 100 milliseconds between pings to prevent floods
		time.Sleep(100 * time.Millisecond)
	}
	return r.buildResults(el)
}

func (r *run) ValidateParameters() (err error) {
	// check if Protocol is a valid one that we support with this module
	switch r.Parameters.Protocol {
	case "icmp", "udp", "tcp":
		break
	default:
		return fmt.Errorf("%s is not a supported ping protocol", r.Parameters.Protocol)
	}
	// tcp and udp pings must have a destination port
	if r.Parameters.Protocol != "icmp" && (r.Parameters.DestinationPort < 0 || r.Parameters.DestinationPort > 65535) {
		return fmt.Errorf("%s ping requires a valid destination port between 0 and 65535, got %.0f",
			r.Parameters.Protocol, r.Parameters.DestinationPort)
	}
	// if the destination is a FQDN, resolve it and take the first IP returned as the dest
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

	// check the format of the destination IP
	ip_parsed := net.ParseIP(ip)
	if ip_parsed == nil {
		return fmt.Errorf("destination IP is invalid: %v", ip)
	}
	r.Parameters.ipDest = ip

	// if timeout is not set, default to 5 seconds
	if r.Parameters.Timeout == 0.0 {
		r.Parameters.Timeout = 5.0
	}

	// if count of pings is not set, default to 3
	if r.Parameters.Count == 0.0 {
		r.Parameters.Count = 3
	}

	return
}

// pingIcmp performs a ping to a destination. It select between ipv4 or ipv6 ping based
// on the format of the destination ip.
func (r *run) pingIcmp() (err error) {
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
		// If connection timed out, we return E_Timeout
		if e := err.(*net.OpError).Timeout(); e {
			return fmt.Errorf(E_Timeout)
		}
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf(E_ConnRefused)
		}
		return fmt.Errorf("Conn.Read failed: %v", err)
	}

	_, err = icmp.ParseMessage(icmpType.Protocol(), rb)
	if err != nil {
		return fmt.Errorf("ParseICMPMessage failed: %v", err)
	}

	return
}

// pingTcp performs a straighfoward connection attempt on a destination ip:port and returns
// an error if the attempt failed
func (r *run) pingTcp() (err error) {
	conn, err := net.DialTimeout("tcp",
		fmt.Sprintf("%s:%d", r.Parameters.Destination, int(r.Parameters.DestinationPort)),
		time.Duration(r.Parameters.Timeout)*time.Second)
	defer conn.Close()
	if err != nil {
		// If connection timed out, we return E_Timeout
		if e := err.(*net.OpError).Timeout(); e {
			return fmt.Errorf(E_Timeout)
		}
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf(E_ConnRefused)
		}
		return fmt.Errorf("Dial Error: %v", err)
	}
	return
}

// pingUdp sends a UDP packet to a destination ip:port to determine if it is open or closed.
// Because UDP does not reply to connection requests, a lack of response may indicate that the
// port is open, or that the packet got dropped. We chose to be optimistic and treat lack of
// response (connection timeout) as an open port.
func (r *run) pingUdp() (err error) {
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
		// If connection timed out, we return E_Timeout
		if e := err.(*net.OpError).Timeout(); e {
			return fmt.Errorf(E_Timeout)
		}
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf(E_ConnRefused)
		}
		return fmt.Errorf("Read Error: %v", err.Error())
	}
	return nil
}

func (r *run) buildResults(el elements) string {
	r.Results.Elements = el
	if len(r.Results.Errors) == 0 {
		r.Results.Success = true
	}
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}

func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var el elements
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Print Error: %v", e)
		}
	}()

	err = result.GetElements(&el)
	if err != nil {
		panic(err)
	}
	if result.FoundAnything {
		prints = append(prints,
			fmt.Sprintf("%s ping of %s succeeded. Target is reachable.",
				el.Protocol,
				el.ResolvedHost,
			),
		)
	}
	// if we don't care about results where the target was not reachable, stop here
	if foundOnly {
		return
	}
	if !result.FoundAnything {
		prints = append(prints,
			fmt.Sprintf("%s ping of %s failed. Target is no reachable.",
				el.Protocol,
				el.ResolvedHost,
			),
		)
	}
	for i, lat := range el.Latencies {
		switch lat {
		case -1:
			prints = append(prints, fmt.Sprintf("ping #%d failed, target was unreachable", i+1))
		case 0:
			prints = append(prints, fmt.Sprintf("ping #%d failed, reason unknown", i+1))
		case 9999999:
			if el.Protocol == "udp" {
				prints = append(prints, fmt.Sprintf("ping #%d may have succeeded (no udp response)", i+1))
			} else {
				prints = append(prints, fmt.Sprintf("ping #%d failed, connection timed out", i+1))
			}
		default:
			prints = append(prints, fmt.Sprintf("ping #%d succeeded in %.0fms", i+1, lat))
		}
	}
	return
}
