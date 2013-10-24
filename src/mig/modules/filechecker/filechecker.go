/* Inspect files on the local system

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
Portions created by the Initial Developer are Copyright (C) 2013
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
package filechecker

import (
	"bufio"
	"code.google.com/p/go.crypto/sha3"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"encoding/json"
	"os"
	"regexp"
)

var DEBUG bool = false
var VERBOSE bool = false

/* BitMask for the type of check to apply to a given file
   see documentation about iota for more info
*/
const (
	CheckContains = 1 << iota
	CheckNamed
	CheckMD5
	CheckSHA1
	CheckSHA256
	CheckSHA384
	CheckSHA512
	CheckSHA3_224
	CheckSHA3_256
	CheckSHA3_384
	CheckSHA3_512
)

/* Representation of a File Check.
- Path is the file system path to inspect
- Type is the name of the type of check
- Value is the value of the check, such as a md5 hash
- CodeType is the type of check in integer form
- FilesCount is the total number of files inspected for each Check
- MatchCount is a counter of positive results for this Check
- Result is a boolean set to True when the Check has matched once or more
- Files is an slice of string that contains paths of matching files
*/
type FileCheck struct {
	ID, Path, Type, Value			string
	CodeType, FilesCount, MatchCount	int
	Result					bool
	Files					map[string]int
	Re					*regexp.Regexp
}

type CheckResult struct {
	TestedFiles, MatchCount int
	Files			 []string
}

/* Statistic counters:
- CheckCount is the total numbers of Checks tested
- FilesCount is the total number of files inspected
- ChecksMatch is the number of Checks that matched at least once
- UniqueFiles is the number of files that matches at least one Check once
- TotalHits is the total number of Checks hits
*/
type Stats struct {
	CheckCount    int
	FilesCount  int
	ChecksMatch   int
	UniqueFiles int
	TotalHits   int
}

/* ParseCheck verifies and populate checks passed as arguments
*/
func ParseCheck(check FileCheck, id string) (FileCheck) {
	check.ID = id
	switch check.Type {
	case "contains":
		check.CodeType = CheckContains
		// compile the value into a regex
		check.Re = regexp.MustCompile(check.Value)
	case "named":
		check.CodeType = CheckNamed
		// compile the value into a regex
		check.Re = regexp.MustCompile(check.Value)
	case "md5":
		check.CodeType = CheckMD5
	case "sha1":
		check.CodeType = CheckSHA1
	case "sha256":
		check.CodeType = CheckSHA256
	case "sha384":
		check.CodeType = CheckSHA384
	case "sha512":
		check.CodeType = CheckSHA512
	case "sha3_224":
		check.CodeType = CheckSHA3_224
	case "sha3_256":
		check.CodeType = CheckSHA3_256
	case "sha3_384":
		check.CodeType = CheckSHA3_384
	case "sha3_512":
		check.CodeType = CheckSHA3_512
	default:
		err := fmt.Sprintf("ParseCheck: Invalid check '%s'", check.Type)
		panic(err)
	}
	// allocate the map
	check.Files = make(map[string]int)
	return check
}

