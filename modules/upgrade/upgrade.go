// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// The upgrade module is used to download and install a new version of the
// mig-agent. It retrieves a binary from an HTTP location, validates its
// checksum. Verifies that the binary version is different from the currently
// running version. Install the binary and run it.
//
// At the end of the run, two mig-agent will be running on the same endpoint,
// and the scheduler will take care of killing one of them. This module does
// not attempt to kill the current mig-agent, in case the new one does not
// connect properly.
package upgrade /* import "mig.ninja/mig/modules/upgrade" */

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/kardianos/osext"
	"hash"
	"io"
	"io/ioutil"
	"mig.ninja/mig/modules"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"time"
)

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("upgrade", new(module))
}

type run struct {
	Parameters params
	Results    modules.Result
}

// JSON sample:
//        {
//            "module": "upgrade",
//            "parameters": {
//                "to_version": "201403031435-b9536d2",
//                "location": "https://download.mig.example.net/mig-agent-b9536d2-201403031435",
//                "checksum": "c59d4eaeac728671c635ff645014e2afa935bebffdb5fbd207ffdeab"
//            }
//        }
type params map[string]map[string]string

type elements struct {
	OldPID int `json:"oldpid"`
}

var stats statistics

type statistics struct {
	DownloadTime string `json:"downloadtime"`
	DownloadSize int64  `json:"downloadsize"`
}

func (r run) ValidateParameters() (err error) {
	locre := regexp.MustCompile(`^https?://`)
	checksumre := regexp.MustCompile(`^[a-zA-Z0-9]{64}$`)
	for k, v := range r.Parameters {
		if v["to_version"] == "" {
			return fmt.Errorf("In %s, parameter 'to_version' is empty. Expecting version.", k, v["to_version"])
		}
		if !locre.MatchString(v["location"]) {
			return fmt.Errorf("In %s, parameter 'location' with value '%s' is invalid. Expecting URL.", k, v["location"])
		}
		if !checksumre.MatchString(v["checksum"]) {
			return fmt.Errorf("In %s, parameter 'checksum' with value '%s' is invalid. Expecting SHA256 checksum.", k, v["checksum"])
		}
	}
	return
}

func (r run) Run(in io.Reader) (out string) {
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			buf, _ := json.Marshal(r.Results)
			out = string(buf[:])
		}
	}()
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	// Extract the parameters that apply to this OS and Arch
	key := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	el, ok := r.Parameters[key]
	if !ok {
		panic("no upgrade instruction found for " + key)
	}

	// Verify that the version we're told to upgrade to isn't the current one
	cversion, err := getCurrentVersion()
	if err != nil {
		panic(err)
	}
	if cversion == el["to_version"] {
		panic("Agent is already running version " + cversion)
	}

	// Download new agent binary from provided location
	binfd, err := downloadBinary(el["location"])
	if err != nil {
		panic(err)
	}

	// Verify checksum of the binary
	err = verifyChecksum(binfd, el["checksum"])
	if err != nil {
		panic(err)
	}

	// grab the path before closing the file descriptor
	binPath := binfd.Name()

	err = binfd.Close()
	if err != nil {
		panic(err)
	}

	// Dry run of the binary to verify that the version is correct
	// but also that it can run without error
	err = verifyVersion(binPath, el["to_version"])
	if err != nil {
		panic(err)
	}

	// Move the binary of the new agent from tmp, to the correct destination
	agentBinPath, err := moveBinary(binPath, el["to_version"])
	if err != nil {
		panic(err)
	}

	// Launch the new agent and exit the module
	cmd := exec.Command(agentBinPath, "-u")
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	out = r.buildResults()
	return
}

// Run the agent binary to obtain the current version
func getCurrentVersion() (cversion string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getCurrentVersion() -> %v", e)
		}
	}()
	bin, err := osext.Executable()
	if err != nil {
		panic(err)
	}
	out, err := exec.Command(bin, "-V").Output()
	if err != nil {
		panic(err)
	}
	if len(out) < 2 {
		panic("Failed to retrieve agent version.")
	}
	cversion = string(out[:len(out)-1])
	return
}

// downloadBinary retrieves the data from a location and saves it to a temp file
func downloadBinary(loc string) (tmpfd *os.File, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("downloadBinary() -> %v", e)
		}
	}()
	tmpfd, err = ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	start := time.Now()
	resp, err := http.Get(loc)
	if err != nil {
		panic(err)
	}
	stats.DownloadSize, err = io.Copy(tmpfd, resp.Body)
	stats.DownloadTime = time.Since(start).String()
	resp.Body.Close()
	return
}

// verifyChecksum computes the hash of a file and compares it
// to a checksum. If comparison fails, it returns an error.
func verifyChecksum(fd *os.File, checksum string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("verifyChecksum() -> %v", e)
		}
	}()
	var h hash.Hash
	h = sha256.New()
	buf := make([]byte, 4096)
	var offset int64 = 0
	for {
		block, err := fd.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if block == 0 {
			break
		}
		h.Write(buf[:block])
		offset += int64(block)
	}
	hexhash := fmt.Sprintf("%x", h.Sum(nil))
	if hexhash != checksum {
		return fmt.Errorf("Checksum validation failed. Got '%s', Expected '%s'.",
			hexhash, checksum)
	}
	return
}

// verifyVersion runs a binary and compares the returned version
func verifyVersion(binPath, expectedVersion string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("verifyVersion() -> %v", e)
		}
	}()
	os.Chmod(binPath, 0500)
	out, err := exec.Command(binPath, "-V").Output()
	if err != nil {
		panic(err)
	}
	binVersion := string(out[:len(out)-1])
	if binVersion != expectedVersion {
		return fmt.Errorf("Version mismatch. Got '%s', Expected '%s'.",
			binVersion, expectedVersion)
	}
	return
}

func moveBinary(binPath, version string) (linkloc string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("moveBinary() -> %v", e)
		}
	}()
	var target string
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		target = fmt.Sprintf("/sbin/mig-agent-%s", version)
		linkloc = "/sbin/mig-agent"
	case "windows":
		target = fmt.Sprintf("C:\\Program Files\\mig\\mig-agent-%s.exe", version)
		linkloc = "C:\\Program Files\\mig\\mig-agent.exe"
	default:
		err = fmt.Errorf("'%s' isn't a supported OS", runtime.GOOS)
		return
	}
	// copy the file (rename may not work if we're crossing partitions)
	srcfd, err := os.Open(binPath)
	if err != nil {
		panic(err)
	}
	dstfd, err := os.Create(target)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(dstfd, srcfd)
	if err != nil {
		panic(err)
	}
	srcfd.Close()
	dstfd.Close()
	err = os.Remove(binPath)
	if err != nil {
		panic(err)
	}
	err = os.Chmod(target, 0750)
	if err != nil {
		panic(err)
	}
	// don't fail on removal of existing link, it may not exist
	os.Remove(linkloc)
	// create a symlink
	err = os.Symlink(target, linkloc)
	if err != nil {
		panic(err)
	}
	return
}

// buildResults transforms the ConnectedIPs map into a Results
// map that is serialized in JSON and returned as a string
func (r run) buildResults() string {
	var el elements
	el.OldPID = os.Getppid()
	r.Results.Elements = el
	if len(r.Results.Errors) == 0 {
		r.Results.Success = true
	}
	r.Results.Statistics = stats
	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
