// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mig.ninja/mig"
	"mig.ninja/mig/pgp"
	"time"
)

// makePubring retrieves GPG public keys of active investigators from the database
// and makes a GPG keyring out of it
func makePubring(ctx Context) (pubring io.ReadSeeker, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makePubring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving makePubring()"}.Debug()
	}()
	keys, err := ctx.DB.ActiveInvestigatorsKeys()
	if err != nil {
		panic(err)
	}
	pubring, keycount, err := pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("loaded %d keys from active investigators", keycount)}.Debug()
	return
}

// getPubring copy an io.Reader from the master pubring. If the keyring hasn't been refreshed
// in a while, it also gets a fresh copy from the database
func getPubring(ctx Context) (kr io.Reader, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getPubring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving getPubring()"}.Debug()
	}()
	// make sure we don't competing Seek calls or interfering copies by setting a mutex
	ctx.PGP.PubMutex.Lock()
	defer ctx.PGP.PubMutex.Unlock()
	// refresh keyring from DB if older than 5 minutes
	if ctx.PGP.PubringUpdateTime.Before(time.Now().Add(-5 * time.Minute)) {
		ctx.PGP.Pubring, err = makePubring(ctx)
		if err != nil {
			panic(err)
		}
		ctx.PGP.PubringUpdateTime = time.Now()
	} else {
		// rewind the master keyring
		_, err = ctx.PGP.Pubring.Seek(0, 0)
		if err != nil {
			panic(err)
		}
	}
	// copy the master keyring over to a local one
	buf, err := ioutil.ReadAll(ctx.PGP.Pubring)
	if err != nil {
		panic(err)
	}
	kr = bytes.NewBuffer(buf)
	return
}

// makeSecring retrieves the GPG private key of the scheduler from the database
// and makes a GPG keyring out of it
func makeSecring(ctx Context) (secring io.ReadSeeker, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makeSecring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving makeSecring()"}.Debug()
	}()
	key, err := ctx.DB.GetSchedulerPrivKey()
	if err != nil {
		panic(err)
	}
	keys := make([][]byte, 1)
	keys[0] = key
	secring, _, err = pgp.ArmoredKeysToKeyring(keys)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: "loaded scheduler private key from database"}.Debug()
	return
}

// getSecring copy an io.Reader from the master secring
func getSecring(ctx Context) (kr io.Reader, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getSecring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving getSecring()"}.Debug()
	}()
	// make sure we don't competing Seek calls or interfering copies by setting a mutex
	ctx.PGP.SecMutex.Lock()
	defer ctx.PGP.SecMutex.Unlock()
	// rewind the master keyring
	_, err = ctx.PGP.Secring.Seek(0, 0)
	if err != nil {
		panic(err)
	}
	// copy the master keyring over to a local one
	buf, err := ioutil.ReadAll(ctx.PGP.Secring)
	if err != nil {
		panic(err)
	}
	kr = bytes.NewBuffer(buf)
	return
}

// makeSchedulerInvestigator generates a new investigator for user migscheduler
// and stores its private key in the database
func makeSchedulerInvestigator(orig_ctx Context) (inv mig.Investigator, err error) {
	ctx := orig_ctx
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makeSchedulerInvestigator() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving makeSchedulerInvestigator()"}.Debug()
	}()
	inv.Name = "migscheduler"
	pubkey, privkey, fp, err := pgp.GenerateKeyPair(inv.Name, "MIG Scheduler action signing key", "scheduler@mig")
	if err != nil {
		panic(err)
	}
	inv.PublicKey = pubkey
	inv.PrivateKey = privkey
	inv.PGPFingerprint = fp
	iid, err := ctx.DB.InsertSchedulerInvestigator(inv)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("created migscheduler identity with ID %d and key ID %s", iid, inv.PGPFingerprint)}
	return
}
