// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Sushant Dinesh <sushant.dinesh94@gmail.com>

package ping

import (
	"flag"
	"fmt"
	"strconv"
)

const help string = `
Ping module is used to check connection between an endpoint and a remote host. Currently supports for icmp, tcp and udp ping.
Please check: https://github.com/mozilla/mig/blob/master/src/mig/modules/ping/doc.rst for a complete documentation.

-d  <destination address>   Destination Address can be ipv4, ipv6 or FQDN of the destination host to which the connectivity has to be checked.
                            example: -d www.mozilla.org
							         -d 63.245.217.105

-dp <destination port>      Port on the destination host to which the connectivity has to be checked.
                            Leave this parameter blank if protocol is ICMP. For UDP and TCP destination port defaults to 80.
							example: -dp 53

-p  <protocol>              Protocol to use for the ping. This can be "icmp", "tcp" or "udp"
							example: -p udp

-c  <count>                 Number of times to repeat the test. Default 3.
                            example: -c 5

-t  <timeout>               Time (in seconds) to wait for response before the module has to timeout. Defaults to 5s.
                            example: -t 10
`

// ParamsParser implements a command line parameter parser for the ping module
func (r Runner) ParamsParser(args []string) (interface{}, error) {
	var (
		err            error
		d, dp, p, c, t flagParam
		fs             flag.FlagSet
	)
	if len(args) < 1 || args[0] == "" || args[0] == "help" {
		fmt.Println(help)
		return nil, fmt.Errorf("help printed")
	}
	fs.Init("file", flag.ContinueOnError)
	fs.Var(&d, "d", "see help")
	fs.Var(&dp, "dp", "see help")
	fs.Var(&p, "p", "see help")
	fs.Var(&c, "c", "see help")
	fs.Var(&t, "t", "see help")

	err = fs.Parse(args)
	if err != nil {
		return nil, err
	}

	port, err := strconv.ParseFloat(dp.String(), 64)
	count, err := strconv.ParseFloat(c.String(), 64)
	timeout, err := strconv.ParseFloat(t.String(), 64)

	if err != nil {
		fmt.Println(help)
		return nil, fmt.Errorf("help printed")
	}

	var pa params
	pa.Destination = d.String()
	pa.DestinationPort = port
	pa.Protocol = p.String()
	pa.Count = count
	pa.Timeout = timeout
	return pa, nil
}

type flagParam []string

func (f *flagParam) String() string {
	return fmt.Sprint([]string(*f))
}

func (f *flagParam) Set(value string) error {
	*f = append(*f, value)
	return nil
}