/* GetHash calculates the hash of a file.
   It opens a file, reads it block by block, and updates a
   sum with each block. This method plays nice with big files
   parameters:
	- fd is an open file descriptor that points to the file to inspect
	- HashType is an integer that define the type of hash
   return:
	- hexhash, the hex encoded hash of the file found at fp
*/
func GetHash(fd *os.File, HashType int) (hexhash string) {
	if DEBUG {
		fmt.Printf("GetHash: computing hash for '%s'\n", fd.Name())
	}
	var h hash.Hash
	switch HashType {
	case CheckMD5:
		h = md5.New()
	case CheckSHA1:
		h = sha1.New()
	case CheckSHA256:
		h = sha256.New()
	case CheckSHA384:
		h = sha512.New384()
	case CheckSHA512:
		h = sha512.New()
	case CheckSHA3_224:
		h = sha3.NewKeccak224()
	case CheckSHA3_256:
		h = sha3.NewKeccak256()
	case CheckSHA3_384:
		h = sha3.NewKeccak384()
	case CheckSHA3_512:
		h = sha3.NewKeccak512()
	default:
		err := fmt.Sprintf("GetHash: Unkown hash type %d", HashType)
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
	hexhash = fmt.Sprintf("%x", h.Sum(nil))
	return
}

/* VerifyHash compares a file hash with the Checks that apply to the file
   parameters:
	- file is the absolute filename of the file to check
	- hash is the value of the hash being checked
	- check is the type of check
	- ActiveCheckIDs is a slice of int with IDs of active Checks
	- Checks is a map of Check
   returns:
	- IsVerified: true if a match is found, false otherwise
*/
func VerifyHash(file string, hash string, check int, ActiveCheckIDs []string,
		Checks map[string]FileCheck) (IsVerified bool) {
	IsVerified = false
	for _, id := range ActiveCheckIDs {
		tmpcheck := Checks[id]
		if Checks[id].Value == hash {
			IsVerified = true
			tmpcheck.Result = true
			tmpcheck.MatchCount += 1
			tmpcheck.Files[file] = 1
		}
		// update Checks tested files count
		tmpcheck.FilesCount++
		Checks[id] = tmpcheck
	}
	return
}

/* MatchRegexpsOnFile read a file line by line and apply regexp search to each
   line. If a regexp matches, the corresponding Check is updated with the result.
   All regexp are compiled during argument parsing and not here.
   parameters:
	- fd is a file descriptor on the open file
	- ReList is a list of Check IDs to apply to this file
	- Checks is a map of Check
   return:
	- MatchesRegexp is a boolean set to true if at least one regexp matches
*/
func MatchRegexpsOnFile(fd *os.File, ReList []string,
			Checks map[string]FileCheck) (MatchesRegexp bool) {
	MatchesRegexp = false
	Results := make(map[string]int)
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		for _, id := range ReList {
			if Checks[id].Re.MatchString(scanner.Text()) {
				MatchesRegexp = true
				Results[id]++
				break
			}
		}
	}
	if MatchesRegexp {
		for id, count := range Results {
			tmpcheck := Checks[id]
			tmpcheck.Result = true
			tmpcheck.MatchCount += count
			tmpcheck.Files[fd.Name()] = count
			Checks[id] = tmpcheck
		}
	}
	// update Checks tested files count
	for _, id := range ReList {
		tmpcheck := Checks[id]
		tmpcheck.FilesCount++
		Checks[id] = tmpcheck
	}
	return
}

/* MatchRegexpsOnName applies regexp search to a given filename
   parameters:
	- filename is a string that contains a filename
	- ReList is a list of Check IDs to apply to this file
	- Checks is a map of Check
   return:
	- MatchesRegexp is a boolean set to true if at least one regexp matches
*/
func MatchRegexpsOnName(filename string, ReList []string,
			Checks map[string]FileCheck) (MatchesRegexp bool) {
	MatchesRegexp = false
	for _, id := range ReList {
		tmpcheck := Checks[id]
		if Checks[id].Re.MatchString(filename) {
			MatchesRegexp = true
			tmpcheck.Result = true
			tmpcheck.MatchCount++
			tmpcheck.Files[filename] = 1
		}
		// update Checks tested files count
		tmpcheck.FilesCount++
		Checks[id] = tmpcheck
	}
	return
}

