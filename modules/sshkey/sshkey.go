// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

// Package sshkey implements the sshkey module in the agent
package sshkey /* import "mig.ninja/mig/modules/sshkey" */

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"mig.ninja/mig/modules"
	"mig.ninja/mig/modules/file"
)

// candidateRegex is the default content search regex we apply to locate private and
// public SSH keys
var candidateRegex = "^(-----BEGIN (RSA|DSA) PRIVATE KEY|ssh-(rsa|dss) )"

// Constants used in KeyInfo.Type
const (
	KeyTypePrivate        = "private"
	KeyTypePublic         = "public"
	KeyTypeAuthorizedKeys = "authorizedkeys"
)

// softErrors stores any errors that are not fatal we want to report back to the investigator
var softErrors []string

type module struct {
}

// NewRun returns a new instance of a modules.Runner for this module
func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("sshkey", new(module))
}

type run struct {
	Parameters Parameters
	Results    modules.Result
}

func buildResults(e Elements, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	r.Elements = e
	r.Errors = append(r.Errors, softErrors...)
	if len(e.Keys) > 0 {
		r.FoundAnything = true
	}
	buf, err = json.Marshal(r)
	return
}

// Run is the main module execution function. in is a type modules.ModuleReader from which the
// module input parameters are read. The results are returned as modules.Result in JSON format.
func (r *run) Run(in modules.ModuleReader) (resStr string) {
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, softErrors...)
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			err, _ := json.Marshal(r.Results)
			resStr = string(err)
			return
		}
	}()

	// Restrict go runtime processor utilization here, this might be moved
	// into a more generic agent module function at some point.
	runtime.GOMAXPROCS(1)

	// Read module parameters
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	e := &Elements{}

	searchpaths := defaultSearchPaths()
	if len(r.Parameters.Paths) > 0 {
		searchpaths = r.Parameters.Paths
	}
	maxdepth := 8 // Default
	if r.Parameters.MaxDepth != 0 {
		maxdepth = r.Parameters.MaxDepth
	}
	// First find a list of all possible candidate files, we want to try to identify
	// private and public key files
	cands, err := findCandidates(searchpaths, maxdepth)
	if err != nil {
		panic(err)
	}
	for _, x := range cands {
		processCandidate(x, e)
	}

	buf, err := buildResults(*e, &r.Results)
	if err != nil {
		panic(err)
	}
	resStr = string(buf)
	return
}

// ValidateParameters ensures the parameters being provided to the module are correct
func (r *run) ValidateParameters() (err error) {
	if r.Parameters.MaxDepth < 0 || r.Parameters.MaxDepth > 1000 {
		return errors.New("maxdepth should be specified from 0-1000")
	}
	return nil
}

// PrintResults returns a list of strings representing the formatted results output for the module
func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		elem Elements
	)

	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}
	for _, x := range elem.Keys {
		ln := fmt.Sprintf("%v [%v]", x.Path, x.Type)
		if x.Type == KeyTypePrivate {
			ln += fmt.Sprintf(" encrypted=%v", x.Encrypted)
		}
		md5fp := "unknown"
		sha256fp := "unknown"
		if x.FingerprintMD5 != "" {
			md5fp = x.FingerprintMD5
		}
		if x.FingerprintSHA256 != "" {
			sha256fp = x.FingerprintSHA256
		}
		ln += fmt.Sprintf(" md5fp=%v sha256fp=%v", md5fp, sha256fp)
		prints = append(prints, ln)
	}
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
	}
	return
}

// findCandidates is the function used for the first pass by the module, it locates any files
// in our default search paths that match candidateRegex, the file module is used to conduct
// the walk of the file system
func findCandidates(paths []string, maxdepth int) (ret []string, err error) {
	run := modules.Available["file"].NewRun()
	args := make([]string, 0)
	for _, x := range paths {
		args = append(args, "-path", x)
	}
	args = append(args, "-content", candidateRegex)
	args = append(args, "-maxdepth", strconv.Itoa(maxdepth))
	args = append(args, "-size", "<1m")
	param, err := run.(modules.HasParamsParser).ParamsParser(args)
	if err != nil {
		return ret, err
	}

	buf, err := modules.MakeMessage(modules.MsgClassParameters, param, false)
	if err != nil {
		return ret, err
	}
	rdr := modules.NewModuleReader(bytes.NewReader(buf))

	res := run.Run(rdr)
	var modresult modules.Result
	var sr file.SearchResults
	err = json.Unmarshal([]byte(res), &modresult)
	if err != nil {
		return ret, err
	}
	err = modresult.GetElements(&sr)
	if err != nil {
		return ret, err
	}

	p0, ok := sr["s1"]
	if !ok {
		return ret, errors.New("result in file module call was missing")
	}
	for _, x := range p0 {
		if x.File == "" {
			continue
		}
		ret = append(ret, x.File)
	}

	return ret, nil
}

