// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent <ulfr@mozilla.com>

package file /* import "mig.ninja/mig/modules/file" */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"mig.ninja/mig/modules"
	"mig.ninja/mig/testutil"
)

// basedir is the base directory for the tests, initialized in createFiles
var basedir string

// subdirs contains the subdirectory structure that will be created to place files in,
// initialized in createFiles
var subdirs string

// subdirEntries is joined in createFiles using os specific path separators to populate subdirs
var subdirEntries = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"}

func TestMain(m *testing.M) {
	createFiles()

	// For all of our test files, dynamically append checks to the test table that tests checksums and
	// sizes
	for _, x := range testFiles {
		if x.link { // Skip symlinks
			continue
		}
		params := []string{"size", "md5", "sha1", "sha2", "sha3"}
		for _, y := range params {
			np := testParams{}
			switch y {
			case "size":
				np.size = append(np.size, strconv.Itoa(x.size))
			case "md5":
				np.md5 = append(np.md5, x.md5)
			case "sha1":
				np.sha1 = append(np.sha1, x.sha1)
			case "sha2":
				np.sha2 = append(np.sha2, x.sha2)
			case "sha3":
				np.sha3 = append(np.sha3, x.sha3)
			}
			np.description = fmt.Sprintf("generated %v test for %v", y, x.filename)
			np.expectedfilesroot = append(np.expectedfilesroot, x.filename)
			np.expectedfilessub = append(np.expectedfilessub, x.filename)
			// See if this file is a destination for a symlink, if so we will also add the link
			for _, z := range testFiles {
				if z.linkdest == x.filename {
					np.expectedfilesroot = append(np.expectedfilesroot, z.filename)
					np.expectedfilessub = append(np.expectedfilessub, z.filename)
				}
			}
			testData = append(testData, np)
		}
	}

	r := m.Run()
	err := os.RemoveAll(basedir)
	if err != nil {
		log.Fatalf("failed to remove %s: %v\n", basedir, err)
	}
	os.Exit(r)
}

func TestRegistration(t *testing.T) {
	testutil.CheckModuleRegistration(t, "file")
}

func TestValidateParameters(t *testing.T) {
	var (
		r run
		s Search
	)
	r.Parameters = *newParameters()
	err := r.ValidateParameters()
	if err != nil {
		t.Fatal("ValidateParameters: %v", err)
	}

	r.Parameters.Searches["s1"] = s
	err = r.ValidateParameters()
	if err == nil {
		t.Fatalf("parameters with empty search path should not validate", err)
	}

	s.Paths = append(s.Paths, "/testing")
	r.Parameters.Searches["s1"] = s
	err = r.ValidateParameters()
	if err != nil {
		t.Fatalf("ValidateParameters: %v", err)
	}
}

func TestParamsParser(t *testing.T) {
	var (
		r    run
		args []string
		err  error
	)
	args = append(args, "-path", "/a/path")
	args = append(args, "-name", "^.*testfile.*$")
	args = append(args, "-content", "content match")
	args = append(args, "-size", "<10k")
	args = append(args, "-size", ">1")
	args = append(args, "-size", "<100000k")
	args = append(args, "-mode", "-rw-r--r--")
	args = append(args, "-mtime", "<1h")
	args = append(args, "-md5", "e499c1912bd9af4f7e8ccaf27f7b04d2")
	args = append(args, "-sha1", "d7bbc3dd7adf6e347c93a4c8b9bfb8ef4748c0fb")
	args = append(args, "-sha2", "4d8ef27c4415d71cbbfad1eaa97d6f2a3ddacc9708b66efbb726133b9fd3d79a")
	args = append(args, "-sha3", "a7ba1e66174848ecea143b612f22168b006979e3827e09f0ae6395e8")
	args = append(args, "-matchany")
	args = append(args, "-matchall")
	args = append(args, "-macroal")
	args = append(args, "-mismatch", "content")
	args = append(args, "-matchlimit", "10")
	args = append(args, "-maxdepth", "2")
	args = append(args, "-maxerrors", "31")
	args = append(args, "-verbose")
	args = append(args, "-decompress")
	_, err = r.ParamsParser(args)
	if err != nil {
		t.Fatalf("ParamsParser: %v", err)
	}
}

func TestMakeChecksBadSize(t *testing.T) {
	var s Search
	s.Paths = append(s.Paths, "/test")
	s.Sizes = []string{"BADSIZE"}
	err := s.makeChecks()
	if err == nil {
		t.Fatalf("makeChecks should have failed with invalid size value")
	}
	s.Sizes = []string{"<BADSIZEg"}
	err = s.makeChecks()
	if err == nil {
		t.Fatalf("makeChecks should have failed with invalid size value")
	}
	s.Sizes = []string{">>>>BADSIZEm"}
	err = s.makeChecks()
	if err == nil {
		t.Fatalf("makeChecks should have failed with invalid size value")
	}
	s.Sizes = []string{"m"}
	err = s.makeChecks()
	if err == nil {
		t.Fatalf("makeChecks should have failed with invalid size value")
	}
	s.Sizes = []string{"<"}
	err = s.makeChecks()
	if err == nil {
		t.Fatalf("makeChecks should have failed with invalid size value")
	}
}

func TestMakeChecksBadMTime(t *testing.T) {
	var s Search
	s.Paths = append(s.Paths, "/test")
	s.Mtimes = append(s.Mtimes, "BADMTIME")
	err := s.makeChecks()
	if err == nil {
		t.Fatalf("makeChecks should have failed with invalid mtime value")
	}
}