/* InspectFile is an orchestration function that runs the individual checks
   against a particular file. It uses the CheckBitMask to select which checks
   to run, and runs the checks in a smart way to minimize effort.
   parameters:
	- fd is an open file descriptor that points to the file to inspect
	- ActiveCheckIDs is a slice that contains the IDs of the Checks
	that all files in that path and below must be checked against
	- CheckBitMask is a bitmask of the checks types currently active
	- Checks is the global list of Checks
   returns:
	- nil on success, error on failure
*/
func InspectFile(fd *os.File, ActiveCheckIDs []string, CheckBitMask int,
		 Checks map[string]FileCheck) error {
	/* Iterate through the entire checklist, and process the checks of
	   each file
	*/
	if DEBUG {
		fmt.Printf("InspectFile: file '%s' CheckMask '%d'\n",
			fd.Name(), CheckBitMask)
	}
	if (CheckBitMask & CheckContains) != 0 {
		// build a list of Checks of check type 'contains'
		var ReList []string
		for _, id := range ActiveCheckIDs {
			if (Checks[id].CodeType & CheckContains) != 0 {
				ReList = append(ReList, id)
			}
		}
		if MatchRegexpsOnFile(fd, ReList, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckNamed) != 0 {
		// build a list of Checks of check type 'contains'
		var ReList []string
		for _, id := range ActiveCheckIDs {
			if (Checks[id].CodeType & CheckNamed) != 0 {
				ReList = append(ReList, id)
			}
		}
		if MatchRegexpsOnName(fd.Name(), ReList, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckMD5) != 0 {
		hash := GetHash(fd, CheckMD5)
		if VerifyHash(fd.Name(), hash, CheckMD5, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA1) != 0 {
		hash := GetHash(fd, CheckSHA1)
		if VerifyHash(fd.Name(), hash, CheckSHA1, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA256) != 0 {
		hash := GetHash(fd, CheckSHA256)
		if VerifyHash(fd.Name(), hash, CheckSHA256, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA384) != 0 {
		hash := GetHash(fd, CheckSHA384)
		if VerifyHash(fd.Name(), hash, CheckSHA384, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA512) != 0 {
		hash := GetHash(fd, CheckSHA512)
		if VerifyHash(fd.Name(), hash, CheckSHA512, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_224) != 0 {
		hash := GetHash(fd, CheckSHA3_224)
		if VerifyHash(fd.Name(), hash, CheckSHA3_224, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_256) != 0 {
		hash := GetHash(fd, CheckSHA3_256)
		if VerifyHash(fd.Name(), hash, CheckSHA3_256, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_384) != 0 {
		hash := GetHash(fd, CheckSHA3_384)
		if VerifyHash(fd.Name(), hash, CheckSHA3_384, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_512) != 0 {
		hash := GetHash(fd, CheckSHA3_512)
		if VerifyHash(fd.Name(), hash, CheckSHA3_512, ActiveCheckIDs, Checks) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	return nil
}

/* GetDownThatPath goes down a directory and build a list of Active Checks that
   apply to the current path. For a given directory, it calls itself for all
   subdirectories fund, recursively walking down the pass. When it find a file,
   it calls the inspection function, and give it the list of Checks to inspect
   the file with.
   parameters:
	- path is the file system path to inspect
	- ActiveCheckIDs is a slice that contains the IDs of the Checks
	that all files in that path and below must be checked against
	- CheckBitMask is a bitmask of the checks types currently active
	- Checks is the global list of Checks
	- ToDoChecks is a map that contains the Checks that are not yet active
	- Statistics is a set of counters
   return:
	- nil on success, error on error
*/
func GetDownThatPath(path string, ActiveCheckIDs []string, CheckBitMask int,
		     Checks map[string]FileCheck, ToDoChecks map[string]FileCheck,
		     Statistics *Stats) error {
	for id, check := range ToDoChecks {
		if check.Path == path {
			/* Found a new Check to apply to the current path, add
			   it to the active list, and delete it from the todo
			*/
			ActiveCheckIDs = append(ActiveCheckIDs, id)
			CheckBitMask |= check.CodeType
			delete(ToDoChecks, id)
			if DEBUG {
				fmt.Printf("GetDownThatPath: Activating Check "+
					"id '%d' for path '%s'\n", id, path)
			}
		}
	}
	var SubDirs []string
	/* Read the content of dir stored in 'path',
	   put all sub-directories in the SubDirs slice, and call
	   the inspection function for all files
	*/
	target, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	targetMode, _ := target.Stat()
	if targetMode.Mode().IsDir() {
		// target is a directory, process its content
		DirContent, err := target.Readdir(-1)
		if err != nil {
			panic(err)
		}
		// loop over the content of the directory
		for _, DirEntry := range DirContent {
			EntryFullPath := path + "/" + DirEntry.Name()
			// this entry is a subdirectory, keep it for later
			if DirEntry.IsDir() {
				SubDirs = append(SubDirs, EntryFullPath)
				// that one is a file, open it and inspect it
			} else if DirEntry.Mode().IsRegular() {
				Entryfd, err := os.Open(EntryFullPath)
				if err != nil {
					panic(err)
				}
				InspectFile(Entryfd, ActiveCheckIDs,
					CheckBitMask, Checks)
				Statistics.FilesCount++
				if err := Entryfd.Close(); err != nil {
					panic(err)
				}
			}
		}
	} else if targetMode.Mode().IsRegular() {
		InspectFile(target, ActiveCheckIDs, CheckBitMask, Checks)
		Statistics.FilesCount++
	}
	// close the current target, we are done with it
	if err := target.Close(); err != nil {
		panic(err)
	}
	// if we found any sub directories, go down the rabbit hole
	for _, dir := range SubDirs {
		GetDownThatPath(dir, ActiveCheckIDs, CheckBitMask, Checks,
			ToDoChecks, Statistics)
	}
	return nil
}

/* BuildResults iterates on the map of Checks and print the results to stdout (if
   VERBOSE is set) and into JSON format
   parameters:
	- Checks is a map of FileCheck
	- Statistics is a set of counters
   returns:
	- nil on success, error on failure
*/
func BuildResults(Checks map[string]FileCheck, Statistics *Stats) (string) {
	Results := make(map[string]CheckResult)
	FileHistory := make(map[string]int)
	for _, check := range Checks {
		if VERBOSE {
			fmt.Printf("Main: Check '%d' returned %d positive match\n",
				check.ID, check.MatchCount)
			if check.Result {
				for file, hits := range check.Files {
					if VERBOSE {
						fmt.Printf("\t- %d hits on %s\n",
							hits, file)
					}
					Statistics.TotalHits += hits
					if _, ok := FileHistory[file]; !ok {
						Statistics.UniqueFiles++
					}
				}
				Statistics.ChecksMatch++
			}
		}
		var listPosFiles []string
		for f, _ := range check.Files {
			listPosFiles = append(listPosFiles, f)
		}
		Results[check.ID] = CheckResult{
			TestedFiles: check.FilesCount,
			MatchCount: check.MatchCount,
			Files: listPosFiles,
		}
	}
	if VERBOSE {
		fmt.Printf("Tested Checks:\t%d\n"+
			"Tested files:\t%d\n"+
			"Checks Match:\t%d\n"+
			"Unique Files:\t%d\n"+
			"Total hits:\t%d\n",
			Statistics.CheckCount, Statistics.FilesCount,
			Statistics.ChecksMatch, Statistics.UniqueFiles,
			Statistics.TotalHits)
	}
	JsonResults, err := json.Marshal(Results)
	if err != nil { panic(err) }
	return string(JsonResults[:])
}

/* The Main logic of filechecker parses command line arguments into a list of
   individual FileChecks, stored in a map.
   Each Check contains a path, which is inspected in the GetDownThatPath function.
   The results are stored in the Checks map and built and display at the end.
*/
func Run(Args []byte) (string) {
	if DEBUG {
		VERBOSE = true
	}
	/* Checks is a map of individual Checks and associated results
	Checks = {
		<id> = { <struct FileCheck> },
		<id> = { <struct FileCheck> },
		...
	}
	*/
	Checks := make(map[string]FileCheck)
	err := json.Unmarshal(Args, &Checks)
	if err != nil { panic(err) }
	// ToDoChecks is a list of Checks to process, dequeued when done
	ToDoChecks := make(map[string]FileCheck)
	var Statistics Stats
	// parse the arguments, split on the space
	for id, check := range Checks {
		if DEBUG {
			fmt.Printf("Main: Parsing check id: '%d'\n", id)
		}
		Checks[id] = ParseCheck(check, id)
		ToDoChecks[id] = Checks[id]
		Statistics.CheckCount++
	}
	for id, check := range Checks {
		if DEBUG {
			fmt.Printf("Main: Inspecting path '%s' from Check id "+
				"'%d'\n", check.Path, id)
		}
		// loop through the list of Check, and only process the Checks that
		// are still in the todo list
		if _, ok := ToDoChecks[id]; !ok {
			continue
		}
		var EmptyActiveChecks []string
		GetDownThatPath(check.Path, EmptyActiveChecks, 0, Checks,
			ToDoChecks, &Statistics)
	}
	return BuildResults(Checks, &Statistics)
}
