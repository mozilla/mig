// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package client /* import "github.com/mozilla/mig/client" */

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/jvehent/cljs"
	"golang.org/x/crypto/openpgp"
	"gopkg.in/gcfg.v1"
	"github.com/mozilla/mig"
	"github.com/mozilla/mig/modules"
	"github.com/mozilla/mig/pgp"
)

var version string

// Client is a type used to interact with the MIG API.
type Client struct {
	API     *http.Client
	Token   string
	Conf    Configuration
	Version string
	debug   bool
}

// Configuration stores the live configuration and global parameters of a client
type Configuration struct {
	API     APIConf    // location of the MIG API
	Homedir string     // location of the user's home directory
	GPG     GpgConf    // location of the user's secring
	Targets TargetConf // Target macro specification
}

// APIConf stores configuration values related to the API connectivity.
type APIConf struct {
	URL            string
	SkipVerifyCert bool
}

// GpgConf stores configuration values related to client keyring access.
type GpgConf struct {
	Home          string // Path to GPG keyrings
	KeyID         string // GPG key ID to use for X-PGPAUTHORIZATION and action signing
	Keyserver     string // Key server to fetch keys from if needed in mig-console
	UseAPIKeyAuth string // Prefer X-MIGAPIKEY authentication for API access, set to API key
}

// TargetConf stores macros present in the configuration file that can be used
// as short form targeting strings.
type TargetConf struct {
	Macro  []string
	macros map[string]string
}

// addMacro is used by the macro parser to add processed target macros to the client
// configuration
func (t *TargetConf) addMacro(name string, tgt string) {
	if t.macros == nil {
		t.macros = make(map[string]string)
	}
	t.macros[name] = tgt
}

// getMacro returns a macro from the targeting configuration by name
func (t *TargetConf) getMacro(name string) (string, error) {
	if val, ok := t.macros[name]; ok {
		return val, nil
	}
	return "", fmt.Errorf("macro %v not found", name)
}

// Can store the passphrase used to decrypt a GPG private key so the client
// does not attempt to prompt for it. We do not store it in the client
// configuration, as under normal usage passphrases for MIG should not be
// stored in cleartext. However in some cases such as with mig-runner this
// behavior is required for automated operation.
var clientPassphrase string

// ClientPassphrase sets the GPG passphrase to be used by the client for secret key operations,
// this can be used to cache a passphrase to avoid being prompted.
func ClientPassphrase(s string) {
	clientPassphrase = s
}

// NewClient initiates a new instance of a Client
func NewClient(conf Configuration, version string) (cli Client, err error) {
	cli.Version = version
	cli.Conf = conf
	tr := &http.Transport{
		DisableCompression: false,
		DisableKeepAlives:  false,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
			InsecureSkipVerify: conf.API.SkipVerifyCert,
		},
		Proxy: http.ProxyFromEnvironment,
	}
	cli.API = &http.Client{Transport: tr}
	// If the client is using API key authentication to access the API, we don't have
	// anything left to do here.
	if conf.GPG.UseAPIKeyAuth != "" {
		return
	}
	// if the env variable to the gpg agent socket isn't set, try to
	// find the socket and set the variable
	if os.Getenv("GPG_AGENT_INFO") == "" {
		_, err = os.Stat(conf.GPG.Home + "/S.gpg-agent")
		if err == nil {
			// socket was found, set it
			os.Setenv("GPG_AGENT_INFO", conf.GPG.Home+"/S.gpg-agent")
		}
	}
	if clientPassphrase != "" {
		pgp.CachePassphrase(clientPassphrase)
	}
	// try to make a signed token, just to check that we can access the private key
	_, err = cli.MakeSignedToken()
	if err != nil {
		err = fmt.Errorf("failed to generate a security token using key %v from %v: %v",
			conf.GPG.KeyID, conf.GPG.Home+"/secring.gpg", err)
		return
	}
	return
}

// ReadConfiguration loads a client configuration from a local configuration file
// and verifies that GnuPG's secring is available
func ReadConfiguration(file string) (conf Configuration, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ReadConfiguration() -> %v", e)
		}
	}()
	_, err = os.Stat(file)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		// Otherwise, the file did not exist so we can create it if requested
		fmt.Fprintf(os.Stderr, "no configuration found at %v\n", file)
		err = MakeConfiguration(file)
		if err != nil {
			panic(err)
		}
	}
	err = gcfg.ReadFileInto(&conf, file)
	if err != nil {
		panic(err)
	}
	// If standard PGP based authentication is being used, validate these settings
	// from the configuration file
	if conf.GPG.UseAPIKeyAuth == "" {
		if conf.GPG.Home == "" {
			gnupgdir := os.Getenv("GNUPGHOME")
			if gnupgdir == "" {
				gnupgdir = "/.gnupg"
			}
			hdir, err := FindHomedir()
			if err != nil {
				panic(err)
			}
			conf.GPG.Home = path.Join(hdir, gnupgdir)
		}
		_, err = os.Stat(conf.GPG.Home + "/secring.gpg")
		if err != nil {
			panic("secring.gpg not found")
		}
	}
	// if trailing slash is missing from API url, add it
	n := len(conf.API.URL)
	if n > 1 {
		if conf.API.URL[n-1] != '/' {
			conf.API.URL += "/"
		}
	} else {
		err = fmt.Errorf("config API URL too short or undefined: len %d", n)
		panic(err)
	}
	err = addTargetMacros(&conf)
	if err != nil {
		panic(err)
	}
	return
}

