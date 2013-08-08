/* Look for file IOCs on the local system

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
Portions created by the Initial Developer are Copyright (C) 2012
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
package main

import (
	"bufio"
	"code.google.com/p/go.crypto/sha3"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"regexp"
	"strings"
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

/* Representation of a File IOC.
- Raw is the raw IOC string received from the program arguments
- Path is the file system path to inspect
- Value is the value of the check, such as a md5 hash
- Check is the type of check in integer form
- FilesCount is the total number of files inspected for each IOC
- ResultCount is a counter of positive results for this IOC
- Result is a boolean set to True when the IOC has matched once or more
- Files is an slice of string that contains paths of matching files
*/
type FileIOC struct {
	Raw, Path, Value			string
	ID, Check, FilesCount, ResultCount	int
	Result					bool
	Files					map[string]int
	Re					*regexp.Regexp
}

type IOCResult struct {
	TestedFiles, ResultCount int
	Files			 []string
}

/* Statistic counters:
- IOCCount is the total numbers of IOCs tested
- FilesCount is the total number of files inspected
- IOCsMatch is the number of IOCs that matched at least once
- UniqueFiles is the number of files that matches at least one IOC once
- TotalHits is the total number of IOCs hits
*/
type Stats struct {
	IOCCount    int
	FilesCount  int
	IOCsMatch   int
	UniqueFiles int
	TotalHits   int
}

/* ParseIOC parses an IOC from the command line into a FileIOC struct
   parameters:
	- raw_ioc is a string that contains the IOC from the command line in
	the format <path>:<check>=<value>
	eg. /usr/bin/vim:md5=8680f252cabb7f4752f8927ce0c6f9bd
	- id is an integer used as a ID reference
   return:
	- a FileIOC structure
*/
func ParseIOC(raw_ioc string, id int) (ioc FileIOC) {
	ioc.Raw = raw_ioc
	ioc.ID = id
	// split on the first ':' and use the left part as the Path
	tmp := strings.Split(raw_ioc, ":")
	ioc.Path = tmp[0]
	// split the right part on '=', left is the check, right is the value
	tmp = strings.Split(tmp[1], "=")
	ioc.Value = tmp[1]
	// the check string is transformed into a bitmask and stored
	checkstring := tmp[0]
	switch checkstring {
	case "contains":
		ioc.Check = CheckContains
		// compile the value into a regex
		ioc.Re = regexp.MustCompile(ioc.Value)
	case "named":
		ioc.Check = CheckNamed
		// compile the value into a regex
		ioc.Re = regexp.MustCompile(ioc.Value)
	case "md5":
		ioc.Check = CheckMD5
	case "sha1":
		ioc.Check = CheckSHA1
	case "sha256":
		ioc.Check = CheckSHA256
	case "sha384":
		ioc.Check = CheckSHA384
	case "sha512":
		ioc.Check = CheckSHA512
	case "sha3_224":
		ioc.Check = CheckSHA3_224
	case "sha3_256":
		ioc.Check = CheckSHA3_256
	case "sha3_384":
		ioc.Check = CheckSHA3_384
	case "sha3_512":
		ioc.Check = CheckSHA3_512
	default:
		err := fmt.Sprintf("ParseIOC: Invalid check '%s'", checkstring)
		panic(err)
	}
	// allocate the map
	ioc.Files = make(map[string]int)
	return
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

/* VerifyHash compares a file hash with the IOCs that apply to the file
   parameters:
	- file is the absolute filename of the file to check
	- hash is the value of the hash being checked
	- check is the type of check
	- ActiveIOCIDs is a slice of int with IDs of active IOCs
	- IOCs is a map of IOC
   returns:
	- IsVerified: true if a match is found, false otherwise
*/
func VerifyHash(file string, hash string, check int, ActiveIOCIDs []int,
	IOCs map[int]FileIOC) (IsVerified bool) {
	IsVerified = false
	for _, id := range ActiveIOCIDs {
		tmpioc := IOCs[id]
		if IOCs[id].Value == hash {
			IsVerified = true
			tmpioc.Result = true
			tmpioc.ResultCount += 1
			tmpioc.Files[file] = 1
		}
		// update IOCs tested files count
		tmpioc.FilesCount++
		IOCs[id] = tmpioc
	}
	return
}

/* MatchRegexpsOnFile read a file line by line and apply regexp search to each
   line. If a regexp matches, the corresponding IOC is updated with the result.
   All regexp are compiled during argument parsing and not here.
   parameters:
	- fd is a file descriptor on the open file
	- ReList is a integer list of IOC IDs to apply to this file
	- IOCs is a map of IOC
   return:
	- MatchesRegexp is a boolean set to true if at least one regexp matches
*/
func MatchRegexpsOnFile(fd *os.File, ReList []int,
	IOCs map[int]FileIOC) (MatchesRegexp bool) {
	MatchesRegexp = false
	Results := make(map[int]int)
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		for _, id := range ReList {
			if IOCs[id].Re.MatchString(scanner.Text()) {
				MatchesRegexp = true
				Results[id]++
				break
			}
		}
	}
	if MatchesRegexp {
		for id, count := range Results {
			tmpioc := IOCs[id]
			tmpioc.Result = true
			tmpioc.ResultCount += count
			tmpioc.Files[fd.Name()] = count
			IOCs[id] = tmpioc
		}
	}
	// update IOCs tested files count
	for _, id := range ReList {
		tmpioc := IOCs[id]
		tmpioc.FilesCount++
		IOCs[id] = tmpioc
	}
	return
}

