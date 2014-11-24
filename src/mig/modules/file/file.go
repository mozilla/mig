// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

// file provides functions to scan a file system. It can look into files
// using regexes. It can search files by name. It can match hashes in md5, sha1,
// sha256, sha384, sha512, sha3_224, sha3_256, sha3_384 and sha3_512.
// The filesystem can be searches using pattern, as described in the Parameters
// documentation.
package file

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
	"mig"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"code.google.com/p/go.crypto/sha3"
)

var debug bool = false

func init() {
	mig.RegisterModule("file", func() interface{} {
		return new(Runner)
	})
}

type Runner struct {
	Parameters Parameters
	Results    Results
}

type Parameters struct {
	Searches  map[string]search `json:"searches,omitempty"`
	Condition string            `json:"condition,omitempty"`
}

type search struct {
	Description string   `json:"description,omitempty"`
	Paths       []string `json:"paths"`
	Contents    []string `json:"contents,omitempty"`
	Names       []string `json:"names,omitempty"`
	MD5         []string `json:"md5,omitempty"`
	SHA1        []string `json:"sha1,omitempty"`
	SHA256      []string `json:"sha256,omitempty"`
	SHA384      []string `json:"sha384,omitempty"`
	SHA512      []string `json:"sha512,omitempty"`
	SHA3_224    []string `json:"sha3_224,omitempty"`
	SHA3_256    []string `json:"sha3_256,omitempty"`
	SHA3_384    []string `json:"sha3_384,omitempty"`
	SHA3_512    []string `json:"sha3_512,omitempty"`
	//Options     options  `json:"options,omitempty"`
}

//type options struct {
//	MaxDepth float64 `json:"maxdepth,omitempty"`
//	CrossFS  bool    `json:"crossfs,omitempty"`
//}

// Create a new Parameters
func newParameters() *Parameters {
	var p Parameters
	p.Searches = make(map[string]search)
	return &p
}