// ReadEnvConfiguration reads any possible configuration values from the environment;
// currently we only load a passphrase here if provided, but conf is passed/returned
// for any future requirements to override file based configuration using environment options.
func ReadEnvConfiguration(inconf Configuration) (conf Configuration, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ReadEnvConfiguration() -> %v", e)
		}
	}()
	conf = inconf
	pf := os.Getenv("MIGPGPPASSPHRASE")
	if pf != "" {
		ClientPassphrase(pf)
	}
	return
}

// FindHomedir attempts to locate the home directory for the current user
func FindHomedir() (ret string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("FindHomedir() -> %v", e)
		}
	}()
	ret = os.Getenv("HOME")
	if ret != "" {
		return
	}
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return u.HomeDir, nil
}

// MakeConfiguration generates a new configuration file for the current user
func MakeConfiguration(file string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("MakeConfiguration() -> %v", e)
		}
	}()
	var cfg Configuration
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("generate a new configuration file at %v? (Y/n)> ", file)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	if scanner.Text() != "y" && scanner.Text() != "Y" && scanner.Text() != "" {
		panic("aborting creating configuration file")
	}
	cfg.Homedir, err = FindHomedir()
	if err != nil {
		panic(err)
	}
	cfg.GPG.Home = path.Join(cfg.Homedir, ".gnupg")
	sr, err := os.Open(path.Join(cfg.GPG.Home, "secring.gpg"))
	if err != nil {
		// We were unable to open the secring to identify the private key for the MIG
		// client tools to use. Display some additional info here and abort.
		fmt.Printf("\nIt was not possible to open your secring.gpg. The configuration generator expects\n"+
			"to find the keyring in the standard GPG location, and is looking in %v. If your keyring\n"+
			"is located elsewhere, you may need to create the configuration file yourself. Consult the\n"+
			"documentation for additional details.\n\n",
			path.Join(cfg.GPG.Home))
		panic(fmt.Sprintf("error opening secring: %v", err))
	}
	defer sr.Close()
	keyring, err := openpgp.ReadKeyRing(sr)
	if err != nil {
		panic(err)
	}
	for _, entity := range keyring {
		fingerprint := strings.ToUpper(hex.EncodeToString(entity.PrivateKey.PublicKey.Fingerprint[:]))
		// get the first name from the key identity
		var name string
		for _, identity := range entity.Identities {
			name = identity.Name
			break
		}
		fmt.Printf("found key %q with fingerprint %q.\nuse this key? (Y/n)> ", name, fingerprint)
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		if scanner.Text() == "y" || scanner.Text() == "Y" || scanner.Text() == "" {
			fmt.Printf("using key %q\n", fingerprint)
			cfg.GPG.KeyID = fingerprint
			break
		}
	}
	if cfg.GPG.KeyID == "" {
		panic("no suitable key found in " + sr.Name())
	}
	for {
		fmt.Printf("URL of API? (e.g., https://mig.example.net/api/v1/)> ")
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		cfg.API.URL = scanner.Text()
		_, err := http.Get(cfg.API.URL)
		if err != nil {
			fmt.Printf("API connection failed: %v\n", err)
			continue
		}
		break
	}
	fd, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(fd, "[api]\n\turl = \"%v\"\n", cfg.API.URL)
	fmt.Fprintf(fd, "[gpg]\n\thome = \"%v\"\n\tkeyid = \"%v\"\n", cfg.GPG.Home, cfg.GPG.KeyID)
	// Add one initial target macro
	fmt.Fprintf(fd, "[targets]\n\tmacro = allonline:status='online'\n")
	fd.Close()
	fmt.Printf("\ncreated configuration file at %v\n\n", file)
	return
}

