// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

/* The file module provides functions to scan a file system. It can look into files
using regexes. It can search files by name. It can match hashes in md5, sha1,
sha256, sha384, sha512, sha3_224, sha3_256, sha3_384 and sha3_512.
The filesystem can be searches using pattern, as described in the Parameters
documentation at http://mig.mozilla.org/doc/module_file.html .
*/
package file /* import "mig.ninja/mig/modules/file" */

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
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

var debug bool = false

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
	Parameters Parameters
	Results    modules.Result
}

type Parameters struct {
	Searches map[string]search `json:"searches,omitempty"`
}

func newParameters() *Parameters {
	var p Parameters
	p.Searches = make(map[string]search)
	return &p
}

type search struct {
	Description  string   `json:"description,omitempty"`
	Paths        []string `json:"paths"`
	Contents     []string `json:"contents,omitempty"`
	Names        []string `json:"names,omitempty"`
	Sizes        []string `json:"sizes,omitempty"`
	Modes        []string `json:"modes,omitempty"`
	Mtimes       []string `json:"mtimes,omitempty"`
	MD5          []string `json:"md5,omitempty"`
	SHA1         []string `json:"sha1,omitempty"`
	SHA2         []string `json:"sha2,omitempty"`
	SHA3         []string `json:"sha3,omitempty"`
	Options      options  `json:"options,omitempty"`
	checks       []check
	checkmask    checkType
	isactive     bool
	iscurrent    bool
	currentdepth uint64
}

type options struct {
	MaxDepth     float64  `json:"maxdepth"`
	RemoteFS     bool     `json:"remotefs,omitempty"`
	MatchAll     bool     `json:"matchall"`
	Macroal      bool     `json:"macroal"`
	Mismatch     []string `json:"mismatch"`
	MatchLimit   float64  `json:"matchlimit"`
	Debug        string   `json:"debug,omitempty"`
	ReturnSHA256 bool     `json:"returnsha256,omitempty"`
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
)

type check struct {
	code                   checkType
	matched                uint64
	matchedfiles           []string
	value                  string
	regex                  *regexp.Regexp
	minsize, maxsize       uint64
	minmtime, maxmtime     time.Time
	inversematch, mismatch bool
}

func (s *search) makeChecks() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makeChecks() -> %v", e)
		}
	}()
	if s.Options.Debug == "print" {
		debug = true
	}
	if s.Options.MaxDepth == 0 {
		s.Options.MaxDepth = 1000
	}
	if s.Options.MatchLimit == 0 {
		s.Options.MatchLimit = 1000
	}
	for _, v := range s.Contents {
		var c check
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
		var c check
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
		var c check
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
		var c check
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
		var c check
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
		var c check
		c.code = checkMD5
		c.value = strings.ToUpper(v)
		if s.hasMismatch("md5") {
			c.mismatch = true
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.SHA1 {
		var c check
		c.code = checkSHA1
		c.value = strings.ToUpper(v)
		if s.hasMismatch("sha1") {
			c.mismatch = true
		}
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
	}
	for _, v := range s.SHA2 {
		var c check
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
		var c check
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
	return
}

func (s *search) hasMismatch(filter string) bool {
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
		n          uint64 = 0
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
		isDays bool   = false
		n      uint64 = 0
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

func (s *search) activate() {
	s.isactive = true
	return
}

func (s *search) deactivate() {
	s.isactive = false
	return
}

func (s *search) increasedepth() {
	s.currentdepth++
	return
}

func (s *search) decreasedepth() {
	s.currentdepth--
	return
}

func (s *search) markcurrent() {
	s.iscurrent = true
	return
}

func (s *search) unmarkcurrent() {
	s.iscurrent = false
	return
}

func (c *check) storeMatch(file string) {
	store := true
	for _, storedFile := range c.matchedfiles {
		// only store files once per check
		if file == storedFile {
			store = false
		}
	}
	if store {
		c.matched++
		c.matchedfiles = append(c.matchedfiles, file)
	}
	return
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
		for _, mismatch := range s.Options.Mismatch {
			debugprint("validating mismatch '%s'\n", mismatch)
			err = validateMismatch(mismatch)
			if err != nil {
				return
			}
		}
	}
	return
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

func (r *run) Run(in io.Reader) (resStr string) {
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
		// check if a search needs to be activated by comparing
		// the search paths with the current path. if one matches,
		// then the search is activated.
		for _, p := range search.Paths {
			debugprint("comparing current path '%s' with candidate search '%s'\n", path, p)
			if len(path) >= len(p) && p == path[:len(p)] {
				search.activate()
				search.markcurrent()
				search.increasedepth()
			} else {
				search.unmarkcurrent()
			}
		}
		// we're entering a new directory, increase the depth counter
		// of active searches, and deactivate a search that is too deep
		if search.isactive {
			if search.currentdepth > uint64(search.Options.MaxDepth) {
				debugprint("deactivating search '%s' because depth %d > %.0f\n", label, search.currentdepth, search.Options.MaxDepth)
				search.deactivate()
			} else {
				activesearches++
			}
		}
		// if we reached the limit of matches we're allowed to return, deactivate this search
		if stats.Totalhits >= search.Options.MatchLimit {
			search.deactivate()
			search.unmarkcurrent()
			activesearches--
			r.Parameters.Searches[label] = search
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
			entryAbsPath := path
			// append path separator if missing
			if entryAbsPath[len(entryAbsPath)-1] != os.PathSeparator {
				entryAbsPath += string(os.PathSeparator)
			}
			entryAbsPath += dirEntry.Name()
			// this entry is a subdirectory, keep it for later
			if dirEntry.IsDir() {
				// append trailing slash
				if entryAbsPath[len(entryAbsPath)-1] != os.PathSeparator {
					entryAbsPath += string(os.PathSeparator)
				}
				subdirs = append(subdirs, entryAbsPath)
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
			panic(err)
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
	r.checkContent(file)
	r.checkHash(file, checkMD5)
	r.checkHash(file, checkSHA1)
	r.checkHash(file, checkSHA256)
	r.checkHash(file, checkSHA384)
	r.checkHash(file, checkSHA512)
	r.checkHash(file, checkSHA3_224)
	r.checkHash(file, checkSHA3_256)
	r.checkHash(file, checkSHA3_384)
	r.checkHash(file, checkSHA3_512)
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
			} else {
				debugprint("wantThis=false\n")
				return false
			}
		} else {
			if c.mismatch {
				debugprint("wantThis=false\n")
				return false
			} else {
				debugprint("wantThis=true\n")
				return true
			}
		}
	} else {
		if c.inversematch {
			if c.mismatch {
				debugprint("wantThis=false\n")
				return false
			} else {
				debugprint("wantThis=true\n")
				return true
			}
		} else {
			if c.mismatch {
				debugprint("wantThis=true\n")
				return true
			} else {
				debugprint("wantThis=false\n")
				return false
			}
		}
	}
}

func (s search) checkName(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkName) != 0 {
		for i, c := range s.checks {
			if (c.code & checkName) == 0 {
				continue
			}
			match := c.regex.MatchString(path.Base(fi.Name()))
			if match {
				debugprint("file name '%s' matches regex '%s'\n", fi.Name(), c.value)
			}
			if c.wantThis(match) {
				c.storeMatch(file)
			} else {
				matchedall = false
			}
			s.checks[i] = c
		}
	}
	return
}

func (s search) checkMode(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkMode) != 0 {
		for i, c := range s.checks {
			if (c.code & checkMode) == 0 {
				continue
			}
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
			s.checks[i] = c
		}
	}
	return
}

