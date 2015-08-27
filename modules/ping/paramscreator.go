// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Sushant Dinesh <sushant.dinesh94@gmail.com>

/* The Ping module implements icmp, tcp and udp pings/
Usage doc is online at http://mig.mozilla.org/doc/module_ping.html
*/
package ping /* import "mig.ninja/mig/modules/ping" */

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func printHelp(isCmd bool) {
	dash := ""
	if isCmd {
		dash = "-"
	}
	fmt.Printf(`Ping module checks connectivity between an endpoint and a remote host. It supports
icmp, tcp and udp ping. See doc at http://mig.mozilla.org/doc/module_ping.html

%sd <ip/fqdn>	Destination Address can be ipv4, ipv6 or FQDN
		example: %sd www.mozilla.org
			 %sd 63.245.217.105

%sdp <port>	For TCP and UDP, specifies the port to test connectivity to
		example: %sdp 53

%sp <protocol>	Protocol to use for the ping. This can be "icmp", "tcp" or "udp"
		example: %sp udp

%sc <count>	Number of ping/connection attempts. Defaults to 3.
		example: %sc 5

%st <timeout>	Connection timeout in seconds. Defaults to 5.
		example: %st 10
`, dash, dash, dash, dash, dash, dash, dash, dash, dash, dash, dash)

	return
}

// ParamsParser implements a command line parameter parser for the ping module
func (r *run) ParamsParser(args []string) (interface{}, error) {
	var (
		err      error
		pa       params
		d, p     string
		dp, c, t float64
		fs       flag.FlagSet
	)
	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		printHelp(true)
		return nil, fmt.Errorf("help printed")
	}
	fs.Init("ping", flag.ContinueOnError)
	fs.StringVar(&d, "d", "www.google.com", "see help")
	fs.Float64Var(&dp, "dp", -1, "see help")
	fs.StringVar(&p, "p", "icmp", "see help")
	fs.Float64Var(&c, "c", 3, "see help")
	fs.Float64Var(&t, "t", 5, "see help")

	err = fs.Parse(args)
	if err != nil {
		return nil, err
	}
	pa.Destination = d
	pa.DestinationPort = dp
	pa.Protocol = p
	pa.Count = c
	pa.Timeout = t
	r.Parameters = pa
	return pa, r.ValidateParameters()
}

// ParamsCreator implements an interactive interface for the console
func (r *run) ParamsCreator() (interface{}, error) {
	var (
		err error
		p   params
	)
	printHelp(false)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("ping> ")
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			fmt.Println("Invalid input. Try again")
			continue
		}
		input := scanner.Text()
		splitted := strings.SplitN(input, " ", 2)
		if splitted[0] != "" && splitted[0] != "help" && splitted[0] != "done" && len(splitted) != 2 {
			fmt.Println("invalid input format")
			continue
		}
		switch splitted[0] {
		case "d":
			p.Destination = splitted[1]
		case "dp":
			p.DestinationPort, err = strconv.ParseFloat(splitted[1], 64)
			if err != nil {
				fmt.Println("invalid destination port: %v", err)
				continue
			}
		case "p":
			p.Protocol = splitted[1]
		case "c":
			p.Count, err = strconv.ParseFloat(splitted[1], 64)
			if err != nil {
				fmt.Println("invalid count: %v", err)
				continue
			}
		case "t":
			p.Timeout, err = strconv.ParseFloat(splitted[1], 64)
			if err != nil {
				fmt.Println("invalid timeout: %v", err)
				continue
			}
		case "help":
			printHelp(false)
		case "done":
			goto done
		default:
			fmt.Println("enter 'done' to exit")
		}
	}
done:
	r.Parameters = p
	return p, r.ValidateParameters()
}