// validate a Parameters
func (r Runner) ValidateParameters() (err error) {
	var labels []string
	for label, s := range r.Parameters.Searches {
		labels = append(labels, label)
		err = validateLabel(label)
		if err != nil {
			return
		}
		for _, r := range s.Contents {
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, r := range s.Names {
			err = validateRegex(r)
			if err != nil {
				return
			}
		}
		for _, hash := range s.MD5 {
			err = validateHash(hash, checkMD5)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA1 {
			err = validateHash(hash, checkSHA1)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA256 {
			err = validateHash(hash, checkSHA256)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA384 {
			err = validateHash(hash, checkSHA384)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA512 {
			err = validateHash(hash, checkSHA512)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA3_224 {
			err = validateHash(hash, checkSHA3_224)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA3_256 {
			err = validateHash(hash, checkSHA3_256)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA3_384 {
			err = validateHash(hash, checkSHA3_384)
			if err != nil {
				return
			}
		}
		for _, hash := range s.SHA3_512 {
			err = validateHash(hash, checkSHA3_512)
			if err != nil {
				return
			}
		}
	}
	if r.Parameters.Condition != "" {
		// evaluate the condition
		condcomp := strings.Split(r.Parameters.Condition, " ")
		lencond := len(condcomp)
		opre := regexp.MustCompile("(?i)^(and|or)$")
		hasLabel := false
		for pos, comp := range condcomp {
			// is the current component a label?
			for _, label := range labels {
				if comp == label || comp == "!"+label {
					if pos > 0 && hasLabel {
						return fmt.Errorf("Invalid condition. Labels must be separated by operators at pos %d", pos)
					}
					hasLabel = true
					goto next
				}
			}
			// is the current component an operator?
			if opre.MatchString(comp) {
				// check that operator is not first or last
				if pos == 0 || pos == lencond-1 {
					return fmt.Errorf("Invalid condition. A condition cannot start or stop with an operator")
				}
				if !hasLabel {
					return fmt.Errorf("Invalid condition. Operator '%s' must be preceded by a label at pos. %d", comp, pos)
				}
				hasLabel = false
				goto next
			}
			// if we are here, the component is invalid
			return fmt.Errorf("Invalid component '%s' in condition. Must be a valid label or an operator (and|or).", comp)
		next:
		}
	}
	return
}

func validateLabel(label string) (err error) {
	labelre := regexp.MustCompile("^[a-zA-Z0-9_-]{1,64}$")
	if !labelre.MatchString(label) {
		return fmt.Errorf("The syntax of label '%s' is invalid. Must match regex ^[a-zA-Z0-9_-]{1,64}$", label)
	}
	return
}

func validateRegex(regex string) (err error) {
	_, err = regexp.Compile(regex)
	if err != nil {
		return fmt.Errorf("Invalid regexp '%s'. Must be a regexp. Compilation failed with '%v'", regex, err)
	}
	return
}

func validateHash(hash string, hashType uint64) (err error) {
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
	return
}

/* Statistic counters:
- CheckCount is the total numbers of checklist tested
- FilesCount is the total number of files inspected
- Checksmatch is the number of checks that matched at least once
- YniqueFiles is the number of files that matches at least one Check once
- Totalhits is the total number of checklist hits
*/
type statistics struct {
	Checkcount  float64 `json:"checkcount"`
	Filescount  float64 `json:"filescount"`
	Openfailed  float64 `json:"openfailed"`
	Checksmatch float64 `json:"checksmatch"`
	Uniquefiles float64 `json:"uniquefiles"`
	Totalhits   float64 `json:"totalhits"`
	Exectime    string  `json:"exectime"`
}

// stats is a global variable
var stats statistics

// Representation of a filecheck.
// label is a string that identifies the search
// path is the file system path to inspect
// method is the name of the type of check
// test is the value of the check, such as a md5 hash
// code is the type of test in integer form
// filecount is the total number of files inspected for each Check
// matchcount is a counter of positive results for this Check
// hasmatched is a boolean set to True when the Check has matched once or more
// files is an slice of string that contains paths of matching files
// regex is a regular expression
type filecheck struct {
	label, path, method, test string
	code                      uint64
	filecount, matchcount     float64
	hasmatched                bool
	files                     map[string]float64
	regex                     *regexp.Regexp
}

func newFileCheck(label, path, method, test string, code uint64) *filecheck {
	var fc filecheck
	fc.files = make(map[string]float64)
	fc.hasmatched = false
	fc.filecount = 0
	fc.matchcount = 0
	fc.label = label
	fc.path = path
	fc.method = method
	fc.test = test
	fc.code = code
	if code == checkRegex || code == checkFilename {
		fc.regex = regexp.MustCompile(test)
	}
	return &fc
}

// Results contains the details of what was inspected on the file system.
// The `Elements` parameter contains 5 level-deep structure that represents
// the original search parameters, plus the detailled result of each test.
// To help with results parsing, if any of the check matches at least once,
// the flag `FoundAnything` will be set to true.
//
// JSON sample:
//	{
//		"elements": {
//			"/usr/*bin/*": {
//			    "filename": {
//			        "module names": {
//			            "atddd": {
//			                "filecount": 1992,
//			                "files": {},
//			                "matchcount": 0
//			            },
//			            "cupsdd": {
//			                "filecount": 1992,
//			                "files": {},
//			                "matchcount": 0
//			            }
//			        }
//			    },
//			    "md5": {
//			        "atddd": {
//			            "fade6e3ab4b396553b191f23d8c04cf1": {
//			                "filecount": 996,
//			                "files": {},
//			                "matchcount": 0
//			            }
//			        },
//			        "cupsdd": {
//			            "ce607e782faa5ace379c13a5de8052a3": {
//			                "filecount": 996,
//			                "files": {},
//			                "matchcount": 0
//			            }
//			        }
//			    }
//			}
//		},
//		"error": [
//			"ERROR: followSymLink() -\u003e lstat /usr/lib/vmware-tools/bin64/vmware-user-wrapper: no such file or directory"
//		],
//		"foundanything": false,
//		"statistics": {
//			"checkcount": 52,
//			"checksmatch": 0,
//			"exectime": "4.67603983s",
//			"filescount": 6574,
//			"openfailed": 1,
//			"totalhits": 0,
//			"uniquefiles": 0
//		}
//	}
type Results struct {
	FoundAnything bool                                                     `json:"foundanything"`
	Success       bool                                                     `json:"success"`
	Elements      map[string]map[string]map[string]map[string]singleresult `json:"elements"`
	Statistics    statistics                                               `json:"statistics"`
	Errors        []string                                                 `json:"error"`
}

// singleresult contains information on the result of a single test
type singleresult struct {
	Filecount  float64            `json:"filecount"`
	Matchcount float64            `json:"matchcount"`
	Files      map[string]float64 `json:"files"`
}

// newResults allocates a Results structure
func newResults() *Results {
	return &Results{Elements: make(map[string]map[string]map[string]map[string]singleresult), FoundAnything: false}
}

var walkingErrors []string

// Run() is file's entry point. It parses command line arguments into a list of
// individual checks, stored in a map.
// Each Check contains a path, which is inspected in the pathWalk function.
// The results are stored in the checklist map and sent to stdout at the end.
func (r Runner) Run(Args []byte) (resStr string) {
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
	//r.Parameters = newParameters()
	err := json.Unmarshal(Args, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	// walk through the parameters and generate a checklist of filechecks
	checklist := make(map[float64]filecheck)
	todolist := make(map[float64]filecheck)
	var i float64 = 0
	for label, search := range r.Parameters.Searches {
		checks, err := createChecks(label, search)
		if err != nil {
			panic(err)
		}
		for _, check := range checks {
			if debug {
				fmt.Printf("check %.0f: label='%s'; path='%s'; method='%s'; test='%s'\n",
					i, check.label, check.path, check.method, check.test)
			}
			checklist[i] = check
			todolist[i] = check
			i++
			stats.Checkcount++
		}
	}

	// From all the checks, grab a list of root path sorted small sortest
	// to longest, and then enter each path iteratively
	var roots []string
	for id, check := range checklist {
		root := findRootPath(check.path)
		if debug {
			fmt.Printf("Main: Found root path at '%s' in check '%d':'%s'\n", root, id, check.test)
		}
		exist := false
		for _, p := range roots {
			if root == p {
				exist = true
			}
		}
		if !exist {
			roots = append(roots, root)
		}
		// sorting the array is useful in case the same command contains "/some/thing"
		// and then "/some". By starting with the smallest root, we ensure that all the
		// checks for both "/some" and "/some/thing" will be processed.
		sort.Strings(roots)
	}
	// enter each root one by one
	for _, root := range roots {
		interestedlist := make(map[float64]filecheck)
		err = pathWalk(root, checklist, todolist, interestedlist)
		if err != nil {
			panic(err)
			if debug {
				fmt.Printf("pathWalk failed with error '%v'\n", err)
			}
		}
	}

	resStr, err = buildResults(checklist, t0)
	if err != nil {
		panic(err)
	}

	if debug {
		// pretty printing
		printedResults, err := r.PrintResults([]byte(resStr), false)
		if err != nil {
			panic(err)
		}
		for _, res := range printedResults {
			fmt.Println(res)
		}
	}
	return
}

// BitMask for the type of check to apply to a given file
// see documentation about iota for more info
const (
	checkRegex = 1 << iota
	checkFilename
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

// createCheck creates a new filecheck
func createChecks(label string, s search) (checks []filecheck, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createChecks() -> %v", e)
		}
	}()
	for _, path := range s.Paths {
		for _, re := range s.Contents {
			check := newFileCheck(label, path, "regex", re, checkRegex)
			checks = append(checks, *check)
		}
		for _, re := range s.Names {
			check := newFileCheck(label, path, "filename", re, checkFilename)
			checks = append(checks, *check)
		}
		for _, hash := range s.MD5 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "md5", hash, checkMD5)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA1 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha1", hash, checkSHA1)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA256 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha256", hash, checkSHA256)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA384 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha384", hash, checkSHA384)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA512 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha512", hash, checkSHA512)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA3_224 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha3_224", hash, checkSHA3_224)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA3_256 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha3_256", hash, checkSHA3_256)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA3_384 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha3_384", hash, checkSHA3_384)
			checks = append(checks, *check)
		}
		for _, hash := range s.SHA3_512 {
			hash = strings.ToUpper(hash)
			check := newFileCheck(label, path, "sha3_512", hash, checkSHA3_512)
			checks = append(checks, *check)
		}
	}
	return
}

// findRootPath takes a path pattern and extracts the root, that is the
// directory we can start our pattern search from.
// example: pattern='/etc/cron.*/*' => root='/etc/'
func findRootPath(pattern string) string {
	// if pattern has no metacharacter, use as-is
	if strings.IndexAny(pattern, "*?[") < 0 {
		return pattern
	}
	// find the root path before the first pattern character.
	// seppos records the position of the latest path separator
	// before the first pattern.
	seppos := 0
	for cursor := 0; cursor < len(pattern); cursor++ {
		char := pattern[cursor]
		switch char {
		case '*', '?', '[':
			// found pattern character. but ignore it if preceded by backslash
			if cursor > 0 {
				if pattern[cursor-1] == '\\' {
					break
				}
			}
			goto exit
		case os.PathSeparator:
			if cursor > 0 {
				seppos = cursor
			}
		}
	}
exit:
	if seppos == 0 {
		return string(pattern[0])
	} else {
		return pattern[0 : seppos+1]
	}
}

// pathWalk goes down a directory and build a list of Active checklist that
// apply to the current path. For a given directory, it calls itself for all
// subdirectories fund, recursively walking down the pass. When it find a file,
// it calls the inspection function, and give it the list of checklist to inspect
// the file with.
// parameters:
//      - path is the file system path to inspect
//      - checklist is the global list of checklist
//      - todolist is a map that contains the checklist that are not yet active
//      - interestedlist is a map that contains checks that are interested in the
//	  current path but not yet active
// return:
//      - nil on success, error on error
func pathWalk(path string, checklist, todolist, interestedlist map[float64]filecheck) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("pathWalk() -> %v", e)
		}
	}()
	if debug {
		fmt.Printf("pathWalk: walking into '%s'\n", path)
	}
	for id, check := range todolist {
		if pathIncludes(path, check.path) {
			/* Found a new Check to apply to the current path, add
			   it to the interested list, and delete it from the todo
			*/
			interestedlist[id] = todolist[id]
			if debug {
				fmt.Printf("pathWalk: adding check '%d':'%s':'%s':'%s' to interestedlist, removing from todolist\n",
					id, check.path, check.method, check.test)
			}
		}
	}
	var subdirs []string
	// Read the content of dir stored in 'path',
	// put all sub-directories in the SubDirs slice, and call
	// the inspection function for all files
	target, err := os.Open(path)
	if err != nil {
		// do not panic when open fails, just increase a counter
		stats.Openfailed++
		walkingErrors = append(walkingErrors, fmt.Sprintf("ERROR: %v", err))
		return nil
	}
	targetMode, _ := os.Lstat(path)
	if targetMode.Mode().IsDir() {
		// target is a directory, process its content
		dirContent, err := target.Readdir(-1)
		if err != nil {
			panic(err)
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
				if debug {
					fmt.Printf("'%s' links to '%s'\n", entryAbsPath, linkpath)
				}
				if linkmode.IsRegular() {
					isLinkedFile = true
				}
			}
			if dirEntry.Mode().IsRegular() || isLinkedFile {
				err = evaluateFile(entryAbsPath, interestedlist, checklist)
				if err != nil {
					panic(err)
				}
			}
		}
	}

	// target is a symlink, expand it
	isLinkedFile := false
	if targetMode.Mode()&os.ModeSymlink == os.ModeSymlink {
		linkmode, linkpath, err := followSymLink(path)
		if err != nil {
			// reading the link failed, count and continue
			stats.Openfailed++
			walkingErrors = append(walkingErrors, fmt.Sprintf("ERROR: %v", err))
			return nil
		}
		if debug {
			fmt.Printf("'%s' links to '%s'\n", path, linkpath)
		}
		if linkmode.IsRegular() {
			isLinkedFile = true
		}
	}

	// target is a file or a symlink to a file, evaluate it
	if targetMode.Mode().IsRegular() || isLinkedFile {
		err = evaluateFile(path, interestedlist, checklist)
		if err != nil {
			panic(err)
		}
	}

	// close the current target, we are done with it
	if err := target.Close(); err != nil {
		panic(err)
	}
	// if we found any sub directories, go down the rabbit hole recursively,
	// but only if one of the check is interested in going
	for _, dir := range subdirs {
		interested := false
		for _, check := range interestedlist {
			if pathIncludes(dir, check.path) {
				interested = true
				break
			}
		}
		if interested {
			err = pathWalk(dir, checklist, todolist, interestedlist)
			if err != nil {
				panic(err)
			}
		}
	}
	return nil
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

