// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package mig /* import "mig.ninja/mig" */

import (
	"errors"
	"fmt"
	mrand "math/rand"
	"regexp"
	"time"
)

// Describes a loader entry stored in the database
type LoaderEntry struct {
	ID        float64   `json:"id"`        // Loader ID
	Name      string    `json:"name"`      // Loader name
	Prefix    string    `json:"prefix"`    // Loader key prefix
	Key       string    `json:"key"`       // Loader key (only populated during creation)
	AgentName string    `json:"agentname"` // Loader environment, agent name
	LastSeen  time.Time `json:"lastseen"`  // Last time loader was used
	Enabled   bool      `json:"enabled"`   // Loader entry is active
	ExpectEnv string    `json:"expectenv"` // Expected environment
}

func (le *LoaderEntry) Validate() (err error) {
	if le.Key != "" {
		err = ValidateLoaderPrefixAndKey(le.Prefix + le.Key)
	}
	return nil
}

// Small helper type used primarily during the loader authentication
// process between the API and database code, temporarily stores
// authentication information
type LoaderAuthDetails struct {
	ID   float64
	Hash []byte
	Salt []byte
}

func (lad *LoaderAuthDetails) Validate() error {
	if len(lad.Hash) != LoaderHashedKeyLength ||
		len(lad.Salt) != LoaderSaltLength {
		return fmt.Errorf("contents of LoaderAuthDetails are invalid")
	}
	return nil
}

// Generate a new loader prefix value
func GenerateLoaderPrefix() string {
	return RandLoaderKeyString(LoaderPrefixLength)
}

// Generate a new loader key value
func GenerateLoaderKey() string {
	return RandLoaderKeyString(LoaderKeyLength)
}

// RandLoaderKeyString is used for prefix and key generation, and just
// returns a random string consisting of alphanumeric characters of
// length characters long
func RandLoaderKeyString(length int) string {
	ret := make([]byte, length)
	lset := []byte("abcdefghijklmnopqrstuvwxyzABCDEFCHIJKLMNOPQRSTUVWXYZ0123456789")
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	for i := 0; i < len(ret); i++ {
		ret[i] = lset[r.Int()%len(lset)]
	}
	return string(ret[:len(ret)])
}

// Various constants related to properties of the loader keys
const LoaderPrefixAndKeyLength = 40 // Key length including prefix
const LoaderPrefixLength = 8        // Prefix length
const LoaderKeyLength = 32          // Length excluding prefix
const LoaderHashedKeyLength = 32    // Length of hashed key in the database
const LoaderSaltLength = 16         // Length of salt

// Validate a loader key, returns nil if it is valid
func ValidateLoaderKey(key string) error {
	repstr := fmt.Sprintf("^[A-Za-z0-9]{%v}$", LoaderKeyLength)
	ok, err := regexp.MatchString(repstr, key)
	if err != nil || !ok {
		return errors.New("loader key format is invalid")
	}
	return nil
}

// Validate a loader prefix value, returns nil if it is valid
func ValidateLoaderPrefix(prefix string) error {
	repstr := fmt.Sprintf("^[A-Za-z0-9]{%v}$", LoaderPrefixLength)
	ok, err := regexp.MatchString(repstr, prefix)
	if err != nil || !ok {
		return errors.New("loader prefix format is invalid")
	}
	return nil
}

// Validate a loader key that includes the prefix
func ValidateLoaderPrefixAndKey(pk string) error {
	if len(pk) != LoaderPrefixAndKeyLength {
		return fmt.Errorf("loader key is incorrect length")
	}
	err := ValidateLoaderPrefix(pk[:LoaderPrefixLength])
	if err != nil {
		return err
	}
	err = ValidateLoaderKey(pk[LoaderPrefixLength:])
	if err != nil {
		return err
	}
	return nil
}
