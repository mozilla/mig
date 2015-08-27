// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
//              Sushant Dinesh sushant.dinesh94@gmail.com [:sushant94]

/* The memory module implements scanning of the memory of processes
using the Masche memory scanning package.
Documentation of this module is online at http://mig.mozilla.org/doc/module_memory.html
*/
package memory /* import "mig.ninja/mig/modules/memory" */

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mozilla/masche/listlibs"
	"github.com/mozilla/masche/memaccess"
	"github.com/mozilla/masche/process"
	"io"
	"mig.ninja/mig/modules"
	"regexp"
	"time"
)

var debug bool = false

type module struct {
}

func (m *module) NewRun() modules.Runner {
	return new(run)
}

func init() {
	modules.Register("memory", new(module))
}

type run struct {
	Parameters params
	Results    modules.Result
}

type params struct {
	Searches map[string]search `json:"searches,omitempty"`
}

func newParameters() *params {
	var p params
	p.Searches = make(map[string]search)
	return &p
}

type search struct {
	Description string   `json:"description,omitempty"`
	Names       []string `json:"names,omitempty"`
	Libraries   []string `json:"libraries,omitempty"`
	Bytes       []string `json:"bytes,omitempty"`
	Contents    []string `json:"contents,omitempty"`
	Options     options  `json:"options,omitempty"`
	checks      []check
	checkmask   checkType
	isactive    bool
}

type options struct {
	Offset      float64 `json:"offset,omitempty"`
	MaxLength   float64 `json:"maxlength,omitempty"`
	LogFailures bool    `json:"logfailures,omitempty"`
	MatchAll    bool    `json:"matchall,omitempty"`
}

type checkType uint64

// BitMask for the type of check to apply to a given file
// see documentation about iota for more info
const (
	checkName checkType = 1 << (64 - 1 - iota)
	checkLib
	checkByte
	checkContent
)

type check struct {
	code      checkType
	matched   uint64
	matchedPs []process.Process
	value     string
	bytes     []byte
	regex     *regexp.Regexp
}

type searchResults map[string]searchresult

type searchresult []matchedps

type matchedps struct {
	Process psres  `json:"process"`
	Search  search `json:"search"`
}

type psres struct {
	Name string  `json:"name"`
	Pid  float64 `json:"pid"`
}

/* Statistic counters:
- ProcessCount is the total numbers of processes inspected
- MemoryRead is the total number of bytes of memory inspected
- Totalhits is the total number of checks that hit
- Failures is an array of soft errors encountered during inspection
- Exectime is the total execution time of the module
*/
type statistics struct {
	ProcessCount float64  `json:"processcount"`
	MemoryRead   float64  `json:"memoryread"`
	TotalHits    float64  `json:"totalhits"`
	Failures     []string `json:"failures,omitempty"`
	Exectime     string   `json:"exectime"`
}

// global stats variable
var stats statistics

// newResults allocates a Results structure
func newResults() *modules.Result {
	return &modules.Result{Elements: make(searchResults)}
}

