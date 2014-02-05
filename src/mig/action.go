/* Mozilla InvestiGator

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

package mig

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"mig/pgp/verify"
	"strconv"
	"time"
)

// a MetaAction is a json object that extends an Action with
// additional parameters. It is used to track the completion
// of an action on agents.
type MetaAction struct {
	Action                                                          Action
	Status                                                          string
	StartTime, FinishTime, LastUpdateTime                           time.Time
	CommandIDs                                                      []uint64
	CmdCompleted, CmdSucceeded, CmdCancelled, CmdFailed, CmdTimeOut int
}

// an Action is the json object that is created by an investigator
// and provided to the MIG platform. It must be PGP signed.
type Action struct {
	// meta
	ID           uint64
	Name, Target string
	Description  description
	Threat       threat
	// time window
	ValidFrom, ExpireAfter time.Time
	// operation to perform
	Operations []operation
	// action signature
	PGPSignature     string
	PGPSignatureDate time.Time
	// action syntax version
	SyntaxVersion int
}

// a description is a simple object that contains detail about the
// action's author, and it's revision.
type description struct {
	Author, Email, URL string
	Revision           int
}

// a threat provides the investigator with an idea of how dangerous
// a the compromission might be, if the indicators return positive
type threat struct {
	Level, Family string
}

// an operation is an object that map to an agent module.
// the parameters of the operation are passed to the module as argument,
// and thus their format depend on the module itself.
type operation struct {
	Module     string
	Parameters interface{}
}

// ActionFromFile() reads an action from a local file on the file system
// and returns a mig.ExtendedAction structure
func ActionFromFile(path string) (a Action, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mig.ActionFromFile(): %v", e)
		}
	}()
	// parse the json of the action into a mig.ExtendedAction
	fd, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(fd, &a)
	if err != nil {
		panic(err)
	}

	return
}

// GenID returns an ID composed of a unix timestamp and a random CRC32
func GenID() uint64 {
	h := crc32.NewIEEE()
	t := time.Now().UTC().Format(time.RFC3339Nano)
	r := rand.New(rand.NewSource(65537))
	rand := string(r.Intn(1000000000))
	h.Write([]byte(t + rand))
	// concatenate timestamp and hash into 64 bits ID
	// id = <32 bits unix ts><32 bits CRC hash>
	id := uint64(time.Now().Unix())
	id = id << 32
	id += uint64(h.Sum32())
	return id
}

// GenHexID returns a string with an hexadecimal encoded ID
func GenB32ID() string {
	id := GenID()
	return strconv.FormatUint(id, 32)
}

// checkAction verifies that the Action received contained all the
// necessary fields, and returns an error when it doesn't.
func (a Action) Validate(keyring io.Reader) (err error) {
	if a.Name == "" {
		return errors.New("Action.Name is empty. Expecting string.")
	}
	if a.Target == "" {
		return errors.New("Action.Target is empty. Expecting string.")
	}
	if a.SyntaxVersion < 1 {
		return errors.New("Action.Order is empty. Expecting string.")
	}
	if a.ValidFrom.String() == "" {
		return errors.New("Action.RunDate is empty. Expecting string.")
	}
	if a.ExpireAfter.String() == "" {
		return errors.New("Action.Expiration is empty. Expecting string.")
	}
	if a.ValidFrom.After(a.ExpireAfter) {
		return errors.New("Action.ExpirationDate is set before Action.ScheduledDate.")
	}
	if time.Now().After(a.ExpireAfter) {
		return errors.New("Action.ExpirationDate is passed. Action has expired.")
	}
	if a.Operations == nil {
		return errors.New("Action.Arguments is nil. Expecting string.")
	}
	if a.PGPSignature == "" {
		return errors.New("Action.PGPSignature is empty. Expecting string.")
	}

	// Verify the signature
	astr, err := a.String()
	if err != nil {
		return errors.New("Failed to stringify action")
	}
	valid, _, err := verify.Verify(astr, a.PGPSignature, keyring)
	if err != nil {
		return errors.New("Failed to verify PGP Signature")
	}
	if !valid {
		return errors.New("Invalid PGP Signature")
	}

	return nil
}

//  concatenates Action components into a string
func (a Action) String() (str string, err error) {
	str = "name=" + a.Name + "; "
	str += "target=" + a.Target + "; "
	str += "validfrom=" + a.ValidFrom.String() + "; "
	str += "expireafter=" + a.ExpireAfter.String() + "; "

	args, err := json.Marshal(a.Operations)
	str += "operations='" + fmt.Sprintf("%s", args) + "';"

	return
}
