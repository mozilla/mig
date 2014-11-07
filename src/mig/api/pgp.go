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
	"mig"
	"mig/pgp"
	"strings"
	"time"
)

// makeKeyring retrieves GPG keys of active investigators from the database and makes
// a GPG keyring out of it
func makeKeyring() (keyring io.ReadSeeker, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makeKeyring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving makeKeyring()"}.Debug()
	}()
	keys, err := ctx.DB.ActiveInvestigatorsKeys()
	if err != nil {
		panic(err)
	}
	keyring, keycount, err := pgp.ArmoredPubKeysToKeyring(keys)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("loaded %d keys from active investigators", keycount)}.Debug()
	return
}

// getKeyring copy an io.Reader from the master keyring. If the keyring hasn't been refreshed
// in a while, it also gets a fresh copy from the database
func getKeyring() (kr io.Reader, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getKeyring() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving getKeyring()"}.Debug()
	}()
	// make sure we don't competing Seek calls or interfering copies by setting a mutex
	ctx.Keyring.Mutex.Lock()
	defer ctx.Keyring.Mutex.Unlock()
	// refresh keyring from DB if older than 5 minutes
	if ctx.Keyring.UpdateTime.Before(time.Now().Add(-5 * time.Minute)) {
		ctx.Keyring.Reader, err = makeKeyring()
		if err != nil {
			panic(err)
		}
		ctx.Keyring.UpdateTime = time.Now()
	} else {
		// rewind the master keyring
		_, err = ctx.Keyring.Reader.Seek(0, 0)
		if err != nil {
			panic(err)
		}
	}
	// copy the master keyring over to a local one
	buf, err := ioutil.ReadAll(ctx.Keyring.Reader)
	if err != nil {
		panic(err)
	}
	kr = bytes.NewBuffer(buf)
	return
}

// verifySignedToken verifies the signature from an authentication token and return
// the investigator that signed it
func verifySignedToken(token string) (inv mig.Investigator, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("verifySignedToken() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving verifySignedToken()"}.Debug()
	}()
	parts := strings.Split(token, ";")
	if len(parts) != 3 {
		panic("invalid token format")
	}
	// verify that token timestamp is recent enough
	tstr := parts[0]
	ts, err := time.Parse("2006-01-02T15:04:05Z", tstr)
	if err != nil {
		panic(err)
	}
	early := time.Now().Add(-ctx.Authentication.duration)
	late := time.Now().Add(ctx.Authentication.duration)
	if ts.Before(early) || ts.After(late) {
		panic("token timestamp is not within acceptable time limits")
	}
	nonce := parts[1]
	sig := parts[2]
	keyring, err := getKeyring()
	if err != nil {
		panic(err)
	}
	fp, err := pgp.GetFingerprintFromSignature(tstr+";"+nonce+"\n", sig, keyring)
	if err != nil {
		panic(err)
	}
	if fp == "" {
		panic("token verification failed")
	}
	inv, err = ctx.DB.InvestigatorByFingerprint(fp)
	if err != nil {
		panic(err)
	}
	return
}
