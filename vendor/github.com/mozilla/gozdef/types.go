// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package gozdef

// ExternalEvent provides a generalized interface that all event types
// must provide
type ExternalEvent interface {
	Validate() error
}

// ComplianceItem measures the compliance of a target
// with particular requirement. The item must be send to mozdef
// in the details of a regular Event.
type ComplianceItem struct {
	Utctimestamp string            `json:"utctimestamp"`
	Target       string            `json:"target"`
	Compliance   bool              `json:"compliance"`
	Link         string            `json:"link"`
	Tags         map[string]string `json:"tags"`

	Policy struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Level string `json:"level"`
	} `json:"policy"`

	Check struct {
		Ref         string `json:"ref"`
		Description string `json:"description"`
		Name        string `json:"name"`
		Location    string `json:"location"`

		Test struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"test"`
	} `json:"check"`
}
