// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

package mig /* import "mig.ninja/mig" */

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"mig.ninja/mig/pgp"
)

// ActionVersion is the version of the syntax that is expected
const ActionVersion uint16 = 2

// Action is the json object that is created by an investigator
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

// ActionCounters are counters used to track the completion of an action
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

// Description is a simple object that contains detail about the
// action's author, and it's revision.
type Description struct {
	Author   string  `json:"author,omitempty"`
	Email    string  `json:"email,omitempty"`
	URL      string  `json:"url,omitempty"`
	Revision float64 `json:"revision,omitempty"`
}

// Threat provides the investigator with details on a threat indicator
// if included in an action
type Threat struct {
	Ref    string `json:"ref,omitempty"`
	Level  string `json:"level,omitempty"`
	Family string `json:"family,omitempty"`
	Type   string `json:"type,omitempty"`
}

// Operation is an object that maps to an agent module.
// The parameters of the operation are passed to the module as an argument,
// and thus their format depends on the module itself.
type Operation struct {
	Module     string      `json:"module"`
	Parameters interface{} `json:"parameters"`

	// If WantCompressed is set in the operation, the parameters
	// will be compressed in PostAction() when the client sends the
	// action to the API. This will also result in IsCompressed being
	// marked as true, so the receiving agent knows it must decompress
	// the parameter data.
	IsCompressed   bool `json:"is_compressed,omitempty"`
	WantCompressed bool `json:"want_compressed,omitempty"`
}

// CompressOperationParam compresses the parameters stored within an operation
func (op *Operation) CompressOperationParam() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("CompressOperationParam() -> %v", e)
		}
	}()
	jb, err := json.Marshal(op.Parameters)
	if err != nil {
		panic(err)
	}
	var b bytes.Buffer
	wb64 := base64.NewEncoder(base64.StdEncoding, &b)
	w := gzip.NewWriter(wb64)
	_, err = w.Write(jb)
	if err != nil {
		panic(err)
	}
	w.Close()
	wb64.Close()
	op.Parameters = string(b.Bytes())
	op.IsCompressed = true
	return
}

// DecompressOperationParam decompresses the parameters stored within an operation
func (op *Operation) DecompressOperationParam() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("DecompressOperationParam() -> %v", e)
		}
	}()
	if !op.IsCompressed {
		return nil
	}
	pstr, ok := op.Parameters.(string)
	if !ok {
		panic("Compressed parameter was not a string")
	}
	b := bytes.NewBuffer([]byte(pstr))
	rb64 := base64.NewDecoder(base64.StdEncoding, b)
	r, err := gzip.NewReader(rb64)
	if err != nil {
		panic(err)
	}
	rb, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(rb, &op.Parameters)
	if err != nil {
		panic(err)
	}
	op.IsCompressed = false
	return
}

// ActionFromFile reads an action from a local file on the file system
// and returns a mig.Action structure
func ActionFromFile(path string) (Action, error) {
	var err error
	var a Action
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("ActionFromFile() -> %v", e)
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

type id struct {
	value float64
	sync.Mutex
}

var globalID id

// GenID returns a float64 ID number that is unique to this process. The ID is initialized
// at the number of seconds since MIG's creation date, shifted 16 bits to the right and incremented
// by one every time a new ID is requested. The resulting value must fit in 53 bits of precision
// provided by the float64 type.
func GenID() float64 {
	globalID.Lock()
	defer globalID.Unlock()
	if globalID.value < 1 {
		// if id hasn't been initialized yet, set it to number of seconds since
		// MIG's inception, plus one
		tmpid := int64(time.Since(time.Unix(1367258400, 0)).Seconds() + 1)
		tmpid = tmpid << 16
		globalID.value = float64(tmpid)
		return globalID.value
	}
	globalID.value++
	return globalID.value
}

// Validate verifies that the Action received contained all the
// necessary fields, and returns an error when it doesn't.
func (a Action) Validate() (err error) {
	if a.Name == "" {
		return errors.New("action name is empty")
	}
	if a.Target == "" {
		return errors.New("action target is empty")
	}
	if a.SyntaxVersion != ActionVersion {
		return fmt.Errorf("wring syntax version, expected %v", ActionVersion)
	}
	if a.ValidFrom.String() == "" {
		return errors.New("action validfrom is empty")
	}
	if a.ExpireAfter.String() == "" {
		return errors.New("action expireafter is empty")
	}
	if a.ValidFrom.After(a.ExpireAfter) {
		return errors.New("action expireafter is set before action validfrom")
	}
	if time.Now().After(a.ExpireAfter) {
		return errors.New("action expireafter is in the past")
	}
	if a.Operations == nil {
		return errors.New("action operations is empty")
	}
	if len(a.PGPSignatures) < 1 {
		return errors.New("action pgpsignatures is empty")
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
	err = os.Remove(filename)
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
		return errors.New("failed to stringify action")
	}
	// If the action does not contain any signatures, treat this as an error condition
	if a.PGPSignatures == nil || len(a.PGPSignatures) == 0 {
		return errors.New("action contained no valid signatures")
	}
	// Create a copy of the keyring we can use during validation of each
	// signature. We don't want to use the keyring reader directly as it is
	// backed by a buffer and will be drained after verification of the first
	// signature.
	keycopy, err := ioutil.ReadAll(keyring)
	if err != nil {
		panic(err)
	}
	for _, sig := range a.PGPSignatures {
		keyrdr := bytes.NewBuffer(keycopy)
		valid, _, err := pgp.Verify(astr, sig, keyrdr)
		if err != nil {
			return errors.New("failed to verify PGP Signature")
		}
		if !valid {
			return errors.New("invalid PGP Signature")
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

// VerifyACL validates that an action has been issued by investigators that have adequate
// permissions to execute the action. It takes a look at each operation included in the
// action and verifies the signatures in the action against the agents ACL.
func (a Action) VerifyACL(acl ACL, keyring io.Reader, onlyVerifyPubKey bool) (err error) {
	// first, verify all signatures and get a list of PGP
	// fingerprints of the signers
	var fingerprints []string
	astr, err := a.String()
	if err != nil {
		return errors.New("failed to stringify action")
	}
	// Create a copy of the keyring we can use during validation of each
	// signature. We don't want to use the keyring reader directly as it is
	// backed by a buffer and will be drained after verification of the first
	// signature.
	keycopy, err := ioutil.ReadAll(keyring)
	if err != nil {
		panic(err)
	}
	for _, sig := range a.PGPSignatures {
		keyrdr := bytes.NewBuffer(keycopy)
		fp, err := pgp.GetFingerprintFromSignature(astr, sig, keyrdr)
		if err != nil {
			return fmt.Errorf("failed to retrieve fingerprint from signatures: %v", err)
		}
		fingerprints = append(fingerprints, fp)
	}

	if len(fingerprints) == 0 {
		return errors.New("no valid fingerprints found")
	}

	if onlyVerifyPubKey {
		return
	}

	// Authorize access to the operation by verifying the fingerprints present against our
	// ACLs
	for _, operation := range a.Operations {
		err = verifyPermission(operation.Module, acl, fingerprints)
		if err != nil {
			return err
		}
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

// IndentedString returns an indented JSON string representing the action suitable for
// display
func (a Action) IndentedString() (string, error) {
	buf, err := json.MarshalIndent(a, "", "    ")
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