// Do is a thin wrapper around http.Client.Do() that inserts an authentication header
// to the outgoing request
func (cli Client) Do(r *http.Request) (resp *http.Response, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Do() -> %v", e)
		}
	}()
	r.Header.Set("User-Agent", "MIG Client "+cli.Version)
	// If API key authentication has been configured, use that rather than
	// X-PGPAUTHORIZATION authentication for the API
	if cli.Conf.GPG.UseAPIKeyAuth != "" {
		r.Header.Set("X-MIGAPIKEY", cli.Conf.GPG.UseAPIKeyAuth)
	} else {
		if cli.Token == "" {
			cli.Token, err = cli.MakeSignedToken()
			if err != nil {
				panic(err)
			}
		}
		r.Header.Set("X-PGPAUTHORIZATION", cli.Token)
		if cli.debug {
			fmt.Printf("debug: %s %s %s\ndebug: User-Agent: %s\ndebug: X-PGPAUTHORIZATION: %s\n",
				r.Method, r.URL.String(), r.Proto, r.UserAgent(), r.Header.Get("X-PGPAUTHORIZATION"))
		}
	}
	// execute the request
	resp, err = cli.API.Do(r)
	if err != nil {
		msg := fmt.Errorf("request failed error: %d %s (%v)", resp.StatusCode, resp.Status, err)
		panic(msg)
	}
	// if the request failed because of an auth issue, it may be that the auth token has expired.
	// try the request again with a fresh token
	if resp.StatusCode == http.StatusUnauthorized && cli.Conf.GPG.UseAPIKeyAuth == "" {
		// Make sure we read the entire response body from the previous request before we close it
		// to avoid connection cancellation issues and a panic in API.Do()
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		resp.Body.Close()
		cli.Token, err = cli.MakeSignedToken()
		if err != nil {
			panic(err)
		}
		r.Header.Set("X-PGPAUTHORIZATION", cli.Token)
		if cli.debug {
			fmt.Printf("debug: %s %s %s\ndebug: User-Agent: %s\ndebug: X-PGPAUTHORIZATION: %s\n",
				r.Method, r.URL.String(), r.Proto, r.UserAgent(), r.Header.Get("X-PGPAUTHORIZATION"))
		}
		// execute the request
		resp, err = cli.API.Do(r)
		if err != nil {
			panic(err)
		}
	}
	return
}

// GetAPIResource retrieves a cljs resource from a target endpoint. The target must be the relative
// to the API URL passed in the configuration. For example, if the API URL is `http://localhost:12345/api/v1/`
// then target could only be set to `dashboard` to retrieve `http://localhost:12345/api/v1/dashboard`
func (cli Client) GetAPIResource(target string) (resource *cljs.Resource, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetAPIResource() -> %v", e)
		}
	}()
	r, err := http.NewRequest("GET", cli.Conf.API.URL+target, nil)
	if err != nil {
		panic(err)
	}
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	hasResource := false
	if resp.Body != nil && resp.StatusCode < http.StatusInternalServerError {
		defer resp.Body.Close()
		// unmarshal the body. don't attempt to interpret it, as long as it
		// fits into a cljs.Resource, it's acceptable
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		if len(body) > 1 {
			if cli.debug {
				fmt.Printf("debug: RESPONSE BODY:\ndebug: %s\n", body)
			}
			err = json.Unmarshal(body, &resource)
			if err != nil {
				panic(err)
			}
			hasResource = true
		} else {
			if cli.debug {
				fmt.Printf("debug: RESPONSE BODY: EMPTY\n")
			}
		}
	}
	if resp.StatusCode != http.StatusOK {
		if hasResource {
			err = fmt.Errorf("error: HTTP %d. API call failed with error '%v' (code %s)",
				resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		} else {
			err = fmt.Errorf("error: HTTP %d %s. No response body", resp.StatusCode, http.StatusText(resp.StatusCode))
		}
		panic(err)
	}
	return
}

// GetAction retrieves a MIG Action from the API using its Action ID
func (cli Client) GetAction(aid float64) (a mig.Action, links []cljs.Link, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetAction() -> %v", e)
		}
	}()
	target := fmt.Sprintf("action?actionid=%.0f", aid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "action" {
		panic("API returned something that is not an action... something's wrong.")
	}
	a, err = ValueToAction(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	links = resource.Collection.Items[0].Links
	return
}

// GetManifestRecord retrieves a MIG manifest record from the API using the
// record ID
func (cli Client) GetManifestRecord(mid float64) (mr mig.ManifestRecord, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetManifestRecord() -> %v", e)
		}
	}()
	target := fmt.Sprintf("manifest?manifestid=%.0f", mid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "manifest" {
		panic("API returned something that is not a manifest... something's wrong.")
	}
	mr, err = ValueToManifestRecord(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// GetManifestLoaders retrieves list of known loader entries that will match manifest mid
func (cli Client) GetManifestLoaders(mid float64) (ldrs []mig.LoaderEntry, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetManifestLoaders() -> %v", e)
		}
	}()
	target := fmt.Sprintf("manifest/loaders/?manifestid=%.0f", mid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			if data.Name != "loader" {
				continue
			}
			ldr, err := ValueToLoaderEntry(data.Value)
			if err != nil {
				panic(err)
			}
			ldrs = append(ldrs, ldr)
			break
		}
	}
	return
}