// pathIncludes verifies that a given path matches a given pattern
func pathIncludes(path, pattern string) bool {
	// if pattern has no metacharacter, use as-is
	if strings.IndexAny(pattern, "*?[") < 0 {
		if path == pattern {
			return true
		}
		return false
	}
	// decompose the path into a slice of strings using the PathSeparator to split
	// and compare each component of the pattern with the correspond component of the path
	pathItems := strings.Split(path, string(os.PathSeparator))
	patternItems := strings.Split(pattern, string(os.PathSeparator))
	matchLen := len(patternItems)
	if matchLen > len(pathItems) {
		matchLen = len(pathItems)
	}
	if debug {
		fmt.Printf("Path comparison: ")
	}
	for i := 0; i < matchLen; i++ {
		if i > 0 && pathItems[i] == "" {
			// skip comparison of the last item of the path because it's empty
			break
		}
		match, _ := filepath.Match(patternItems[i], pathItems[i])
		if !match {
			if debug {
				fmt.Printf("'%s'!='%s'\n", pathItems[i], patternItems[i])
			}
			return false
		}
		if debug {
			fmt.Printf("'%s'=~'%s'; ", pathItems[i], patternItems[i])
		}
	}
	if debug {
		fmt.Printf("=> match\n")
	}
	return true
}

