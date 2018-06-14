// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package modules

import (
	"bytes"
	"encoding/json"

	"mig.ninja/mig/modules/netstat"
)

// NetStat contains the configuration parameters required to run the netstat module.
type NetStat struct {
	LocalMACAddress          *string `json:"localMACAddress"`
	NeighborMACAddress       *string `json:"neighbourMACAddress"`
	LocalIPAddress           *string `json:"localIPAddress"`
	NeighborIPAddress        *string `json:"neighborIPAddress"`
	RemoteConnectedIPAddress *string `json:"remoteConnectedIPAddress"`
	ListeningPort            *uint16 `json:"listeningPort"`
	ResolveNamespaces        *bool   `json:"resolveNamespaces"`
}

func (module *NetStat) Name() string {
	return "netstat"
}

func (module *NetStat) ToParameters() (interface{}, error) {
	var localMAC, localIP, neighborMAC, neighborIP, connectedIP, listeningPort []string
	var searchNamespaces bool

	if module.LocalMACAddress != nil {
		localMAC = append(localMAC, *module.LocalMACAddress)
	}
	if module.NeighborMACAddress != nil {
		neighborMAC = append(neighborMAC, *module.NeighborMACAddress)
	}
	if module.LocalIPAddress != nil {
		localIP = append(localIP, *module.LocalIPAddress)
	}
	if module.NeighborIPAddress != nil {
		neighborIP = append(neighborIP, *module.NeighborIPAddress)
	}
	if module.RemoteConnectedIPAddress != nil {
		connectedIP = append(connectedIP, *module.RemoteConnectedIPAddress)
	}
	if module.ListeningPort != nil {
		listeningPort = append(listeningPort, string(*module.ListeningPort))
	}
	if module.ResolveNamespaces != nil {
		searchNamespaces = *module.ResolveNamespaces
	}

	params := netstat.Parameters{
		LocalMAC:         localMAC,
		LocalIP:          localIP,
		NeighborMAC:      neighborMAC,
		NeighborIP:       neighborIP,
		ConnectedIP:      connectedIP,
		ListeningPort:    listeningPort,
		SearchNamespaces: searchNamespaces,
	}

	return params, nil
}

func (module *NetStat) InitFromMap(jsonData map[string]interface{}) error {
	encoded, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(module)
}
