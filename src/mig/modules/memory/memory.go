// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Sushant Dinesh sushant.dinesh94@gmail.com [:sushant94]
//
// Memory scanner module.
// To run this module run 'make go_get_memory_deps' to download and build masche.

package memory

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mozilla/masche/listlibs"
	"github.com/mozilla/masche/memaccess"
	"github.com/mozilla/masche/process"
	"mig"
	"regexp"
)

func init() {
	mig.RegisterModule("memory", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters Parameters
	Results    mig.ModuleResult
}

type Parameters struct {
	Searches map[string]Search `json:"searches,omitempty"`
}

func newParameters() *Parameters {
	var p Parameters
	p.Searches = make(map[string]Search)
	return &p
}

type Search struct {
	Description string   `json:"description,omitempty"`
	Processes   []string `json:"processes"`
	Libs        []string `json:"libs,omitempty"`
	Scans       []Scan   `json:"scans,omitempty"`
	Options     options  `json:"options,omitempty"`
}

type Scan struct {
	Expression string  `json:"expression"`
	Length     float64 `json:"length"`
	MatchCount float64 `json:"matchcount"`
	Active     bool
}

type options struct {
	MatchAll bool `json:"matchall"`
}

type element map[string]SearchResult
type SearchResult []ProcResult

// procList is a map that maps pid of a process to procSearch.
// procSearch stores the labels of the searches that need to be performed on a process.
type ProcList map[uint]ProcSearch

type ProcSearch struct {
	Proc     process.Process
	Libs     []string
	Searches map[string]Search
	Results  map[string]ProcResult
}

// Per-Process result struct.
type ProcResult struct {
	// Process Name
	Name string `json:"name"`
	// Process Id
	Pid float64 `json:"pid"`
	// Libraries loaded
	Libs []string `json:"libs,omitempty"`
	// Result for memory scans. Returns 'n' occurances of the string from memory.
	BytesFound []string `json:"bytesfound,omitempty"`
	// MatchCount keeps a count of number of scans the process matched to.
	MatchedCount float64 `json:"matchedcount"`
}

func (r *Runner) ValidateParameters() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Validate Parameters: %v", e)
			return
		}
	}()

	allowedLabels, _ := regexp.Compile("[a-zA-Z0-9_]+")
	for label, currsearch := range r.Parameters.Searches {
		if ok := allowedLabels.MatchString(label); !ok {
			return fmt.Errorf("Illegal label. Please use only a-z A-Z 0-9 and _ characters in your label.")
		}
		if len(currsearch.Processes) == 0 {
			return fmt.Errorf("Each search must operate on atleast one processes.")
		}
		for i := range currsearch.Processes {
			_ = regexp.MustCompile(currsearch.Processes[i])
		}
		for i := range currsearch.Libs {
			_ = regexp.MustCompile(currsearch.Libs[i])
		}
		for i := range currsearch.Scans {
			_ = regexp.MustCompile(currsearch.Scans[i].Expression)
			if currsearch.Scans[i].MatchCount <= 0 {
				currsearch.Scans[i].MatchCount = 1
			}
			if currsearch.Scans[i].Length < 0 {
				currsearch.Scans[i].Length = 0
			}
		}
	}
	return
}

// needle - regular expression against which a process should be matched
func Pgrep(needle string) (procs []process.Process, harderr error, softerr []error) {
	regex, _ := regexp.Compile(needle)
	procs, harderr, softerr = process.OpenByName(regex)
	if harderr != nil {
		return
	}
	return
}

// Activate all the scans for a process.
func ActivateAllScans(p *ProcSearch) (count int) {
	count = 0
	for _, currSearch := range p.Searches {
		for i := range currSearch.Scans {
			currSearch.Scans[i].Active = true
			count += 1
		}
	}
	return count
}

// Search for libraries loaded by a process.
func SearchLoadedLibs(currproc *ProcSearch) (err error) {
	var (
		libs    []string
		harderr error
	)

	for label, currsearch := range currproc.Searches {
		if len(currsearch.Libs) == 0 {
			continue
		}

		// Compile a single RE for all libs in the same search.
		searchstr := currsearch.Libs[0]
		for i := 1; i < len(currsearch.Libs); i += 1 {
			searchstr = searchstr + "|" + currsearch.Libs[i]
		}
		re, _ := regexp.Compile(searchstr)

		if len(libs) == 0 {
			libs, harderr, _ = listlibs.ListLoadedLibraries(currproc.Proc)
			if harderr != nil {
				return harderr
			}
		}

		for i := range libs {
			loc := re.FindStringIndex(libs[i])
			if loc == nil {
				continue
			}

			var res ProcResult
			if _, ok := currproc.Results[label]; !ok {
				res.Name, _, _ = currproc.Proc.Name()
				res.Pid = float64(currproc.Proc.Pid())
				res.Libs = append(res.Libs, libs[i])
				currproc.Results[label] = res
			} else {
				res = currproc.Results[label]
				res.Libs = append(res.Libs, libs[i])
				currproc.Results[label] = res
			}
		}
	}

	return
}

