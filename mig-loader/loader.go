// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

// The MIG loader is a simple bootstrapping tool for MIG. It can be scheduled
// to run on a host system and download the newest available version of the
// agent. If the loader identifies a newer version of the agent available, it
// will download the required files from the API, replace the existing files,
// and notify any existing agent it should terminate.
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/jvehent/cljs"
	"io"
	"io/ioutil"
	"mig.ninja/mig"
	"mig.ninja/mig/mig-agent/agentcontext"
	"mig.ninja/mig/pgp"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
)

var ctx Context
var haveChanges bool
var apiManifest *mig.ManifestResponse
var wg sync.WaitGroup

func initializeHaveBundle() (ret []mig.BundleDictionaryEntry, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initializeHaveBundle() -> %v", e)
		}
	}()

	ret, err = mig.GetHostBundle()
	if err != nil {
		panic(err)
	}
	ret, err = mig.HashBundle(ret)
	if err != nil {
		panic(err)
	}
	logInfo("initialized local bundle information")
	for _, x := range ret {
		hv := x.SHA256
		if hv == "" {
			hv = "not found"
		}
		logInfo("%v %v -> %v", x.Name, x.Path, hv)
	}
	return
}

func requestManifest() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("requestManifest() -> %v", e)
		}
	}()

	murl := APIURL + "manifest/agent/"
	logInfo("requesting manifest from %v", murl)

	mparam := mig.ManifestParameters{}
	mparam.AgentIdentifier = ctx.AgentIdentifier
	buf, err := json.Marshal(mparam)
	if err != nil {
		panic(err)
	}
	mstring := string(buf)
	data := url.Values{"parameters": {mstring}}
	r, err := http.NewRequest("POST", murl, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("X-LOADERKEY", ctx.LoaderKey)
	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP %v, API call failed with error '%v' (code %s)", resp.StatusCode,
			resource.Collection.Error.Message, resource.Collection.Error.Code)
		panic(err)
	}

	// Extract our manifest from the response.
	manifest, err := valueToManifest(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}
	apiManifest = &manifest
	err = apiManifest.Validate()
	if err != nil {
		panic(err)
	}
	return checkManifestSignature(apiManifest, MANIFESTPGPKEYS[:])
}

func valueToManifest(v interface{}) (m mig.ManifestResponse, err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &m)
	return
}

func valueToFetchResponse(v interface{}) (m mig.ManifestFetchResponse, err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, &m)
	return
}

func fetchFile(n string) (ret []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("fetchFile() -> %v", e)
		}
	}()

	murl := APIURL + "manifest/fetch/"

	logInfo("fetching file from %v", murl)
	mparam := mig.ManifestParameters{}
	mparam.AgentIdentifier = ctx.AgentIdentifier
	mparam.Object = n
	buf, err := json.Marshal(mparam)
	if err != nil {
		panic(err)
	}
	mstring := string(buf)
	data := url.Values{"parameters": {mstring}}
	r, err := http.NewRequest("POST", murl, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("X-LOADERKEY", ctx.LoaderKey)
	client := http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var resource *cljs.Resource
	err = json.Unmarshal(body, &resource)
	if err != nil {
		panic(err)
	}

	// Extract fetch response.
	fetchresp, err := valueToFetchResponse(resource.Collection.Items[0].Data[0].Value)
	if err != nil {
		panic(err)
	}

	// Decompress the returned file and return it as a byte slice.
	b := bytes.NewBuffer(fetchresp.Data)
	gz, err := gzip.NewReader(b)
	if err != nil {
		panic(err)
	}
	ret, err = ioutil.ReadAll(gz)
	if err != nil {
		panic(err)
	}
	return
}

