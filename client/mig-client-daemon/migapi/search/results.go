// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/modules"
)

// APIResultAggregator is a `ResultAggregator` that will search for results
// from the MIG API.
type APIResultAggregator struct {
	baseAddress string
}

// responseStructure is a convenient container type for the response
// format expected from the API.
type responseStructure struct {
	Collection collectionJSON `json:"collection"`
}

type collectionJSON struct {
	Error errorJSON  `json:"error"`
	Items []itemJSON `json:"items"`
}

type errorJSON struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type itemJSON struct {
	Data []itemDataJSON `json:"data"`
}

type itemDataJSON struct {
	Name  string      `json:"name"`
	Value mig.Command `json:"value"`
}

// NewAPIResultAggregator constructs a new `APIResultAggregator`.
func NewAPIResultAggregator(baseAddr string) APIResultAggregator {
	return APIResultAggregator{
		baseAddress: baseAddr,
	}
}

// Search queries the MIG API until it reads all of the results generated as
// a result of an action being executed by agents.
func (aggregator APIResultAggregator) Search(
	actionID actions.InternalActionID,
) ([]modules.Result, error) {
	const limitResultsPerRequest = 50

	resultsReceived := 0
	results := []modules.Result{}

	baseAddr, parseErr := url.Parse(aggregator.baseAddress)
	if parseErr != nil {
		err := errors.New("result aggregator configured with an invalid base address for the MIG API")
		return []modules.Result{}, err
	}

	for {
		// Approach copied from FetchActionResults in client/client.go
		apiEndptPath := fmt.Sprintf(
			"search?type=command&limit=%d&offset=%d&actionid=%d",
			limitResultsPerRequest,
			resultsReceived,
			actionID)
		endptPath, _ := url.Parse(apiEndptPath)
		endptURL := baseAddr.ResolveReference(endptPath)

		response, err := http.Get(endptURL.String())
		if err != nil {
			err = fmt.Errorf("failed to send request to the MIG API. Error: %s", err.Error())
			return []modules.Result{}, err
		}

		respData := responseStructure{}
		decoder := json.NewDecoder(response.Body)
		defer response.Body.Close()
		decodeErr := decoder.Decode(&respData)
		if decodeErr != nil {
			err = fmt.Errorf("unexpected data in response from MIG API. Error: %s", decodeErr.Error())
			return []modules.Result{}, err
		}

		if respData.Collection.Error.Message == "no results found" {
			break
		}
		if respData.Collection.Error.Code != "" {
			err = fmt.Errorf("got an error from the MIG API: %s", respData.Collection.Error.Message)
			return []modules.Result{}, err
		}

		resultsReceivedThisIter := 0
		for _, item := range respData.Collection.Items {
			for _, data := range item.Data {
				if data.Name != "command" {
					continue
				}
				for _, result := range data.Value.Results {
					resultsReceivedThisIter++
					results = append(results, result)
				}
			}
		}

		if resultsReceivedThisIter == 0 {
			break
		}
		resultsReceived += resultsReceivedThisIter
	}

	return results, nil
}
