// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// Package file provides functions to scan a file system as an agent module.
// It can look into files using regexes. It can search files by name. It can
// match hashes in md5, sha1, sha256, sha384, sha512, sha3_224, sha3_256, sha3_384
// and sha3_512.  The filesystem can be searched using patterns, as described in
// the Parameters documentation at http://mig.mozilla.org/doc/module_file.html.
package file /* import "mig.ninja/mig/modules/file" */

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
	"mig.ninja/mig/modules"
)

var debug = false
var tryDecompress = false

func debugprint(format string, a ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("file", new(module))
}

type run struct {
	Parameters parameters
	Results    modules.Result
}

// parameters describes the parameters the file module uses as input upon
// invocation
type parameters struct {
	Searches map[string]*Search `json:"searches,omitempty"`
}

func newParameters() *parameters {
	var p parameters
	p.Searches = make(map[string]*Search)
	return &p
}

// Search contains the fields used to execute an individual search
type Search struct {
	Description      string   `json:"description,omitempty"`
	Paths            []string `json:"paths"`
	Contents         []string `json:"contents,omitempty"`
	Names            []string `json:"names,omitempty"`
	Sizes            []string `json:"sizes,omitempty"`
	Modes            []string `json:"modes,omitempty"`
	Mtimes           []string `json:"mtimes,omitempty"`
	MD5              []string `json:"md5,omitempty"`
	SHA1             []string `json:"sha1,omitempty"`
	SHA2             []string `json:"sha2,omitempty"`
	SHA3             []string `json:"sha3,omitempty"`
	Options          options  `json:"options,omitempty"`
	checks           []check
	checkmask        checkType
	isactive         bool
	iscurrent        bool
	currentdepth     uint64
	matchChan        chan checkMatchNotify // Channel to notify search processor of a check hit
	filesMatchingAll []string              // If Options.MatchAll, stores files matching all checks
}

type options struct {
	MaxDepth     float64  `json:"maxdepth"`
	MaxErrors    float64  `json:"maxerrors"`
	RemoteFS     bool     `json:"remotefs,omitempty"`
	MatchAll     bool     `json:"matchall"`
	Macroal      bool     `json:"macroal"`
	Mismatch     []string `json:"mismatch"`
	MatchLimit   float64  `json:"matchlimit"`
	Debug        string   `json:"debug,omitempty"`
	ReturnSHA256 bool     `json:"returnsha256,omitempty"`
	Decompress   bool     `json:"decompress,omitempty"`
}

type checkType uint64

// BitMask for the type of check to apply to a given file
// see documentation about iota for more info
const (
	checkContent checkType = 1 << (64 - 1 - iota)
	checkName
	checkSize
	checkMode
	checkMtime
	checkMD5
	checkSHA1
	checkSHA256
	checkSHA384
	checkSHA512
	checkSHA3_224
	checkSHA3_256
	checkSHA3_384
	checkSHA3_512
	checkBytes
)

// check represents an individual check that is part of a search.
type check struct {
	checkid                int // Internal check ID, set by the search parent
	code                   checkType
	matched                uint64
	matchedfiles           []string
	value                  string
	bytes                  []byte
	regex                  *regexp.Regexp
	minsize, maxsize       uint64
	minmtime, maxmtime     time.Time
	inversematch, mismatch bool
	matchChan              chan checkMatchNotify
	waitNotify             chan bool
}

// checkMatchNotify is sent from the check to the parent Search via the checks matchChan to
// notify the Search type's search processor that a match has been found for an individual check.
type checkMatchNotify struct {
	checkid int
	file    string
}

// pretty much infinity when it comes to file searches
const unlimited float64 = 1125899906842624

// processMatch processes incoming matches from individual checks which are part of the search. It
// also manages the total hit statistics. The match processor does some preprocessing, such as identifying
// files that match all checks for a search if MatchAll is set, to make building the results simpler.
//
// Although this function runs in a goroutine, execution is serialized via a wait channel this function
// will write to when its ready for the next result.
func (s *Search) processMatch() {
	for {
		var c *check
		match := <-s.matchChan

		c = nil
		for i := range s.checks {
			if s.checks[i].checkid == match.checkid {
				c = &s.checks[i]
			}
		}
		if c == nil {
			// This is fatal, and means we received a result for a check which we
			// do not know about
			panic("processMatch received check result for invalid check id")
		}
		// See if we need to add the file for this check, if it already exists we are done
		found := false
		for _, x := range c.matchedfiles {
			if x == match.file {
				found = true
				break
			}
		}
		if found {
			c.waitNotify <- true
			continue
		}
		c.matchedfiles = append(c.matchedfiles, match.file)
		c.matched++

		// If this search has MatchAll set, see if this file now matches all checks in
		// the search. If so, add it to the allMatched list.
		if s.Options.MatchAll && !s.allChecksMatched(match.file) {
			allmatch := true
			for _, c := range s.checks {
				if !c.hasMatch(match.file) {
					allmatch = false
					break
				}
			}
			if allmatch {
				s.filesMatchingAll = append(s.filesMatchingAll, match.file)
				// Since this should be considered a match now, increment the hits
				// counter
				stats.Totalhits++
			}
		} else {
			// MatchAll isn't set, so we just count every hit here as a match
			stats.Totalhits++
		}

		c.waitNotify <- true
	}
}

// allChecksMatched returns true if the file is in the filesMatchingAll list for a search
func (s *Search) allChecksMatched(file string) bool {
	for _, f := range s.filesMatchingAll {
		if f == file {
			return true
		}
	}
	return false
}