// ManifestRecordStatus changes the status of an existing manifest record
func (cli Client) ManifestRecordStatus(mr mig.ManifestRecord, status string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ResetManifestRecord() -> %v", e)
		}
	}()
	data := url.Values{"manifestid": {fmt.Sprintf("%.0f", mr.ID)}, "status": {status}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"manifest/status/",
		strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. Status update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// PostNewManifest posts a new manifest record for storage through the API
func (cli Client) PostNewManifest(mr mig.ManifestRecord) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostNewManifest() -> %v", e)
		}
	}()
	mrbuf, err := json.Marshal(mr)
	if err != nil {
		panic(err)
	}
	data := url.Values{"manifest": {string(mrbuf)}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"manifest/new/",
		strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusCreated {
		err = fmt.Errorf("error: HTTP %d. Manifest create failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// PostManifestSignature adds a new signature to an existing manifest known to the API
func (cli Client) PostManifestSignature(mr mig.ManifestRecord, sig string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostManifestSignature() -> %v", e)
		}
	}()
	data := url.Values{"manifestid": {fmt.Sprintf("%.0f", mr.ID)}, "signature": {sig}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"manifest/sign/",
		strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. Signature update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// GetLoaderEntry retrieves a MIG loader entry from the API using the record ID
func (cli Client) GetLoaderEntry(lid float64) (le mig.LoaderEntry, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetLoaderEntry() -> %v", e)
		}
	}()
	target := fmt.Sprintf("loader?loaderid=%.0f", lid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "loader" {
		panic("API returned something that is not a loader... something's wrong.")
	}
	le, err = ValueToLoaderEntry(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// LoaderEntryExpect changes the expect fields of an existing loader entry
func (cli Client) LoaderEntryExpect(le mig.LoaderEntry, eval string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("LoaderEntryExpect() -> %v", e)
		}
	}()
	data := url.Values{"loaderid": {fmt.Sprintf("%.0f", le.ID)},
		"expectenv": {eval},
	}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"loader/expect/",
		strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. Expect update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// LoaderEntryStatus changes the status of an existing loader entry
func (cli Client) LoaderEntryStatus(le mig.LoaderEntry, status bool) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("LoaderEntryStatus() -> %v", e)
		}
	}()
	statusval := "disabled"
	if status {
		statusval = "enabled"
	}
	data := url.Values{"loaderid": {fmt.Sprintf("%.0f", le.ID)}, "status": {statusval}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"loader/status/",
		strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. Status update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// LoaderEntryKey changes the key on an existing loader entry
func (cli Client) LoaderEntryKey(le mig.LoaderEntry) (newle mig.LoaderEntry, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("LoaderEntryKey() -> %v", e)
		}
	}()
	data := url.Values{"loaderid": {fmt.Sprintf("%.0f", le.ID)}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"loader/key/",
		strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. Key update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	newle, err = ValueToLoaderEntry(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// PostNewLoader posts a new loader entry for storage through the API
func (cli Client) PostNewLoader(le mig.LoaderEntry) (newle mig.LoaderEntry, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostNewLoader() -> %v", e)
		}
	}()
	// When adding a new loader entry, the prefix and key values should
	// be "", since the server will be generating them.
	if le.Prefix != "" || le.Key != "" {
		panic("loader key and prefix must be unset")
	}
	lebuf, err := json.Marshal(le)
	if err != nil {
		panic(err)
	}
	data := url.Values{"loader": {string(lebuf)}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"loader/new/",
		strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusCreated {
		err = fmt.Errorf("error: HTTP %d. Loader create failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	newle, err = ValueToLoaderEntry(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// PostAction submits a MIG Action to the API and returns the reflected action with API ID
func (cli Client) PostAction(a mig.Action) (a2 mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostAction() -> %v", e)
		}
	}()
	a.SyntaxVersion = mig.ActionVersion
	// serialize
	ajson, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	actionstr := string(ajson)
	data := url.Values{"action": {actionstr}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"action/create/", strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		err = fmt.Errorf("error: HTTP %d. action creation failed", resp.StatusCode)
		panic(err)
	}
	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		panic(err)
	}
	a2, err = ValueToAction(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// ValueToAction converts JSON data in interface v into a mig.Action
func ValueToAction(v interface{}) (a mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ValueToAction() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &a)
	if err != nil {
		panic(err)
	}
	return
}

// ValueToLoaderEntry converts JSON data in interface v into a mig.LoaderEntry
func ValueToLoaderEntry(v interface{}) (l mig.LoaderEntry, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ValueToLoaderEntries() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &l)
	if err != nil {
		panic(err)
	}
	return
}

// ValueToManifestRecord converts JSON data in interface v into a mig.ManifestRecord
func ValueToManifestRecord(v interface{}) (m mig.ManifestRecord, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ValueToManifestRecord() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &m)
	if err != nil {
		panic(err)
	}
	return
}

// GetCommand fetches the specified command ID from the API and returns it
func (cli Client) GetCommand(cmdid float64) (cmd mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetCommand() -> %v", e)
		}
	}()
	target := "command?commandid=" + fmt.Sprintf("%.0f", cmdid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "command" {
		panic("API returned something that is not a command... something's wrong.")
	}
	cmd, err = ValueToCommand(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// ValueToCommand converts JSON data in interface v into a mig.Command
func ValueToCommand(v interface{}) (cmd mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ValueToCommand() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &cmd)
	if err != nil {
		panic(err)
	}
	return
}

// GetAgent fetches the specified agent ID from the API and returns it
func (cli Client) GetAgent(agtid float64) (agt mig.Agent, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetAgent() -> %v", e)
		}
	}()
	target := "agent?agentid=" + fmt.Sprintf("%.0f", agtid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "agent" {
		panic("API returned something that is not an agent... something's wrong.")
	}
	agt, err = ValueToAgent(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// ValueToAgent converts JSON data in interface v into a mig.Agent
func ValueToAgent(v interface{}) (agt mig.Agent, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("valueToAgent() -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &agt)
	if err != nil {
		panic(err)
	}
	return
}

// GetInvestigator fetches the specified investigator ID from the API and returns it
func (cli Client) GetInvestigator(iid float64) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("GetInvestigator() -> %v", e)
		}
	}()
	target := "investigator?investigatorid=" + fmt.Sprintf("%.0f", iid)
	resource, err := cli.GetAPIResource(target)
	if err != nil {
		panic(err)
	}
	if resource.Collection.Items[0].Data[0].Name != "investigator" {
		panic("API returned something that is not an investigator... something's wrong.")
	}
	inv, err = ValueToInvestigator(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// PostInvestigator creates an Investigator and returns the reflected investigator. If pubkey
// is zero-length, the investigator will be created without a PGP public key.
func (cli Client) PostInvestigator(name string, pubkey []byte, pset mig.InvestigatorPerms) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostInvestigator() -> %v", e)
		}
	}()
	// build the body into buf using a multipart writer
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	// set the name form value
	err = writer.WriteField("name", name)
	if err != nil {
		panic(err)
	}
	pbuf, err := json.Marshal(&pset)
	if err != nil {
		panic(err)
	}
	err = writer.WriteField("permissions", string(pbuf))
	if err != nil {
		panic(err)
	}
	if len(pubkey) > 0 {
		// set the publickey form value
		part, err := writer.CreateFormFile("publickey", fmt.Sprintf("%s.asc", name))
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(part, bytes.NewReader(pubkey))
		if err != nil {
			panic(err)
		}
	}
	// must close the writer to write trailing boundary
	err = writer.Close()
	if err != nil {
		panic(err)
	}
	// post the request
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"investigator/create/", buf)
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	// get the investigator back from the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusCreated {
		err = fmt.Errorf("HTTP %d: %v (code %s)", resp.StatusCode,
			resource.Collection.Error.Message, resource.Collection.Error.Code)
		return
	}
	inv, err = ValueToInvestigator(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// PostInvestigatorPerms sets permission on an investigator
func (cli Client) PostInvestigatorPerms(iid float64, perm mig.InvestigatorPerms) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostInvestigatorPerms() -> %v", e)
		}
	}()
	permbuf, err := json.Marshal(&perm)
	if err != nil {
		panic(err)
	}
	data := url.Values{"id": {fmt.Sprintf("%.0f", iid)}, "permissions": {fmt.Sprintf("%v", string(permbuf))}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"investigator/update/", strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. permission update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// PostInvestigatorStatus updates the status of an Investigator
func (cli Client) PostInvestigatorStatus(iid float64, newstatus string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostInvestigatorStatus() -> %v", e)
		}
	}()
	data := url.Values{"id": {fmt.Sprintf("%.0f", iid)}, "status": {newstatus}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"investigator/update/", strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. status update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	return
}