func TestBadRunParameters(t *testing.T) {
	var (
		r  run
		s  Search
		mr modules.Result
	)

	// Reinitialize various globals in the file module upon each test run
	debug = false
	walkingErrors = make([]string, 0)
	tryDecompress = false
	stats.Filescount = 0
	stats.Openfailed = 0
	stats.Totalhits = 0
	stats.Exectime = ""

	r.Parameters = *newParameters()
	s.Paths = append(s.Paths, basedir)
	s.SHA1 = append(s.SHA1, "NOTASHA1")
	r.Parameters.Searches["s1"] = s
	msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters, false)
	if err != nil {
		t.Fatalf("modules.MakeMessage: %v", err)
	}
	out := r.Run(modules.NewModuleReader(bytes.NewBuffer(msg)))
	if len(out) == 0 {
		t.Fatal("module returned no output")
	}
	err = json.Unmarshal([]byte(out), &mr)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if mr.Success {
		t.Fatal("module should have indicated it failed")
	}
	if mr.FoundAnything {
		t.Fatal("module should have indicated it found nothing")
	}
}

// createFiles creates the file structure the tests will be executed against
func createFiles() {
	bdname := fmt.Sprintf("migfiletest%v", time.Now().Format("15-04-05.99999999"))
	basedir = path.Join(os.TempDir(), bdname)

	subdirEntries = append([]string{basedir}, subdirEntries...)
	subdirs = path.Join(subdirEntries...)

	err := os.MkdirAll(subdirs, 0700)
	if err != nil {
		log.Fatalf("MkDirAll: %v", err)
	}
	for _, dir := range []string{basedir, subdirs} {
		for _, tp := range testFiles {
			tfpath := path.Join(dir, tp.filename)
			if tp.link {
				linkdestpath := path.Join(dir, tp.linkdest)
				err = os.Symlink(linkdestpath, tfpath)
				if err != nil {
					log.Fatalf("Symlink: %v", err)
				}
				continue
			}
			fd, err := os.Create(tfpath)
			if err != nil {
				log.Fatalf("Create: %v", err)
			}
			os.Chmod(tfpath, 0644)
			n, err := fd.Write(tp.data)
			if err != nil {
				log.Fatalf("Write: %v: %v", fd.Name(), err)
			}
			if n != len(tp.data) {
				log.Fatalf("Write: short write, wanted %v wrote %v", len(tp.data), n)
			}
			fd.Close()
		}
	}
	return
}

func TestRunTestParams(t *testing.T) {
	for _, x := range testData {
		x.runTest(t)
	}
}

// testParams is used to define standard test cases for the file module
type testParams struct {
	description string

	// Search parameters, if set these values will be applied to a given search
	name                []string
	size                []string
	mode                []string
	mtime               []string
	content             []string
	md5                 []string
	sha1                []string
	sha2                []string
	sha3                []string
	decompressedcontent []string
	decompressedmd5     []string
	macroal             bool
	mismatch            []string
	maxdepth            float64

	searchpath        []string // Override search in normal test path
	expectedfilesroot []string // The files we expect to find for this search in the root path
	expectedfilessub  []string // The files we expect to find in the subdirectory path
}

func (tp *testParams) getExpectedFiles() (ret []string) {
	for _, x := range tp.expectedfilesroot {
		rp := path.Join(basedir, x)
		ret = append(ret, rp)
	}
	for _, x := range tp.expectedfilessub {
		rp := path.Join(subdirs, x)
		ret = append(ret, rp)
	}
	return
}

