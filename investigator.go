// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"fmt"
	"time"
)

type Investigator struct {
	ID             float64   `json:"id,omitempty"`
	Name           string    `json:"name"`
	PGPFingerprint string    `json:"pgpfingerprint"`
	PublicKey      []byte    `json:"publickey,omitempty"`
	PrivateKey     []byte    `json:"privatekey,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdat"`
	LastModified   time.Time `json:"lastmodified"`

	Permissions InvestigatorPerms `json:"permissions"`
}

// Check an investigator has given permission pv
func (i *Investigator) CheckPermission(pv int64) bool {
	switch pv {
	case PermSearch:
		return i.Permissions.Search
	case PermAction:
		return i.Permissions.Action
	case PermActionCreate:
		return i.Permissions.ActionCreate
	case PermCommand:
		return i.Permissions.Command
	case PermAgent:
		return i.Permissions.Agent
	case PermDashboard:
		return i.Permissions.Dashboard
	case PermLoader:
		return i.Permissions.Loader
	case PermLoaderStatus:
		return i.Permissions.LoaderStatus
	case PermLoaderExpect:
		return i.Permissions.LoaderExpect
	case PermLoaderKey:
		return i.Permissions.LoaderKey
	case PermLoaderNew:
		return i.Permissions.LoaderNew
	case PermManifest:
		return i.Permissions.Manifest
	case PermManifestSign:
		return i.Permissions.ManifestSign
	case PermManifestNew:
		return i.Permissions.ManifestNew
	case PermManifestLoaders:
		return i.Permissions.ManifestLoaders
	case PermInvestigator:
		return i.Permissions.Investigator
	case PermInvestigatorCreate:
		return i.Permissions.InvestigatorCreate
	case PermInvestigatorUpdate:
		return i.Permissions.InvestigatorUpdate
	}
	return false
}

// Describes permissions assigned to an investigator
type InvestigatorPerms struct {
	Search             bool `json:"search"`
	Action             bool `json:"action"`
	ActionCreate       bool `json:"action_create"`
	Command            bool `json:"command"`
	Agent              bool `json:"agent"`
	Dashboard          bool `json:"dashboard"`
	Loader             bool `json:"loader"`
	LoaderStatus       bool `json:"loader_status"`
	LoaderExpect       bool `json:"loader_expect"`
	LoaderKey          bool `json:"loader_key"`
	LoaderNew          bool `json:"loader_new"`
	Manifest           bool `json:"manifest"`
	ManifestSign       bool `json:"manifest_sign"`
	ManifestStatus     bool `json:"manifest_status"`
	ManifestNew        bool `json:"manifest_new"`
	ManifestLoaders    bool `json:"manifest_loaders"`
	Investigator       bool `json:"investigator"`
	InvestigatorCreate bool `json:"investigator_create"`
	InvestigatorUpdate bool `json:"investigator_update"`
}

// Convert a permission bit mask into a boolean permission set
func (ip *InvestigatorPerms) FromMask(mask int64) {
	if (mask & PermSearch) != 0 {
		ip.Search = true
	}
	if (mask & PermAction) != 0 {
		ip.Action = true
	}
	if (mask & PermActionCreate) != 0 {
		ip.ActionCreate = true
	}
	if (mask & PermCommand) != 0 {
		ip.Command = true
	}
	if (mask & PermAgent) != 0 {
		ip.Agent = true
	}
	if (mask & PermDashboard) != 0 {
		ip.Dashboard = true
	}
	if (mask & PermLoader) != 0 {
		ip.Loader = true
	}
	if (mask & PermLoaderStatus) != 0 {
		ip.LoaderStatus = true
	}
	if (mask & PermLoaderExpect) != 0 {
		ip.LoaderExpect = true
	}
	if (mask & PermLoaderKey) != 0 {
		ip.LoaderKey = true
	}
	if (mask & PermLoaderNew) != 0 {
		ip.LoaderNew = true
	}
	if (mask & PermManifest) != 0 {
		ip.Manifest = true
	}
	if (mask & PermManifestSign) != 0 {
		ip.ManifestSign = true
	}
	if (mask & PermManifestNew) != 0 {
		ip.ManifestNew = true
	}
	if (mask & PermManifestStatus) != 0 {
		ip.ManifestStatus = true
	}
	if (mask & PermManifestLoaders) != 0 {
		ip.ManifestLoaders = true
	}
	if (mask & PermInvestigator) != 0 {
		ip.Investigator = true
	}
	if (mask & PermInvestigatorCreate) != 0 {
		ip.InvestigatorCreate = true
	}
	if (mask & PermInvestigatorUpdate) != 0 {
		ip.InvestigatorUpdate = true
	}
}

