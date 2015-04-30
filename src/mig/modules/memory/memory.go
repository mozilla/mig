// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Sushant Dinesh sushant.dinesh94@gmail.com [:sushant94]
//
// Memory scanner module.

package memory

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mozilla/masche/listlibs"
	"github.com/mozilla/masche/memaccess"
	"github.com/mozilla/masche/process"
	"mig/modules"
	"regexp"
	"time"
)

func init() {
	modules.Register("memory", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters parameters
	Results    modules.Result
}

type parameters struct {
	Searches map[string]search `json:"searches,omitempty"`
}

func newParameters() *parameters {
	var p parameters
	p.Searches = make(map[string]search)
	return &p
}

type search struct {
	Description string           `json:"description,omitempty"`
	Processes   []string         `json:"processes"`
	Libs        []string         `json:"libs,omitempty"` // Regular expression to match against loaded libs of a process.
	Scans       []scan           `json:"scans,omitempty"`
	Options     options          `json:"options,omitempty"`
	reLibs      []*regexp.Regexp // Store compiled regular expressions for []Libs
}

type scan struct {
	Bytes      string         `json:"bytes,omitempty"`      // Scan for raw bytes.
	Regexp     string         `json:"regexp,omitempty"`     // Scan against a regular expression.
	MatchCount float64        `json:"matchcount,omitempty"` // Maximum number of matches to look for in a processes memory before deactivating the scan.
	compiledRe *regexp.Regexp // Compiled regexp for this scan
	active     bool
}

type options struct {
	MatchAll bool `json:"matchall"`
}

type procList map[uint]procSearch // procList is a map that maps pid of a process to procSearch.
type procSearch struct {          // procSearch stores the labels of the searches that need to be performed on a process.
	Proc     process.Process
	Searches map[string]search     // Map from search "label" to the search struct.
	Results  map[string]procResult // Map from search "label" to the corrsponding result.
}

type elements map[string]searchResult
type searchResult []procResult
type procResult struct { // Per-Process result struct.
	Name         string   `json:"name"`                 // Process Name
	Pid          float64  `json:"pid"`                  // Process Id
	Libs         []string `json:"libs,omitempty"`       // Libraries loaded by the process which match the given search criteria.
	Found        bool     `json:"bytesfound,omitempty"` // Result for memory scans. Returns true if atleast one occurance is found.
	MatchedCount float64  `json:"matchedcount"`         // MatchCount keeps a count of number of scans the process matched to.
}

type statistics struct {
	ExecTime float64  `json:"exectime"` // Time for which the module ran before stopping.
	SoftErrs []string `json:"softerrs"` // softerrors which occurred during the execution of the module
	Failures []string `json:"failures"` // Errors due to which some scans might have been abandoned.
}

// Global var to store statistics.
var stats statistics

func (r *Runner) ValidateParameters() (err error) {
	for label, currsearch := range r.Parameters.Searches {
		err = validateLabel(label)
		if err != nil {
			return err
		}
		err = validateProcs(currsearch.Processes)
		if err != nil {
			return err
		}
		currsearch.reLibs, err = validateLibs(currsearch.Libs)
		if err != nil {
			return err
		}
		err = validateScans(currsearch.Scans)
		if err != nil {
			return err
		}
	}
	return
}

func validateProcs(procs []string) (err error) {
	if len(procs) == 0 {
		return fmt.Errorf("Each search must operate on atleast one processes.")
	}
	for i := range procs {
		_, err = regexp.Compile(procs[i])
		if err != nil {
			return err
		}
	}
	return
}

// validateLibs functions validate the regular expressions used for search and returns the compiled re for the same.
func validateLibs(libs []string) (compiledRe []*regexp.Regexp, err error) {
	for i := range libs {
		re, err := regexp.Compile(libs[i])
		if err != nil {
			return nil, err
		}
		compiledRe = append(compiledRe, re)
	}
	return
}

func validateScans(scans []scan) (err error) {
	for i := range scans {
		var re *regexp.Regexp
		// Check if the current scan is a scan for regexp or raw bytes.
		if scans[i].Regexp != "" {
			re, err = regexp.Compile(scans[i].Regexp)
		} else {
			_, err = hex.DecodeString(scans[i].Bytes)
		}
		if err != nil {
			return err
		}
		if scans[i].MatchCount <= 0 {
			scans[i].MatchCount = 1
		}
		scans[i].compiledRe = re
	}
	return
}

func validateLabel(label string) (err error) {
	allowedLabels, _ := regexp.Compile("[a-zA-Z0-9_]")
	if ok := allowedLabels.MatchString(label); !ok {
		err = fmt.Errorf("Illegal label. Please use only a-z A-Z 0-9 and _ characters in your label.")
	}
	return
}

func addSoftErrors(softerr []error) {
	for i := range softerr {
		stats.SoftErrs = append(stats.SoftErrs, fmt.Sprintf("%v", softerr[i]))
	}
}

// pgrep returns a list of process whose name match a given regular expression
func pgrep(name string) (procs []process.Process, harderr error, softerr []error) {
	regex := regexp.MustCompile(name)
	procs, harderr, softerr = process.OpenByName(regex)
	if harderr != nil {
		return
	}
	return
}

// Activate all the scans for a process.
func activateAllScans(p *procSearch) (count int) {
	count = 0
	for _, currSearch := range p.Searches {
		for i := range currSearch.Scans {
			currSearch.Scans[i].active = true
			count += 1
		}
	}
	return count
}

// Search for libraries loaded by a process.
func searchLoadedLibs(currproc *procSearch) (err error) {
	var libs []string
	for label, currsearch := range currproc.Searches {
		if len(currsearch.Libs) == 0 {
			continue
		}

		// Get a list of all libs loaded by the currproc.
		if len(libs) == 0 {
			libs, harderr, softerr := listlibs.ListLoadedLibraries(currproc.Proc)
			if harderr != nil {
				return harderr
			}

			addSoftErrors(softerr)
			if len(libs) == 0 {
				return
			}
		}

		for j := 0; j < len(currsearch.Libs); j += 1 {
			re := currsearch.reLibs[j]
			for i := range libs {
				loc := re.FindStringIndex(libs[i])
				if loc == nil {
					continue
				}

				// Check if we already have a result associated with this search label in the result hash.
				// If we don't we create a new procResult with the process name and pid and add the libs found to it.
				var res procResult
				res, ok := currproc.Results[label]

				if !ok {
					pname, harderr, softerr := currproc.Proc.Name()
					if harderr != nil {
						return harderr
					}
					addSoftErrors(softerr)
					res.Name = pname
					res.Pid = float64(currproc.Proc.Pid())
				}

				res.Libs = append(res.Libs, libs[i])
				currproc.Results[label] = res
			}
		}
	}
	return
}

func scanProcMemory(currproc *procSearch) (harderr error) {
	scancount := activateAllScans(currproc)
	buffer_size := uint(4096)

	walkfn :=
		func(address uintptr, buf []byte) (keepSearching bool) {
			// Iterate through the searches to perform the scan
			for label, currsearch := range currproc.Searches {
				for i := range currsearch.Scans {
					currscan := currsearch.Scans[i]
					if !currscan.active {
						continue
					}
					// Check if we need to scan for rawbytes or for a regexp.
					if currscan.Bytes != "" {
						b, _ := hex.DecodeString(currscan.Bytes)
						if index := bytes.Index(buf, b); index == -1 {
							continue
						}
					} else {
						regex := currscan.compiledRe
						if loc := regex.FindIndex(buf); loc == nil {
							continue
						}
					}

					// If we are here, it means that we have found a match.
					var res procResult
					if _, ok := currproc.Results[label]; !ok {
						procname, harderr, softerr := currproc.Proc.Name()
						if harderr != nil {
							return false
						}

						addSoftErrors(softerr)
						res.Name = procname
						res.Pid = float64(currproc.Proc.Pid())
						res.Found = true
						res.MatchedCount = 1
						currproc.Results[label] = res
					} else {
						res = currproc.Results[label]
						res.MatchedCount++
						currproc.Results[label] = res
					}
					// Check if 'n' matches have been found. Deactivate the scan if we've found 'n' matches.
					if int(currproc.Results[label].MatchedCount) == int(currscan.MatchCount) {
						currscan.active = false
						scancount--
					}
					// If we have no more scans to perform on this process. Stop.
					if scancount == 0 {
						return false
					}
				}
			}
			return true
		}

	harderror, softerror := memaccess.SlidingWalkMemory(currproc.Proc, 0, buffer_size, walkfn)

	if harderror != nil {
		return harderror
	}

	addSoftErrors(softerror)
	return
}

func (r Runner) Run() (resStr string) {
	var (
		startTime time.Time
		err       error
	)

	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			stats.ExecTime = time.Since(startTime).Seconds() * 1000
			r.Results.Statistics = stats
			err, _ := json.Marshal(r.Results)
			resStr = string(err[:])
			return
		}
	}()

	err = modules.ReadInputParameters(&r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	// Used to calculate the ExecTime for the module.
	startTime = time.Now()

	// Aggregate all the process.
	proclist := make(procList)
	for label, currsearch := range r.Parameters.Searches {
		for _, process := range currsearch.Processes {
			procs, hard, soft := pgrep(process[j])
			if hard != nil {
				panic(hard)
			}
			addSoftErrors(soft)

			// Add currsearch to the processes returned by pgrep.
			for _, proc := range procs {
				_, ok := proclist[proc.Pid()]
				// Check if we have this process in the proclist already. If we don't we create a new procSearch and add it to the list.
				if !ok {
					procsearch := procSearch{Searches: make(map[string]search), Results: make(map[string]procResult)}
					procsearch.Proc = plist[i]
					proclist[pid] = procsearch
				}
				proclist[pid].Searches[label] = currsearch
			}
		}
	}

	for _, currproc := range proclist {
		err := searchLoadedLibs(&currproc)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
	}

	for _, currproc := range proclist {
		err := scanProcMemory(&currproc)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
	}

	stats.ExecTime = time.Since(startTime).Seconds() * 1000

	// Iterate through all the processes. Grab all the results.
	fres := make(elements)
	for _, currproc := range proclist {
		res := currproc.Results
		for label, cur := range res {
			fres[label] = append(fres[label], cur)
		}
		currproc.Proc.Close()
	}

	r.Results.Elements = fres
	return r.buildResults()
}

func (r Runner) buildResults() string {
	r.Results.Success = true
	if len(r.Results.Errors) > 0 {
		r.Results.Success = false
	}

	if len(r.Results.Elements.(elements)) > 0 {
		r.Results.FoundAnything = true
	}

	r.Results.Statistics = stats

	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