// PostInvestigatorAPIKeyStatus is used to either enable or disable API key based access
// to the MIG API for an investigator. API key based access to the API can be used in
// place of X-PGPAUTHORIZATION API authentication.
//
// If an API key is being set, the returned investigator APIKey value will contain the
// assigned key. newstatus should be set to either 'active' or 'disabled'. If a key already
// exists for an investigator, calling this with a status of 'active' will cause the existing
// key to be replaced.
func (cli Client) PostInvestigatorAPIKeyStatus(iid float64, newstatus string) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PostInvestigatorAPIKey() -> %v", e)
		}
	}()
	data := url.Values{"id": {fmt.Sprintf("%.0f", iid)}, "apikey": {newstatus}}
	r, err := http.NewRequest("POST", cli.Conf.API.URL+"investigator/update/", strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cli.Do(r)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var resource *cljs.Resource
	if len(body) > 1 {
		err = json.Unmarshal(body, &resource)
		if err != nil {
			panic(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error: HTTP %d. apikey update failed with error '%v' (code %s)",
			resp.StatusCode, resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}
	inv, err = ValueToInvestigator(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	return
}

// ValueToInvestigator converts JSON data in interface v into a mig.Investigator
func ValueToInvestigator(v interface{}) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("valueToInvestigator) -> %v", e)
		}
	}()
	bData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bData, &inv)
	if err != nil {
		panic(err)
	}
	return
}

