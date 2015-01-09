// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package mig

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"mig/pgp"
	"os"
	"strconv"
	"time"
)

// ActionVersion is the version of the syntax that is expected
const ActionVersion uint16 = 2

// an Action is the json object that is created by an investigator
// and provided to the MIG platform. It must be PGP signed.
type Action struct {
	ID             float64        `json:"id"`
	Name           string         `json:"name"`
	Target         string         `json:"target"`
	Description    Description    `json:"description,omitempty"`
	Threat         Threat         `json:"threat,omitempty"`
	ValidFrom      time.Time      `json:"validfrom"`
	ExpireAfter    time.Time      `json:"expireafter"`
	Operations     []Operation    `json:"operations"`
	PGPSignatures  []string       `json:"pgpsignatures"`
	Investigators  []Investigator `json:"investigators,omitempty"`
	Status         string         `json:"status,omitempty"`
	StartTime      time.Time      `json:"starttime,omitempty"`
	FinishTime     time.Time      `json:"finishtime,omitempty"`
	LastUpdateTime time.Time      `json:"lastupdatetime,omitempty"`
	Counters       ActionCounters `json:"counters,omitempty"`
	SyntaxVersion  uint16         `json:"syntaxversion,omitempty"`
}

// Some counters used to track the completion of an action
type ActionCounters struct {
	Sent      int `json:"sent,omitempty"`
	Done      int `json:"done,omitempty"`
	InFlight  int `json:"inflight,omitempty"`
	Success   int `json:"success,omitempty"`
	Cancelled int `json:"cancelled,omitempty"`
	Expired   int `json:"expired,omitempty"`
	Failed    int `json:"failed,omitempty"`
	TimeOut   int `json:"timeout,omitempty"`
}

// a description is a simple object that contains detail about the
// action's author, and it's revision.
type Description struct {
	Author   string  `json:"author,omitempty"`
	Email    string  `json:"email,omitempty"`
	URL      string  `json:"url,omitempty"`
	Revision float64 `json:"revision,omitempty"`
}

// a threat provides the investigator with an idea of how dangerous
// a the compromission might be, if the indicators return positive
type Threat struct {
	Ref    string `json:"ref,omitempty"`
	Level  string `json:"level,omitempty"`
	Family string `json:"family,omitempty"`
	Type   string `json:"type,omitempty"`
}

// an operation is an object that map to an agent module.
// the parameters of the operation are passed to the module as argument,
// and thus their format depend on the module itself.
type Operation struct {
	Module     string      `json:"module"`
	Parameters interface{} `json:"parameters"`
}

// ActionFromFile() reads an action from a local file on the file system
// and returns a mig.Action structure
func ActionFromFile(path string) (Action, error) {
	var err error
	var a Action
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.ActionFromFile(): %v", e)
		}
	}()
	// parse the json of the action into a mig.Action
	fd, err := ioutil.ReadFile(path)
	if err != nil {
		return a, err
	}
	err = json.Unmarshal(fd, &a)
	if err != nil {
		return a, err
	}

	return a, err
}

// ToTempFile writes an action into a generated temporary file and returns its filename
func (a Action) ToTempFile() (filename string, err error) {
	var (
		data []byte
		fd   *os.File
		fi   os.FileInfo
	)
	data, err = json.Marshal(a)
	if err != nil {
		return
	}
	fd, err = ioutil.TempFile("", "migaction_")
	defer fd.Close()
	if err != nil {
		return
	}
	_, err = fd.Write(data)
	if err != nil {
		return
	}
	fi, err = fd.Stat()
	if err != nil {
		return
	}
	filename = fmt.Sprintf("%s/%s", os.TempDir(), fi.Name())
	return
}

const MAXINT = int(^uint(0) >> 1)

// GenID returns an ID composed of a unix timestamp and a random CRC32
func GenID() float64 {
	h := crc32.NewIEEE()
	t := time.Now().UTC().Format(time.RFC3339Nano)
	r := rand.New(rand.NewSource(65537))
	rand := string(r.Intn(MAXINT))
	h.Write([]byte(t + rand))
	// concatenate timestamp and hash into 64 bits ID
	// id = <32 bits unix ts><32 bits CRC hash>
	uid := uint64(time.Now().Unix())
	uid = uid << 32
	sum := float64(h.Sum32())
	id := float64(uid) + sum
	return id
}

// GenHexID returns a string with an hexadecimal encoded ID
func GenB32ID() string {
	id := GenID()
	return strconv.FormatUint(uint64(id), 32)
}

