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
	"path/filepath"
	"testing"
	"time"

	"mig.ninja/mig/modules"
	"mig.ninja/mig/testutil"
)

var basedir string

func TestMain(m *testing.M) {
	basedir = createFiles()
	r := m.Run()
	//err := os.RemoveAll(basedir)
	//if err != nil {
	//	log.Fatalf("failed to remove %s: %v\n", basedir, err)
	//}
	os.Exit(r)
}

func TestRegistration(t *testing.T) {
	testutil.CheckModuleRegistration(t, "file")
}

func TestNameSearch(t *testing.T) {
	for _, tp := range TESTDATA {
		var (
			r run
			s search
		)
		var expectedfiles = []string{
			basedir + "/" + tp.name,
			basedir + subdirs + tp.name,
		}
		r.Parameters = *newParameters()
		s.Paths = append(s.Paths, basedir)
		s.Names = append(s.Names, "^"+tp.name+"$")
		s.Names = append(s.Names, "!^"+tp.name+"FOOBAR$")
		s.Options.MatchAll = true
		r.Parameters.Searches["s1"] = s
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestContentSearch(t *testing.T) {
	for _, tp := range TESTDATA {
		var (
			r run
			s search
		)
		var expectedfiles = []string{
			basedir + "/" + tp.name,
			basedir + subdirs + tp.name,
		}
		r.Parameters = *newParameters()
		s.Paths = append(s.Paths, basedir)
		s.Contents = append(s.Contents, tp.content)
		s.Contents = append(s.Contents, "!^FOOBAR$")
		s.Options.MatchAll = true
		r.Parameters.Searches["s1"] = s
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSize(t *testing.T) {
	for _, tp := range TESTDATA {
		var (
			r run
			s search
		)
		var expectedfiles = []string{
			basedir + "/" + tp.name,
			basedir + subdirs + tp.name,
		}
		r.Parameters = *newParameters()
		s.Paths = append(s.Paths, basedir)
		s.Sizes = append(s.Sizes, tp.size)
		r.Parameters.Searches["s1"] = s
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMTime(t *testing.T) {
	for _, tp := range TESTDATA {
		var (
			r run
			s search
		)
		var expectedfiles = []string{
			basedir + "/" + tp.name,
			basedir + subdirs + tp.name,
		}
		r.Parameters = *newParameters()
		s.Paths = append(s.Paths, basedir)
		s.Names = append(s.Names, tp.name)
		s.Mtimes = append(s.Mtimes, tp.mtime)
		s.Options.MatchAll = true
		r.Parameters.Searches["s1"] = s
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMode(t *testing.T) {
	for _, tp := range TESTDATA {
		var (
			r run
			s search
		)
		var expectedfiles = []string{
			basedir + "/" + tp.name,
			basedir + subdirs + tp.name,
		}
		r.Parameters = *newParameters()
		s.Paths = append(s.Paths, basedir)
		s.Names = append(s.Names, tp.name)
		s.Modes = append(s.Modes, tp.mode)
		s.Options.MatchAll = true
		r.Parameters.Searches["s1"] = s
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestHashes(t *testing.T) {
	for _, hashtype := range []string{`md5`, `sha1`, `sha256`, `sha384`, `sha512`,
		`sha3_224`, `sha3_256`, `sha3_384`, `sha3_512`} {
		for _, tp := range TESTDATA {
			var (
				r run
				s search
			)
			var expectedfiles = []string{
				basedir + "/" + tp.name,
				basedir + subdirs + tp.name,
			}
			r.Parameters = *newParameters()
			s.Paths = append(s.Paths, basedir)
			switch hashtype {
			case `md5`:
				s.MD5 = append(s.MD5, tp.md5)
			case `sha1`:
				s.SHA1 = append(s.SHA1, tp.sha1)
			case `sha256`:
				s.SHA256 = append(s.SHA256, tp.sha256)
			case `sha384`:
				s.SHA384 = append(s.SHA384, tp.sha384)
			case `sha512`:
				s.SHA512 = append(s.SHA512, tp.sha512)
			case `sha3_224`:
				s.SHA3_224 = append(s.SHA3_224, tp.sha3_224)
			case `sha3_256`:
				s.SHA3_256 = append(s.SHA3_256, tp.sha3_256)
			case `sha3_384`:
				s.SHA3_384 = append(s.SHA3_384, tp.sha3_384)
			case `sha3_512`:
				s.SHA3_512 = append(s.SHA3_512, tp.sha3_512)
			}
			r.Parameters.Searches["s1"] = s
			msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s\n", msg)
			out := r.Run(bytes.NewBuffer(msg))
			if len(out) == 0 {
				t.Fatal("run failed")
			}
			t.Log(out)
			err = evalResults([]byte(out), expectedfiles)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestAllHashes(t *testing.T) {
	for _, tp := range TESTDATA {
		var (
			r run
			s search
		)
		var expectedfiles = []string{
			basedir + "/" + tp.name,
			basedir + subdirs + tp.name,
		}
		r.Parameters = *newParameters()
		s.Paths = append(s.Paths, basedir)
		s.MD5 = append(s.MD5, tp.md5)
		s.SHA1 = append(s.SHA1, tp.sha1)
		s.SHA256 = append(s.SHA256, tp.sha256)
		s.SHA384 = append(s.SHA384, tp.sha384)
		s.SHA512 = append(s.SHA512, tp.sha512)
		s.SHA3_224 = append(s.SHA3_224, tp.sha3_224)
		s.SHA3_256 = append(s.SHA3_256, tp.sha3_256)
		s.SHA3_384 = append(s.SHA3_384, tp.sha3_384)
		s.SHA3_512 = append(s.SHA3_512, tp.sha3_512)
		s.Options.MatchAll = true
		r.Parameters.Searches["s1"] = s
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMaxDepth(t *testing.T) {
	var (
		r run
		s search
	)
	var expectedfiles = []string{
		basedir + "/" + TESTDATA[0].name,
	}
	r.Parameters = *newParameters()
	s.Paths = append(s.Paths, basedir)
	s.Names = append(s.Names, "^"+TESTDATA[0].name+"$")
	s.Contents = append(s.Contents, TESTDATA[0].content)
	s.Options.MatchAll = true
	s.Options.MaxDepth = 5
	r.Parameters.Searches["s1"] = s
	msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s\n", msg)
	out := r.Run(bytes.NewBuffer(msg))
	if len(out) == 0 {
		t.Fatal("run failed")
	}
	t.Log(out)
	if evalResults([]byte(out), expectedfiles) != nil {
		t.Fatal(err)
	}
}

/* Test all cases of macroal
 Regex     | Inverse | MACROAL | Result
-----------+---------+---------+--------
 Match     |  False  |  True   | pass	-> must match all lines and current line matched
 Match     |  True   |  True   | fail	-> must match no line but current line matches
 Not Match |  True   |  True   | pass	-> must match no line and current line didn't match
 Not Match |  False  |  True   | fail	-> much match all lines and current line didn't match
*/
type macroaltest struct {
	desc, name, content string
	expectedfiles       []string
}

func TestMacroal(t *testing.T) {
	var MacroalTestCases = []macroaltest{
		macroaltest{
			desc:    "want testfile0 with all lines matching '^(.+)?$', should find 2 files",
			name:    "^" + TESTDATA[0].name + "$",
			content: "^(.+)?$",
			expectedfiles: []string{
				basedir + "/" + TESTDATA[0].name,
				basedir + subdirs + TESTDATA[0].name},
		},
		macroaltest{
			desc:          "want testfile0 with no line matching '^(.+)?$', should find 0 file",
			name:          "^" + TESTDATA[0].name + "$",
			content:       "!^(.+)?$",
			expectedfiles: []string{""},
		},
		macroaltest{
			desc:    "want testfile0 with no line matching '!FOOBAR', should find 2 files",
			name:    "^" + TESTDATA[0].name + "$",
			content: "!FOOBAR",
			expectedfiles: []string{
				basedir + "/" + TESTDATA[0].name,
				basedir + subdirs + TESTDATA[0].name},
		},
		macroaltest{
			desc:          "want testfile0 with all lines matching 'FOOBAR', should find 0 file",
			name:          "^" + TESTDATA[0].name + "$",
			content:       "FOOBAR",
			expectedfiles: []string{""},
		},
	}
	for _, mt := range MacroalTestCases {
		t.Log(mt.desc)
		var r run
		var s search
		r.Parameters = *newParameters()
		s.Paths = append(s.Paths, basedir)
		s.Names = append(s.Names, mt.name)
		s.Contents = append(s.Contents, mt.content)
		s.Options.MatchAll = true
		s.Options.Macroal = true
		r.Parameters.Searches["s1"] = s
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), mt.expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}
}

type mismatchtest struct {
	desc          string
	search        search
	expectedfiles []string
}

func TestMismatch(t *testing.T) {
	var MismatchTestCases = []mismatchtest{
		mismatchtest{
			desc: "want files that don't match name '^testfile0' with maxdept=1, should find testfile1 and testfile2",
			search: search{
				Paths: []string{basedir},
				Names: []string{"^" + TESTDATA[0].name + "$"},
				Options: options{
					MaxDepth: 1,
					Mismatch: []string{"name"},
				},
			},
			expectedfiles: []string{
				basedir + "/" + TESTDATA[1].name,
				basedir + "/" + TESTDATA[2].name},
		},
		mismatchtest{
			desc: "want files that don't have a size of 190 bytes or larger than 10{k,m,g,t} or smaller than 10 bytes, should find testfile1 and testfile2",
			search: search{
				Paths: []string{basedir},
				Sizes: []string{"190", ">10k", ">10m", ">10g", ">10t", "<10"},
				Options: options{
					MaxDepth: 1,
					MatchAll: true,
					Mismatch: []string{"size"},
				},
			},
			expectedfiles: []string{
				basedir + "/" + TESTDATA[1].name,
				basedir + "/" + TESTDATA[2].name},
		},
		mismatchtest{
			desc: "want files that have not been modified in the last hour ago, should find nothing",
			search: search{
				Paths:  []string{basedir + subdirs, basedir},
				Mtimes: []string{"<1h"},
				Options: options{
					Mismatch: []string{"mtime"},
				},
			},
			expectedfiles: []string{""},
		},
		mismatchtest{
			desc: "want files that don't have 644 permissions, should find nothing",
			search: search{
				Paths: []string{basedir},
				Modes: []string{"-rw-r--r--"},
				Options: options{
					Mismatch: []string{"mode"},
				},
			},
			expectedfiles: []string{""},
		},
		mismatchtest{
			desc: "want files that don't a name different than testfile0, should find testfile0",
			search: search{
				Paths: []string{basedir},
				Names: []string{"!^testfile0$"},
				Options: options{
					Mismatch: []string{"name"},
				},
			},
			expectedfiles: []string{
				basedir + "/" + TESTDATA[0].name,
				basedir + subdirs + TESTDATA[0].name},
		},
		mismatchtest{
			desc: "test matchall+macroal+mismatch: want file where at least one line fails to match the regex on testfile0, should find testfile1 that has the extra line 'some other other text'",
			search: search{
				Paths:    []string{basedir},
				Names:    []string{"^testfile(0|1)$"},
				Contents: []string{`^((---.+)|(#.+)|(\s+)|(some (other )?text))?$`},
				Options: options{
					MatchAll: true,
					Macroal:  true,
					Mismatch: []string{"content"},
				},
			},
			expectedfiles: []string{
				basedir + "/" + TESTDATA[1].name,
				basedir + subdirs + TESTDATA[1].name},
		},
		mismatchtest{
			desc: "want files that don't match the hashes of testfile2, should find testfile0 & 1",
			search: search{
				Paths:    []string{basedir},
				MD5:      []string{TESTDATA[2].md5},
				SHA1:     []string{TESTDATA[2].sha1},
				SHA256:   []string{TESTDATA[2].sha256},
				SHA384:   []string{TESTDATA[2].sha384},
				SHA512:   []string{TESTDATA[2].sha512},
				SHA3_224: []string{TESTDATA[2].sha3_224},
				SHA3_256: []string{TESTDATA[2].sha3_256},
				SHA3_384: []string{TESTDATA[2].sha3_384},
				SHA3_512: []string{TESTDATA[2].sha3_512},
				Options: options{
					MaxDepth: 1,
					MatchAll: true,
					Mismatch: []string{`md5`, `sha1`, `sha256`, `sha384`, `sha512`,
						`sha3_224`, `sha3_256`, `sha3_384`, `sha3_512`},
				},
			},
			expectedfiles: []string{
				basedir + "/" + TESTDATA[0].name,
				basedir + "/" + TESTDATA[1].name},
		},
	}

	for _, mt := range MismatchTestCases {
		t.Log(mt.desc)
		var r run
		r.Parameters = *newParameters()
		r.Parameters.Searches["s1"] = mt.search
		msg, err := modules.MakeMessage(modules.MsgClassParameters, r.Parameters)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", msg)
		out := r.Run(bytes.NewBuffer(msg))
		if len(out) == 0 {
			t.Fatal("run failed")
		}
		t.Log(out)
		err = evalResults([]byte(out), mt.expectedfiles)
		if err != nil {
			t.Fatal(err)
		}
	}

}

func TestParamsParser(t *testing.T) {
	var (
		r    run
		args []string
		err  error
	)
	args = append(args, "-path", basedir+"/")
	args = append(args, "-name", TESTDATA[0].name)
	args = append(args, "-content", TESTDATA[0].content)
	args = append(args, "-size", TESTDATA[0].size)
	args = append(args, "-size", ">1")
	args = append(args, "-size", "<100000k")
	args = append(args, "-mode", TESTDATA[0].mode)
	args = append(args, "-mtime", TESTDATA[0].mtime)
	args = append(args, "-md5", TESTDATA[0].md5)
	args = append(args, "-sha1", TESTDATA[0].sha1)
	args = append(args, "-sha256", TESTDATA[0].sha256)
	args = append(args, "-sha384", TESTDATA[0].sha384)
	args = append(args, "-sha512", TESTDATA[0].sha512)
	args = append(args, "-sha3_224", TESTDATA[0].sha3_224)
	args = append(args, "-sha3_256", TESTDATA[0].sha3_256)
	args = append(args, "-sha3_384", TESTDATA[0].sha3_384)
	args = append(args, "-sha3_512", TESTDATA[0].sha3_512)
	args = append(args, "-matchany")
	args = append(args, "-matchall")
	args = append(args, "-macroal")
	args = append(args, "-mismatch", "content")
	args = append(args, "-matchlimit", "10")
	args = append(args, "-maxdepth", "2")
	args = append(args, "-verbose")
	t.Logf("%s\n", args)
	_, err = r.ParamsParser(args)
	if err != nil {
		t.Fatal(err)
	}
}

func evalResults(jsonresults []byte, expectedfiles []string) error {
	var (
		mr modules.Result
		sr SearchResults
	)
	err := json.Unmarshal(jsonresults, &mr)
	if err != nil {
		return err
	}
	if !mr.Success {
		return fmt.Errorf("failed to run file search")
	}
	if !mr.FoundAnything {
		return fmt.Errorf("should have found %d files in '%s' but didn't",
			len(expectedfiles), basedir)
	}
	if mr.GetElements(&sr) != nil {
		return fmt.Errorf("failed to retrieve search results")
	}
	if len(expectedfiles) == 1 && expectedfiles[0] == "" {
		// should not have found anything to succeed
		if len(sr["s1"]) != 1 {
			return fmt.Errorf("expected to find nothing but found %d files",
				len(sr["s1"]))
		} else if sr["s1"][0].File != "" {
			return fmt.Errorf("expected to find nothing but found file '%s'",
				sr["s1"][0].File)
		}
	}
	if len(sr["s1"]) != len(expectedfiles) {
		if len(sr["s1"]) == 1 && sr["s1"][0].File == "" {
			return fmt.Errorf("expected to find %d files but found nothing",
				len(expectedfiles))
		}
		return fmt.Errorf("expected to find %d files but found %d",
			len(expectedfiles), len(sr["s1"]))
	}
	for _, found := range sr["s1"] {
		for i, expectedfile := range expectedfiles {
			if filepath.Clean(found.File) == filepath.Clean(expectedfile) {
				// good result, remove expected file from list of expected files
				expectedfiles = expectedfiles[:i+copy(expectedfiles[i:], expectedfiles[i+1:])]
			}
		}
	}
	if len(expectedfiles) != 0 {
		return fmt.Errorf("did not find %d files: %s", len(expectedfiles), expectedfiles)
	}
	return nil
}

func createFiles() (basedir string) {
	basedir = os.TempDir() + "/migfiletest" + time.Now().Format("15-04-05.99999999")
	err := os.MkdirAll(basedir+subdirs, 0700)
	if err != nil {
		log.Fatalf("failed to create test directories %s%s: %v\n",
			basedir, subdirs, err)
	}
	for _, dir := range []string{basedir, basedir + subdirs} {
		for i, tp := range TESTDATA {
			fd, err := os.Create(fmt.Sprintf("%s/testfile%d", dir, i))
			if err != nil {
				log.Fatalf("failed to create testfile1: %v\n", err)
			}
			os.Chmod(fmt.Sprintf("%s/testfile%d", dir, i), 0644)
			n, err := fd.Write(tp.data)
			if err != nil {
				log.Fatalf("failed to write content to %s: %v\n", fd.Name(), err)
			}
			if n != len(tp.data) {
				log.Fatalf("wrote %d bytes when content had %d\n", n, len(tp.data))
			}
			fd.Close()
		}
	}
	return
}

const subdirs string = `/a/b/c/d/e/f/g/h/i/j/k/l/m/n/`

type testParams struct {
	data []byte
	name, size, mode, mtime, content,
	md5, sha1, sha256, sha384, sha512,
	sha3_224, sha3_256, sha3_384, sha3_512 string
}

var TESTDATA = []testParams{
	testParams{
		data: []byte(`--- header for first file ---
# this is a comment
                                       
# above is an line filled with spaces

# above is an empty line, no spaces
some text
some other text`),
		name:     `testfile0`,
		size:     `190`,
		mode:     `-rw-r--r--`,
		mtime:    `<1m`,
		content:  `^--- header for first file ---$`,
		md5:      `e499c1912bd9af4f7e8ccaf27f7b04d2`,
		sha1:     `d7bbc3dd7adf6e347c93a4c8b9bfb8ef4748c0fb`,
		sha256:   `4d8ef27c4415d71cbbfad1eaa97d6f2a3ddacc9708b66efbb726133b9fd3d79a`,
		sha384:   `8bf7ca66a8cd73b252e1431e350ef415034b211ea4d7711189b0b3f664c6fd372ed4a8f454ffc7e577a828a97a30074b`,
		sha512:   `bd6e6a312a5fe4998df5d6ace15837355e1465ed3d32188ec56551279f70b51cf168e5c83d1f60bf66c15b70c0b2e51b4a728f3a0046d46db9a9e566c2db3daf`,
		sha3_224: `a7ba1e66174848ecea143b612f22168b006979e3827e09f0ae6395e8`,
		sha3_256: `091dbb7c04406fb5d95dc1c3c1fbc0378a63f19472f42fdd133b826a2a5ea3a7`,
		sha3_384: `5b33c1fff06dff46b62b89922dfbab786a7763601028a741b7d7f1c75b584ae88acaf07f672bd4902929e7168fd9de28`,
		sha3_512: `c9cf248748858b3b1ea752f9c778889a9cf0abc23529da20147b9ffbd7254a82d949c85a399730b40b3603bb2bc41b9585de147d2cd7080938388615501c4a5e`,
	},
	testParams{
		data: []byte(`--- header for second file ---
# this is a comment
                                       
# above is an line filled with spaces

# above is an empty line, no spaces
some text
some other other text`),
		name:     `testfile1`,
		size:     `197`,
		mode:     `-rw-r--r--`,
		mtime:    `<1m`,
		content:  `^--- header for second file ---$`,
		md5:      `63c7fa8ec03e72343d434835ff95c8a7`,
		sha1:     `14dcc657c3362bc9adb12ff8c23e14940df42b6f`,
		sha256:   `b665fabb0c6c5cd9fabfd3fdd222aa4cd56dceda82485acc263546d30a825634`,
		sha384:   `fdd9460795c000f9143e5bdd8d7ffb153f7541c154682179a131f557fa0a878db51f0046672e486a9bdcb64cdaf76ca1`,
		sha512:   `e40b2f00f2a4097b3f53bc33c60cd04750ce87016ec3c6ef05bea05f0c5f49c56f7d634448012b2bbb879c2ede43d5bd3bc0ce20873129c2caad9cb4d8bbe6da`,
		sha3_224: `bae8d23a49eb7ac8c5c8589e6d089d4b127478132711d164d92ad244`,
		sha3_256: `92d0f8878baff9ff926bb752de4e830d60ef05146be90e0b857a58402940f839`,
		sha3_384: `f8b736cdc7e14afb264bafb287805a2d05397142cabe3a8d1b17c13f6b5bf62006b413814fdb7d04cd63ebe7a8c59542`,
		sha3_512: `c501a1809064bf480b6260c0af7430e81547a854a41ce900707134210123db4ddfefd58f73a41b3072cef0a034b39d8d4ce01265d3ce30d0bf11e0ea26ec2dbd`,
	},
	testParams{
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
		name:     `testfile2`,
		size:     `1024`,
		mode:     `-rw-r--r--`,
		mtime:    `<1m`,
		content:  `skZ0`,
		md5:      `8d3a7afb7e59693b383d52396243a5b8`,
		sha1:     `d82bc1145d471714b056940b268032f9ab0df2ae`,
		sha256:   `3b495fae5bae9751ea4706c29e992002ba277bce30bd83a827b01ba977eabc2f`,
		sha384:   `e778dda037764db51a4aaaf1511f8415aa9e6b5f9e012d1fef4cfe5492bf11410cb37a5db2acf3580460a265bd0ace2e`,
		sha512:   `36d988e223f086c95d45c804f3d4b0ab95e74b69c36d5bc8801dcd9d71c0e252e4987d8e2bcab348811e559c454bd9e18527fd66c3b0be1d53463c5d7a80e9f2`,
		sha3_224: `fdb23afa808c265284c3199013e4ded9704eebf54ffdc1f016dacc12`,
		sha3_256: `bb84ecae0ebff542bef1478e4f19523c910905a88669abb38fe86f8b1b1cc7a8`,
		sha3_384: `5053ccfd9cc72aead52742ea89ef4ab87c7e8fac92d09983d6ea0b43d8f1e247338c6460a66a7e5f53293888b82e2720`,
		sha3_512: `674b6d6b4868e7bf848c4ce9be4fa964e3907a78c82152dd7f009778015043810e0e6fd75f58fb4a706893f22f70cabab449ebde37b88cb645675c3df16ea347`,
	},
}