// MakeSignedToken encrypts a timestamp and a random number with the users GPG key
// to use as an auth token with the API
func (cli Client) MakeSignedToken() (token string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("MakeSignedToken() -> %v", e)
		}
	}()
	tokenVersion := 1
	str := fmt.Sprintf("%d;%s;%.0f", tokenVersion, time.Now().UTC().Format(time.RFC3339), mig.GenID())
	secringFile, err := os.Open(cli.Conf.GPG.Home + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer secringFile.Close()
	sig, err := pgp.Sign(str+"\n", cli.Conf.GPG.KeyID, secringFile)
	if err != nil {
		panic(err)
	}
	token = str + ";" + sig
	return
}

// CompressAction takens a MIG action, and applies compression to any operations
// within the action for which compression is requested.
//
// This function should be called on the action prior to signing it for submission
// to the API.
func (cli Client) CompressAction(a mig.Action) (compAction mig.Action, err error) {
	compAction = a
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("CompressAction() -> %v", e)
		}
	}()
	for i := range compAction.Operations {
		if !compAction.Operations[i].WantCompressed {
			continue
		}
		if compAction.Operations[i].IsCompressed {
			continue
		}
		err = compAction.Operations[i].CompressOperationParam()
		if err != nil {
			panic(err)
		}
	}
	return
}

// SignAction takes a MIG Action, signs it with the key identified in the configuration
// and returns the signed action
func (cli Client) SignAction(a mig.Action) (signedAction mig.Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("SignAction() -> %v", e)
		}
	}()
	secring, err := os.Open(cli.Conf.GPG.Home + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer secring.Close()
	sig, err := a.Sign(cli.Conf.GPG.KeyID, secring)
	if err != nil {
		panic(err)
	}
	a.PGPSignatures = append(a.PGPSignatures, sig)
	signedAction = a
	return
}

// SignManifest takes a MIG manifest record, signs it with the key identified
// in the configuration and returns the signature
func (cli Client) SignManifest(m mig.ManifestRecord) (ret string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("SignManifest() -> %v", e)
		}
	}()
	secring, err := os.Open(cli.Conf.GPG.Home + "/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer secring.Close()
	ret, err = m.Sign(cli.Conf.GPG.KeyID, secring)
	if err != nil {
		panic(err)
	}
	return
}

// ResolveTargetMacro resolves macros specified by a client; clients should pass the
// action target string here, and this function will return the resolved target if it
// is a valid macro, otherwise it just returns the passed target string
func (cli Client) ResolveTargetMacro(target string) string {
	v, err := cli.Conf.Targets.getMacro(target)
	if err != nil {
		return target
	}
	return v
}

// EvaluateAgentTarget runs a search against the api to find all agents that match an action target string
func (cli Client) EvaluateAgentTarget(target string) (agents []mig.Agent, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("EvaluateAgentTarget() -> %v", e)
		}
	}()
	query := "search?type=agent&limit=1000000&target=" + url.QueryEscape(target)
	resource, err := cli.GetAPIResource(query)
	if err != nil {
		panic(err)
	}
	for _, item := range resource.Collection.Items {
		for _, data := range item.Data {
			if data.Name != "agent" {
				continue
			}
			agt, err := ValueToAgent(data.Value)
			if err != nil {
				panic(err)
			}
			agents = append(agents, agt)
		}
	}
	return
}

