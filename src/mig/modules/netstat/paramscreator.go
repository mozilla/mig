// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package netstat

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const help string = `To add search parameters, use the following syntax:
localmac <regex>	search for mac addresses on the local interfaces that match <regex>
			example: > localmac ^8c:70:[0-9a-f]

neighbormac <regex>	search for neighbors mac addresses in the ARP table that match <regex>
			example: > neighbormac ^8c:70:[0-9a-f]

localip <ip|cidr>	search for ip addresses on the local interfaces that match <cidr>
			if a cidr is specified, return all ip addresses included in it
			example: > localip 10.0.0.0/8
				 > localip 2001:db8::/32

neighborip <ip|cidr>	search for neighbors ip address in the ARP table that match <cidr>
			if a cidr is specified, return all ip addresses included in it
			example: > neighborip 10.1.2.3
				 > neighborip fdda:5cc1:23:4::1f

connectedip <ip|cidr>	search for connections with the given IP address or within the given <cidr>
			return localip:localport remoteip:remoteport
			example: > connectedip 80.70.60.0/24
				 > connectedip 2001:0db8:0123:4567:89ab:cdef:1234:0/116

listeningport <port>	search for an open socket on the local system listening on <port>, tcp and udp
			example: > listeningport 443

When done, enter 'done'`

// ParamsCreator implements an interactive parameters creation interface, which
// receives user input,  stores it into a Parameters structure, validates it,
// and returns that structure as an interface. It is mainly used by the MIG Console
func (r Runner) ParamsCreator() (interface{}, error) {
	fmt.Println("initializing netstat parameters creation")
	var err error
	var p params
	fmt.Printf("%s\n", help)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("search> ")
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			fmt.Println("Invalid input. Try again")
			continue
		}
		input := scanner.Text()
		if input == "done" {
			break
		}
		if input == "help" {
			fmt.Printf("%s\n", help)
			continue
		}
		arr := strings.SplitN(input, " ", 2)
		if len(arr) != 2 {
			fmt.Printf("Invalid input format!\n%s\n", help)
			continue
		}
		checkType := arr[0]
		checkValue := arr[1]
		switch checkType {
		case "localmac":
			err = validateMAC(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.LocalMAC = append(p.LocalMAC, checkValue)
			fmt.Printf("Stored %s '%s'. Enter another search or 'done'.\n", checkType, checkValue)
		case "neighbormac":
			err = validateMAC(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.NeighborMAC = append(p.NeighborMAC, checkValue)
			fmt.Printf("Stored %s '%s'. Enter another search or 'done'.\n", checkType, checkValue)
		case "localip":
			err = validateIP(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.LocalIP = append(p.LocalIP, checkValue)
			fmt.Printf("Stored %s '%s'. Enter another search or 'done'.\n", checkType, checkValue)
		case "neighborip":
			err = validateIP(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.NeighborIP = append(p.NeighborIP, checkValue)
			fmt.Printf("Stored %s '%s'. Enter another search or 'done'.\n", checkType, checkValue)
		case "connectedip":
			err = validateIP(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.ConnectedIP = append(p.ConnectedIP, checkValue)
			fmt.Printf("Stored %s '%s'. Enter another search or 'done'.\n", checkType, checkValue)
		case "listeningport":
			err = validatePort(checkValue)
			if err != nil {
				fmt.Printf("ERROR: %v\nTry again.\n", err)
				continue
			}
			p.ListeningPort = append(p.ListeningPort, checkValue)
			fmt.Printf("Stored %s '%s'. Enter another search or 'done'.\n", checkType, checkValue)
		default:
			fmt.Printf("Invalid method!\nTry 'help'\n")
			continue
		}
	}
	return p, nil
}
