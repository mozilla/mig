/* Upgrade a MIG agent

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package upgrade

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"time"
)

type Parameters struct {
	Elements map[string]map[string]string `json:"elements"`
}

func NewParameters() (p Parameters) {
	return
}

type Results struct {
	Success    bool       `json:"success"`
	OldPID     int        `json:"oldpid"`
	Error      string     `json:"error,omitempty"`
	Statistics Statistics `json:"statistics,omitempty"`
}

func (p Parameters) Validate() (err error) {
	versionre := regexp.MustCompile(`^[a-z0-9]{7}-[0-9]{12}$`)
	locre := regexp.MustCompile(`^https?://`)
	checksumre := regexp.MustCompile(`^[a-zA-Z0-9]{64}$`)
	for k, el := range p.Elements {
		if !versionre.MatchString(el["to_version"]) {
			return fmt.Errorf("In %s, parameter 'to_version' with value '%s' is invalid. Expecting version.", k, el["to_version"])
		}
		if !locre.MatchString(el["location"]) {
			return fmt.Errorf("In %s, parameter 'location' with value '%s' is invalid. Expecting URL.", k, el["location"])
		}
		if !checksumre.MatchString(el["checksum"]) {
			return fmt.Errorf("In %s, parameter 'checksum' with value '%s' is invalid. Expecting SHA256 checksum.", k, el["checksum"])
		}
	}
	return
}

var stats Statistics

type Statistics struct {
	DownloadTime string `json:"downloadtime"`
	DownloadSize int64  `json:"downloadsize"`
}

func Run(Args []byte) string {
	p := NewParameters()

	err := json.Unmarshal(Args, &p.Elements)
	if err != nil {
		panic(err)
	}

	err = p.Validate()
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}

	// Extract the parameters that apply to this OS and Arch
	key := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	el, ok := p.Elements[key]
	if !ok {
		return buildResults(p, fmt.Sprintf("No parameter found for %s", key))
	}

	// Verify that the version we're told to upgrade to isn't the current one
	cversion, err := getCurrentVersion()
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}
	fmt.Println(cversion)
	if cversion == el["to_version"] {
		return buildResults(p, fmt.Sprintf("Agent is already running version '%s'", cversion))
	}

	// Download new agent binary from provided location
	binfd, err := downloadBinary(el["location"])
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}

	// Verify checksum of the binary
	err = verifyChecksum(binfd, el["checksum"])
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}

	// grab the path before closing the file descriptor
	binPath := binfd.Name()

	err = binfd.Close()
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}

	// Dry run of the binary to verify that the version is correct
	// but also that it can run without error
	err = verifyVersion(binPath, el["to_version"])
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}

	// Move the binary of the new agent from tmp, to the correct destination
	agentBinPath, err := moveBinary(binPath, el["to_version"])
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}

	// Launch the new agent and exit the module
	_, err = exec.Command(agentBinPath).Output()
	if err != nil {
		return buildResults(p, fmt.Sprintf("%v", err))
	}

	return buildResults(p, "")
}

// Run the agent binary to obtain the current version
func getCurrentVersion() (cversion string, err error) {
	cdir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	bin := cdir + "/" + os.Args[0]
	out, err := exec.Command(bin, "-V").Output()
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
	os.Chmod(binPath, 0750)
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
		target = fmt.Sprintf("C:/Windows/mig-agent-%s.exe", version)
		linkloc = "C:/Windows/mig-agent"
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
func buildResults(params Parameters, errors string) string {
	var results Results
	results.OldPID = os.Getppid()
	if errors != "" {
		results.Error = errors
	} else {
		results.Success = true
	}
	results.Statistics = stats
	jsonOutput, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