// Validate verifies that the Action received contained all the
// necessary fields, and returns an error when it doesn't.
func (a Action) Validate() (err error) {
	if a.Name == "" {
		return errors.New("Action.Name is empty. Expecting string.")
	}
	if a.Target == "" {
		return errors.New("Action.Target is empty. Expecting string.")
	}
	if a.SyntaxVersion != ActionVersion {
		return fmt.Errorf("Wrong Syntax Version integer. Expection version %d", ActionVersion)
	}
	if a.ValidFrom.String() == "" {
		return errors.New("Action.ValidFrom is empty. Expecting string.")
	}
	if a.ExpireAfter.String() == "" {
		return errors.New("Action.ExpireAfter is empty. Expecting string.")
	}
	if a.ValidFrom.After(a.ExpireAfter) {
		return errors.New("Action.ExpireAfter is set before Action.ValidFrom.")
	}
	if time.Now().After(a.ExpireAfter) {
		return errors.New("Action.ExpireAfter is passed. Action has expired.")
	}
	if a.Operations == nil {
		return errors.New("Action.Operations is nil. Expecting string.")
	}
	if len(a.PGPSignatures) < 1 {
		return errors.New("Action.PGPSignatures is empty. Expecting array of strings.")
	}
	return
}

// Sign computes and returns the GPG signature of a MIG action in its stringified form
func (a Action) Sign(keyid string, secring io.Reader) (sig string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Sign() -> %v", e)
		}
	}()
	filename, err := a.ToTempFile()
	if err != nil {
		panic(err)
	}
	a2, err := ActionFromFile(filename)
	if err != nil {
		panic(err)
	}
	str, err := a2.String()
	if err != nil {
		panic(err)
	}
	sig, err = pgp.Sign(str, keyid, secring)
	if err != nil {
		panic(err)
	}
	return
}

// VerifySignatures verifies that the Action contains valid signatures from
// known investigators. It does not verify permissions.
func (a Action) VerifySignatures(keyring io.Reader) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("VerifySignatures() -> %v", e)
		}
	}()
	astr, err := a.String()
	if err != nil {
		return errors.New("Failed to stringify action")
	}
	for _, sig := range a.PGPSignatures {
		valid, _, err := pgp.Verify(astr, sig, keyring)
		if err != nil {
			return errors.New("Failed to verify PGP Signature")
		}
		if !valid {
			return errors.New("Invalid PGP Signature")
		}
	}
	return
}

//  concatenates Action components into a string
func (a Action) String() (str string, err error) {
	args, err := json.Marshal(a.Operations)
	if err != nil {
		return
	}
	str += fmt.Sprintf("name=%s;target=%s;validfrom=%d;expireafter=%s;operations=%s;",
		a.Name, a.Target, a.ValidFrom.UTC().Unix(), a.ExpireAfter.UTC().Unix(), args)
	return
}

// VerifyACL controls that an action has been issued by investigators
// that have the right permissions. This function looks at each operation
// listed in the action, and find the corresponding permission. If no
// permission is found, the default one `default` is used.
// The first permission that is found to apply to an operation, but
// doesn't allow the operation to run, will fail the verification globally
func (a Action) VerifyACL(acl ACL, keyring io.Reader) (err error) {
	// first, verify all signatures and get a list of PGP
	// fingerprints of the signers
	var fingerprints []string
	astr, err := a.String()
	if err != nil {
		return errors.New("Failed to stringify action")
	}
	for _, sig := range a.PGPSignatures {
		fp, err := pgp.GetFingerprintFromSignature(astr, sig, keyring)
		if err != nil {
			return fmt.Errorf("Failed to retrieve fingerprint from signatures: %v", err)
		}
		fingerprints = append(fingerprints, fp)
	}

	// Then, for each operation contained in the action, look for
	// a permission that apply to it, by comparing the operation name
	// with permission name. If none is found, use the default permission.
	for _, operation := range a.Operations {
		for _, permission := range acl {
			for permName, _ := range permission {
				if permName == operation.Module {
					return verifyPermission(operation, permName, permission, fingerprints)
				}
			}
		}
		// no specific permission found, apply the default permission
		var defaultPerm Permission
		for _, permission := range acl {
			for permName, _ := range permission {
				if permName == "default" {
					defaultPerm = permission
					break
				}
			}
		}
		return verifyPermission(operation, "default", defaultPerm, fingerprints)
	}
	return
}

// PrintCounters prints the counters of an action to stderr
func (a Action) PrintCounters() {
	out := fmt.Sprintf("%d sent, %d done", a.Counters.Sent, a.Counters.Done)
	if a.Counters.InFlight > 0 {
		out += fmt.Sprintf(", %d inflight", a.Counters.InFlight)
	}
	if a.Counters.Success > 0 {
		out += fmt.Sprintf(", %d succeeded", a.Counters.Success)
	}
	if a.Counters.Cancelled > 0 {
		out += fmt.Sprintf(", %d cancelled", a.Counters.Cancelled)
	}
	if a.Counters.Expired > 0 {
		out += fmt.Sprintf(", %d expired", a.Counters.Expired)
	}
	if a.Counters.Failed > 0 {
		out += fmt.Sprintf(", %d failed", a.Counters.Failed)
	}
	if a.Counters.TimeOut > 0 {
		out += fmt.Sprintf(", %d timed out", a.Counters.TimeOut)
	}
	fmt.Fprintf(os.Stderr, "%s\n", out)
}
