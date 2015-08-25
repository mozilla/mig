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

type VulnEvent struct {
	Description  string    `json:"description"`
	UTCTimestamp time.Time `json:"utctimestamp"`
	SourceName   string    `json:"sourcename"`
	Asset        VulnAsset `json:"asset"`
	Vuln         VulnVuln  `json:"vuln"`
	OS           string    `json:"os"`
}

type VulnAsset struct {
	AssetID   int    `json:"assetid"`
	IPv4      string `json:"ipv4address"`
	Hostname  string `json:"hostname"`
	MAC       string `json:"macaddress"`
	Autogroup string `json:"autogroup"`
	Operator  string `json:"operator"`
}

type VulnVuln struct {
	Status        string   `json:"status"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Proof         string   `json:"proof"`
	ImpactLabel   string   `json:"impact_label"`
	KnownExp      bool     `json:"known_exploits"`
	KnownMal      bool     `json:"known_malware"`
	Age           float64  `json:"age_days"`
	DiscoveryTime int      `json:"discovery_time"`
	PatchIn       float64  `json:"patch_in"`
	VulnID        string   `json:"vulnid"`
	CVE           []string `json:"cves"`
	CVEText       []string `json:"cvetext"`
	CVSS          float64  `json:"cvss"`
	CVSSVector    VulnCVSS `json:"cvss_vector"`
}

type VulnCVSS struct {
	AccessComplexity      string `json:"access_complexity"`
	AvailabilityImpact    string `json:"availability_impact"`
	ConfidentialityImpact string `json:"confidentiality_impact"`
	AccessVector          string `json:"access_vector"`
	Authentication        string `json:"authentication"`
}

func NewVulnEvent() (e VulnEvent, err error) {
	e.UTCTimestamp = time.Now().UTC()
	return
}

// Validate verifies that an event is formatted correctly
func (e VulnEvent) Validate() error {
	if e.SourceName == "" {
		return fmt.Errorf("must set SourceName in event")
	}
	if e.Asset.AssetID == 0 {
		return fmt.Errorf("must set AssetID in event")
	}
	if e.Vuln.VulnID == "" {
		return fmt.Errorf("must set VulnID in event")
	}
	return nil
}
