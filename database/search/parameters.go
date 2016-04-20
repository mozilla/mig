// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package search /* import "mig.ninja/mig/database/search" */

import (
	"fmt"
	"strings"
	"time"
)

// SearchParameters contains fields used to perform database searches
type Parameters struct {
	ActionID         string    `json:"actionid"`
	ActionName       string    `json:"actionname"`
	After            time.Time `json:"after"`
	AgentID          string    `json:"agentid"`
	AgentName        string    `json:"agentname"`
	AgentVersion     string    `json:"agentversion"`
	Before           time.Time `json:"before"`
	CommandID        string    `json:"commandid"`
	FoundAnything    bool      `json:"foundanything"`
	InvestigatorID   string    `json:"investigatorid"`
	InvestigatorName string    `json:"investigatorname"`
	Limit            float64   `json:"limit"`
	LoaderID         string    `json:"loaderid"`
	LoaderName       string    `json:"loadername"`
	ManifestID       string    `json:"manifestid"`
	ManifestName     string    `json:"manifestname"`
	Offset           float64   `json:"offset"`
	Report           string    `json:"report"`
	Status           string    `json:"status"`
	Target           string    `json:"target"`
	ThreatFamily     string    `json:"threatfamily"`
	Type             string    `json:"type"`
}

// by default, search all records 10 years prior and after today
const DefaultWindow time.Duration = 39600 * time.Hour

// NewParameters initializes search parameters
func NewParameters() (p Parameters) {
	p.ActionID = "∞"
	p.ActionName = "%"
	p.After = time.Now().Add(-DefaultWindow).UTC()
	p.AgentID = "∞"
	p.AgentName = "%"
	p.AgentVersion = "%"
	p.Before = time.Now().Add(DefaultWindow).UTC()
	p.CommandID = "∞"
	p.InvestigatorID = "∞"
	p.InvestigatorName = "%"
	p.Limit = 100
	p.LoaderID = "∞"
	p.LoaderName = "%"
	p.ManifestID = "∞"
	p.ManifestName = "%"
	p.Offset = 0
	p.Status = "%"
	p.ThreatFamily = "%"
	p.Type = "action"
	return
}

// String() returns a query string with the current search parameters
func (p Parameters) String() (query string) {
	query = fmt.Sprintf("type=%s&after=%s&before=%s", p.Type, p.After.Format(time.RFC3339), p.Before.Format(time.RFC3339))
	if p.ActionID != "∞" {
		query += fmt.Sprintf("&actionid=%s", p.ActionID)
	}
	if p.ActionName != "%" {
		query += fmt.Sprintf("&actionname=%s", p.ActionName)
	}
	if p.AgentID != "∞" {
		query += fmt.Sprintf("&agentid=%s", p.AgentID)
	}
	if p.AgentName != "%" {
		query += fmt.Sprintf("&agentname=%s", p.AgentName)
	}
	if p.AgentVersion != "%" {
		query += fmt.Sprintf("&agentversion=%s", p.AgentVersion)
	}
	if p.CommandID != "∞" {
		query += fmt.Sprintf("&commandid=%s", p.CommandID)
	}
	if p.InvestigatorID != "∞" {
		query += fmt.Sprintf("&investigatorid=%s", p.InvestigatorID)
	}
	if p.InvestigatorName != "%" {
		query += fmt.Sprintf("&investigatorname=%s", p.InvestigatorName)
	}
	if p.LoaderID != "∞" {
		query += fmt.Sprintf("&loaderid=%s", p.LoaderID)
	}
	if p.LoaderName != "%" {
		query += fmt.Sprintf("&loadername=%s", p.LoaderName)
	}
	if p.ManifestName != "%" {
		query += fmt.Sprintf("&manifestname=%s", p.ManifestName)
	}
	if p.ManifestID != "∞" {
		query += fmt.Sprintf("&manifestid=%s", p.ManifestID)
	}
	query += fmt.Sprintf("&limit=%.0f", p.Limit)
	if p.Offset != 0 {
		query += fmt.Sprintf("&offset=%.0f", p.Offset)
	}
	if p.Status != "%" {
		query += fmt.Sprintf("&status=%s", p.Status)
	}
	if p.ThreatFamily != "%" {
		query += fmt.Sprintf("&threatfamily=%s", p.ThreatFamily)
	}
	// urlencode % and * wildcard characters
	query = strings.Replace(query, "%", "%25", -1)
	query = strings.Replace(query, "*", "%25", -1)
	// replace + character with a wildcard
	query = strings.Replace(query, "+", "%25", -1)
	return
}