var testData = []testParams{
	// Simple content tests
	testParams{
		description:       "find testfile0 by content, should see testfile9 which is a link too",
		content:           []string{"^--- header for first file ---$"},
		expectedfilesroot: []string{"testfile0", "testfile9"},
		expectedfilessub:  []string{"testfile0", "testfile9"},
	},
	testParams{
		description:       "find testfile0 by content using file itself as a path",
		content:           []string{"^--- header for first file ---$"},
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{},
		searchpath:        []string{"SEARCHBASE+testfile0"},
	},
	testParams{
		description:       "find testfile0 by content using file itself as a path via symlink",
		content:           []string{"^--- header for first file ---$"},
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{},
		searchpath:        []string{"SEARCHBASE+testfile9"},
	},
	testParams{
		description:       "find testfile1 by content",
		content:           []string{"^--- header for second file ---$"},
		expectedfilesroot: []string{"testfile1"},
		expectedfilessub:  []string{"testfile1"},
	},
	testParams{
		description:       "find testfile2 by content",
		content:           []string{"skZ0"},
		expectedfilesroot: []string{"testfile2"},
		expectedfilessub:  []string{"testfile2"},
	},
	testParams{
		description:       "find testfile3 by content",
		content:           []string{"^--- header for fourth file ---$"},
		expectedfilesroot: []string{"testfile3"},
		expectedfilessub:  []string{"testfile3"},
	},
	testParams{
		description:       "find testfile4 by content",
		content:           []string{"^--- header for fifth file ---$"},
		expectedfilesroot: []string{"testfile4"},
		expectedfilessub:  []string{"testfile4"},
	},
	testParams{
		description:       "find testfile5 by content",
		content:           []string{"^--- header for sixth file ---$"},
		expectedfilesroot: []string{"testfile5"},
		expectedfilessub:  []string{"testfile5"},
	},
	testParams{
		description:       "find testfile6 by content",
		content:           []string{"KO3B"},
		expectedfilesroot: []string{"testfile6"},
		expectedfilessub:  []string{"testfile6"},
	},
	testParams{
		description:       "find testfile7 by content",
		content:           []string{"t6Pl"},
		expectedfilesroot: []string{"testfile7"},
		expectedfilessub:  []string{"testfile7"},
	},
	testParams{
		description:       "find testfile8 by content",
		content:           []string{",'XL"},
		expectedfilesroot: []string{"testfile8"},
		expectedfilessub:  []string{"testfile8"},
	},
	testParams{
		description:       "find testfile2 by size",
		size:              []string{"1024"},
		expectedfilesroot: []string{"testfile2"},
		expectedfilessub:  []string{"testfile2"},
	},
	testParams{
		description:       "find testfile4 by md5",
		md5:               []string{"5d5a4fdeafc1677dca8255ef9624d522"},
		expectedfilesroot: []string{"testfile4"},
		expectedfilessub:  []string{"testfile4"},
	},
	testParams{
		description:       "find testfile4 by sha2",
		sha2:              []string{"a4001843158a7a374e5ddcc22644c0e37738bc64ffd50179fc18fb443e0a62393b43384d9ac734e7a64c204e862ae3424094381afb33dfc639c52517afad1f32"},
		expectedfilesroot: []string{"testfile4"},
		expectedfilessub:  []string{"testfile4"},
	},
	testParams{
		description:       "find testfile4 by sha3",
		sha3:              []string{"2028feaccf974066aa7c47070f24c72d349ed6a6575cb801cc606c4a2b59020af4339b60dbedd0049a7341edde14133ee6f8b199f1a7c6ef36493fd217501607"},
		expectedfilesroot: []string{"testfile4"},
		expectedfilessub:  []string{"testfile4"},
	},
	testParams{
		description:       "find testfile6 by sha2",
		sha2:              []string{"bb4e449df74edae0292d60d2733a3b1801d90ae23560484b1e04fb52f111a14f"},
		expectedfilesroot: []string{"testfile6"},
		expectedfilessub:  []string{"testfile6"},
	},
	testParams{
		description:       "find testfile7 by all hashes",
		md5:               []string{"52fa96013b5c6aa9302d39ee7fe2f6a5"},
		sha1:              []string{"31952c0d2772c302ec94b303c2b80b67cf830060"},
		sha2:              []string{"f6032dc9b4ba112397a6f8bcb778ab10708c1acd38e48f637ace15c3ae417ded"},
		sha3:              []string{"3f4dacf0b2347d0a0ab6f09b7d7c98fd12cb2030d4af8baeacaf55a9"},
		expectedfilesroot: []string{"testfile7"},
		expectedfilessub:  []string{"testfile7"},
	},
	testParams{
		description:       "find testfile0 and testfile6",
		name:              []string{".*testfile(0|6)$"},
		expectedfilesroot: []string{"testfile0", "testfile6"},
		expectedfilessub:  []string{"testfile0", "testfile6"},
	},
	testParams{
		description: "find all files by modification time in minutes",
		mtime:       []string{"<1m"},
		expectedfilesroot: []string{"testfile0", "testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8",
			"testfile9"},
		expectedfilessub: []string{"testfile0", "testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8",
			"testfile9"},
	},
	testParams{
		description:       "find testfile0 by modification time in days with name",
		name:              []string{"^testfile0$"},
		mtime:             []string{"<1d"},
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{"testfile0"},
	},
	testParams{
		description:       "find no files with modification time which should not match",
		mtime:             []string{">1h", ">30m", ">15m"},
		expectedfilesroot: []string{},
		expectedfilessub:  []string{},
	},
	testParams{
		description:       "find testfile0 using maxdepth",
		name:              []string{"^.*0$"},
		maxdepth:          1,
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{},
	},
	testParams{
		description: "inverted content match on testfile0",
		content:     []string{"!^--- header for first file ---$"},
		maxdepth:    1,
		expectedfilesroot: []string{"testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8"},
		expectedfilessub: []string{},
	},
	testParams{
		description: "find files with specific mode",
		mode:        []string{"-rw-r--r--"},
		expectedfilesroot: []string{"testfile0", "testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8", "testfile9"},
		expectedfilessub: []string{"testfile0", "testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8", "testfile9"},
	},
	testParams{
		description:       "find testfile0 with two checks which will match the same file",
		name:              []string{".*0", "^testfile0$"},
		maxdepth:          1,
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{},
	},
	// Various error conditions
	testParams{
		description:       "search a nonexistent root",
		name:              []string{".*testfile.*"},
		expectedfilesroot: []string{},
		expectedfilessub:  []string{},
		searchpath:        []string{"/doesnotexist"},
	},
	// MACROAL tests
	// Regex     | Inverse | MACROAL | Result
	// -----------+---------+---------+--------
	// Match     |  False  |  True   | pass	-> must match all lines and current line matched
	// Match     |  True   |  True   | fail	-> must match no line but current line matches
	// Not Match |  True   |  True   | pass	-> must match no line and current line didn't match
	// Not Match |  False  |  True   | fail	-> much match all lines and current line didn't match
	testParams{
		description:       "macroal match on testfile0",
		macroal:           true,
		content:           []string{"^(.+)?$"},
		name:              []string{"^testfile0$"},
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{"testfile0"},
	},
	testParams{
		description:       "macroal match on testfile0, negated",
		macroal:           true,
		content:           []string{"!^(.+)?$"},
		name:              []string{"^testfile0$"},
		expectedfilesroot: []string{},
		expectedfilessub:  []string{},
	},
	testParams{
		description:       "macroal match on testfile0, negated with non-matching expression",
		macroal:           true,
		content:           []string{"!FOOBAR"},
		name:              []string{"^testfile0$"},
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{"testfile0"},
	},
	testParams{
		description:       "macroal match on testfile0, with non-matching expression",
		macroal:           true,
		content:           []string{"FOOBAR"},
		name:              []string{"^testfile0$"},
		expectedfilesroot: []string{},
		expectedfilessub:  []string{},
	},
	// Mismatch tests
	testParams{
		description: "mismatch match files that do not match name testfile0",
		name:        []string{"^testfile0$"},
		mismatch:    []string{"name"},
		expectedfilesroot: []string{"testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8",
			"testfile9"},
		expectedfilessub: []string{"testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8",
			"testfile9"},
	},
	testParams{
		description: "mismatch match files that do not meet specified size criteria",
		size:        []string{"190", ">10k", ">10m", ">10g", ">10t", "<10"},
		mismatch:    []string{"size"},
		expectedfilesroot: []string{"testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8"},
		expectedfilessub: []string{"testfile1", "testfile2", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8"},
	},
	testParams{
		description:       "mismatch match files that have not been modified in last hour",
		mtime:             []string{"<1h"},
		mismatch:          []string{"mtime"},
		expectedfilesroot: []string{},
		expectedfilessub:  []string{},
	},
	testParams{
		description:       "mismatch match files that do not have mode 0644",
		mode:              []string{"-rw-r--r--"},
		mismatch:          []string{"mode"},
		expectedfilesroot: []string{},
		expectedfilessub:  []string{},
	},
	testParams{
		description:       "mismatch match files that do not have name different than !testfile0",
		name:              []string{"!^testfile0$"},
		mismatch:          []string{"name"},
		expectedfilesroot: []string{"testfile0"},
		expectedfilessub:  []string{"testfile0"},
	},
	testParams{
		description:       "mismatch match content test with macroal and matchall",
		name:              []string{"^testfile(0|1)$"},
		content:           []string{"^((---.+)|(#.+)|(\\s+)|(some (other )?text))?$"},
		macroal:           true,
		mismatch:          []string{"content"},
		expectedfilesroot: []string{"testfile1"},
		expectedfilessub:  []string{"testfile1"},
	},
	testParams{
		description: "mismatch match files that do not have testfile2 hash value",
		mismatch:    []string{"md5", "sha1", "sha2", "sha3"},
		md5:         []string{"8d3a7afb7e59693b383d52396243a5b8"},
		sha1:        []string{"d82bc1145d471714b056940b268032f9ab0df2ae"},
		sha2:        []string{"3b495fae5bae9751ea4706c29e992002ba277bce30bd83a827b01ba977eabc2f"},
		sha3:        []string{"fdb23afa808c265284c3199013e4ded9704eebf54ffdc1f016dacc12"},
		expectedfilesroot: []string{"testfile0", "testfile1", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8",
			"testfile9"},
		expectedfilessub: []string{"testfile0", "testfile1", "testfile3",
			"testfile4", "testfile5", "testfile6", "testfile7", "testfile8",
			"testfile9"},
	},
}

func (tp *testParams) runTest(t *testing.T) {
	var (
		r  run
		s  Search
		mr modules.Result
		sr SearchResults
	)

	// Reinitialize various globals in the file module upon each test run
	debug = false
	walkingErrors = make([]string, 0)
	tryDecompress = false
	stats.Filescount = 0
	stats.Openfailed = 0
	stats.Totalhits = 0
	stats.Exectime = ""

	t.Logf("runTest: %v", tp.description)
	r.Parameters = *newParameters()
	if len(tp.searchpath) != 0 {
		s.Paths = append(s.Paths, tp.searchpath...)
	} else {
		s.Paths = append(s.Paths, basedir)
	}
	// Substitute any tokens out of the search paths, used for some special path
	// value manipulation in certain tests
	for i := range s.Paths {
		if !strings.HasPrefix(s.Paths[i], "SEARCHBASE+") {
			continue
		}
		fcomponent := s.Paths[i][11:]
		s.Paths[i] = path.Join(basedir, fcomponent)
	}
	if len(tp.content) != 0 {
		s.Contents = append(s.Contents, tp.content...)
	}
	if len(tp.name) != 0 {
		s.Names = append(s.Names, tp.name...)
	}
	if len(tp.size) != 0 {
		s.Sizes = append(s.Sizes, tp.size...)
	}
	if len(tp.mode) != 0 {
		s.Modes = append(s.Modes, tp.mode...)
	}
	if len(tp.mtime) != 0 {
		s.Mtimes = append(s.Mtimes, tp.mtime...)
	}
	if len(tp.md5) != 0 {
		s.MD5 = append(s.MD5, tp.md5...)
	}
	if len(tp.sha1) != 0 {
		s.SHA1 = append(s.SHA1, tp.sha1...)
	}
	if len(tp.sha2) != 0 {
		s.SHA2 = append(s.SHA2, tp.sha2...)
	}
	if len(tp.sha3) != 0 {
		s.SHA3 = append(s.SHA3, tp.sha3...)
	}
	if tp.maxdepth != 0 {
		s.Options.MaxDepth = tp.maxdepth
	}
	if tp.macroal {
		s.Options.Macroal = true
	}
	if len(tp.mismatch) != 0 {
		s.Options.Mismatch = append(s.Options.Mismatch, tp.mismatch...)
	}
	s.Options.MatchAll = true
	r.Parameters.Searches["s1"] = s
	msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters, false)
	if err != nil {
		t.Fatalf("modules.MakeMessage: %v", err)
	}
	out := r.Run(modules.NewModuleReader(bytes.NewBuffer(msg)))
	if len(out) == 0 {
		t.Fatal("module returned no output")
	}

	err = json.Unmarshal([]byte(out), &mr)
	if err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !mr.Success {
		t.Fatal("module result indicated it was not successful")
	}
	err = mr.GetElements(&sr)
	if err != nil {
		t.Fatalf("GetElements: %v", err)
	}
	// Build a list of the files we got back from the search
	gotfiles := make([]string, 0)
	for _, x := range sr["s1"] {
		if x.File == "" {
			continue
		}
		gotfiles = append(gotfiles, x.File)
	}
	expected := tp.getExpectedFiles()
	// If the number of expected files is 0, than FoundAnything should be false
	if len(expected) == 0 {
		if mr.FoundAnything {
			t.Logf("%v", out)
			t.Fatal("expected 0 files, but module run indicating it found something")
		}
	}
	if len(gotfiles) != len(expected) {
		t.Fatalf("test should have returned %v files, but returned %v", len(expected),
			len(gotfiles))
	}
	for _, x := range expected {
		found := false
		for _, y := range gotfiles {
			if x == y {
				found = true
			}
		}
		if !found {
			t.Fatalf("%v not found in results", x)
		}
	}
}

