// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package gozdef

type ExternalEvent interface {
	Validate() error
}

// An HostAssetHint describes information about a host that can be used
// to correlate asset information in MozDef. This is primarily used by MIG
type HostAssetHint struct {
	Type      string   `json:"type"`
	Name      string   `json:"name"`
	IPv4      []string `json:"ipv4"`
	IPv6      []string `json:"ipv6"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	Ident     string   `json:"ident"`
	Init      string   `json:"init"`
	IsProxied bool     `json:"isproxied"`
	Operator  string   `json:"operator"`
	Team      string   `json:"team"`
}

// a ComplianceItem measures the compliance of a target
// with particular requirement. The item must be send to mozdef
// in the details of a regular Event.
type ComplianceItem struct {
	Utctimestamp string           `json:"utctimestamp"`
	Target       string           `json:"target"`
	Policy       CompliancePolicy `json:"policy"`
	Check        ComplianceCheck  `json:"check"`
	Compliance   bool             `json:"compliance"`
	Link         string           `json:"link"`
	Tags         interface{}      `json:"tags"`
}

type CompliancePolicy struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Level string `json:"level"`
}

type ComplianceCheck struct {
	Ref         string         `json:"ref"`
	Description string         `json:"description"`
	Name        string         `json:"name"`
	Location    string         `json:"location"`
	Test        ComplianceTest `json:"test"`
}

type ComplianceTest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
