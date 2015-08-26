// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"mig.ninja/mig"
	"net"
)

type CommandLocation struct {
	Endpoint      string  `json:"endpoint"`
	CommandID     float64 `json:"commandid"`
	ActionID      float64 `json:"actionid"`
	FoundAnything bool    `json:"foundanything"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	City          string  `json:"city"`
	Country       string  `json:"country"`
}

func commandsToGeolocations(commands []mig.Command) (items []CommandLocation, err error) {
	if ctx.MaxMind.r == nil {
		return items, fmt.Errorf("maxmind database not initialized")
	}
	var cl CommandLocation
	for _, cmd := range commands {
		if cmd.Agent.Env.PublicIP == "" {
			continue
		}
		record, err := ctx.MaxMind.r.City(net.ParseIP(cmd.Agent.Env.PublicIP))
		if err != nil {
			return items, err
		}
		cl.Latitude = record.Location.Latitude
		cl.Longitude = record.Location.Longitude
		cl.City = record.City.Names["en"]
		cl.Country = record.Country.Names["en"]
		cl.Endpoint = cmd.Agent.Name
		cl.CommandID = cmd.ID
		cl.ActionID = cmd.Action.ID
		for _, r := range cmd.Results {
			if r.FoundAnything {
				cl.FoundAnything = true
			}
		}
		items = append(items, cl)
	}
	return
}