// evaluateFile looks for patterns that match a file and build a list of checks
// passed to inspectFile
// '/etc/' will grep into /etc/ without going further down. '/etc/*' will go further down.
// '/etc/*sswd' or '/etc/*yum*/*.repo' work as expected.
func evaluateFile(file string, interestedlist, checklist map[float64]filecheck) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("evaluateFile() -> %v", e)
		}
	}()
	if debug {
		fmt.Printf("evaluateFile: evaluating '%s' against %d checks\n", file, len(interestedlist))
	}
	if len(interestedlist) < 1 {
		if debug {
			fmt.Printf("evaluateFile: interestedlist is empty\n")
		}
		return nil
	}
	// that one is a file, see if it matches one of the pattern
	inspect := false
	var checkBitmask uint64 = 0
	var activechecks []float64
	for id, check := range interestedlist {
		match := false
		subfile := file
		if strings.IndexAny(check.path, "*?[") < 0 {
			// check.path doesn't contain metacharacters,
			// do a direct comparison
			if check.path[len(check.path)-1] == os.PathSeparator {
				if len(file) >= len(check.path) {
					if check.path[0:len(check.path)-1] == file[0:len(check.path)-1] {
						match = true
					}
				}
			} else if file == check.path {
				match = true
			}
		} else {
			match, err = filepath.Match(check.path, file)
			if err != nil {
				return err
			}
			if !match && (len(check.path) < len(file)) && (check.path[len(check.path)-1] == '*') {
				// 2nd chance to match if check.path is shorter than file and ends
				// with a wildcard.
				// filepath.Match isn't very tolerant: a pattern such as '/etc*'
				// will not match the file '/etc/passwd'.
				// We work around that by attempting to match on equal length.
				subfile = file[0 : len(check.path)-1]
				match, err = filepath.Match(check.path, subfile)
				if err != nil {
					return err
				}
			}
		}
		if match {
			if debug {
				fmt.Printf("evaluateFile: activated check id '%d' '%s' on '%s'\n", id, check.path, file)
			}
			activechecks = append(activechecks, id)
			checkBitmask |= check.code
			inspect = true
		} else {
			if debug {
				fmt.Printf("evaluateFile: '%s' is NOT interested in '%s'\n", check.path, file)
			}
		}
	}
	if inspect {
		// it matches, open the file and inspect it
		entryfd, err := os.Open(file)
		if err != nil {
			// woops, open failed. update counters and move on
			stats.Openfailed++
			return nil
		}
		inspectFile(entryfd, activechecks, checkBitmask, checklist)
		stats.Filescount++
		if err := entryfd.Close(); err != nil {
			panic(err)
		}
		stats.Filescount++
	}
	return
}