func fetchAndReplace(entry mig.BundleDictionaryEntry, sig string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("fetchAndReplace() -> %v", e)
		}
	}()

	// Grab the new file from the API.
	filebuf, err := fetchFile(entry.Name)
	if err != nil {
		panic(err)
	}

	// Stage the new file. Write the file recieved from the API to the
	// file system and validate the signature of the new file to make
	// sure it matches the signature from the manifest.
	//
	// Append .loader to the file name to use as the staged file path.
	reppath := entry.Path + ".loader"
	oldpath := entry.Path + ".old"
	fd, err := os.OpenFile(reppath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		entry.Perm)
	if err != nil {
		panic(err)
	}
	_, err = fd.Write(filebuf)
	if err != nil {
		panic(err)
	}
	fd.Close()

	// Validate the signature on the new file.
	logInfo("validating staged file signature")
	h := sha256.New()
	fd, err = os.Open(reppath)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 4096)
	for {
		n, err := fd.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fd.Close()
			panic(err)
		}
		if n > 0 {
			h.Write(buf[:n])
		}
	}
	fd.Close()
	if sig != fmt.Sprintf("%x", h.Sum(nil)) {
		panic("staged file signature mismatch")
	}

	// Got this far, OK to proceed with the replacement.
	// Rename existing file first
	logInfo("renaming existing file")
	os.Rename(entry.Path, oldpath)
	// Replace target file
	logInfo("installing staged file")
	err = os.Rename(reppath, entry.Path)
	if err != nil {
		panic(err)
	}
	return
}

func checkEntry(entry mig.BundleDictionaryEntry) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("checkEntry() -> %v", e)
		}
	}()

	var compare mig.ManifestEntry
	logInfo("comparing %v %v", entry.Name, entry.Path)
	found := false
	for _, x := range apiManifest.Entries {
		if x.Name == entry.Name {
			compare = x
			found = true
			break
		}
	}
	if !found {
		logInfo("entry not in API manifest, ignoring")
		return
	}
	hv := entry.SHA256
	if hv == "" {
		hv = "not found"
	}
	logInfo("we have %v", hv)
	logInfo("they have %v", compare.SHA256)
	if entry.SHA256 == compare.SHA256 {
		logInfo("we have correct file, no need to replace")
		return entryPermCheck(entry)
	}
	haveChanges = true
	logInfo("refreshing %v", entry.Name)
	err = fetchAndReplace(entry, compare.SHA256)
	if err != nil {
		panic(err)
	}
	return
}

// Check the file permissions set on a bundle entry and if they are
// incorrect fix the permissions
func entryPermCheck(entry mig.BundleDictionaryEntry) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("entryPermCheck() -> %v", e)
		}
	}()
	var fi os.FileInfo
	fi, err = os.Stat(entry.Path)
	if err != nil {
		panic(err)
	}
	if fi.Mode() != entry.Perm {
		logInfo("%v has incorrect permissions, fixing", entry.Path)
		err = os.Chmod(entry.Path, entry.Perm)
		if err != nil {
			panic(err)
		}
	}
	return
}

// Compare the manifest that the API sent with our knowledge of what is
// currently installed. For each case there is a difference, we will
// request the new file and replace the existing entry.
func compareManifest(have []mig.BundleDictionaryEntry) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("compareManifest() -> %v", e)
		}
	}()

	for _, x := range have {
		err := checkEntry(x)
		if err != nil {
			panic(err)
		}
	}
	return
}

func checkManifestSignature(mr *mig.ManifestResponse, keylist []string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("checkManifestSignature() -> %v", e)
		}
	}()

	var keys [][]byte
	for _, pk := range keylist {
		keys = append(keys, []byte(pk))
	}
	keyring, _, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		panic(err)
	}
	cnt, err := mr.VerifySignatures(keyring)
	if err != nil {
		panic(err)
	}
	logInfo("%v valid signatures in manifest", cnt)
	if cnt < REQUIREDSIGNATURES {
		err = fmt.Errorf("Not enough valid signatures (got %v, need %v), rejecting",
			cnt, REQUIREDSIGNATURES)
		panic(err)
	}

	return
}

func initContext() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initContext() -> %v", e)
		}
	}()

	ctx.Channels.Log = make(chan mig.Log, 37)
	ctx.Logging, err = getLoggingConf()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	ctx.Logging, err = mig.InitLogger(ctx.Logging, "mig-loader")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	wg.Add(1)
	go func() {
		var stop bool
		for event := range ctx.Channels.Log {
			// Also write the message to stderr to ease debugging
			fmt.Fprintf(os.Stderr, "%v\n", event.Desc)
			stop, err = mig.ProcessLog(ctx.Logging, event)
			if err != nil {
				panic("unable to process log")
			}
			if stop {
				break
			}
		}
		wg.Done()
	}()
	logInfo("logging routine started")

	ctx.LoaderKey = LOADERKEY

	hints := agentcontext.AgentContextHints{
		DiscoverPublicIP: DISCOVERPUBLICIP,
		DiscoverAWSMeta:  DISCOVERAWSMETA,
		APIUrl:           APIURL,
		Proxies:          PROXIES[:],
	}
	actx, err := agentcontext.NewAgentContext(ctx.Channels.Log, hints)
	if err != nil {
		panic(err)
	}
	ctx.AgentIdentifier = actx.ToAgent()
	ctx.AgentIdentifier.Tags = TAGS

	return
}

