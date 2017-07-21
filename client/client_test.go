// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package client /* import "mig.ninja/mig/client" */

import (
	"net/http"

	"github.com/jvehent/cljs"
	"mig.ninja/mig"
)

func init() {
	GetClientFunction = NewTestClient
}

type TestClient struct {
	Conf Configuration
}

func NewTestClient(conf Configuration, version string) (ret Client, err error) {
	cli := &TestClient{}
	return cli, nil
}

func (cli TestClient) CompressAction(act mig.Action) (ret mig.Action, err error) {
	return
}

func (cli TestClient) DisableDebug() {
}

func (cli TestClient) Do(req *http.Request) (*http.Response, error) {
	return nil, nil
}

func (cli TestClient) EnableDebug() {
}

func (cli TestClient) EvaluateAgentTarget(target string) ([]mig.Agent, error) {
	return nil, nil
}

func (cli TestClient) FetchActionResults(act mig.Action) ([]mig.Command, error) {
	return nil, nil
}

func (cli TestClient) FollowAction(act mig.Action, total int) error {
	return nil
}

func (cli TestClient) GetAction(aid float64) (act mig.Action, links []cljs.Link, err error) {
	return
}

func (cli TestClient) GetAgent(agtid float64) (ret mig.Agent, err error) {
	return
}

func (cli TestClient) GetAPIResource(target string) (*cljs.Resource, error) {
	return nil, nil
}

func (cli TestClient) GetConfiguration() Configuration {
	return cli.Conf
}

func (cli TestClient) GetCommand(cid float64) (ret mig.Command, err error) {
	return
}

func (cli TestClient) GetInvestigator(iid float64) (ret mig.Investigator, err error) {
	return
}

func (cli TestClient) GetLoaderEntry(lid float64) (ret mig.LoaderEntry, err error) {
	return
}

func (cli TestClient) GetManifestLoaders(mid float64) ([]mig.LoaderEntry, error) {
	return nil, nil
}

func (cli TestClient) GetManifestRecord(mid float64) (ret mig.ManifestRecord, err error) {
	return
}

func (cli TestClient) LoaderEntryExpect(le mig.LoaderEntry, expect string) error {
	return nil
}

func (cli TestClient) LoaderEntryKey(le mig.LoaderEntry) (ret mig.LoaderEntry, err error) {
	return
}

func (cli TestClient) LoaderEntryStatus(le mig.LoaderEntry, status bool) error {
	return nil
}

func (cli TestClient) ManifestRecordStatus(mr mig.ManifestRecord, status string) error {
	return nil
}

func (cli TestClient) PostAction(act mig.Action) (ret mig.Action, err error) {
	return
}

func (cli TestClient) PostInvestigator(name string, pubkey []byte, perms mig.InvestigatorPerms) (inv mig.Investigator, err error) {
	return
}

func (cli TestClient) PostInvestigatorAPIKeyStatus(iid float64, status string) (ret mig.Investigator, err error) {
	return
}

func (cli TestClient) PostInvestigatorPerms(iid float64, perms mig.InvestigatorPerms) error {
	return nil
}

func (cli TestClient) PostInvestigatorStatus(iid float64, status string) error {
	return nil
}

func (cli TestClient) PostManifestSignature(mr mig.ManifestRecord, sig string) error {
	return nil
}

func (cli TestClient) PostNewLoader(le mig.LoaderEntry) (ret mig.LoaderEntry, err error) {
	return
}

func (cli TestClient) PostNewManifest(mr mig.ManifestRecord) error {
	return nil
}

func (cli TestClient) PrintActionResults(act mig.Action, show string, render string) error {
	return nil
}

func (cli TestClient) ResolveTargetMacro(target string) string {
	return ""
}

func (cli TestClient) SignAction(act mig.Action) (ret mig.Action, err error) {
	return
}

func (cli TestClient) SignManifest(mr mig.ManifestRecord) (ret string, err error) {
	return
}
