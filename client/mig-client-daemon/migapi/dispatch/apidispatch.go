// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Zack Mullaly zmullaly@mozilla.com [:zack]

package dispatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"bytes"
	"io/ioutil"

	"mig.ninja/mig"
	"mig.ninja/mig/client/mig-client-daemon/actions"
	"mig.ninja/mig/client/mig-client-daemon/migapi/authentication"
)

const createActionEndpt string = "/api/v1/action/create/"

// APIDispatcher is a `Dispatcher` that will send actions to the MIG API.
type APIDispatcher struct {
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
	Name  string            `json:"name"`
	Value itemDataValueJSON `json:"value"`
}

type itemDataValueJSON struct {
	ID actions.InternalActionID `json:"id"`
}

// NewAPIDispatcher constructs a new `APIDispatcher`.
// The dispatcher needs to be informed of the base address of the MIG API,
// e.g. http://api.mig.ninja
func NewAPIDispatcher(serverURL string) APIDispatcher {
	return APIDispatcher{
		baseAddress: serverURL,
	}
}

// Dispatch sends a POST request to the MIG API to create an action.
func (dispatcher APIDispatcher) Dispatch(
	action mig.Action,
	auth authentication.Authenticator,
) (actions.InternalActionID, error) {
	// Construct the full path to the create action endpoint for v1 of the API.
	baseURL, parseErr := url.Parse(dispatcher.baseAddress)
	if parseErr != nil {
		return actions.InvalidID, parseErr
	}
	reqPath, _ := url.Parse(createActionEndpt)
	fullURL := baseURL.ResolveReference(reqPath)

	// Create a reader for the JSON-encoded action.
	body, encodeErr := json.Marshal(action)
	if encodeErr != nil {
		return actions.InvalidID, encodeErr
	}
	// This isn't made clear in the documentation, but this is how the body
	// of this request has to be formatted. See
	// https://github.com/mozilla/mig/blob/master/client/client.go#L859
	bodyStr := url.Values{"action": {string(body)}}.Encode()
	bodyReader := strings.NewReader(bodyStr)

	// Create an authenticated request.
	request, createReqErr := http.NewRequest("POST", fullURL.String(), bodyReader)
	if createReqErr != nil {
		return actions.InvalidID, createReqErr
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	authErr := auth.Authenticate(request)
	if authErr != nil {
		return actions.InvalidID, authErr
	}

	// Perform the request.
	client := &http.Client{}
	response, reqErr := client.Do(request)
	if reqErr != nil {
		return actions.InvalidID, reqErr
	}

	// Check for an error in the response.
	respData := responseStructure{}
	//decoder := json.NewDecoder(response.Body)
	// Debugging
	respBody, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return actions.InvalidID, readErr
	}
	fmt.Printf("Got raw response body\n%s\n-----\n", string(respBody))
	decoder := json.NewDecoder(bytes.NewReader(respBody))
	// Debugging
	defer response.Body.Close()
	decodeErr := decoder.Decode(&respData)
	if decodeErr != nil {
		return actions.InvalidID, decodeErr
	}
	fmt.Printf("Got response data\n%v\n-----\n", respData)
	fmt.Printf("Length of error code == %d\n", len(respData.Collection.Error.Message))
	// We test this case last because returning an error here gives us the
	// least amount of information.
	if response.StatusCode != http.StatusAccepted {
		return actions.InvalidID, errors.New("request not accepted by API")
	}
	actionID, err := extractActionID(respData.Collection)
	if err != nil {
		return actions.InvalidID, err
	}

	return actionID, nil
}

func extractActionID(data collectionJSON) (actions.InternalActionID, error) {
	missingData := errors.New("response to dispatch request does not contain expected data")

	if data.Error.Code != "" {
		return actions.InvalidID, errors.New(data.Error.Message)
	}
	if len(data.Items) == 0 {
		return actions.InvalidID, missingData
	}
	if len(data.Items[0].Data) == 0 {
		return actions.InvalidID, missingData
	}
	if strings.HasPrefix(data.Items[0].Data[0].Name, "action ID") {
		return data.Items[0].Data[0].Value.ID, nil
	}
	return actions.InvalidID, missingData
}