// FollowAction continuously loops over an action and prints its completion status in os.Stderr.
// When the action reaches its expiration date, FollowAction prints its final status and returns.
//
// a represents the action being followed, and total indicates the total number of agents
// the action was submitted to and is used to initialize the progress meter.
//
// stop is of type chan bool, and passing a value to this channel will cause the routine to return
// immediately.
func (cli Client) FollowAction(a mig.Action, total int, stop chan bool) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("followAction() -> %v", e)
		}
	}()
	fmt.Fprintf(os.Stderr, "\x1b[34mFollowing action ID %.0f.\x1b[0m\n", a.ID)
	previousctr := 0
	status := ""
	attempts := 0
	cancelfollow := false
	var completion float64
	bar := pb.New(total)
	bar.ShowSpeed = true
	bar.SetMaxWidth(80)
	bar.Output = os.Stderr
	bar.Start()
	go func() {
		_ = <-stop
		bar.Postfix(" [cancelling]")
		cancelfollow = true
		bar.Finish()
	}()
	for {
		a, _, err = cli.GetAction(a.ID)
		if err != nil {
			attempts++
			time.Sleep(time.Second)
			if attempts >= 30 {
				panic("failed to retrieve action after 30 seconds. launch may have failed")
			}
			continue
		}
		if status == "" {
			status = a.Status
		}
		// exit follower mode if status isn't one we follow,
		// or enough commands have returned
		// or expiration time has passed
		if (status != "pending" && status != "scheduled" && status != "preparing" && status != "inflight") ||
			(a.Counters.Done > 0 && a.Counters.Done >= a.Counters.Sent) ||
			(time.Now().After(a.ExpireAfter.Add(10 * time.Second))) {
			goto finish
			break
		}
		if cancelfollow {
			// We have been asked to stop, just return
			return nil
		}
		if a.Counters.Done > 0 && a.Counters.Done > previousctr {
			completion = (float64(a.Counters.Done) / float64(a.Counters.Sent)) * 100
			if completion < 99.5 {
				bar.Add(a.Counters.Done - previousctr)
				bar.Update()
				previousctr = a.Counters.Done
			}
		}
		time.Sleep(2 * time.Second)
	}
finish:
	bar.Add(total - previousctr)
	bar.Update()
	bar.Finish()
	a, _, err = cli.GetAction(a.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[error] failed to retrieve action counters\n")
	} else {
		completion = (float64(a.Counters.Done) / float64(a.Counters.Sent)) * 100
		fmt.Fprintf(os.Stderr, "\x1b[34m%2.1f%% done in %s\x1b[0m\n", completion, time.Now().Sub(a.StartTime).String())
	}
	fmt.Fprintf(os.Stderr, "\x1b[34m")
	a.PrintCounters()
	fmt.Fprintf(os.Stderr, "\x1b[0m")
	return
}

// FetchActionResults retrieves mig command results associated with a
// particular action. This function differs from PrintActionResults in
// that it returns a slice of mig.Command structs, rather then printing
// results to stdout.
//
// XXX Note in the future it may be worth refactoring the action print
// functions to make use of this, but it would require additional work.
func (cli Client) FetchActionResults(a mig.Action) (ret []mig.Command, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("FetchActionResults() -> %v", e)
		}
	}()

	limit := 37
	offset := 0
	ret = make([]mig.Command, 0)

	for {
		target := fmt.Sprintf("search?type=command&limit=%d&offset=%d", limit, offset)
		target = target + fmt.Sprintf("&actionid=%.0f", a.ID)

		resource, err := cli.GetAPIResource(target)
		if resource.Collection.Error.Message == "no results found" {
			err = nil
			break
		} else if err != nil {
			panic(err)
		}
		count := 0
		for _, item := range resource.Collection.Items {
			for _, data := range item.Data {
				if data.Name != "command" {
					continue
				}
				cmd, err := ValueToCommand(data.Value)
				if err != nil {
					panic(err)
				}
				ret = append(ret, cmd)
				count++
			}
		}
		if count == 0 {
			break
		}
		offset += limit
	}

	return ret, nil
}