// Convert a boolean permission set to a permission bit mask
func (ip *InvestigatorPerms) ToMask() (ret int64) {
	if ip.Search {
		ret |= PermSearch
	}
	if ip.Action {
		ret |= PermAction
	}
	if ip.ActionCreate {
		ret |= PermActionCreate
	}
	if ip.Command {
		ret |= PermCommand
	}
	if ip.Agent {
		ret |= PermAgent
	}
	if ip.Dashboard {
		ret |= PermDashboard
	}
	if ip.Loader {
		ret |= PermLoader
	}
	if ip.LoaderStatus {
		ret |= PermLoaderStatus
	}
	if ip.LoaderExpect {
		ret |= PermLoaderExpect
	}
	if ip.LoaderKey {
		ret |= PermLoaderKey
	}
	if ip.LoaderNew {
		ret |= PermLoaderNew
	}
	if ip.Manifest {
		ret |= PermManifest
	}
	if ip.ManifestSign {
		ret |= PermManifestSign
	}
	if ip.ManifestNew {
		ret |= PermManifestNew
	}
	if ip.ManifestStatus {
		ret |= PermManifestStatus
	}
	if ip.ManifestLoaders {
		ret |= PermManifestLoaders
	}
	if ip.Investigator {
		ret |= PermInvestigator
	}
	if ip.InvestigatorCreate {
		ret |= PermInvestigatorCreate
	}
	if ip.InvestigatorUpdate {
		ret |= PermInvestigatorUpdate
	}
	return ret
}

// Convert an existing boolean permission set to a descriptive string, used
// primarily in mig-console for summarizing permissions assigned to an
// investigator
func (ip *InvestigatorPerms) ToDescriptive() string {
	cf := func(want int64, have int64) (bool, int64) {
		var (
			wantcnt, havecnt int64
			n                uint = 64
			sb               uint
		)
		for sb = 0; sb < n; sb++ {
			wantcnt += (want >> sb) & 0x01
		}
		for sb = 0; sb < n; sb++ {
			havecnt += ((have & want) >> sb) & 0x01
		}
		return (havecnt == wantcnt), havecnt
	}
	ret := ""
	// Compare the permissions applied to the investigator to the various
	// permission sets; if the user has the full set of permissions from the
	// set we note that
	tv := InvestigatorPerms{}
	tv.DefaultSet()
	fs, part := cf(tv.ToMask(), ip.ToMask())
	if fs {
		ret = "Default"
	} else if part > 0 {
		ret = "Default(partial)"
	}

	av := ""
	tv = InvestigatorPerms{}
	tv.AdminSet()
	fs, part = cf(tv.ToMask(), ip.ToMask())
	if fs {
		av = "PermAdmin"
	} else if part > 0 {
		av = "PermAdmin(partial)"
	}
	if ret != "" && av != "" {
		ret += ","
	}
	ret += av

	av = ""
	tv = InvestigatorPerms{}
	tv.LoaderSet()
	fs, part = cf(tv.ToMask(), ip.ToMask())
	if fs {
		av = "PermLoader"
	} else if part > 0 {
		av = "PermLoader(partial)"
	}
	if ret != "" && av != "" {
		ret += ","
	}
	ret += av

	av = ""
	tv = InvestigatorPerms{}
	tv.ManifestSet()
	fs, part = cf(tv.ToMask(), ip.ToMask())
	if fs {
		av = "PermManifest"
	} else if part > 0 {
		av = "PermManifest(partial)"
	}
	if ret != "" && av != "" {
		ret += ","
	}
	ret += av
	return ret
}

// Describe permission sets that can be applied; note default is omitted as this
// is currently always applied
var PermSets = []string{"PermManifest", "PermLoader", "PermAdmin"}

// Apply permission sets in slice sl to the investigator
func (ip *InvestigatorPerms) FromSetList(sl []string) error {
	for _, x := range sl {
		switch x {
		case "PermManifest":
			ip.ManifestSet()
		case "PermLoader":
			ip.LoaderSet()
		case "PermAdmin":
			ip.AdminSet()
		default:
			return fmt.Errorf("invalid permission %q", x)
		}
	}
	return nil
}

// Set a default set of permissions on the investigator
func (ip *InvestigatorPerms) DefaultSet() {
	ip.Search = true
	ip.Action = true
	ip.ActionCreate = true
	ip.Command = true
	ip.Agent = true
	ip.Dashboard = true
}

// Set manifest related permissions on the investigator
func (ip *InvestigatorPerms) ManifestSet() {
	ip.Manifest = true
	ip.ManifestSign = true
	ip.ManifestNew = true
	ip.ManifestStatus = true
	ip.ManifestLoaders = true
}

// Set loader related permissions on the investigator
func (ip *InvestigatorPerms) LoaderSet() {
	ip.Loader = true
	ip.LoaderStatus = true
	ip.LoaderExpect = true
	ip.LoaderKey = true
	ip.LoaderNew = true
}

// Set administrative permissions on the investigator
func (ip *InvestigatorPerms) AdminSet() {
	ip.Investigator = true
	ip.InvestigatorCreate = true
	ip.InvestigatorUpdate = true
}

// Permissions that can be assigned to investigators
const (
	PermSearch = 1 << iota
	PermAction
	PermActionCreate
	PermCommand
	PermAgent
	PermDashboard
	PermLoader
	PermLoaderStatus
	PermLoaderExpect
	PermLoaderKey
	PermLoaderNew
	PermManifest
	PermManifestSign
	PermManifestNew
	PermManifestStatus
	PermManifestLoaders
	PermInvestigator
	PermInvestigatorCreate
	PermInvestigatorUpdate
)

const (
	StatusActiveInvestigator   string = "active"
	StatusDisabledInvestigator string = "disabled"
)