// inspectFile is an orchestration function that runs the individual checks
// against a selected file. It uses checkBitmask to find the checks it needs
// to run. The file is opened once, and all the checks are ran against it,
// minimizing disk IOs.
// parameters:
//      - fd is an open file descriptor that points to the file to inspect
//      - activechecks is a slice that contains the IDs of the checklist
//      that all files in that path and below must be checked against
//      - checkBitmask is a bitmask of the checks types currently active
//      - checklist is the global list of checklist
// returns:
//      - nil on success, error on failure
func inspectFile(fd *os.File, activechecks []float64, checkBitmask uint64, checklist map[float64]filecheck) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("inspectFile() -> %v", e)
		}
	}()
	// Iterate through the entire checklist, and process the checks of each file
	if debug {
		fmt.Printf("InspectFile: file '%s' CheckMask '%d'\n",
			fd.Name(), checkBitmask)
	}
	if (checkBitmask & checkRegex) != 0 {
		// build a list of checklist of check type 'contains'
		var ReList []float64
		for _, id := range activechecks {
			if (checklist[id].code & checkRegex) != 0 {
				ReList = append(ReList, id)
			}
		}
		match, err := matchRegexOnFile(fd, ReList, checklist)
		if err != nil {
			panic(err)
		}
		if match {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkFilename) != 0 {
		// build a list of checklist of check type 'contains'
		var ReList []float64
		for _, id := range activechecks {
			if (checklist[id].code & checkFilename) != 0 {
				ReList = append(ReList, id)
			}
		}
		if matchRegexOnName(fd.Name(), ReList, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkMD5) != 0 {
		hash, err := getHash(fd, checkMD5)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkMD5, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA1) != 0 {
		hash, err := getHash(fd, checkSHA1)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA1, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA256) != 0 {
		hash, err := getHash(fd, checkSHA256)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA256, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA384) != 0 {
		hash, err := getHash(fd, checkSHA384)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA384, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA512) != 0 {
		hash, err := getHash(fd, checkSHA512)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA512, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA3_224) != 0 {
		hash, err := getHash(fd, checkSHA3_224)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA3_224, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA3_256) != 0 {
		hash, err := getHash(fd, checkSHA3_256)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA3_256, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA3_384) != 0 {
		hash, err := getHash(fd, checkSHA3_384)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA3_384, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	if (checkBitmask & checkSHA3_512) != 0 {
		hash, err := getHash(fd, checkSHA3_512)
		if err != nil {
			panic(err)
		}
		if verifyHash(fd.Name(), hash, checkSHA3_512, activechecks, checklist) {
			if debug {
				fmt.Printf("InspectFile: Positive result found for '%s'\n", fd.Name())
			}
		}
	}
	return
}

// getHash calculates the hash of a file.
// It reads a file block by block, and updates a hashsum with each block.
// Reading by blocks consume very little memory, which is needed for large files.
// parameters:
//      - fd is an open file descriptor that points to the file to inspect
//      - hashType is an integer that define the type of hash
// return:
//      - hexhash, the hex encoded hash of the file found at fp
func getHash(fd *os.File, hashType float64) (hexhash string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getHash() -> %v", e)
		}
	}()
	if debug {
		fmt.Printf("getHash: computing hash for '%s'\n", fd.Name())
	}
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

// verifyHash compares a file hash with the checklist that apply to the file
// parameters:
//      - file is the absolute filename of the file to check
//      - hash is the value of the hash being checked
//      - check is the type of check
//      - activechecks is a slice of int with IDs of active checklist
//      - checklist is a map of Check
// returns:
//      - IsVerified: true if a match is found, false otherwise
func verifyHash(file string, hash string, check float64, activechecks []float64, checklist map[float64]filecheck) (IsVerified bool) {
	IsVerified = false
	for _, id := range activechecks {
		tmpcheck := checklist[id]
		if checklist[id].test == hash {
			IsVerified = true
			tmpcheck.hasmatched = true
			tmpcheck.matchcount++
			tmpcheck.files[file] = 1
		}
		// update checklist tested files count
		tmpcheck.filecount++
		checklist[id] = tmpcheck
	}
	return
}

// matchRegexOnFile read a file line by line and apply regexp search to each
// line. If a regexp matches, the corresponding Check is updated with the result.
// All regexp are compiled during argument parsing and not here.
// parameters:
//      - fd is a file descriptor on the open file
//      - ReList is a list of Check IDs to apply to this file
//      - checklist is a map of Check
// return:
//      - hasmatched is a boolean set to true if at least one regexp matches
func matchRegexOnFile(fd *os.File, ReList []float64, checklist map[float64]filecheck) (hasmatched bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("matchRegexOnFile() -> %v", e)
		}
	}()
	hasmatched = false
	// temp map to store the results
	results := make(map[float64]float64)
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		for _, id := range ReList {
			if checklist[id].regex.MatchString(scanner.Text()) {
				if debug {
					fmt.Printf("matchRegexOnFile: regex '%s' match on line '%s'\n",
						checklist[id].test, scanner.Text())
				}
				hasmatched = true
				results[id]++
			}
		}
	}
	if hasmatched {
		for id, count := range results {
			tmpcheck := checklist[id]
			tmpcheck.hasmatched = true
			tmpcheck.matchcount += count
			tmpcheck.files[fd.Name()] = count
			checklist[id] = tmpcheck
		}
	}
	// update checklist tested files count
	for _, id := range ReList {
		tmpcheck := checklist[id]
		tmpcheck.filecount++
		checklist[id] = tmpcheck
	}
	return
}