func (s *search) makeChecks() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("makeChecks() -> %v", e)
		}
	}()
	if s.Options.MaxLength == 0 {
		s.Options.MaxLength = float64(^uint64(0))
	}
	for _, v := range s.Names {
		var c check
		c.code = checkName
		c.value = v
		c.regex = regexp.MustCompile(v)
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
		if debug {
			fmt.Printf("adding name check with value '%s'\n", c.value)
		}
	}
	for _, v := range s.Libraries {
		var c check
		c.code = checkLib
		c.value = v
		c.regex = regexp.MustCompile(v)
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
		if debug {
			fmt.Printf("adding library check with value '%s'\n", c.value)
		}
	}
	for _, v := range s.Contents {
		var c check
		c.code = checkContent
		c.value = v
		c.regex = regexp.MustCompile(v)
		s.checks = append(s.checks, c)
		s.checkmask |= c.code
		if debug {
			fmt.Printf("adding content check with value '%s'\n", c.value)
		}
	}
	for _, v := range s.Bytes {
		var c check
		c.code = checkByte
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

func (s *search) activate() {
	s.isactive = true
	return
}

func (s *search) deactivate() {
	s.isactive = false
	return
}

func (c *check) storeMatch(proc process.Process) {
	if debug {
		fmt.Printf("storing process id %d that matched check %d\n",
			proc.Pid(), c.code)
	}
	store := true
	for _, storedPs := range c.matchedPs {
		// only store files once per check
		if proc.Pid() == storedPs.Pid() {
			store = false
		}
	}
	if store {
		c.matched++
		c.matchedPs = append(c.matchedPs, proc)
	}
	return
}

func (r *run) ValidateParameters() (err error) {
	var labels []string
	for label, s := range r.Parameters.Searches {
		labels = append(labels, label)
		if debug {
			fmt.Printf("validating label '%s'\n", label)
		}
		err = validateLabel(label)
		if err != nil {
			return
		}
		for _, r := range s.Contents {
			if debug {
				fmt.Printf("validating content '%s'\n", r)
			}
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Names {
			if debug {
				fmt.Printf("validating name '%s'\n", r)
			}
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Libraries {
			if debug {
				fmt.Printf("validating library '%s'\n", r)
			}
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Bytes {
			if debug {
				fmt.Printf("validating bytes '%s'\n", r)
			}
			err = validateBytes(r)
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
	labelregexp := `^([a-zA-Z0-9_-]){1,64}$`
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
	_, err := regexp.Compile(regex)
	if err != nil {
		return fmt.Errorf("Invalid regexp '%s'. Must be a regexp. Compilation failed with '%v'", regex, err)
	}
	return nil
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

func (r *run) Run(in io.Reader) (out string) {
	var ts statistics
	stats = ts
	// in debug mode, we just panic
	if !debug {
		defer func() {
			if e := recover(); e != nil {
				// return error in json
				res := newResults()
				res.Statistics = stats
				res.Errors = append(res.Errors, fmt.Sprintf("%v", e))
				res.Success = false
				err, _ := json.Marshal(res)
				out = string(err[:])
				return
			}
		}()
	}
	t0 := time.Now()
	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}
	// create the checks based on the search parameters
	for label, search := range r.Parameters.Searches {
		if debug {
			fmt.Println("making checks for label", label)
		}
		err := search.makeChecks()
		if err != nil {
			panic(err)
		}
		r.Parameters.Searches[label] = search
	}
	// evaluate each process one by one
	pids, err, serr := process.GetAllPids()
	if err != nil {
		panic(err)
	}
	if debug {
		fmt.Println("found", len(pids), "processes to evaluate")
	}
	for _, err = range serr {
		stats.Failures = append(stats.Failures, err.Error())
	}
	for _, pid := range pids {
		// activate all searches
		for label, search := range r.Parameters.Searches {
			search.activate()
			r.Parameters.Searches[label] = search
		}
		proc, err, serr := process.OpenFromPid(pid)
		if err != nil {
			// if we encounter a hard failure, skip this process
			stats.Failures = append(stats.Failures, err.Error())
			continue
		}
		for _, err = range serr {
			// soft failures are just logged but we continue inspection
			stats.Failures = append(stats.Failures, err.Error())
		}
		err = r.evaluateProcess(proc)
		if err != nil {
			stats.Failures = append(stats.Failures, err.Error())
		}
		stats.ProcessCount++
	}

	out, err = r.buildResults(t0)
	if err != nil {
		panic(err)
	}

	if debug {
		fmt.Println("---- results ----")
		var tmpres modules.Result
		err = json.Unmarshal([]byte(out), &tmpres)
		printedResults, err := r.PrintResults(tmpres, false)
		if err != nil {
			panic(err)
		}
		for _, res := range printedResults {
			fmt.Println(res)
		}
	}
	return
}

// evaluateProcess takes a single process and applies searches to it. All searches are evaluated. The `name` and `library`
// checks are run first, and if needed, the memory of the process is read to run the checks on `contents` and `bytes`.
// The logic is optimized to only read the process memory once and apply all the checks to it.
func (r *run) evaluateProcess(proc process.Process) (err error) {
	if !debug {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("evaluateProcess() -> %v", e)
			}
		}()
	}
	procname, err, serr := proc.Name()
	if err != nil {
		return
	}
	for _, err = range serr {
		stats.Failures = append(stats.Failures, err.Error())
		if debug {
			fmt.Printf("evaluateProcess: soft error -> %v\n", err)
		}
	}
	if debug {
		fmt.Printf("evaluateProcess: evaluating proc %s\n", procname)
	}
	// first pass: apply all name & library checks against the current process
	for label, search := range r.Parameters.Searches {
		if !search.isactive {
			goto skip
		}
		if !search.checkName(proc, procname) && search.Options.MatchAll {
			if debug {
				fmt.Printf("evaluateProcess: proc %s does not match the names of search %s and matchall is set\n",
					procname, label)
			}
			search.deactivate()
			goto skip
		}
		if !search.checkLibraries(proc, procname) && search.Options.MatchAll {
			if debug {
				fmt.Printf("evaluateProcess: proc %s does not match the libraries of search %s and matchall is set\n",
					procname, label)
			}
			search.deactivate()
			goto skip
		}
	skip:
		r.Parameters.Searches[label] = search
	}
	// second pass: walk the memory of the process and apply contents regexes and bytes searches
	return r.walkProcMemory(proc, procname)
}

// checkName compares the "name" (binary full path) of a process against name checks
func (s search) checkName(proc process.Process, procname string) (matchedall bool) {
	matchedall = true
	if s.checkmask&checkName == 0 {
		// this search has no name check
		return
	}
	for i, c := range s.checks {
		if c.code&checkName == 0 {
			continue
		}
		if debug {
			fmt.Println("checkName: evaluating", procname, proc.Pid(), "against check", c.value)
		}
		if c.regex.MatchString(procname) {
			if debug {
				fmt.Printf("checkName: proc name '%s' pid %d matches regex '%s'\n",
					procname, proc.Pid(), c.value)
			}
			c.storeMatch(proc)
		} else {
			if debug {
				fmt.Printf("checkName: proc name '%s' pid %d does not match regex '%s'\n",
					procname, proc.Pid(), c.value)
			}
			matchedall = false
		}
		s.checks[i] = c
	}
	return
}

// checkLibraries retrieves the linked libraries of a process and compares them with the
// regexes of library checks
func (s search) checkLibraries(proc process.Process, procname string) (matchedall bool) {
	matchedall = true
	if s.checkmask&checkLib == 0 {
		// this search has no library check
		return
	}
	for i, c := range s.checks {
		if c.code&checkLib == 0 {
			continue
		}
		libs, err, serr := listlibs.GetMatchingLoadedLibraries(proc, c.regex)
		if err != nil {
			stats.Failures = append(stats.Failures, err.Error())
		}
		if len(serr) > 0 && s.Options.LogFailures {
			stats.Failures = append(stats.Failures, err.Error())
			if debug {
				for _, err := range serr {
					fmt.Printf("checkLibraries: soft error -> %v\n", err)
				}
			}
		}
		if len(libs) > 0 {
			if debug {
				fmt.Printf("checkLibraries: proc name '%s' pid %d has libraries matching regex '%s'\n",
					procname, proc.Pid(), c.value)
			}
			c.storeMatch(proc)
		} else {
			matchedall = false
		}
		s.checks[i] = c
	}
	return
}

func (r *run) walkProcMemory(proc process.Process, procname string) (err error) {
	// find longest byte string to search for, which determines the buffer size
	bufsize := uint(4096)
	// find lowest offset, which determines start address
	offset := ^uintptr(0) >> 1
	// verify that at least one search is interested in inspecting the memory
	// of this process
	shouldWalkMemory := false
	// if at least one search wants to log failures, do so, otherwise omit them
	logFailures := false
	for label, search := range r.Parameters.Searches {
		// if the search is not active or the search as no content or by check to run, skip it
		if !search.isactive || (search.checkmask&checkContent == 0 && search.checkmask&checkByte == 0) {
			search.deactivate()
			r.Parameters.Searches[label] = search
			continue
		}
		shouldWalkMemory = true
		// find the largest bufsize needed
		for _, c := range search.checks {
			if c.code&checkByte != 0 {
				if uint(len(c.bytes)) > (bufsize / 2) {
					bufsize = 2 * uint(len(c.bytes))
					// pad to always have an even bufsize
					if bufsize%2 != 0 {
						bufsize++
					}
				}
			}
		}
		// find the smallest offset needed
		if uintptr(search.Options.Offset) < offset {
			offset = uintptr(search.Options.Offset)
		}
		if search.Options.LogFailures {
			logFailures = true
		}
	}
	if !shouldWalkMemory {
		if debug {
			fmt.Println("walkProcMemory: no check needs to read the memory of process", proc.Pid(), procname)
		}
		return
	}
	// keep track of the number of bytes read to exit when maxlength is reached
	var readBytes float64
	walkfn := func(curStartAddr uintptr, buf []byte) (keepSearching bool) {
		if readBytes == 0 {
			readBytes += float64(len(buf))
		} else {
			readBytes += float64(len(buf) / 2)
		}
		if debug {
			fmt.Println("walkProcMemory: reading", bufsize, "bytes starting at addr", curStartAddr, "; read", readBytes, "bytes so far")
		}
		for label, search := range r.Parameters.Searches {
			matchedall := true
			if !search.isactive {
				continue
			}
			// if the search is meant to stop at a given address, and we're passed
			// that point then deactivate the search now
			if readBytes >= search.Options.MaxLength {
				search.deactivate()
				goto skip
			}
			keepSearching = true
			for i, c := range search.checks {
				switch c.code {
				case checkContent:
					if c.regex.FindIndex(buf) == nil {
						// not found
						matchedall = false
						continue
					}
					c.storeMatch(proc)
					search.checks[i] = c
				case checkByte:
					if bytes.Index(buf, c.bytes) < 0 {
						// not found
						matchedall = false
						continue
					}
					c.storeMatch(proc)
					search.checks[i] = c
				}
			}
			// if all the checks have matched on this search, deactivate it
			if matchedall {
				search.deactivate()
			}
		skip:
			r.Parameters.Searches[label] = search
		}
		if debug && !keepSearching {
			fmt.Println("walkProcMemory: stopping the memory search for", proc.Pid(), procname)
		}
		return
	}
	if debug {
		fmt.Println("walkProcMemory: reading memory of", proc.Pid(), procname)
	}
	err, serr := memaccess.SlidingWalkMemory(proc, offset, bufsize, walkfn)
	if err != nil {
		return err
	}
	if logFailures {
		for _, err = range serr {
			stats.Failures = append(stats.Failures, err.Error())
			if debug {
				fmt.Printf("walkProcMemory: soft error -> %v\n", err)
			}
		}
	}
	stats.MemoryRead += readBytes
	return
}

func (r *run) buildResults(t0 time.Time) (resStr string, err error) {
	// in debug mode, we just panic
	if !debug {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("buildResults() -> %v", e)
			}
		}()
	}
	res := newResults()
	for label, search := range r.Parameters.Searches {
		var sr searchresult
		// if matchall is set
		//
		// all checks in the search must have match on any given process for it
		// to be included in the results.
		//
		// First: we build a list of processes that matched at list one check
		// in `allProcesses`.
		// Second: we prune the list of `allProcesses` to only keep the ones that
		// have matched on all the checks and store it in `matchedProcesses`
		// Third: we take the list in `matchedProcesses` and retrieve extra details
		// about the process itself (owner, ...) and store it into results
		//
		// An edge case: if the search has not matched on any process, a process
		// with an empty name and pid 0 is inserted to store the results. This
		// fake process always indicates that the search has failed to match.
		if search.Options.MatchAll {
			var allProcesses, matchedProcesses []process.Process
			// First: collect all the processes that were found across all
			// checks of this search, don't store duplicates
			for _, c := range search.checks {
				for _, matchedPs := range c.matchedPs {
					store := true
					for _, aps := range allProcesses {
						if aps.Pid() == matchedPs.Pid() {
							store = false
						}
					}
					if store {
						allProcesses = append(allProcesses, matchedPs)
					}
				}
			}
			// Second: prune the list to only keep the processes that matched
			// all checks
			for _, aps := range allProcesses {
				if debug {
					fmt.Println("checking if process", aps.Pid(), "matched all checks")
				}
				matchedallchecks := true
				for _, c := range search.checks {
					found := false
					for _, matchedPs := range c.matchedPs {
						if aps.Pid() == matchedPs.Pid() {
							found = true
						}
					}
					if !found {
						if debug {
							fmt.Println("check", c.code, "did not match")
						}
						matchedallchecks = false
						break
					}
				}
				if matchedallchecks {
					matchedProcesses = append(matchedProcesses, aps)
				}
			}
			if len(matchedProcesses) == 0 {
				var nullPs process.Process
				matchedProcesses = append(matchedProcesses, nullPs)
			}
			// Third: we have a clean list of files that matched all checks,
			// store it as search results
			for _, matchedPs := range matchedProcesses {
				var mps matchedps
				if matchedPs == nil {
					mps.Process.Name = ""
					mps.Process.Pid = 0
				} else {
					// we don't check for errors here, that's been done before
					mps.Process.Name, _, _ = matchedPs.Name()
					mps.Process.Pid = float64(matchedPs.Pid())
					stats.TotalHits++
					// TODO: get detailed info about process here
				}
				mps.Search = search
				// reset option fields so they get omitted
				mps.Search.Options.Offset = 0.0
				mps.Search.Options.MaxLength = 0.0
				mps.Search.Options.MatchAll = search.Options.MatchAll
				sr = append(sr, mps)
			}
			// done with this search, go to the next one
			goto nextsearch
		}

		// if matchall is not set (and the goto above wasn't called)
		//
		// now that `matchall` is handled, go through the list of processes and store
		// them into the searchresults structure, with the corresponding checks that matched
		for _, c := range search.checks {
			// if this check matched nothing, store it in a search result
			// where the File value is the empty string
			if len(c.matchedPs) == 0 {
				var nullps process.Process
				c.matchedPs = append(c.matchedPs, nullps)
			}
			for _, matchedPs := range c.matchedPs {
				var mps matchedps
				if matchedPs == nil {
					mps.Process.Name = ""
					mps.Process.Pid = 0
				} else {
					// we don't check for errors here, that's been done before
					mps.Process.Name, _, _ = matchedPs.Name()
					mps.Process.Pid = float64(matchedPs.Pid())
					stats.TotalHits++
					// TODO: get detailed info about process here
				}
				// reset option fields so they get omitted
				mps.Search.Options.Offset = 0.0
				mps.Search.Options.MaxLength = 0.0
				mps.Search.Options.MatchAll = search.Options.MatchAll
				switch c.code {
				case checkContent:
					mps.Search.Contents = append(mps.Search.Contents, c.value)
				case checkName:
					mps.Search.Names = append(mps.Search.Names, c.value)
				case checkLib:
					mps.Search.Libraries = append(mps.Search.Libraries, c.value)
				case checkByte:
					mps.Search.Bytes = append(mps.Search.Bytes, c.value)
				}
				sr = append(sr, mps)
			}
		}
	nextsearch:
		res.Elements.(searchResults)[label] = sr
	}

	// calculate execution time
	t1 := time.Now()
	stats.Exectime = t1.Sub(t0).String()
	if debug {
		fmt.Printf("storing exectime %s\n", stats.Exectime)
	}
	// store the stats in the response
	res.Statistics = stats
	// execution succeeded, set Success to true
	res.Success = true
	if stats.TotalHits > 0 {
		res.FoundAnything = true
	}
	if debug {
		fmt.Printf("Processes Count:  %.0f\n"+
			"Memory read:      %.0f bytes\n"+
			"Total hits:       %.0f\n"+
			"Failures:         %d\n"+
			"Execution time:   %s\n",
			stats.ProcessCount, stats.MemoryRead,
			stats.TotalHits, len(stats.Failures), stats.Exectime)
	}
	JsonResults, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	resStr = string(JsonResults[:])
	if debug {
		fmt.Println(resStr)
	}
	return
}

// PrintResults() returns results in a human-readable format. if foundOnly is set,
// only results that have at least one match are returned.
// If foundOnly is not set, all results are returned, along with errors and
// statistics.
func (r *run) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var (
		el    searchResults
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
		for _, mps := range sr {
			var out string
			if mps.Process.Name == "" {
				if foundOnly {
					continue
				}
				out = fmt.Sprintf("0 match found in search '%s'", label)
			} else {
				out = fmt.Sprintf("%s [pid:%.0f] in search '%s'",
					mps.Process.Name, mps.Process.Pid, label)
			}
			if mps.Search.Options.MatchAll {
				prints = append(prints, out)
				continue
			}
			out += " on checks"
			// if matchany, print the detail of the checks that matched with the filename
			for _, v := range mps.Search.Names {
				out += fmt.Sprintf(" name='%s'", v)
			}
			for _, v := range mps.Search.Libraries {
				out += fmt.Sprintf(" library='%s'", v)
			}
			for _, v := range mps.Search.Contents {
				out += fmt.Sprintf(" content='%s'", v)
			}
			for _, v := range mps.Search.Bytes {
				out += fmt.Sprintf(" byte='%s'", v)
			}
			prints = append(prints, out)
		}
	}
	if !foundOnly {
		for _, e := range stats.Failures {
			prints = append(prints, fmt.Sprintf("Failure: %v", e))
		}
		for _, e := range result.Errors {
			prints = append(prints, e)
		}
		stat := fmt.Sprintf("Statistics: %.0f processes checked, %.0f matched, %d failures, ran in %s.",
			stats.ProcessCount, stats.TotalHits, len(stats.Failures), stats.Exectime)
		prints = append(prints, stat)
	}
	return
}