func (s search) checkSize(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkSize) != 0 {
		for i, c := range s.checks {
			if (c.code & checkSize) == 0 {
				continue
			}
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
			s.checks[i] = c
		}
	}
	return
}

func (s search) checkMtime(file string, fi os.FileInfo) (matchedall bool) {
	matchedall = true
	if (s.checkmask & checkMtime) != 0 {
		for i, c := range s.checks {
			if (c.code & checkMtime) == 0 {
				continue
			}
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
			s.checks[i] = c
		}
	}
	return
}

func (r *run) checkContent(file string) {
	var (
		err error
		fd  *os.File
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
		for i, c := range search.checks {
			if c.code&checkContent == 0 {
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
	fd, err = os.Open(file)
	if err != nil {
		stats.Openfailed++
		panic(err)
	}
	defer fd.Close()
	// iterate over the file content
	scanner := bufio.NewScanner(fd)
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
			for i, c := range search.checks {
				// skip this check if it's not a content check or if it has already matched
				if c.code&checkContent == 0 || (checksstatus[label][i] && !search.Options.Macroal) {
					continue
				}
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
								c.storeMatch(file)
							}
						}
					} else {
						if search.Options.Macroal {
							debugprint("checkContent: [fail] must match no line but current line matched. regex='%s', line='%s'\n",
								c.value, scanner.Text())
							macroalstatus[label] = false
							if c.wantThis(true) {
								c.storeMatch(file)
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
								c.storeMatch(file)
							}
						} else {
							debugprint("checkContent: [fail] much match any line and current line didn't match. regex='%s', line='%s'\n",
								c.value, scanner.Text())
						}
					}
				}
				search.checks[i] = c
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
				label, file)
			// we match all content regexes on all lines of the file,
			// as requested via the Macroal flag
			// now store the filename in all checks
			for i, c := range search.checks {
				if c.code&checkContent == 0 {
					continue
				}
				if c.wantThis(checksstatus[label][i]) {
					c.storeMatch(file)
				}
				search.checks[i] = c
			}
			// we're done with this search
			continue
		}
		// 2. If any check with inversematch=true failed to match, record that as a success
		for i, c := range search.checks {
			if c.code&checkContent == 0 {
				continue
			}
			if !checksstatus[label][i] && c.inversematch {
				debugprint("in search '%s' on file '%s', check '%s' has not matched and is set to inversematch, record this as a positive result\n",
					label, file, c.value)
				if c.wantThis(checksstatus[label][i]) {
					c.storeMatch(file)
				}
				// adjust check status to true because the check did in fact match as an inverse
				checksstatus[label][i] = true
				search.checks[i] = c
			}
		}
		// 3. deactivate searches that have matchall=true, but did not match against
		if search.isactive && (search.checkmask&checkContent != 0) && search.Options.MatchAll {
			for i, c := range search.checks {
				if c.code&checkContent == 0 {
					continue
				}
				// check hasn't matched, or has matched and we didn't want it to, deactivate the search
				if !checksstatus[label][i] || (checksstatus[label][i] && c.inversematch) {
					if c.wantThis(checksstatus[label][i]) {
						c.storeMatch(file)
						search.checks[i] = c
					}
					search.deactivate()
				}
			}
		}
		r.Parameters.Searches[label] = search
	}
	return
}