/* MatchRegexpsOnName applies regexp search to a given filename
   parameters:
	- filename is a string that contains a filename
	- ReList is a integer list of IOC IDs to apply to this file
	- IOCs is a map of IOC
   return:
	- MatchesRegexp is a boolean set to true if at least one regexp matches
*/
func MatchRegexpsOnName(filename string, ReList []int,
	IOCs map[int]FileIOC) (MatchesRegexp bool) {
	MatchesRegexp = false
	for _, id := range ReList {
		tmpioc := IOCs[id]
		if IOCs[id].Re.MatchString(filename) {
			MatchesRegexp = true
			tmpioc.Result = true
			tmpioc.ResultCount++
			tmpioc.Files[filename] = 1
		}
		// update IOCs tested files count
		tmpioc.FilesCount++
		IOCs[id] = tmpioc
	}
	return
}

/* InspectFile is an orchestration function that runs the individual checks
   against a particular file. It uses the CheckBitMask to select which checks
   to run, and runs the checks in a smart way to minimize effort.
   parameters:
	- fd is an open file descriptor that points to the file to inspect
	- ActiveIOCIDs is a slice of integer that contains the IDs of the IOCs
	that all files in that path and below must be checked against
	- CheckBitMask is a bitmask of the checks types currently active
	- IOCs is the global list of IOCs
   returns:
	- nil on success, error on failure
*/
func InspectFile(fd *os.File, ActiveIOCIDs []int, CheckBitMask int,
	         IOCs map[int]FileIOC) error {
	/* Iterate through the entire checklist, and process the checks of
	   each file
	*/
	if DEBUG {
		fmt.Printf("InspectFile: file '%s' CheckMask '%d'\n",
			fd.Name(), CheckBitMask)
	}
	if (CheckBitMask & CheckContains) != 0 {
		// build a list of IOCs of check type 'contains'
		var ReList []int
		for _, id := range ActiveIOCIDs {
			if (IOCs[id].Check & CheckContains) != 0 {
				ReList = append(ReList, id)
			}
		}
		if MatchRegexpsOnFile(fd, ReList, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckNamed) != 0 {
		// build a list of IOCs of check type 'contains'
		var ReList []int
		for _, id := range ActiveIOCIDs {
			if (IOCs[id].Check & CheckNamed) != 0 {
				ReList = append(ReList, id)
			}
		}
		if MatchRegexpsOnName(fd.Name(), ReList, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckMD5) != 0 {
		hash := GetHash(fd, CheckMD5)
		if VerifyHash(fd.Name(), hash, CheckMD5, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA1) != 0 {
		hash := GetHash(fd, CheckSHA1)
		if VerifyHash(fd.Name(), hash, CheckSHA1, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA256) != 0 {
		hash := GetHash(fd, CheckSHA256)
		if VerifyHash(fd.Name(), hash, CheckSHA256, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA384) != 0 {
		hash := GetHash(fd, CheckSHA384)
		if VerifyHash(fd.Name(), hash, CheckSHA384, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA512) != 0 {
		hash := GetHash(fd, CheckSHA512)
		if VerifyHash(fd.Name(), hash, CheckSHA512, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_224) != 0 {
		hash := GetHash(fd, CheckSHA3_224)
		if VerifyHash(fd.Name(), hash, CheckSHA3_224, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_256) != 0 {
		hash := GetHash(fd, CheckSHA3_256)
		if VerifyHash(fd.Name(), hash, CheckSHA3_256, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_384) != 0 {
		hash := GetHash(fd, CheckSHA3_384)
		if VerifyHash(fd.Name(), hash, CheckSHA3_384, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	if (CheckBitMask & CheckSHA3_512) != 0 {
		hash := GetHash(fd, CheckSHA3_512)
		if VerifyHash(fd.Name(), hash, CheckSHA3_512, ActiveIOCIDs, IOCs) {
			if DEBUG{
				fmt.Printf("InspectFile: Positive result " +
					"found for '%s'\n", fd.Name())
			}
		}
	}
	return nil
}

/* GetDownThatPath goes down a directory and build a list of Active IOCs that
   apply to the current path. For a given directory, it calls itself for all
   subdirectories fund, recursively walking down the pass. When it find a file,
   it calls the inspection function, and give it the list of IOCs to inspect
   the file with.
   parameters:
	- path is the file system path to inspect
	- ActiveIOCIDs is a slice of integer that contains the IDs of the IOCs
	that all files in that path and below must be checked against
	- CheckBitMask is a bitmask of the checks types currently active
	- IOCs is the global list of IOCs
	- ToDoIOCs is a map that contains the IOCs that are not yet active
	- Statistics is a set of counters
   return:
	- nil on success, error on error
*/
func GetDownThatPath(path string, ActiveIOCIDs []int, CheckBitMask int,
	IOCs map[int]FileIOC, ToDoIOCs map[int]FileIOC,
	Statistics *Stats) error {
	for id, ioc := range ToDoIOCs {
		if ioc.Path == path {
			/* Found a new IOC to apply to the current path, add
			   it to the active list, and delete it from the todo
			*/
			ActiveIOCIDs = append(ActiveIOCIDs, id)
			CheckBitMask |= ioc.Check
			delete(ToDoIOCs, id)
			if DEBUG {
				fmt.Printf("GetDownThatPath: Activating IOC "+
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
				InspectFile(Entryfd, ActiveIOCIDs,
					CheckBitMask, IOCs)
				Statistics.FilesCount++
				if err := Entryfd.Close(); err != nil {
					panic(err)
				}
			}
		}
	} else if targetMode.Mode().IsRegular() {
		InspectFile(target, ActiveIOCIDs, CheckBitMask, IOCs)
		Statistics.FilesCount++
	}
	// close the current target, we are done with it
	if err := target.Close(); err != nil {
		panic(err)
	}
	// if we found any sub directories, go down the rabbit hole
	for _, dir := range SubDirs {
		GetDownThatPath(dir, ActiveIOCIDs, CheckBitMask, IOCs,
			ToDoIOCs, Statistics)
	}
	return nil
}

/* BuildResults iterates on the map of IOCs and print the results to stdout (if
   VERBOSE is set) and into JSON format
   parameters:
	- IOCs is a map of FileIOC
	- Statistics is a set of counters
   returns:
	- nil on success, error on failure
*/
func BuildResults(IOCs map[int]FileIOC, Statistics *Stats) error {
	Results := make(map[string]IOCResult)
	FileHistory := make(map[string]int)
	for _, ioc := range IOCs {
		if VERBOSE {
			fmt.Printf("Main: IOC '%s' returned %d positive match\n",
				ioc.Raw, ioc.ResultCount)
			if ioc.Result {
				for file, hits := range ioc.Files {
					if VERBOSE {
						fmt.Printf("\t- %d hits on %s\n",
							hits, file)
					}
					Statistics.TotalHits += hits
					if _, ok := FileHistory[file]; !ok {
						Statistics.UniqueFiles++
					}
				}
				Statistics.IOCsMatch++
			}
		}
		var listPosFiles []string
		for f, _ := range ioc.Files {
			listPosFiles = append(listPosFiles, f)
		}
		Results[ioc.Raw] = IOCResult{
			TestedFiles: ioc.FilesCount,
			ResultCount: ioc.ResultCount,
			Files: listPosFiles,
		}
	}
	if VERBOSE {
		fmt.Printf("Tested IOCs:\t%d\n"+
			"Tested files:\t%d\n"+
			"IOCs Match:\t%d\n"+
			"Unique Files:\t%d\n"+
			"Total hits:\t%d\n",
			Statistics.IOCCount, Statistics.FilesCount,
			Statistics.IOCsMatch, Statistics.UniqueFiles,
			Statistics.TotalHits)
	}
	JsonResults, err := json.Marshal(Results)
	if err != nil { panic(err) }
	os.Stdout.Write(JsonResults)
	return nil
}

/* The Main logic of filechecker parses command line arguments into a list of
   individual FileIOCs, stored in a map.
   Each IOC contains a path, which is inspected in the GetDownThatPath function.
   The results are stored in the IOCs map and built and display at the end.
*/
func main() {
	if DEBUG {
		VERBOSE = true
	}
	/* IOCs is a map of individual IOCs and associated results
	IOCs = {
		<id> = { <struct FileIOC> },
		<id> = { <struct FileIOC> },
		...
	}
	*/
	IOCs := make(map[int]FileIOC)

	// list of IOCs to process, remove from list when processed
	ToDoIOCs := make(map[int]FileIOC)

	var Statistics Stats
	flag.Parse()
	for i := 0; flag.Arg(i) != ""; i++ {
		if DEBUG {
			fmt.Printf("Main: Parsing IOC id '%d': '%s'\n",
				i, flag.Arg(i))
		}
		raw_ioc := flag.Arg(i)
		IOCs[i] = ParseIOC(raw_ioc, i)
		ToDoIOCs[i] = IOCs[i]
		Statistics.IOCCount++
	}
	for id, ioc := range IOCs {
		if DEBUG {
			fmt.Printf("Main: Inspecting path '%s' from IOC id "+
				"'%d'\n", ioc.Path, id)
		}
		// loop through the list of IOC, and only process the IOCs that
		// are still in the todo list
		if _, ok := ToDoIOCs[id]; !ok {
			continue
		}
		var EmptyActiveIOCs []int
		GetDownThatPath(ioc.Path, EmptyActiveIOCs, 0, IOCs,
			ToDoIOCs, &Statistics)
	}
	BuildResults(IOCs, &Statistics)
}
