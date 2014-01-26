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

type Action struct {
	ID uint64
	Name, Target, Order string
	ScheduledDate, ExpirationDate time.Time
	Arguments interface{}
	PGPSignature string
	PGPSignatureDate time.Time
}

type ExtendedAction struct{
	Action Action
	Status string
	StartTime, FinishTime, LastUpdateTime time.Time
	CommandIDs []uint64
	CmdCompleted, CmdCancelled, CmdTimedOut int
}

// ActionFromFile() reads an action from a local file on the file system
// and returns a mig.ExtendedAction structure
func ActionFromFile(path string) (ea ExtendedAction, err error){
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
	err = json.Unmarshal(fd, &ea.Action)
	if err != nil {
		panic(err)
	}

	// Populate the Extended attributes of the action
	ea.StartTime = time.Now().UTC()

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
	if a.Order == "" {
		return errors.New("Action.Order is empty. Expecting string.")
	}
	if a.ScheduledDate.String() == "" {
		return errors.New("Action.RunDate is empty. Expecting string.")
	}
	if a.ExpirationDate.String() == "" {
		return errors.New("Action.Expiration is empty. Expecting string.")
	}
	if a.ScheduledDate.After(a.ExpirationDate) {
		return errors.New("Action.ExpirationDate is set before Action.ScheduledDate.")
	}
	if time.Now().After(a.ExpirationDate) {
		return errors.New("Action.ExpirationDate is passed. Action has expired.")
	}
	if a.Arguments == nil {
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
	str += "order=" + a.Order + "; "
	str += "scheduleddate=" + a.ScheduledDate.String() + "; "
	str += "expirationdate=" + a.ExpirationDate.String() + "; "

	args, err := json.Marshal(a.Arguments)
	str += "arguments='" + fmt.Sprintf("%s", args) + "';"

	return
}