// PrintActionResults fetches the results of action a from the API and prints
// the results on stdout.
//
// show can either be found, notfound, or all and can be used to control which results
// are fetched and displayed for a given action.
func (cli Client) PrintActionResults(a mig.Action, show string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PrintActionResults() -> %v", e)
		}
	}()
	var (
		found                   bool
		report, foundQ          string
		limit, offset, agtCount int = 37, 0, 0
	)
	switch show {
	case "found":
		found = true
		foundQ = "&foundanything=true"
	case "notfound":
		found = false
		foundQ = "&foundanything=false"
	case "all":
		found = false
	default:
		return fmt.Errorf("invalid parameter '%s'", show)
	}

	// loop until all results have been retrieved using paginated queries
	for {
		target := fmt.Sprintf("search?type=command&limit=%d&offset=%d&actionid=%.0f%s%s", limit, offset, a.ID, foundQ, report)
		resource, err := cli.GetAPIResource(target)
		// because we query using pagination, the last query will return a 404 with no result.
		// When that happens, GetAPIResource returns an error which we do not report to the user
		switch resource.Collection.Error.Message {
		case "", "no results found":
			err = nil
		default:
			panic(err)
		}
		count := 0
		for _, item := range resource.Collection.Items {
			for _, data := range item.Data {
				if data.Name != "command" {
					continue
				}
				cmd, err := ValueToCommand(data.Value)
				if err != nil {
					panic(err)
				}
				err = PrintCommandResults(cmd, found, true)
				if err != nil {
					panic(err)
				}
				count++
			}
		}
		// if count is still at zero, we didn't get any results from the query and exit the loop
		if count == 0 {
			break
		}
		// else increase limit and offset and continue
		offset += limit
		agtCount += count
	}
	s := "agent has"
	if agtCount > 1 {
		s = "agents have"
	}
	fmt.Fprintf(os.Stderr, "\x1b[31m%d %s %s results\x1b[0m\n", agtCount, s, show)
	if show != "all" {
		var unsuccessful map[string][]string
		unsuccessful = make(map[string][]string)
		for _, status := range []string{mig.StatusCancelled, mig.StatusExpired, mig.StatusFailed, mig.StatusTimeout} {
			offset = 0
			for {
				// print commands that have not returned successfully
				target := fmt.Sprintf("search?type=command&limit=%d&offset=%d&actionid=%.0f&status=%s", limit, offset, a.ID, status)
				resource, err := cli.GetAPIResource(target)
				// because we query using pagination, the last query will return a 404 with no result.
				// When that happens, GetAPIResource returns an error which we do not report to the user
				switch resource.Collection.Error.Message {
				case "":
					break
				case "no results found":
					// 404, move one
					err = nil
					goto nextunsuccessful
				default:
					// something else happened, exit with error
					panic(err)
				}
				for _, item := range resource.Collection.Items {
					for _, data := range item.Data {
						if data.Name != "command" {
							continue
						}
						cmd, err := ValueToCommand(data.Value)
						if err != nil {
							panic(err)
						}
						unsuccessful[status] = append(unsuccessful[status], cmd.Agent.Name)
					}
				}
				offset += limit
			}
		nextunsuccessful:
		}
		for status, agents := range unsuccessful {
			fmt.Fprintf(os.Stderr, "\x1b[35m%s: %s\x1b[0m\n", status, strings.Join(agents, ", "))
		}
	}
	return
}

// PrintCommandResults prints the results of mig.Command cmd.
func PrintCommandResults(cmd mig.Command, onlyFound, showAgent bool) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("PrintCommandResults() -> %v", e)
		}
	}()
	var prefix string
	if showAgent {
		prefix = cmd.Agent.Name + " "
	}
	if cmd.Status != mig.StatusSuccess {
		if !onlyFound {
			fmt.Fprintf(os.Stderr, "%scommand did not succeed. status=%s\n", prefix, cmd.Status)
		}
		return
	}
	for i, result := range cmd.Results {
		if len(cmd.Action.Operations) <= i {
			if !onlyFound {
				fmt.Fprintf(os.Stderr, "%s[error] operation %d did not return results\n", prefix, i)
			}
			continue
		}
		// verify that we know the module
		moduleName := cmd.Action.Operations[i].Module
		if _, ok := modules.Available[moduleName]; !ok {
			if !onlyFound {
				fmt.Fprintf(os.Stderr, "%s[error] unknown module '%s'\n", prefix, moduleName)
			}
			continue
		}
		run := modules.Available[moduleName].NewRun()
		// look for a result printer in the module
		if _, ok := run.(modules.HasResultsPrinter); ok {
			outRes, err := run.(modules.HasResultsPrinter).PrintResults(result, onlyFound)
			if err != nil {
				panic(err)
			}
			for _, resLine := range outRes {
				fmt.Printf("%s%s\n", prefix, resLine)
			}
		} else {
			if !onlyFound {
				fmt.Fprintf(os.Stderr, "%s[error] no printer available for module '%s'\n", prefix, moduleName)
			}
		}
	}
	if !onlyFound {
		fmt.Printf("%scommand %s\n", prefix, cmd.Status)
	}
	return
}

// EnableDebug enables debugging mode in the client
func (cli *Client) EnableDebug() {
	cli.debug = true
	return
}

// DisableDebug disables debugging mode in the client
func (cli *Client) DisableDebug() {
	cli.debug = false
	return
}