// matchRegexOnName applies regexp search to a given filename
// parameters:
//      - filename is a string that contains a filename
//      - ReList is a list of Check IDs to apply to this file
//      - checklist is a map of Check
// return:
//      - hasmatched is a boolean set to true if at least one regexp matches
func matchRegexOnName(filename string, ReList []float64, checklist map[float64]filecheck) (hasmatched bool) {
	hasmatched = false
	for _, id := range ReList {
		tmpcheck := checklist[id]
		if checklist[id].regex.MatchString(path.Base(filename)) {
			hasmatched = true
			tmpcheck.hasmatched = true
			tmpcheck.matchcount++
			tmpcheck.files[filename] = tmpcheck.matchcount
		}
		// update checklist tested files count
		tmpcheck.filecount++
		checklist[id] = tmpcheck
	}
	return
}

// buildResults iterates on the map of checklist and print the results to stdout (if
// debug is set) and into JSON format
func buildResults(checklist map[float64]filecheck, t0 time.Time) (resStr string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("buildResults() -> %v", e)
		}
	}()
	res := newResults()
	history := make(map[string]float64)

	// iterate through the checklist and parse the results
	// into a Response object
	for id, check := range checklist {
		if debug {
			fmt.Printf("Main: Check %s:%d returned %d positive match\n", check.label, id, check.matchcount)
		}
		if check.hasmatched {
			for file, hits := range check.files {
				if debug {
					fmt.Printf("\t- %d hits on %s\n", hits, file)
				}
				stats.Totalhits += float64(hits)
				if _, ok := history[file]; !ok {
					stats.Uniquefiles++
				}
			}
			stats.Checksmatch++
		}

		// build a single results and insert it into the result structure
		r := singleresult{
			Filecount:  check.filecount,
			Matchcount: check.matchcount,
			Files:      check.files,
		}
		// to avoid overwriting existing elements, we test each level before inserting the result
		if _, ok := res.Elements[check.path]; !ok {
			res.Elements[check.path] = map[string]map[string]map[string]singleresult{
				check.method: map[string]map[string]singleresult{
					check.label: map[string]singleresult{
						check.test: r,
					},
				},
			}
		} else if _, ok := res.Elements[check.path][check.method]; !ok {
			res.Elements[check.path][check.method] = map[string]map[string]singleresult{
				check.label: map[string]singleresult{
					check.test: r,
				},
			}
		} else if _, ok := res.Elements[check.path][check.method][check.label]; !ok {
			res.Elements[check.path][check.method][check.label] = map[string]singleresult{
				check.test: r,
			}
		} else if _, ok := res.Elements[check.path][check.method][check.label][check.test]; !ok {
			res.Elements[check.path][check.method][check.label][check.test] = r
		}
	}

	// if something matched anywhere, set the global boolean to true
	if stats.Checksmatch > 0 {
		res.FoundAnything = true
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
	if debug {
		fmt.Printf("Tested checklist: %d\n"+
			"Tested files:     %d\n"+
			"checklist Match:  %d\n"+
			"Unique Files:     %d\n"+
			"Total hits:       %d\n"+
			"Execution time:   %s\n",
			stats.Checkcount, stats.Filescount,
			stats.Checksmatch, stats.Uniquefiles,
			stats.Totalhits, stats.Exectime)
	}
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
func (r Runner) PrintResults(rawResults []byte, foundOnly bool) (prints []string, err error) {
	var results Results
	err = json.Unmarshal(rawResults, &results)
	if err != nil {
		panic(err)
	}
	for path, _ := range results.Elements {
		for method, _ := range results.Elements[path] {
			for label, _ := range results.Elements[path][method] {
				for value, _ := range results.Elements[path][method][label] {
					if foundOnly {
						if results.Elements[path][method][label][value].Matchcount < 1 {
							// go to next value
							continue
						}
					}
					if len(results.Elements[path][method][label][value].Files) == 0 {
						res := fmt.Sprintf("0 found on '%s' in check '%s':'%s':'%s'",
							value, path, method, label)
						prints = append(prints, res)
						continue
					}
					for file, cnt := range results.Elements[path][method][label][value].Files {
						res := fmt.Sprintf("%.0f found in '%s' on '%s' for file '%s':'%s':'%s'",
							cnt, file, value, path, method, label)
						prints = append(prints, res)
					}
				}
			}
		}
	}
	if !foundOnly {
		for _, we := range results.Errors {
			prints = append(prints, we)
		}
		stat := fmt.Sprintf("Statistics: %.0f checks tested on %.0f files. %.0f failed to open. %.0f checks matched on %.0f files. %.0f total hits. ran in %s.",
			results.Statistics.Checkcount, results.Statistics.Filescount, results.Statistics.Openfailed, results.Statistics.Checksmatch, results.Statistics.Uniquefiles,
			results.Statistics.Totalhits, results.Statistics.Exectime)
		prints = append(prints, stat)
	}
	return
}