func logInfo(s string, args ...interface{}) {
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf(s, args...)}.Info()
}

func logError(s string, args ...interface{}) {
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf(s, args...)}.Err()
}

func doExit(v int) {
	close(ctx.Channels.Log)
	wg.Wait()
	os.Exit(v)
}

// Return the path to the expected loader key file location
func getLoaderKeyfile() string {
	switch runtime.GOOS {
	case "linux":
		return "/etc/mig/mig-loader.key"
	case "darwin":
		return "/etc/mig/mig-loader.key"
	}
	panic("loader does not support this operating system")
	return ""
}

func loaderInitializePathLinux() error {
	path := os.Getenv("PATH")
	if path != "" {
		path = path + ":"
	}
	path = path + "/bin:/sbin:/usr/bin:/usr/sbin"
	return os.Setenv("PATH", path)
}

func loaderInitializePathDarwin() error {
	path := os.Getenv("PATH")
	if path != "" {
		path = path + ":"
	}
	path = path + "/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin"
	return os.Setenv("PATH", path)
}

func loaderInitializePath() error {
	switch runtime.GOOS {
	case "linux":
		return loaderInitializePathLinux()
	case "darwin":
		return loaderInitializePathDarwin()
	}
	return fmt.Errorf("loader does not support this operating system")
}

// Attempt to obtain the loader key from the file system and override the
// built-in secret
func loadLoaderKey() error {
	fd, err := os.Open(getLoaderKeyfile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer fd.Close()
	buf, _, err := bufio.NewReader(fd).ReadLine()
	if err != nil {
		// Nothing in the loader key file
		if err == io.EOF {
			return nil
		}
		return err
	}
	// Also trim any leading and trailing spaces from the loader key
	LOADERKEY = strings.Trim(string(buf), " ")
	err = mig.ValidateLoaderPrefixAndKey(LOADERKEY)
	if err != nil {
		return err
	}
	ctx.LoaderKey = LOADERKEY
	return nil
}

func main() {
	var (
		initialMode bool
		err         error
	)
	runtime.GOMAXPROCS(1)

	flag.BoolVar(&initialMode, "i", false, "initialization mode")
	flag.Parse()

	err = initContext()
	if err != nil {
		logError("%v", err)
		doExit(1)
	}

	err = loaderInitializePath()
	if err != nil {
		logError("%v", err)
		doExit(1)
	}
	err = loadLoaderKey()
	if err != nil {
		logError("%v", err)
		doExit(1)
	}

	// Do any service installation that might be required for this platform
	if initialMode {
		err = serviceDeploy()
		if err != nil {
			logError("%v", err)
			doExit(1)
		}
	}

	// Get our current status from the file system.
	have, err := initializeHaveBundle()
	if err != nil {
		logError("%v", err)
		doExit(1)
	}

	// Retrieve our manifest from the API.
	err = requestManifest()
	if err != nil {
		logError("%v", err)
		doExit(1)
	}

	err = compareManifest(have)
	if err != nil {
		logError("%v", err)
		doExit(1)
	}

	if haveChanges {
		err = runTriggers()
		if err != nil {
			logError("%v", err)
			doExit(1)
		}
	} else {
		// If we don't have changes, just validate the agent is running,
		// if it is not we will also execute the triggers to try to
		// bump it.
		err = agentRunning()
		if err != nil {
			logInfo("agent does not appear to be running, trying to start")
			err = runTriggers()
			if err != nil {
				logError("%v", err)
				doExit(1)
			}
		} else {
			logInfo("agent looks like it is running")
		}
	}
	doExit(0)
}