func (s *Search) makeChecks() (err error) {
	var nextCheckID int
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makeChecks() -> %v", e)
		}
	}()
	nextCID := func() check {
		ret := check{}
		nextCheckID++
		ret.checkid = nextCheckID
		ret.matchChan = s.matchChan
		ret.waitNotify = make(chan bool, 0)
		return ret
	}
	if s.Options.Debug == "print" {
		debug = true
	}
	if s.Options.MaxDepth == 0 {
		s.Options.MaxDepth = unlimited
	}
	if s.Options.MaxErrors == 0 {
		s.Options.MaxErrors = unlimited
	}
	if s.Options.MatchLimit == 0 {
		s.Options.MatchLimit = unlimited
	}
	for _, v := range s.Contents {
		c := nextCID()
		c.code = checkContent
		c.value = v
		if len(v) > 1 && v[:1] == "!" {
			c.inversematch = true
			v = v[1:]
		}
		if s.hasMismatch("content") {
			c.mismatch = true
		}
		c.regex = regexp.MustCompile(v)
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.Names {
		c := nextCID()
		c.code = checkName
		c.value = v
		if len(v) > 1 && v[:1] == "!" {
			c.inversematch = true
			v = v[1:]
		}
		if s.hasMismatch("name") {
			c.mismatch = true
		}
		c.regex = regexp.MustCompile(v)
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.Sizes {
		c := nextCID()
		c.code = checkSize
		c.value = v
		c.minsize, c.maxsize, err = parseSize(v)
		if err != nil {
			panic(err)
		}
		if s.hasMismatch("size") {
			c.mismatch = true
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.Modes {
		c := nextCID()
		c.code = checkMode
		c.value = v
		if s.hasMismatch("mode") {
			c.mismatch = true
		}
		c.regex = regexp.MustCompile(v)
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.Mtimes {
		c := nextCID()
		c.code = checkMtime
		c.value = v
		if s.hasMismatch("mtime") {
			c.mismatch = true
		}
		c.minmtime, c.maxmtime, err = parseMtime(v)
		if err != nil {
			panic(err)
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.MD5 {
		c := nextCID()
		c.code = checkMD5
		c.value = strings.ToUpper(v)
		if s.hasMismatch("md5") {
			c.mismatch = true
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.SHA1 {
		c := nextCID()
		c.code = checkSHA1
		c.value = strings.ToUpper(v)
		if s.hasMismatch("sha1") {
			c.mismatch = true
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.SHA2 {
		c := nextCID()
		c.value = strings.ToUpper(v)
		if s.hasMismatch("sha2") {
			c.mismatch = true
		}
		switch len(v) {
		case 64:
			c.code = checkSHA256
		case 96:
			c.code = checkSHA384
		case 128:
			c.code = checkSHA512
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.SHA3 {
		c := nextCID()
		c.value = strings.ToUpper(v)
		if s.hasMismatch("sha3") {
			c.mismatch = true
		}
		switch len(v) {
		case 56:
			c.code = checkSHA3_224
		case 64:
			c.code = checkSHA3_256
		case 96:
			c.code = checkSHA3_384
		case 128:
			c.code = checkSHA3_512
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.Bytes {
		var c check
		c.code = checkBytes
		c.value = v
		c.bytes, err = hex.DecodeString(v)
		if err != nil {
			return
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
		if debug {
			fmt.Printf("adding byte check with value '%s'\n", c.value)
		}
	}
	return
}

func (s *Search) hasMismatch(filter string) bool {
	for _, fi := range s.Options.Mismatch {
		if fi == filter {
			return true
		}
	}
	return false
}

func parseSize(size string) (minsize, maxsize uint64, err error) {
	var (
		multiplier uint64 = 1
		n          uint64
	)
	switch size[len(size)-1] {
	case 'k':
		multiplier = 1024
	case 'm':
		multiplier = 1024 * 1024
	case 'g':
		multiplier = 1024 * 1024 * 1024
	case 't':
		multiplier = 1024 * 1024 * 1024 * 1024
	}
	up := len(size)
	if multiplier > 1 {
		up--
	}
	switch size[0] {
	case '<':

		// must not exceed size
		n, err = strconv.ParseUint(size[1:up], 10, 64)
		if err != nil {
			return
		}
		minsize = 0
		maxsize = n * multiplier
	case '>':
		// must not be smaller than
		n, err = strconv.ParseUint(size[1:up], 10, 64)
		if err != nil {
			return
		}
		minsize = n * multiplier
		maxsize = uint64(^uint64(0) >> 1)
	default:
		// must be exactly this size
		n, err = strconv.ParseUint(size[0:up], 10, 64)
		if err != nil {
			return
		}
		minsize = n * multiplier
		maxsize = n * multiplier
	}
	return
}

func parseMtime(mtime string) (minmtime, maxmtime time.Time, err error) {
	var (
		isDays = false
		n      uint64
	)
	suffix := mtime[len(mtime)-1]
	if suffix == 'd' {
		isDays = true
		suffix = 'h'
	}
	n, err = strconv.ParseUint(mtime[1:len(mtime)-1], 10, 64)
	if err != nil {
		return
	}
	if isDays {
		n = n * 24
	}
	duration := fmt.Sprintf("%d%c", n, suffix)
	d, err := time.ParseDuration(duration)
	switch mtime[0] {
	case '<':
		// modification date is between date and now (or future)
		minmtime = time.Now().Add(-d)
		maxmtime = time.Date(9998, time.January, 11, 11, 11, 11, 11, time.UTC)
	case '>':
		// modification date is older than date
		minmtime = time.Date(1111, time.January, 11, 11, 11, 11, 11, time.UTC)
		maxmtime = time.Now().Add(-d)
	}
	debugprint("Parsed mtime filter with minmtime '%s' and maxmtime '%s'\n",
		minmtime.String(), maxmtime.String())
	return
}

func (s *Search) activate() {
	s.isactive = true
	return
}

func (s *Search) deactivate() {
	s.isactive = false
	return
}

func (s *Search) increasedepth() {
	s.currentdepth++
	return
}

func (s *Search) decreasedepth() {
	s.currentdepth--
	return
}

func (s *Search) markcurrent() {
	s.iscurrent = true
	return
}

func (s *Search) unmarkcurrent() {
	s.iscurrent = false
	return
}

// storeMatch writes a matched file to the check's parent Search type results
// processor, where it can be processed and stored with the check.
func (c *check) storeMatch(file string) {
	c.matchChan <- checkMatchNotify{
		file:    file,
		checkid: c.checkid,
	}
	_ = <-c.waitNotify
}

// hasMatch returns true if a check has matched against a file
func (c *check) hasMatch(file string) bool {
	for _, x := range c.matchedfiles {
		if x == file {
			return true
		}
	}
	return false
}

func (r *run) ValidateParameters() (err error) {
	var labels []string
	for label, s := range r.Parameters.Searches {
		labels = append(labels, label)
		debugprint("validating label '%s'\n", label)
		err = validateLabel(label)
		if err != nil {
			return
		}
		if len(s.Paths) == 0 {
			return fmt.Errorf("invalid empty search path, must have at least one search path")
		}
		for _, r := range s.Contents {
			debugprint("validating content '%s'\n", r)
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Names {
			debugprint("validating name '%s'\n", r)
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Sizes {
			debugprint("validating size '%s'\n", r)
			err = validateSize(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Modes {
			debugprint("validating mode '%s'\n", r)
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Mtimes {
			debugprint("validating mtime '%s'\n", r)
			err = validateMtime(r)
			if err != nil {
				return
			}
		}
		for _, hash := range s.MD5 {
			debugprint("validating hash '%s'\n", hash)
			err = validateHash(hash, checkMD5)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA1 {
			debugprint("validating hash '%s'\n", hash)
			err = validateHash(hash, checkSHA1)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA2 {
			debugprint("validating hash '%s'\n", hash)
			switch len(hash) {
			case 64:
				err = validateHash(hash, checkSHA256)
			case 96:
				err = validateHash(hash, checkSHA384)
			case 128:
				err = validateHash(hash, checkSHA512)
			default:
				fmt.Printf("ERROR: Invalid hash length")
			}
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA3 {
			debugprint("validating hash '%s'\n", hash)
			switch len(hash) {
			case 56:
				err = validateHash(hash, checkSHA3_224)
			case 64:
				err = validateHash(hash, checkSHA3_256)
			case 96:
				err = validateHash(hash, checkSHA3_384)
			case 128:
				err = validateHash(hash, checkSHA3_512)
			default:
				fmt.Printf("ERROR: Invalid hash length")
			}
			if err != nil {
				return
			}
		}
		for _, bytes := range s.Bytes {
			debugprint("validating bytes '%s'\n", bytes)
			err = validateBytes(bytes)
			if err != nil {
				return
			}
		}

		for _, mismatch := range s.Options.Mismatch {
			debugprint("validating mismatch '%s'\n", mismatch)
			err = validateMismatch(mismatch)
			if err != nil {
				return
			}
		}
		if s.Options.Decompress {
			tryDecompress = true
		} else {
			tryDecompress = false
		}
	}
	return
}

func validateBytes(bytes string) error {
	if len(bytes) < 1 {
		return fmt.Errorf("Empty values are not permitted")
	}
	_, err := hex.DecodeString(bytes)
	if err != nil {
		return fmt.Errorf("Invalid bytes '%s'. Must be an hexadecimal byte string without leading 0x. ex: 'ff00d1ab'", bytes)
	}
	return nil
}

func validateLabel(label string) error {
	if len(label) < 1 {
		return fmt.Errorf("empty labels are not permitted")
	}
	labelregexp := `^([a-zA-Z0-9_-]|.){1,64}$`
	labelre := regexp.MustCompile(labelregexp)
	if !labelre.MatchString(label) {
		return fmt.Errorf("The syntax of label '%s' is invalid. Must match regex %s", label, labelregexp)
	}
	return nil
}

func validateRegex(regex string) error {
	if len(regex) < 1 {
		return fmt.Errorf("Empty values are not permitted")
	}
	if len(regex) > 1 && regex[:1] == "!" {
		// remove heading ! before compiling the regex
		regex = regex[1:]
	}
	_, err := regexp.Compile(regex)
	if err != nil {
		return fmt.Errorf("Invalid regexp '%s'. Must be a regexp. Compilation failed with '%v'", regex, err)
	}
	return nil
}

// Size accepts the prefixes '<', '>' for lower than and greater than. if no prefix is specified, equality is assumed.
// Size accepts the suffixes 'k', 'm', 'g', 't' for kilobyte, megabyte, gigabyte and terabyte. if not suffix is specified,
// bytes are assumed. example: '>50m' will find files with a size greater than 50 megabytes
func validateSize(size string) error {
	if len(size) < 1 {
		return fmt.Errorf("Empty values are not permitted")
	}
	re := "^(<|>)?[0-9]*(k|m|g|t)?$"
	sizere := regexp.MustCompile(re)
	if !sizere.MatchString(size) {
		return fmt.Errorf("Invalid size format for size '%s'. Must match regex %s", size, re)
	}
	return nil
}

func validateMtime(mtime string) error {
	if len(mtime) < 1 {
		return fmt.Errorf("Empty values are not permitted")
	}
	re := "^(<|>)[0-9]*(d|h|m)$"
	mtimere := regexp.MustCompile(re)
	if !mtimere.MatchString(mtime) {
		return fmt.Errorf("Invalid mtime format for mtime '%s'. Must match regex %s", mtime, re)
	}
	return nil
}

func validateHash(hash string, hashType checkType) error {
	if len(hash) < 1 {
		return fmt.Errorf("Empty values are not permitted")
	}
	hash = strings.ToUpper(hash)
	var re string
	switch hashType {
	case checkMD5:
		re = "^[A-F0-9]{32}$"
	case checkSHA1:
		re = "^[A-F0-9]{40}$"
	case checkSHA256:
		re = "^[A-F0-9]{64}$"
	case checkSHA384:
		re = "^[A-F0-9]{96}$"
	case checkSHA512:
		re = "^[A-F0-9]{128}$"
	case checkSHA3_224:
		re = "^[A-F0-9]{56}$"
	case checkSHA3_256:
		re = "^[A-F0-9]{64}$"
	case checkSHA3_384:
		re = "^[A-F0-9]{96}$"
	case checkSHA3_512:
		re = "^[A-F0-9]{128}$"
	default:
		return fmt.Errorf("Invalid hash type %d for hash '%s'", hashType, hash)
	}
	hashre := regexp.MustCompile(re)
	if !hashre.MatchString(hash) {
		return fmt.Errorf("Invalid checksum format for hash '%s'. Must match regex %s", hash, re)
	}
	return nil
}

func validateMismatch(filter string) error {
	if len(filter) < 1 {
		return fmt.Errorf("empty filters are not permitted")
	}
	filterregexp := `^(name|size|mode|mtime|content|md5|sha1|sha2|sha3)$`
	re := regexp.MustCompile(filterregexp)
	if !re.MatchString(filter) {
		return fmt.Errorf("The syntax of filter '%s' is invalid. Must match regex %s", filter, filterregexp)
	}
	return nil
}

/* Statistic counters:
- FilesCount is the total number of files inspected
- Openfailed is the count of files that could not be opened
- Totalhits is the total number of checklist hits
- Exectim is the total runtime of all the searches
*/
type statistics struct {
	Filescount float64 `json:"filescount"`
	Openfailed float64 `json:"openfailed"`
	Totalhits  float64 `json:"totalhits"`
	Exectime   string  `json:"exectime"`
}

// stats is a global variable
var stats statistics

var walkingErrors []string

func (r *run) Run(in modules.ModuleReader) (resStr string) {
	var (
		roots     []string
		traversed []string
	)
	defer func() {
		if e := recover(); e != nil {
			// return error in json
			res := newResults()
			res.Statistics = stats
			for _, we := range walkingErrors {
				res.Errors = append(res.Errors, we)
			}
			res.Errors = append(res.Errors, fmt.Sprintf("%v", e))
			res.Success = false
			err, _ := json.Marshal(res)
			resStr = string(err[:])
			return
		}
	}()
	t0 := time.Now()
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	for label, search := range r.Parameters.Searches {
		// Allocate our match input channel; checks will write the filename and their respective
		// check ID to this channel when a match is identified
		search.matchChan = make(chan checkMatchNotify, 0)
		// Start the incoming match processor for the search entry
		go search.processMatch()
		// Create all the checks for the search
		debugprint("making checks for label %s\n", label)
		err := search.makeChecks()
		if err != nil {
			panic(err)
		}
		var paths []string
		// clean up the paths, store in roots if not already present
		for _, p := range search.Paths {
			p = filepath.Clean(p)
			paths = append(paths, p)
			alreadyPresent := false
			for _, r := range roots {
				if p == r {
					alreadyPresent = true
				}
			}
			if !alreadyPresent {
				debugprint("adding path %s to list of locations to traverse\n", p)
				roots = append(roots, p)
			}
		}
		// store the cleaned up paths
		search.Paths = paths
		r.Parameters.Searches[label] = search
		// sorting the array is useful in case the same command contains "/some/thing"
		// and then "/some". By starting with the smallest root, we ensure that all the
		// checks for both "/some" and "/some/thing" will be processed.
		sort.Strings(roots)
	}
	// enter each root one by one
	for _, root := range roots {
		// before entering a root, deactivate all searches a reset the depth counters
		for label, search := range r.Parameters.Searches {
			search.deactivate()
			search.currentdepth = 0
			r.Parameters.Searches[label] = search
		}
		for _, p := range traversed {
			if root == p {
				debugprint("skipping already traversed root: %s\n", root)
				goto skip
			}
		}
		debugprint("entering root %s\n", root)
		traversed, err = r.pathWalk(root, roots)
		if err != nil {
			// log errors and continue
			walkingErrors = append(walkingErrors, fmt.Sprintf("ERROR: %v", err))
			debugprint("pathWalk failed with error '%v'\n", err)
		}
	skip:
	}

	resStr, err = r.buildResults(t0)
	if err != nil {
		panic(err)
	}

	if debug {
		debugprint("---- results ----")
		var tmpres modules.Result
		err = json.Unmarshal([]byte(resStr), &tmpres)
		printedResults, err := r.PrintResults(tmpres, false)
		if err != nil {
			panic(err)
		}
		for _, res := range printedResults {
			debugprint(res)
		}
	}
	return
}

func (r *run) pathWalk(path string, roots []string) (traversed []string, err error) {
	var (
		subdirs []string
		target  *os.File
		t       os.FileInfo
	)
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("pathWalk() -> %v", e)
		}
	}()
	debugprint("pathWalk: walking into '%s'\n", path)
	// as we traversed the directory structure from the shortest path to the longest, we
	// may end up traversing directories that are supposed to be processed later on.
	// when that happens, flag the directory in the traversed list to tell the top-level
	// function to not traverse it again
	for _, p := range roots {
		if p == path {
			traversed = append(traversed, p)
		}
	}
	// verify that we have at least one search interested in the current directory
	activesearches := 0
	for label, search := range r.Parameters.Searches {
		// We need to determine if the search path we are in is in the scope of this
		// search (is current) and also, if we should activate the search. Activating the
		// search means files under the path will be examined. Even if we do not activate, we
		// want to make sure the search is marked as current so we properly track depth
		// changes.
		//
		searchInScope := false
		// First, lets scan the search paths in this given search and compare it to determine
		// if it's in scope.
		for _, p := range search.Paths {
			debugprint("comparing current path '%s' with candidate search '%s'\n", path, p)
			if len(path) >= len(p) && p == path[:len(p)] {
				searchInScope = true
				break
			}
		}
		if searchInScope {
			// The search is in scope, so note the depth change and mark it current.
			search.increasedepth()
			search.markcurrent()

			// Next, see if we can activate the search. For activation, we need to meet the maximum
			// depth criteria and be under our match limit
			if search.currentdepth <= uint64(search.Options.MaxDepth) &&
				stats.Totalhits < search.Options.MatchLimit {
				search.activate()
				activesearches++
			}
		}
		r.Parameters.Searches[label] = search
	}
	debugprint("pathWalk: %d searches are currently active\n", activesearches)
	if activesearches == 0 {
		goto finish
	}
	// Read the content of dir stored in 'path',
	// put all sub-directories in the subdirs slice, and call
	// the inspection function for all files
	target, err = os.Open(path)
	if err != nil {
		// do not panic when open fails, just increase a counter
		stats.Openfailed++
		walkingErrors = append(walkingErrors, fmt.Sprintf("ERROR: %v", err))
		goto finish
	}
	t, _ = os.Lstat(path)
	if t.Mode().IsDir() {
		// target is a directory, process its content
		debugprint("'%s' is a directory, processing its content\n", path)
		dirContent, err := target.Readdir(-1)
		if err != nil {
			walkingErrors = append(walkingErrors, fmt.Sprintf("ERROR: %v", err))
			goto finish
		}
		// loop over the content of the directory
		for _, dirEntry := range dirContent {
			// While we are iterating over the directory content, consult Totalhits to
			// make sure we do not exceed our match limit. If we hit the match limit, we
			// deactivate the search.
			for _, search := range r.Parameters.Searches {
				if stats.Totalhits >= search.Options.MatchLimit {
					search.deactivate()
					activesearches--
				}
			}
			entryAbsPath := path
			// append path separator if missing
			if entryAbsPath[len(entryAbsPath)-1] != os.PathSeparator {
				entryAbsPath += string(os.PathSeparator)
			}
			entryAbsPath += dirEntry.Name()
			// this entry is a subdirectory, keep it for later
			if dirEntry.IsDir() {
				// if not symlinked directory, don't put symlinked directory in subdirectories to follow
				if dirEntry.Mode()&os.ModeSymlink != os.ModeSymlink {
					// append trailing slash
					if entryAbsPath[len(entryAbsPath)-1] != os.PathSeparator {
						entryAbsPath += string(os.PathSeparator)
					}
					subdirs = append(subdirs, entryAbsPath)
				}
				continue
			}
			// if entry is a symlink, evaluate the target
			isLinkedFile := false
			if dirEntry.Mode()&os.ModeSymlink == os.ModeSymlink {
				linkmode, linkpath, err := followSymLink(entryAbsPath)
				if err != nil {
					// reading the link failed, count and continue
					stats.Openfailed++
					walkingErrors = append(walkingErrors, fmt.Sprintf("ERROR: %v", err))
					continue
				}
				debugprint("'%s' links to '%s'\n", entryAbsPath, linkpath)
				if linkmode.IsRegular() {
					isLinkedFile = true
				}
			}
			if dirEntry.Mode().IsRegular() || isLinkedFile {
				err = r.evaluateFile(entryAbsPath)
				if err != nil {
					walkingErrors = append(walkingErrors, err.Error())
				}
			}
		}
	}

	// target is a symlink, expand it. we only follow symlinks to files, not directories
	if t.Mode()&os.ModeSymlink == os.ModeSymlink {
		linkmode, linkpath, err := followSymLink(path)
		if err != nil {
			// reading the link failed, count and continue
			stats.Openfailed++
			walkingErrors = append(walkingErrors, fmt.Sprintf("ERROR: %v", err))
			goto finish
		}
		debugprint("'%s' links to '%s'\n", path, linkpath)
		if linkmode.IsRegular() {
			path = linkpath
		} else {
			walkingErrors = append(walkingErrors, fmt.Sprintf("warning: %s is a link to %s and was not followed", path, linkpath))
		}
	}

	// target is a not a directory
	if !t.Mode().IsDir() {
		err = r.evaluateFile(path)
		if err != nil {
			walkingErrors = append(walkingErrors, err.Error())
			goto finish
		}
	}

	// if we found any sub directories, go down the rabbit hole recursively,
	// but only if one of the check is interested in going
	for _, dir := range subdirs {
		traversed, err = r.pathWalk(dir, roots)
		if err != nil {
			walkingErrors = append(walkingErrors, err.Error())
			continue
		}
	}
finish:
	// close the current target, we are done with it
	target.Close()
	// leaving the directory, decrement the depth counter of active searches
	for label, search := range r.Parameters.Searches {
		if search.iscurrent {
			search.decreasedepth()
			debugprint("decreasing search depth for '%s' to '%d'\n", label, search.currentdepth)
		}
		r.Parameters.Searches[label] = search
	}
	return
}

// followSymLink expands a symbolic link and return the absolute path of the target,
// along with its FileMode and an error
func followSymLink(link string) (mode os.FileMode, path string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("followSymLink() -> %v", e)
		}
	}()
	path, err = filepath.EvalSymlinks(link)
	if err != nil {
		panic(err)
	}
	// make an absolute path
	if !filepath.IsAbs(path) {
		path = filepath.Dir(link) + string(os.PathSeparator) + path
	}
	fi, err := os.Lstat(path)
	if err != nil {
		panic(err)
	}
	mode = fi.Mode()
	return
}

type fileEntry struct {
	filename string
	fd       *os.File
	compRdr  io.Reader
}

func (f *fileEntry) Close() {
	f.fd.Close()
}

// getReader returns an appropriate reader for the file being checked.
// This is done by checking whether the file is compressed or not
func (f *fileEntry) getReader() io.Reader {
	var (
		err error
	)
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getReader() -> %v", err)
		}
	}()
	f.fd, err = os.Open(f.filename)
	if err != nil {
		stats.Openfailed++
		panic(err)
	}
	if tryDecompress != true {
		return f.fd
	}
	magic := make([]byte, 2)
	n, err := f.fd.Read(magic)
	if err != nil {
		panic(err)
	}
	if n != 2 {
		return f.fd
	}
	_, err = f.fd.Seek(0, 0)
	if err != nil {
		panic(err)
	}
	if magic[0] == 0x1f && magic[1] == 0x8b {
		f.compRdr, err = gzip.NewReader(f.fd)
		if err != nil {
			panic(err)
		}
		return f.compRdr
	}
	return f.fd
}

// evaluateFile takes a single file and applies searches to it
func (r *run) evaluateFile(file string) (err error) {
	var activeSearches []string
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("evaluateFile() -> %v", e)
		}
		// restore list of active searches on exit
		for _, label := range activeSearches {
			search := r.Parameters.Searches[label]
			search.activate()
			r.Parameters.Searches[label] = search
		}
	}()
	stats.Filescount++
	debugprint("evaluateFile: evaluating '%s'\n", file)
	// store list of active searches to restore it before leaving
	for label, search := range r.Parameters.Searches {
		if search.isactive {
			debugprint("evaluateFile: search '%s' is active\n", label)
			activeSearches = append(activeSearches, label)
		}
	}
	// First pass: look at the file metadata and if MatchAll is set,
	// deactivate the searches that don't match the current file.
	// If MatchAll is not set, all checks will be performed individually
	fi, err := os.Stat(file)
	if err != nil {
		panic(err)
	}
	for label, search := range r.Parameters.Searches {
		if !search.isactive {
			goto skip
		}
		if !search.checkName(file, fi) && search.Options.MatchAll {
			search.deactivate()
			goto skip
		}
		if !search.checkMode(file, fi) && search.Options.MatchAll {
			search.deactivate()
			goto skip
		}
		if !search.checkSize(file, fi) && search.Options.MatchAll {
			search.deactivate()
			goto skip
		}
		if !search.checkMtime(file, fi) && search.Options.MatchAll {
			search.deactivate()
			goto skip
		}
	skip:
		r.Parameters.Searches[label] = search
	}
	// Second pass: Enter all content & hash checks across all searches.
	// Only perform the searches that are active.
	// Optimize to only read a file once per check type
	f := fileEntry{filename: file}
	r.checkContent(f)
	r.checkBytes(f)
	r.checkHash(f, checkMD5)
	r.checkHash(f, checkSHA1)
	r.checkHash(f, checkSHA256)
	r.checkHash(f, checkSHA384)
	r.checkHash(f, checkSHA512)
	r.checkHash(f, checkSHA3_224)
	r.checkHash(f, checkSHA3_256)
	r.checkHash(f, checkSHA3_384)
	r.checkHash(f, checkSHA3_512)
	return
}

/* wantThis() implements boolean logic to decide if a given check should be a match or not
It's just 2 XOR chained one after the other.

 Match | Inverse | Mismatch | Return
-------+---------+----------+--------
 true  ^  true   ^  true    = true
 true  ^  true   ^  false   = false
 true  ^  false  ^  true    = false
 true  ^  false  ^  false   = true
 false ^  true   ^  true    = false
 false ^  true   ^  false   = true
 false ^  false  ^  true    = true
 false ^  false  ^  false   = false
*/
func (c check) wantThis(match bool) bool {
	if match {
		if c.inversematch {
			if c.mismatch {
				debugprint("wantThis=true\n")
				return true
			}
			debugprint("wantThis=false\n")
			return false
		}
		if c.mismatch {
			debugprint("wantThis=false\n")
			return false
		}
		debugprint("wantThis=true\n")
		return true
	}
	if c.inversematch {
		if c.mismatch {
			debugprint("wantThis=false\n")
			return false
		}
		debugprint("wantThis=true\n")
		return true
	}
	if c.mismatch {
		debugprint("wantThis=true\n")
		return true
	}
	debugprint("wantThis=false\n")
	return false
}

func (s Search) checkName(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkName) != 0 {
		for i := range s.checks {
			if (s.checks[i].code & checkName) == 0 {
				continue
			}
			c := &s.checks[i]
			match := c.regex.MatchString(path.Base(fi.Name()))
			if match {
				debugprint("file name '%s' matches regex '%s'\n", fi.Name(), c.value)
			}
			if c.wantThis(match) {
				c.storeMatch(file)
			} else {
				matchedall = false
			}
		}
	}
	return
}

func (s Search) checkMode(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkMode) != 0 {
		for i := range s.checks {
			if (s.checks[i].code & checkMode) == 0 {
				continue
			}
			c := &s.checks[i]
			match := c.regex.MatchString(fi.Mode().String())
			if match {
				debugprint("file '%s' mode '%s' matches regex '%s'\n",
					fi.Name(), fi.Mode().String(), c.value)
			}
			if c.wantThis(match) {
				c.storeMatch(file)
			} else {
				matchedall = false
			}
		}
	}
	return
}

func (s Search) checkSize(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkSize) != 0 {
		for i := range s.checks {
			if (s.checks[i].code & checkSize) == 0 {
				continue
			}
			c := &s.checks[i]
			match := false
			if fi.Size() >= int64(c.minsize) && fi.Size() <= int64(c.maxsize) {
				match = true
				debugprint("file '%s' size '%d' is between %d and %d\n",
					fi.Name(), fi.Size(), c.minsize, c.maxsize)
			}
			if c.wantThis(match) {
				c.storeMatch(file)
			} else {
				matchedall = false
			}
		}
	}
	return
}

func (s Search) checkMtime(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkMtime) != 0 {
		for i := range s.checks {
			if (s.checks[i].code & checkMtime) == 0 {
				continue
			}
			c := &s.checks[i]
			match := false
			if fi.ModTime().After(c.minmtime) && fi.ModTime().Before(c.maxmtime) {
				match = true
				debugprint("file '%s' mtime '%s' is between %s and %s\n",
					fi.Name(), fi.ModTime().UTC().String(),
					c.minmtime.String(), c.maxmtime.String())
			}
			if c.wantThis(match) {
				c.storeMatch(file)
			} else {
				matchedall = false
			}
		}
	}
	return
}

func (r *run) checkContent(f fileEntry) {
	var (
		err error
	)
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("checkContent() -> %v", e)
			walkingErrors = append(walkingErrors, err.Error())
		}
	}()
	// skip this check if no search has anything to run
	// also used to keep track of the checks to run and exit early if possible
	var checksstatus = make(map[string]map[int]bool)
	// keep track of matches lines
	var macroalstatus = make(map[string]bool)
	continuereadingfile := false
	for label, search := range r.Parameters.Searches {
		if !search.isactive {
			continue
		}
		for i := range search.checks {
			if search.checks[i].code&checkContent == 0 {
				continue
			}
			// init the map
			checksstatus[label] = map[int]bool{i: false}
			continuereadingfile = true
		}
	}
	if !continuereadingfile {
		return
	}
	// iterate over the file content
	reader := f.getReader()
	defer f.Close()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		if !continuereadingfile {
			break
		}
		hasactivechecks := false
		for label, search := range r.Parameters.Searches {
			// skip inactive searches or searches that don't have a content check
			if !search.isactive || (search.checkmask&checkContent == 0) {
				continue
			}
			// macroal is a flag used to keep track of all checks ran against the
			// current line of the file. if one check fails to match on this line,
			// the macroal flag is set to false
			macroalstatus[label] = true
			// apply the content checks regexes to the current scan
			for i := range search.checks {
				// skip this check if it's not a content check or if it has already matched
				if search.checks[i].code&checkContent == 0 || (checksstatus[label][i] && !search.Options.Macroal) {
					continue
				}
				c := &search.checks[i]
				hasactivechecks = true

				/* Matching Logic
				When evaluating a content check against a line in a file, three criteria are considered:
				1. did the regex match the line?
				2. is the inversematch flag set on the check?
				3. is the macroal option set on the search?
				Based on these, the table below indicates the result of the search.

				 Regex     | Inverse | MACROAL | Result
				-----------+---------+---------+--------
				 Match     |  False  |  True   | pass	-> must match all lines and current line matched
				 Match     |  False  |  False  | pass	-> must match any line and current line matched
				 Match     |  True   |  True   | fail	-> must match no line but current line matches
				 Match     |  True   |  False  | fail	-> must not match at least one line but current line matched
				 Not Match |  True   |  True   | pass	-> must match no line and current line didn't match
				 Not Match |  True   |  False  | pass	-> must not match at least one line and current line didn't match
				 Not Match |  False  |  True   | fail	-> much match all lines and current line didn't match
				 Not Match |  False  |  False  | fail	-> much match any line and current line didn't match
				*/
				if c.regex.MatchString(scanner.Text()) {
					// Regex Match
					debugprint("checkContent: regex '%s' match on line '%s'\n", c.value, scanner.Text())
					if !c.inversematch {
						if search.Options.Macroal {
							debugprint("checkContent: [pass] must match all lines and current line matched. regex='%s', line='%s'\n",
								c.value, scanner.Text())
						} else {
							debugprint("checkContent: [pass] must match any line and current line matched. regex='%s', line='%s'\n",
								c.value, scanner.Text())
							if c.wantThis(true) {
								c.storeMatch(f.filename)
							}
						}
					} else {
						if search.Options.Macroal {
							debugprint("checkContent: [fail] must match no line but current line matched. regex='%s', line='%s'\n",
								c.value, scanner.Text())
							macroalstatus[label] = false
							if c.wantThis(true) {
								c.storeMatch(f.filename)
							}
						} else {
							debugprint("checkContent: [fail] must not match at least one line but current line matched. regex='%s', line='%s'\n",
								c.value, scanner.Text())
						}
						debugprint("checkContent: regex '%s' is an inverse match and shouldn't have matched on line '%s'\n", c.value, scanner.Text())
					}
					checksstatus[label][i] = true
				} else {
					// Regex Not Match
					if c.inversematch {
						if search.Options.Macroal {
							debugprint("checkContent: [pass] must match no line and current line didn't match. regex='%s', line='%s'\n",
								c.value, scanner.Text())
						} else {
							debugprint("checkContent: [pass] must not match at least one line and current line didn't match. regex='%s', line='%s'\n",
								c.value, scanner.Text())
						}
					} else {
						if search.Options.Macroal {
							debugprint("checkContent: [fail] much match all lines and current line didn't match. regex='%s', line='%s'\n",
								c.value, scanner.Text())
							macroalstatus[label] = false
							if c.wantThis(false) {
								c.storeMatch(f.filename)
							}
						} else {
							debugprint("checkContent: [fail] much match any line and current line didn't match. regex='%s', line='%s'\n",
								c.value, scanner.Text())
						}
					}
				}
			}
			if search.Options.Macroal && !macroalstatus[label] {
				// we have failed to match all content regexes on this line,
				// no need to continue with this search on this file
				search.deactivate()
				r.Parameters.Searches[label] = search
			}
		}
		if !hasactivechecks {
			continuereadingfile = false
		}
	}
	// done with file content inspection, loop over the checks one more time
	// 1. if MACROAL is set and the search succeeded, store the file in each content check
	// 2. If any check with inversematch=true failed to match, record that as a success
	// 3. deactivate searches that have matchall=true, but did not match against
	for label, search := range r.Parameters.Searches {
		// 1. if MACROAL is set and the search succeeded, store the file in each content check
		if search.Options.Macroal && macroalstatus[label] {
			debugprint("checkContent: macroal is set and search label '%s' passed on file '%s'\n",
				label, f.filename)
			// we match all content regexes on all lines of the file,
			// as requested via the Macroal flag
			// now store the filename in all checks
			for i := range search.checks {
				if search.checks[i].code&checkContent == 0 {
					continue
				}
				c := &search.checks[i]
				if c.wantThis(checksstatus[label][i]) {
					c.storeMatch(f.filename)
				}
			}
			// we're done with this search
			continue
		}
		// 2. If any check with inversematch=true failed to match, record that as a success
		for i := range search.checks {
			if search.checks[i].code&checkContent == 0 {
				continue
			}
			c := &search.checks[i]
			if !checksstatus[label][i] && c.inversematch {
				debugprint("in search '%s' on file '%s', check '%s' has not matched and is set to inversematch, record this as a positive result\n",
					label, f.filename, c.value)
				if c.wantThis(checksstatus[label][i]) {
					c.storeMatch(f.filename)
				}
				// adjust check status to true because the check did in fact match as an inverse
				checksstatus[label][i] = true
			}
		}
		// 3. deactivate searches that have matchall=true, but did not match against
		if search.isactive && (search.checkmask&checkContent != 0) && search.Options.MatchAll {
			for i := range search.checks {
				if search.checks[i].code&checkContent == 0 {
					continue
				}
				c := &search.checks[i]
				// check hasn't matched, or has matched and we didn't want it to, deactivate the search
				if !checksstatus[label][i] || (checksstatus[label][i] && c.inversematch) {
					if c.wantThis(checksstatus[label][i]) {
						c.storeMatch(f.filename)
					}
					search.deactivate()
				}
			}
		}
	}
	return
}

func (r *run) checkHash(f fileEntry, hashtype checkType) {
	var (
		err error
	)
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("checkHash() -> %v", e)
			walkingErrors = append(walkingErrors, err.Error())
		}
	}()
	// skip this check if no search has anything to run
	nothingToDo := true
	for _, search := range r.Parameters.Searches {
		if search.isactive && (search.checkmask&hashtype) != 0 {
			nothingToDo = false
		}
	}
	if nothingToDo {
		return
	}
	hash, err := getHash(f, hashtype)
	if err != nil {
		panic(err)
	}
	for label, search := range r.Parameters.Searches {
		if search.isactive && (search.checkmask&hashtype) != 0 {
			for i := range search.checks {
				if search.checks[i].code&hashtype == 0 {
					continue
				}
				c := &search.checks[i]
				match := false
				if c.value == hash {
					match = true
					debugprint("checkHash: file '%s' matches checksum '%s'\n", f.filename, c.value)
				}
				if c.wantThis(match) {
					c.storeMatch(f.filename)
				} else if search.Options.MatchAll {
					search.deactivate()
				}
			}
		}
		r.Parameters.Searches[label] = search
	}
	return
}

func (r *run) checkBytes(f fileEntry) {
	var (
		err error
	)
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("checkBytes() -> %v", e)
		}
	}()
	reader := f.getReader()
	defer f.Close()

	// Must use the "label,search" for loop iteration because of
	// map[string]search for parameters
	for _, search := range r.Parameters.Searches {
		if !search.isactive {
			continue
		}

		for _, c := range search.checks {

			decodedByte := make([]byte, hex.DecodedLen(len(c.bytes)))
			_, err := hex.Decode(decodedByte, c.bytes)
			if err != nil {
				e := fmt.Sprintf("unable to decode byte query: %v ", err)
				panic(e)
			}
			scansize := len(decodedByte) // Get the actual byte slice length
			blocksize := 4 * (2 << 10)   // 4096, 4k

			var bigbuf = make([]byte, (blocksize + (2 * scansize)))
			// Read slightly more than 4k so we have a full buffer to scan through
			// as we move into the for loop
			_, err = reader.Read(bigbuf)
			if err != nil && err != io.EOF {
				panic(err)
			}

			for err != io.EOF {
				if bytes.Contains(bigbuf, decodedByte) {
					// We do find the intended hex values in our scan
					// Break the loop and return
					// Not certain exactly how to do that here
					c.storeMatch(f.filename)
					break
				} else { //Explicitly, we do not find the hex value, and have not encountered EOF
					tempBuf := make([]byte, blocksize)
					_, err = reader.Read(tempBuf)
					if err != nil && err != io.EOF {
						panic(err)
					}
					// Take a small slice from bigbuf for sliding window
					smallBuf := make([]byte, 2*scansize)
					// Get the last two scan-lengths so as to permit the sliding window through End-of-Read
					copy(smallBuf, bigbuf[blocksize:])
					// Now make bigbuf out of two scan-lengths plus the 4k buffer read
					var bigbuf = make([]byte, (2 * scansize))
					copy(bigbuf, smallBuf)
					bigbuf = append(bigbuf, tempBuf...)
				}
			}
		}
	}

}

// getHash calculates the hash of a file.
func getHash(f fileEntry, hashType checkType) (hexhash string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getHash() -> %v", e)
		}
	}()
	reader := f.getReader()
	defer f.Close()
	debugprint("getHash: computing hash for '%s'\n", f.fd.Name())
	var h hash.Hash
	switch hashType {
	case checkMD5:
		h = md5.New()
	case checkSHA1:
		h = sha1.New()
	case checkSHA256:
		h = sha256.New()
	case checkSHA384:
		h = sha512.New384()
	case checkSHA512:
		h = sha512.New()
	case checkSHA3_224:
		h = sha3.New224()
	case checkSHA3_256:
		h = sha3.New256()
	case checkSHA3_384:
		h = sha3.New384()
	case checkSHA3_512:
		h = sha3.New512()
	default:
		err := fmt.Sprintf("getHash: Unkown hash type %d", hashType)
		panic(err)
	}
	buf := make([]byte, 4096)
	for {
		block, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if block == 0 {
			break
		}
		h.Write(buf[:block])
	}
	hexhash = fmt.Sprintf("%X", h.Sum(nil))
	return
}

// SearchResults is the search result element for an invocation of the file module
type SearchResults map[string]SearchResult

// SearchResult is the results of a single search the file module has executed. It contains
// a list of the files which were matched as a result of the search.
type SearchResult []MatchedFile

// MatchedFile describes a single file matched as a result of a search.
type MatchedFile struct {
	File     string `json:"file"`
	Search   Search `json:"search"`
	FileInfo Info   `json:"fileinfo"`
}

// Info describes the metadata associated with a file matched as a result of a
// search.
type Info struct {
	Size   float64 `json:"size"`
	Mode   string  `json:"mode"`
	Mtime  string  `json:"lastmodified"`
	SHA256 string  `json:"sha256,omitempty"`
}

// newResults allocates a Results structure
func newResults() *modules.Result {
	return &modules.Result{Elements: make(SearchResults), FoundAnything: false}
}

func (r *run) buildResults(t0 time.Time) (resStr string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("buildResults() -> %v", e)
		}
	}()
	res := newResults()
	elements := res.Elements.(SearchResults)
	var maxerrors int
	for label, search := range r.Parameters.Searches {
		var sr SearchResult
		// first pass on the results: if matchall is set, verify that all
		// the checks matched on all the files
		if search.Options.MatchAll {
			// The results processor which is part of the search has already prepared a list
			// of files that match all searches, so we leverage that to build our results.
			if len(search.filesMatchingAll) == 0 {
				search.filesMatchingAll = append(search.filesMatchingAll, "")
			}
			for _, matchedFile := range search.filesMatchingAll {
				var mf MatchedFile
				mf.File = matchedFile
				if mf.File != "" {
					fi, err := os.Stat(mf.File)
					if err != nil {
						panic(err)
					}
					mf.FileInfo.Size = float64(fi.Size())
					mf.FileInfo.Mode = fi.Mode().String()
					mf.FileInfo.Mtime = fi.ModTime().UTC().String()
					if search.Options.ReturnSHA256 {
						f := fileEntry{filename: mf.File}
						mf.FileInfo.SHA256, err = getHash(f, checkSHA256)
						if err != nil {
							panic(err)
						}
					}
				}
				mf.Search = *search
				mf.Search.Options.MatchLimit = 0
				// store the value of maxerrors if greater than the one
				// we already have, we'll need it further down to return
				// the right number of walking errors
				if int(mf.Search.Options.MaxErrors) > maxerrors {
					maxerrors = int(mf.Search.Options.MaxErrors)
				}
				mf.Search.Options.MaxDepth = 0
				mf.Search.Options.MaxErrors = 0
				mf.Search.Options.MatchAll = search.Options.MatchAll
				sr = append(sr, mf)
			}
			// done with this search, go to the next one
			goto nextsearch
		}
		// if matchall is not set, store each file on each check that matched individually
		for _, c := range search.checks {
			// if this check matched nothing, store it in a search result
			// where the File value is the empty string
			if len(c.matchedfiles) == 0 {
				c.matchedfiles = append(c.matchedfiles, "")
			}
			for _, file := range c.matchedfiles {
				var mf MatchedFile
				mf.File = file
				if mf.File != "" {
					fi, err := os.Stat(file)
					if err != nil {
						panic(err)
					}
					mf.FileInfo.Size = float64(fi.Size())
					mf.FileInfo.Mode = fi.Mode().String()
					mf.FileInfo.Mtime = fi.ModTime().UTC().String()
					mf.Search.Paths = []string{filepath.Dir(mf.File)}
				} else {
					mf.Search.Paths = search.Paths
				}
				mf.Search.Options.MatchLimit = 0
				// store the value of maxerrors if greater than the one
				// we already have, we'll need it further down to return
				// the right number of walking errors
				if int(search.Options.MaxErrors) > maxerrors {
					maxerrors = int(search.Options.MaxErrors)
				}
				mf.Search.Options.MaxDepth = 0
				mf.Search.Options.MaxErrors = 0
				mf.Search.Options.MatchAll = search.Options.MatchAll
				switch c.code {
				case checkContent:
					mf.Search.Contents = append(mf.Search.Contents, c.value)
				case checkBytes:
					mf.Search.Bytes = append(mf.Search.Bytes, c.value)
				case checkName:
					mf.Search.Names = append(mf.Search.Names, c.value)
				case checkSize:
					mf.Search.Sizes = append(mf.Search.Sizes, c.value)
				case checkMode:
					mf.Search.Modes = append(mf.Search.Modes, c.value)
				case checkMtime:
					mf.Search.Mtimes = append(mf.Search.Mtimes, c.value)
				case checkMD5:
					mf.Search.MD5 = append(mf.Search.MD5, c.value)
				case checkSHA1:
					mf.Search.SHA1 = append(mf.Search.SHA1, c.value)
				case checkSHA256, checkSHA384, checkSHA512:
					mf.Search.SHA2 = append(mf.Search.SHA2, c.value)
				case checkSHA3_224, checkSHA3_256, checkSHA3_384, checkSHA3_512:
					mf.Search.SHA3 = append(mf.Search.SHA2, c.value)
				}
				sr = append(sr, mf)
			}
		}
	nextsearch:
		elements[label] = sr
	}

	// calculate execution time
	t1 := time.Now()
	stats.Exectime = t1.Sub(t0).String()

	// store the stats in the response
	res.Statistics = stats

	// store the errors encountered along the way
	var errctr int
	for _, we := range walkingErrors {
		res.Errors = append(res.Errors, we)
		errctr++
		if errctr >= maxerrors {
			break
		}
	}
	if len(walkingErrors) > int(maxerrors) {
		res.Errors = append(res.Errors, fmt.Sprintf("%d errors were not returned (max errors = %d)",
			len(walkingErrors)-maxerrors, maxerrors))
	}
	// execution succeeded, set Success to true
	res.Success = true
	if stats.Totalhits > 0 {
		res.FoundAnything = true
	}
	debugprint("Tested files:     %.0f\n"+
		"Open Failed:      %.0f\n"+
		"Total hits:       %.0f\n"+
		"Execution time:   %s\n",
		stats.Filescount, stats.Openfailed,
		stats.Totalhits, stats.Exectime)
	JSONResults, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	resStr = string(JSONResults[:])
	return
}

