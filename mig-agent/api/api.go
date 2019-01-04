package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Each endpoint format string assumes that the base address for the API will
// be something like "https://domain.com/api/v1".
const (
	heartbeatEndptFmt string = "%s/heartbeat"
)

// Client is an HTTP client abstraction for the MIG API.
// It must be configured with:
//
// 1. The base address of the API, such as https://api.mig.mozilla.com/api/v1
// 2. A list of addresses of proxies to try to use.
//
// Note that the base address MUST NOT contain a trailing forward-slash (/).
// Note also that the client will try each proxy in order.
type Client struct {
	baseAddress    string
	proxyAddresses []string
}

// Heartbeat contains information describing an active agent.
type Heartbeat struct {
	Name        string      `json:"name"`
	Mode        string      `json:"mode"`
	Version     string      `json:"version"`
	PID         uint        `json:"pid"`
	QueueLoc    string      `json:"queueLoc"`
	StartTime   time.Time   `json:"startTime"`
	RefreshTime time.Time   `json:"refreshTime"`
	Environment Environment `json:"environment"`
	Tags        []Tag       `json:"tags"`
}

// Tag is a label associated with an agent.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Environment contains information about the environment an agent is running in.
type Environment struct {
	Init      string   `json:"init"`
	Ident     string   `json:"ident"`
	OS        string   `json:"os"`
	Arch      string   `json:"arch"`
	IsProxied bool     `json:"isProxied"`
	Proxy     string   `json:"proxy"`
	Addresses []string `json:"addresses"`
	PublicIP  string   `json:"publicIP"`
	Modules   []string `json:"modules"`
}

// NewClient constructs a client that can be used to make requests to the
// MIG API through proxies.
func NewClient(baseAddr string, proxies []string) Client {
	return Client{
		baseAddress:    baseAddr,
		proxyAddresses: proxies,
	}
}

// Heartbeat posts a heartbeat message to the MIG API, indicating that the
// agent is active.
func (client Client) Heartbeat(hb Heartbeat) error {
	fullURL := fmt.Sprintf(heartbeatEndptFmt, client.baseAddress)
	encoded, _ := json.Marshal(hb)
	body := bytes.NewReader(encoded)
	response := struct {
		Error *string `json:"error"`
	}{nil}
	status, err := client.sendJSON("POST", fullURL, body, &response)

	if err != nil {
		return err
	}
	if status != 200 && response.Error != nil {
		return errors.New(*response.Error)
	}

	return nil
}

func (client Client) sendJSON(
	method string,
	destUrl string,
	body io.Reader,
	response interface{},
) (int, error) {
	var httpClient *http.Client = nil
	var transport http.RoundTripper = http.DefaultTransport

	var didFindProxy = len(client.proxyAddresses) == 0
	for _, proxyAddr := range client.proxyAddresses {
		if !strings.HasPrefix(proxyAddr, "http") {
			proxyAddr = "http://" + proxyAddr
		}
		proxyUrl, err := url.Parse(proxyAddr)
		if err != nil {
			continue
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
		}
		didFindProxy = true
		break
	}

	if !didFindProxy {
		return 0, errors.New("None of the configured proxies was found to be usable")
	}

	httpClient = &http.Client{
		Transport: transport,
	}
	req, err := http.NewRequest(method, destUrl, body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	decoder := json.NewDecoder(res.Body)
	defer res.Body.Close()
	err = decoder.Decode(response)
	if err != nil {
		return 0, err
	}
	return res.StatusCode, nil
}