// processCandidate is called for each candidate file which was found, and it attempts
// to determine if the file is a private or public key, adding any relevant information
// about the file to the Elements
func processCandidate(path string, e *Elements) {
	ski, err := checkPrivateKey(path)
	if err == nil {
		e.Keys = append(e.Keys, ski)
		return
	}
	skilist, err := checkPublicKey(path)
	if err == nil {
		e.Keys = append(e.Keys, skilist...)
		return
	}
}

// checkPrivateKey will check if the file is a private key, and if so populate ret
// otherwise return an error
func checkPrivateKey(path string) (ret KeyInfo, err error) {
	ret.Path = path

	// Try to load the candidate as a private key, if successful we have enough information
	// to process a fingerprint from the file.
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	signer, err := ssh.ParsePrivateKey(buf)
	if err == nil {
		// Parsing was successful, get a fingerprint and return
		ret.FingerprintMD5 = ssh.FingerprintLegacyMD5(signer.PublicKey())
		ret.FingerprintSHA256 = ssh.FingerprintSHA256(signer.PublicKey())
		ret.Type = KeyTypePrivate
		return
	}
	// Unable to parse, see if the file was encrypted; if so we will note it's existence
	if strings.Contains(err.Error(), "cannot decode encrypted private keys") {
		ret.Encrypted = true
		ret.Type = KeyTypePrivate
		return ret, nil
	}

	return
}

// checkPublicKey will check if a file is a public key or an authorized keys file. In the
// first case, ret will contain key info for a single key. In the second case, ret may contain
// one or more key info entries. An error is returned if the candidate is not parsable as a
// public key.
func checkPublicKey(path string) (ret []KeyInfo, err error) {
	isAuthKeys := false

	if filepath.Base(path) == "authorized_keys" || filepath.Base(path) == "authorized_keys2" {
		isAuthKeys = true
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	if !isAuthKeys {
		ski := KeyInfo{Path: path}
		pubkey, _, _, _, err := ssh.ParseAuthorizedKey(buf)
		if err == nil {
			// Parsing was successful, get a fingerprint and return
			ski.FingerprintMD5 = ssh.FingerprintLegacyMD5(pubkey)
			ski.FingerprintSHA256 = ssh.FingerprintSHA256(pubkey)
			ski.Type = KeyTypePublic
			ret = append(ret, ski)
			return ret, nil
		}
		return ret, err
	}
	// Treat this as an authorized keys file, we will line delimit the buffer and parse each
	// entry, creating a key info value for each
	sbuf := bytes.Split(buf, []byte("\n"))
	parseerror := false
	for _, x := range sbuf {
		if len(x) == 0 {
			// Ignore any extra line feeds at the end of the file
			continue
		}
		ski := KeyInfo{Path: path}
		pubkey, _, _, _, err := ssh.ParseAuthorizedKey(x)
		if err == nil {
			ski.FingerprintMD5 = ssh.FingerprintLegacyMD5(pubkey)
			ski.FingerprintSHA256 = ssh.FingerprintSHA256(pubkey)
			ski.Type = KeyTypeAuthorizedKeys
			ret = append(ret, ski)
		} else {
			if !parseerror {
				softErrors = append(softErrors, fmt.Sprintf("%v contained unparsable data", path))
				// Only add this error message once per file
				parseerror = true
			}
		}
	}
	return ret, nil
}

func defaultSearchPaths() []string {
	switch runtime.GOOS {
	case "linux", "darwin":
		return []string{
			"/root",
			"/home",
		}
	case "windows":
		return []string{
			"c:\\Users",
		}
	default:
		return []string{}
	}
}

// KeyInfo describes information about a key that has been identified
type KeyInfo struct {
	FingerprintMD5    string `json:"fingerprint_md5"`    // MD5 fingerprint
	FingerprintSHA256 string `json:"fingerprint_sha256"` // SHA256 fingerprint
	Path              string `json:"path"`               // Path to file
	Encrypted         bool   `json:"encrypted"`          // True if private key is encrypted
	Type              string `json:"type"`               // Type of file (e.g., private, public)
}

// Elements is the type that contains the results of a module invocation
type Elements struct {
	Keys []KeyInfo `json:"keys"`
}

// parameters contains the parameters used to control how the module executes a given
// action
type Parameters struct {
	Paths    []string `json:"paths"`    // Used to override default module search paths
	MaxDepth int      `json:"maxdepth"` // Override default maximum search depth
}

func newParameters() *Parameters {
	return &Parameters{}
}