// PrintResults() returns results in a human-readable format. if foundOnly is set,
// only results that have at least one match are returned.
// If foundOnly is not set, all results are returned, along with errors and
// statistics.
func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		el    SearchResults
		stats statistics
	)
	err = result.GetElements(&el)
	if err != nil {
		panic(err)
	}
	err = result.GetStatistics(&stats)
	if err != nil {
		panic(err)
	}

	for label, sr := range el {
		for _, mf := range sr {
			var out string
			if mf.File == "" {
				if foundOnly {
					continue
				}
				out = fmt.Sprintf("0 match found in search '%s'", label)
			} else {
				out = fmt.Sprintf("%s [lastmodified:%s, mode:%s, size:%.0f",
					mf.File, mf.FileInfo.Mtime, mf.FileInfo.Mode, mf.FileInfo.Size)
				if mf.FileInfo.SHA256 != "" {
					out += fmt.Sprintf(", sha256:%s", strings.ToLower(mf.FileInfo.SHA256))
				}
				out += fmt.Sprintf("] in search '%s'", label)
			}
			if mf.Search.Options.MatchAll {
				prints = append(prints, out)
				continue
			}
			out += " on checks"
			// if matchany, print the detail of the checks that matched with the filename
			for _, v := range mf.Search.Names {
				out += fmt.Sprintf(" name='%s'", v)
			}
			for _, v := range mf.Search.Sizes {
				out += fmt.Sprintf(" size='%s'", v)
			}
			for _, v := range mf.Search.Modes {
				out += fmt.Sprintf(" mode='%s'", v)
			}
			for _, v := range mf.Search.Mtimes {
				out += fmt.Sprintf(" mtime='%s'", v)
			}
			for _, v := range mf.Search.Contents {
				out += fmt.Sprintf(" content='%s'", v)
			}
			for _, v := range mf.Search.MD5 {
				out += fmt.Sprintf(" md5='%s'", v)
			}
			for _, v := range mf.Search.SHA1 {
				out += fmt.Sprintf(" sha1='%s'", v)
			}
			for _, v := range mf.Search.SHA2 {
				out += fmt.Sprintf(" sha2='%s'", v)
			}
			for _, v := range mf.Search.SHA3 {
				out += fmt.Sprintf(" sha3='%s'", v)
			}
			prints = append(prints, out)
		}
	}
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
		stat := fmt.Sprintf("Statistics: %.0f files checked, %.0f failed to open, %.0f matched, ran in %s.",
			stats.Filescount, stats.Openfailed,
			stats.Totalhits, stats.Exectime)
		prints = append(prints, stat)
	}
	return
}

// Enhanced privacy mode for file module, mask file names being returned by the module
func (r *run) EnhancePrivacy(in modules.Result) (out modules.Result, err error) {
	var el SearchResults
	out = in
	// Mask errors; it's possible in some circumstances an error might contain file name or
	// path information
	for i := range out.Errors {
		out.Errors[i] = "masked"
	}
	// Mask file name components in elements
	err = out.GetElements(&el)
	if err != nil {
		return
	}
	for k, v := range el {
		for i := range v {
			if v[i].File != "" {
				v[i].File = "masked"
			}
		}
		el[k] = v
	}
	out.Elements = el
	return
}