type testFile struct {
	filename              string
	data                  []byte
	link                  bool   // If true, created file will be a symlink
	linkdest              string // Destination file name for symlink, must be in same directory
	md5, sha1, sha2, sha3 string
	size                  int
}

var testFiles = []testFile{
	testFile{
		filename: "testfile0",
		data: []byte(`--- header for first file ---
# this is a comment
                                       
# above is an line filled with spaces

# above is an empty line, no spaces
some text
some other text`),
		md5:  "e499c1912bd9af4f7e8ccaf27f7b04d2",
		sha1: "d7bbc3dd7adf6e347c93a4c8b9bfb8ef4748c0fb",
		sha2: "4d8ef27c4415d71cbbfad1eaa97d6f2a3ddacc9708b66efbb726133b9fd3d79a",
		sha3: "a7ba1e66174848ecea143b612f22168b006979e3827e09f0ae6395e8",
		size: 190,
	},
	testFile{
		filename: "testfile1",
		data: []byte(`--- header for second file ---
# this is a comment
                                       
# above is an line filled with spaces
# above is an empty line, no spaces
some text
some other other text`),
		md5:  "072841679be61acd27de062da1ad6fdf",
		sha1: "21f4a0f1d86915f9fa676b96a823c4c3142eb22b",
		sha2: "72573e5f095cb29afa2486b519928ed153558a8c036f15a9d1f790c8989e96c3",
		sha3: "7ec2e3b36e220b3c5ea9ad0129a1cdcd6dd7f545c92a90f8419ea05d408ca9d5ec999452fd804df7ede9ca0f0647195ae03eba1be7fae0c2217a8f24eaf7cce0",
		size: 196,
	},
	testFile{
		filename: "testfile2",
		data: []byte("\x35\xF3\x40\xD8\xE9\xCE\x96\x38\xBD\x02\x80\xE4\xED\xA8\xCE\x5F\x5D\xEB\xDB\x92" +
			"\x2A\x63\xB0\x66\x5F\xC7\xCA\x57\xB5\xFC\x76\x9B\x44\x89\x48\x9E\x73\x6B\x5A\x30" +
			"\x8E\xC7\x60\xD3\xF2\xA8\x36\x7F\xED\xCE\xC7\x1E\xE9\xB2\x1B\x73\xC4\x72\xE8\xAE" +
			"\xDB\x0D\x2A\xB2\xDD\x8F\x29\xDB\x98\xF8\xDE\x47\x5F\xEA\x1C\x6C\x2A\xD3\xFB\x70" +
			"\x8C\x03\x5A\x67\x3A\xBF\xEC\x68\x49\x7F\x00\x4B\xE1\x87\x95\xE6\x34\x44\x32\x83" +
			"\x78\xA1\x06\xCB\x57\xB7\xE4\x7E\x16\x49\xFF\x03\x59\xBD\xD0\xA1\x67\xA7\x03\x9E" +
			"\xF5\x99\x3D\x62\xEE\xFE\x93\xE9\xAD\xA2\xD4\x0D\x15\xB5\x9C\x4C\x3A\x44\xD9\xA3" +
			"\xAC\xEF\xF3\x68\xEB\x11\xF2\xC2\xA4\x32\xD1\xC3\xF0\x5C\x60\xCA\x75\x99\xD9\x68" +
			"\x24\x46\x74\x62\x9E\x21\x89\x12\xC5\x74\x8E\xCE\x07\xEF\xC7\xE7\x81\x51\x40\x0E" +
			"\xDD\x48\xD5\xEC\x8E\x17\x8F\x18\xB7\x03\xB2\xFB\x66\x0D\xF8\x45\xCA\x19\x27\xA0" +
			"\x65\x18\xED\x43\x74\x24\xC7\xB4\x61\x91\x21\x63\xD0\x49\x95\xC7\x87\x9C\x7B\x5A" +
			"\xE6\x96\xD1\xBF\x28\x28\x09\xD3\xA3\x18\xB1\x8F\xF6\xA5\xE6\xD9\x69\x77\xD0\x8E" +
			"\xAC\x1A\x2B\xC0\x57\xAB\xFD\x04\x9D\x37\x93\xE0\xBA\x61\x0C\x59\x12\xE4\xAF\x48" +
			"\x91\x47\x2D\x15\xAA\x3F\x8C\x17\xEF\x34\x58\xC1\xD1\x09\xE1\x47\x60\x9A\xD1\xEC" +
			"\x1A\xE2\x59\x1A\xC4\x58\xF5\x38\xE6\x46\x53\xF2\x89\x44\xFD\x8A\xD0\xC6\x4C\x2C" +
			"\x9C\xEA\xC7\xDF\x29\xB8\xAA\x33\x16\xF0\x2A\x3F\x1D\x21\xA2\x08\x8E\xED\x02\x86" +
			"\x80\x48\x75\xF2\xD2\xA8\x3F\x56\x9F\x4A\xB1\x7F\x26\x82\xC5\x2D\x16\xFD\xBD\xE0" +
			"\x00\xD0\x0E\xFA\x4F\x6E\x22\x0B\xFC\xC6\x89\x25\x35\x41\xBC\x84\x2C\x35\x11\x52" +
			"\xCC\x77\xC6\x5A\xB9\x62\xFE\xCC\x82\xEE\x4A\x2A\x8A\x09\x70\xC0\xEA\xE6\x8C\xC8" +
			"\x6F\x2A\x4C\x06\x19\xCF\xDC\x95\x3D\xB9\x67\x0C\x90\xC7\x72\x24\x96\xA9\xCD\x33" +
			"\x76\xD7\xF3\x6E\xFF\x3C\xEE\x9C\x2A\xA8\xE3\x2F\x13\x84\x2F\x5B\xBE\x37\x63\x24" +
			"\xD4\x02\xE8\xD8\x8E\x08\x12\x5E\x6C\x8E\x98\xF1\x3E\x6C\x3E\x4D\xF1\xD7\xF2\xBA" +
			"\x19\x46\xFB\x0F\xCE\xB0\x48\x40\xEA\xF7\x59\x3B\x8A\x92\xDB\xA2\x1C\x8B\xE4\x18" +
			"\xEE\x4A\x04\x3D\x07\xF3\x78\x8C\xFA\xC8\x05\xD1\x76\x44\x86\xE5\x63\x73\x3F\xAC" +
			"\x65\x57\x8B\xC5\x02\x87\xF7\x36\xDD\x90\xE3\xE5\xCE\xC7\x66\x6C\x96\x9A\x1C\x1C" +
			"\x82\x5E\x40\xF1\x84\x15\x00\xF6\x31\x25\xA0\xBB\x95\x83\xD7\xB6\x9C\x7C\x8C\x20" +
			"\x5E\xE6\xF7\x4F\x86\xAE\x58\x88\x56\x22\x6F\xF6\xD9\x77\x9C\x12\x68\xB4\xEF\xF2" +
			"\x3E\xC5\x0B\xB5\xF2\x31\xDB\x9F\xD5\x07\x4E\x84\x6B\x1E\xB6\x72\x79\x8B\x3B\x06" +
			"\xE8\x51\x2B\x7C\x2E\xFD\x15\x88\x66\x72\x9D\x7D\xE0\x9D\xA0\x6D\xF6\x33\xD5\xC1" +
			"\x19\x0D\x9B\x1F\xA7\x87\x97\xAC\x32\x3D\xE3\xC3\xAA\x48\x1D\xDF\x3C\xFE\xDC\x35" +
			"\x15\x7C\x27\x40\x82\xAD\x77\xCD\x77\x7D\x03\x90\x0C\x93\xA8\x29\xCF\xA7\x18\xAC" +
			"\xF7\x6A\x1D\x52\x03\x5F\xA2\xB3\x12\xD3\x64\xF1\x77\xB0\xB5\x27\x18\xFF\x20\x0D" +
			"\x1D\x7E\x39\x25\xE1\x78\x74\x46\x1A\x3E\x70\x80\x14\xA0\xED\x2F\xB9\xE2\xC4\xCA" +
			"\xCB\x19\x09\xED\x11\xE8\x5E\x07\x61\x2E\x5A\x27\xF9\xF4\x60\x64\x78\xDB\x12\x7D" +
			"\xE2\xC5\xB4\xCF\x79\x7C\x4F\xAD\x79\xC7\x18\xFE\xEB\x9B\xE7\x9F\xE7\xEF\x58\x42" +
			"\x93\x01\x1E\x08\x1A\x9C\x65\x75\x63\xCD\x5C\x3C\x53\x8A\xB3\xE8\x52\xBF\x62\x97" +
			"\x73\x41\x35\x3B\x1C\xEF\x27\x64\x46\x9E\x17\x35\x51\x17\x36\x16\x4E\x79\xEA\x9E" +
			"\x5A\xA0\xB6\xFC\xEA\x13\x49\x6C\xB2\x86\x3B\x70\x70\x84\xEB\xF5\xEF\xFB\x57\xC1" +
			"\x1E\x76\x6B\x2A\x5E\x49\xC1\x48\x64\x91\x4F\xF5\x10\xE5\x7A\xE7\x87\xA2\x97\xAA" +
			"\xF4\xF5\xBD\x3B\xA6\x6D\x73\x3B\xFA\x98\x26\xE2\xC4\x08\xDD\xBA\x5A\xC3\xA7\xAB" +
			"\x07\x88\xC2\xA3\x55\x78\xE9\x09\xBA\x0E\xED\xEB\x9B\x8E\xFE\x73\xBB\x63\xED\x33" +
			"\x38\x21\x04\xEA\xE4\x6A\xF6\xE8\x12\xFA\xE8\x91\x4B\x7C\x33\xB9\xAF\x33\x4C\x5B" +
			"\xC0\xD3\x0E\xF2\x4F\x4A\x98\xC5\xEF\x1C\x9D\x08\xC0\x33\x20\xC2\x00\xD1\xE9\xB0" +
			"\xF5\x62\xAB\x05\x52\x66\x04\xC9\xEB\x19\x66\x96\x06\xF5\x48\x55\x0D\x66\xB7\x3D" +
			"\xB4\x50\x7D\xF2\x63\xBB\xBC\xDF\x8C\x4F\x4D\x86\xEA\x52\xFD\x18\x25\xBE\xC2\x13" +
			"\x17\xAA\xFD\x2B\x05\xBA\xB2\xB1\x0B\x6C\xB4\x1B\x47\xFE\x3D\x02\x25\x35\x8E\xF9" +
			"\x3C\x86\x90\x94\x7D\xF4\x98\x56\x2C\xCC\x27\xAD\x9F\xC7\x0A\x8C\x63\x5E\xDC\x83" +
			"\x05\x2A\x57\xB7\x22\x4A\x6A\x78\x18\x0B\xB3\x95\x0A\xE5\xEB\xE0\x57\xF7\xD4\xA5" +
			"\xDF\x88\x8D\x8D\x65\xA6\xA0\x40\x01\x4B\x6D\x2E\x3D\xE5\xE7\x43\x7D\x99\xB2\x0C" +
			"\x00\xF3\x39\x34\x84\x6D\x76\x69\xF0\x7D\x90\x39\x16\x84\x37\x52\xA5\x79\xCF\x20" +
			"\x18\xC2\x00\x31\xCD\x6C\x38\x25\x5D\x47\xB6\x2B\x3F\xA0\x7D\xB3\x69\x85\xBF\xF8" +
			"\x25\x38\x32\x35"),
		md5:  "8d3a7afb7e59693b383d52396243a5b8",
		sha1: "d82bc1145d471714b056940b268032f9ab0df2ae",
		sha2: "3b495fae5bae9751ea4706c29e992002ba277bce30bd83a827b01ba977eabc2f",
		sha3: "fdb23afa808c265284c3199013e4ded9704eebf54ffdc1f016dacc12",
		size: 1024,
	},
	testFile{
		filename: "testfile3",
		data: []byte(`--- header for fourth file ---
# above is an line filled with spaces

# above is an empty line, no spaces
some text
some other text`),
		md5:  "d6b008f34e7cf207cb9bc74a2153fffd",
		sha1: "9ee0213f3227fe4f3658af0c3de315669b36ccf9",
		sha2: "fb9758f30549a282d41a4eb125790704c17309e55443dbb54895379b8e33438f2825b78b938aa3735f99f3305d3b98e8",
		sha3: "fe66d22caa59899c386e0a041f641d1c8130ded8f7365330957cbf69",
		size: 131,
	},
	testFile{
		filename: "testfile4",
		data: []byte(`--- header for fifth file ---
# this is a comment
                                       
# above is an empty line, no spaces
some text
some other text`),
		md5:  "5d5a4fdeafc1677dca8255ef9624d522",
		sha1: "caf4ce81c990785e5041bfc410526f471ea1ba6f",
		sha2: "a4001843158a7a374e5ddcc22644c0e37738bc64ffd50179fc18fb443e0a62393b43384d9ac734e7a64c204e862ae3424094381afb33dfc639c52517afad1f32",
		sha3: "2028feaccf974066aa7c47070f24c72d349ed6a6575cb801cc606c4a2b59020af4339b60dbedd0049a7341edde14133ee6f8b199f1a7c6ef36493fd217501607",
		size: 151,
	},
	testFile{
		filename: "testfile5",
		data: []byte(`--- header for sixth file ---
# this is a comment
                                       
some text
some other text`),
		md5:  "f9132062fccc09cba5f93474724a57e3",
		sha1: "fb03d2d4ac2a82090bc29934f75c1d6914bacc91",
		sha2: "8871b2ff047be05571549398e54c1f36163ae171e05a89900468688ea3bac4f9f3d7c922f0bebc24fdac28d0b2d38fb2718209fb5976c9245e7c837170b79819",
		sha3: "cb086f02b728d57e299651f89e1fb0f89c659db50c7c780ec2689a8143e55c8e5e63ab47fe20897be7155e409151c190",
		size: 115,
	},
	testFile{
		filename: "testfile6",
		data: []byte("\x1f\x8b\x08\x08\xd9\xdc\x88\x56\x00\x03\x74\x65\x73\x74\x00\x8d" +
			"\x8e\xcd\x0a\xc3\x30\x0c\x83\xef\x79\x0a\xc1\xae\xf3\x43\x65\xad" +
			"\x4a\x0c\x49\x1c\x1a\xb3\xbf\xa7\x5f\x96\xb2\x4b\x4f\x33\x42\x18" +
			"\xf3\x49\x58\x44\x90\x18\x57\xee\xd8\x6c\xc7\x5b\x5b\xe3\x8a\x4d" +
			"\x33\x21\x22\xe1\x02\x4f\xda\x31\x14\xb1\x58\x29\xac\x1e\xf0\xdf" +
			"\x8c\x6c\xbc\xd9\x9d\x33\x5c\x91\xb5\xf2\xdb\x9b\x47\xfd\x43\x3d" +
			"\xa1\xb7\xb8\xb0\x87\x13\xc6\xd2\xfc\x35\xe1\x2b\xaa\xfd\xa0\x6e" +
			"\x85\x70\x3e\xfd\xd8\xcc\xd3\xf8\xf7\xf0\x79\xfd\x00\x4c\x08\xa4" +
			"\x7a\xc6\x00\x00\x00"),
		md5:  "31d38eee231318166538e1569631aba9",
		sha1: "bd1f24d8cbb000bbf7bcd618c2aec73280388721",
		sha2: "bb4e449df74edae0292d60d2733a3b1801d90ae23560484b1e04fb52f111a14f",
		sha3: "433b84f162d1b00481e6da022c5738fb4d04c3bb4317f73266746dd1",
		size: 133,
	},
	testFile{
		filename: "testfile7",
		data: []byte("\x1f\x8b\x08\x00\xd9\xe3\x8f\x56\x00\x03\xed\xd4\xcb\x6a\x84\x30" +
			"\x14\x06\x60\xd7\xf3\x14\x07\xba\xad\x60\xe2\xed\x09\xfa\x02\x5d" +
			"\x14\xba\x4c\xf5\x0c\x86\x6a\x22\xe6\xf4\xfa\xf4\x8d\xce\x74\x28" +
			"\x03\x65\xba\xb1\x45\xfa\x7f\x44\x22\x7a\x12\x43\xf0\x8f\x70\x90" +
			"\xbd\xed\x59\x25\xeb\xc9\xa2\xaa\x2a\xe6\x5e\xd5\x65\xf6\xb5\x5f" +
			"\xe4\xaa\x4c\x94\xae\x8a\xbc\xae\x54\xa1\x55\x92\x29\xad\xeb\x2c" +
			"\xa1\x6c\xc5\x35\x9d\x3c\x05\x31\x13\x51\xf2\x68\x43\xe7\xa7\xef" +
			"\xeb\x2e\xbd\xdf\xa8\x34\x4d\xa9\x63\xd3\xf2\x44\x7b\x1f\x2f\x3b" +
			"\x05\xa1\x77\x3b\x8e\xdc\xd2\xfc\x5f\x50\x2c\xd8\x5d\x91\x74\x36" +
			"\x50\x6c\x86\x1a\x3f\x0c\xec\x64\x47\x3f\x13\xc7\x9a\x07\xff\xcc" +
			"\xcb\x60\x47\xbd\x75\x3c\xcf\xdb\xc7\xe9\x5f\xac\x74\x14\x46\xd3" +
			"\x70\xd8\x9d\x95\xf1\x30\xca\xdb\x52\x7c\x4d\xce\x7f\x16\x05\x3f" +
			"\x30\x09\xbf\xca\xe1\xce\x4b\x17\x57\x3d\x19\xd7\xfa\xe1\xf0\xf8" +
			"\xaf\x37\x73\x83\xe4\x98\x7f\xbd\xe2\x37\x2e\xe6\x5f\xe7\xa7\xfc" +
			"\x97\xc5\x31\xff\x39\xf2\xff\x1b\xce\xf2\x1f\xb8\xf1\xae\xdd\xd4" +
			"\x01\x70\x77\x73\x7b\x8f\x53\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
			"\x00\x00\x00\x00\x00\x00\x00\xfe\xaf\x0f\x60\x69\x1f\x15\x00\x28" +
			"\x00\x00"),
		md5:  "52fa96013b5c6aa9302d39ee7fe2f6a5",
		sha1: "31952c0d2772c302ec94b303c2b80b67cf830060",
		sha2: "f6032dc9b4ba112397a6f8bcb778ab10708c1acd38e48f637ace15c3ae417ded",
		sha3: "3f4dacf0b2347d0a0ab6f09b7d7c98fd12cb2030d4af8baeacaf55a9",
		size: 274,
	},
	testFile{
		filename: "testfile8",
		data: []byte("\x1f\x8b\x08\x08\x2c\xe8\x8f\x56\x00\x03\x74\x65\x73\x74\x66\x69" +
			"\x6c\x65\x33\x00\x8d\x8e\x4b\x0a\x03\x31\x0c\x43\xf7\x73\x0a\x41" +
			"\xb7\xf5\xa1\xd2\x19\x0d\x09\x24\x71\x48\xdc\xef\xe9\xeb\xa6\xcc" +
			"\xa6\xab\x1a\x81\x6c\x78\x12\x16\x11\x44\x86\x8d\x1d\xbb\x76\xac" +
			"\xda\xfb\xb5\x19\x37\xbc\x52\x6b\x00\x00\xca\x84\x88\x2c\x27\x58" +
			"\x4c\x03\xae\xe0\x54\x29\xac\xb6\xe0\xbf\xf1\x6c\xb8\xe8\x8d\x33" +
			"\x5c\x91\x53\xe5\xa7\x37\x7b\xfd\x3d\x59\xc4\x68\x61\xe5\x58\x7e" +
			"\x30\x96\x66\xcf\x09\x9f\x51\xf5\x80\x86\x16\xc2\xf8\xb0\xef\xa6" +
			"\x16\xfd\xf3\x79\xbf\x01\x7b\xae\xde\x84\xca\x00\x00\x00"),
		md5:  "df7b577ceb59f700d5b03db9d12d174e",
		sha1: "ea033d30e996ac443bc50e9c37eb25b37505302e",
		sha2: "2f4f81c0920501f178085032cd2784b8aa811b8c8e94da7ff85a43a361cd96cc",
		sha3: "d171566f8026a4ca6b4cdf8e6491a651625f98fbc15f9cb601833b64",
		size: 142,
	},
	testFile{
		filename: "testfile9",
		link:     true,
		linkdest: "testfile0",
	},
}