func (r Runner) Run(Args []byte) (resStr string) {
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			err, _ := json.Marshal(r.Results)
			resStr = string(err[:])
			return
		}
	}()

	err := json.Unmarshal(Args, &r.Parameters)
	if err != nil {
		panic(err)
	}
	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	proclist := make(ProcList)

	// Aggregate all the process.
	for label, currSearch := range r.Parameters.Searches {

		// Concatenate all search strings under a given search into a single regexp.
		searchstr := currSearch.Processes[0]
		for i := 1; i < len(currSearch.Processes); i += 1 {
			searchstr = searchstr + "|" + currSearch.Processes[i]
		}

		plist, hard, _ := Pgrep(searchstr)
		if hard != nil {
			panic(hard)
		}

		for i := range plist {
			pid := plist[i].Pid()
			_, ok := proclist[pid]
			// Add search to an existing process in list or create a new one.
			if !ok {
				procsearch := ProcSearch{Searches: make(map[string]Search), Results: make(map[string]ProcResult)}
				procsearch.Proc = plist[i]
				procsearch.Searches[label] = currSearch
				proclist[pid] = procsearch
			} else {
				proclist[pid].Searches[label] = currSearch
			}
		}
	}

	// Search Libraries
	for _, currproc := range proclist {
		err := SearchLoadedLibs(&currproc)
		if err != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", err))
		}
	}

	// Memory scan.
	for pid, currproc := range proclist {
		scancount := ActivateAllScans(&currproc)

		buffer_size := uint(4096)
		foundAddress := uintptr(0)

		harderror, _ := memaccess.SlidingWalkMemory(currproc.Proc, 0, buffer_size,
			func(address uintptr, buf []byte) (keepSearching bool) {
				// Iterate through the searches to perform the scan
				for label, currsearch := range currproc.Searches {
					for i := range currsearch.Scans {
						// Check if the scan is still active.
						if !currsearch.Scans[i].Active {
							continue
						}
						needle := currsearch.Scans[i].Expression
						// Check if the given search is regexp or a slice of hex bytes.
						if b, err := hex.DecodeString(needle); err == nil {
							index := bytes.Index(buf, b)
							if index == -1 {
								continue
							}
							foundAddress = address + uintptr(index)
						} else {
							regex, _ := regexp.Compile(needle)
							loc := regex.FindIndex(buf)
							if loc == nil {
								continue
							}
							foundAddress = address + uintptr(loc[0])
						}
						// If we are here, it means that foundAddress has the memory location of the required search string.
						found := make([]byte, len(needle)+int(currsearch.Scans[i].Length))
						harderr, _ := memaccess.CopyMemory(currproc.Proc, foundAddress, found)
						if harderr != nil {
							r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", harderr))
							return false
						}
						foundstr := hex.EncodeToString(found)
						var res ProcResult
						if _, ok := currproc.Results[label]; !ok {
							res.Name, _, _ = currproc.Proc.Name()
							res.Pid = float64(pid)
							res.BytesFound = append(res.BytesFound, foundstr)
							currproc.Results[label] = res
						} else {
							res = currproc.Results[label]
							res.BytesFound = append(res.BytesFound, foundstr)
							currproc.Results[label] = res
						}
						// Check if 'n' matches have been found. Deactivate the scan if we've found 'n' matches.
						if len(currproc.Results[label].BytesFound) == int(currsearch.Scans[i].MatchCount) {
							currsearch.Scans[i].Active = false
							scancount -= 1
						}
						// If we have no more scans to perform on this process. Stop.
						if scancount == 0 {
							return false
						}
					}
				}
				return true
			})

		if harderror != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", harderror))
		}
	}

	// Iterate through all the processes. Grab all the results. Add them to results.
	fres := make(element)
	for _, currproc := range proclist {
		res := currproc.Results
		for label, cur := range res {
			fres[label] = append(fres[label], cur)
		}
		currproc.Proc.Close()
	}

	r.Results.Success = true
	r.Results.Elements = fres
	return r.buildResults()
}

func (r Runner) buildResults() string {

	if len(r.Results.Elements.(element)) > 0 {
		r.Results.FoundAnything = true
	}

	jsonOutput, err := json.Marshal(r.Results)
	if err != nil {
		panic(err)
	}
	return string(jsonOutput[:])
}