func (r *run) checkHash(file string, hashtype checkType) {
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
	hash, err := getHash(file, hashtype)
	if err != nil {
		panic(err)
	}
	for label, search := range r.Parameters.Searches {
		if search.isactive && (search.checkmask&hashtype) != 0 {
			for i, c := range search.checks {
				if c.code&hashtype == 0 {
					continue
				}
				match := false
				if c.value == hash {
					match = true
					debugprint("checkHash: file '%s' matches checksum '%s'\n", file, c.value)
				}
				if c.wantThis(match) {
					c.storeMatch(file)
				} else if search.Options.MatchAll {
					search.deactivate()
				}
				search.checks[i] = c
			}
		}
		r.Parameters.Searches[label] = search
	}
	return
}

// getHash calculates the hash of a file.
func getHash(file string, hashType checkType) (hexhash string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getHash() -> %v", e)
		}
	}()
	fd, err := os.Open(file)
	if err != nil {
		stats.Openfailed++
		panic(err)
	}
	defer fd.Close()
	debugprint("getHash: computing hash for '%s'\n", fd.Name())
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
	hexhash = fmt.Sprintf("%X", h.Sum(nil))
	return
}

type SearchResults map[string]searchresult

type searchresult []matchedfile

type matchedfile struct {
	File     string   `json:"file"`
	Search   search   `json:"search"`
	FileInfo fileinfo `json:"fileinfo"`
}

type fileinfo struct {
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
	for label, search := range r.Parameters.Searches {
		var sr searchresult
		// first pass on the results: if matchall is set, verify that all
		// the checks matched on all the files
		if search.Options.MatchAll {
			// collect all the files that were found across all checks of this search
			var allFiles, matchedFiles []string
			for _, c := range search.checks {
				// populate allFiles as a slice of unique files
				for _, matchedFile := range c.matchedfiles {
					store := true
					for _, afile := range allFiles {
						if afile == matchedFile {
							store = false
						}
					}
					if store {
						allFiles = append(allFiles, matchedFile)
					}
				}
			}
			// verify that each file has matched on all the checks
			for _, foundFile := range allFiles {
				debugprint("checking if file %s matched all checks\n", foundFile)
				matchedallchecks := true
				for _, c := range search.checks {
					found := false
					for _, matchedFile := range c.matchedfiles {
						if foundFile == matchedFile {
							found = true
						}
					}
					if !found {
						debugprint("check %d did not match\n", c.code)
						matchedallchecks = false
						break
					}
				}
				if matchedallchecks {
					matchedFiles = append(matchedFiles, foundFile)
				}
			}
			if len(matchedFiles) == 0 {
				matchedFiles = append(matchedFiles, "")
			}
			// now that we have a clean list of files that matched all checks, store it
			for _, matchedFile := range matchedFiles {
				var mf matchedfile
				mf.File = matchedFile
				if mf.File != "" {
					stats.Totalhits++
					fi, err := os.Stat(mf.File)
					if err != nil {
						panic(err)
					}
					mf.FileInfo.Size = float64(fi.Size())
					mf.FileInfo.Mode = fi.Mode().String()
					mf.FileInfo.Mtime = fi.ModTime().UTC().String()
					if search.Options.ReturnSHA256 {
						mf.FileInfo.SHA256, err = getHash(mf.File, checkSHA256)
						if err != nil {
							panic(err)
						}
					}
				}
				mf.Search = search
				mf.Search.Options.MatchLimit = 0
				mf.Search.Options.MaxDepth = 0
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
				var mf matchedfile
				mf.File = file
				if mf.File != "" {
					stats.Totalhits++
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
				mf.Search.Options.MaxDepth = 0
				mf.Search.Options.MatchAll = search.Options.MatchAll
				switch c.code {
				case checkContent:
					mf.Search.Contents = append(mf.Search.Contents, c.value)
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
				case checkSHA256:
				case checkSHA384:
				case checkSHA512:
					mf.Search.SHA2 = append(mf.Search.SHA2, c.value)
				case checkSHA3_224:
				case checkSHA3_256:
				case checkSHA3_384:
				case checkSHA3_512:
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
	for _, we := range walkingErrors {
		res.Errors = append(res.Errors, we)
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
	JsonResults, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	resStr = string(JsonResults[:])
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
