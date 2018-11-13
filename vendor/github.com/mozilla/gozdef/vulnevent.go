// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package gozdef

import (
	"fmt"
	"time"
)

// MozDef vulnerability event handling

// VulnEvent describes a vulnerability event
type VulnEvent struct {
	Description        string     `json:"description"`
	UTCTimestamp       time.Time  `json:"utctimestamp"`
	SourceName         string     `json:"sourcename"`
	CredentialedChecks bool       `json:"credentialed_checks"`
	Vuln               []VulnVuln `json:"vulnerabilities"`
	ExemptVuln         []VulnVuln `json:"exempt_vulnerabilities"`
	Version            int        `json:"version"`
	Zone               string     `json:"zone"`

	Asset struct {
		IPAddress string `json:"ipaddress"`
		Hostname  string `json:"hostname"`
		OS        string `json:"os"`
		Owner     struct {
			Operator string `json:"operator"`
			Team     string `json:"team"`
			V2Bkey   string `json:"v2bkey"`
		} `json:"owner"`
	} `json:"asset"`
}

// VulnVuln describes individual vulnerabilities for inclusion in a vulnerability
// event
type VulnVuln struct {
	Risk                string   `json:"risk"`
	Link                string   `json:"link"`
	CVE                 string   `json:"cve"`
	CVSS                string   `json:"cvss"`
	Name                string   `json:"name"`
	Packages            []string `json:"vulnerable_packages"`
	LikelihoodIndicator string   `json:"likelihood_indicator"`
}

// NewVulnEvent initializes a new VulnEvent that can be populated and submitted
// to MozDef
func NewVulnEvent() (e VulnEvent, err error) {
	e.UTCTimestamp = time.Now().UTC()
	e.Version = 2
	return
}

// Validate verifies that an event is formatted correctly
func (e VulnEvent) Validate() error {
	if e.SourceName == "" {
		return fmt.Errorf("must set SourceName in event")
	}
	if e.Asset.IPAddress == "" && e.Asset.Hostname == "" {
		return fmt.Errorf("must set IPAddress or Hostname in event")
	}
	return nil
}
