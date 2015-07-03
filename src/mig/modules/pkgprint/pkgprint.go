// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package pkgprint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mig/modules"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

func init() {
	modules.Register("pkgprint", func() interface{} {
		return new(Runner)
	})
}

type FPResult struct {
	Name    string       `json:"name"`
	Matches []MatchGroup `json:"matches"`
}

type MatchGroup struct {
	Root    string       `json:"root"`
	Entries []MatchEntry `json:"entries"`
}

type MatchEntry struct {
	Match string `json:"match"`
}

func executeTemplate(tname string, depth int, root string) (FPResult, error) {
	var (
		res FPResult
		err error
	)
	fp := getTemplateFingerprint(tname)
	if fp == nil {
		return res, fmt.Errorf("invalid template name specified")
	}
	res.Name = tname
	s := NewSimpleFileLocator()
	s.maxDepth = depth
	if root != "/" {
		s.root = root
	}

	// If the fingerprint has a forced root, override the root that has
	// been specified.
	if fp.forceRoot != "" {
		s.root = fp.forceRoot
	}

	s.Locate(fp.filename, fp.isRegexp)
	for _, x := range s.matches {
		// If a path filter exists and does not match the file, ignore
		// it.
		if fp.pathFilter != "" {
			var flag bool
			flag, err = regexp.MatchString(fp.pathFilter, x)
			if err != nil || !flag {
				continue
			}
		}

		// If an error occurs here, just ignore it and keep going.
		m, _ := FileContentCheck(x, fp.contentMatch)
		if m == nil || len(m) == 0 {
			continue
		}
		for _, i := range m {
			// The module requires a substring match in the
			// template, so we only process entries that have
			// resulted in substrings being returned.
			if len(i) < 2 {
				continue
			}
			newmatch := MatchGroup{}
			newmatch.Root = x
			for j := 1; j < len(i); j++ {
				nme := MatchEntry{}
				nme.Match, err = fp.transform(i[j])
				if err != nil {
					continue
				}
				newmatch.Entries = append(newmatch.Entries, nme)
			}
			res.Matches = append(res.Matches, newmatch)
		}
	}
	return res, nil
}

// Given a file, identify any lines in the file that match the supplied
// regular expression. Will return a slice of string slices. If the
// supplied regular expression contains substring matches, each string
// slice will contain the full line matched, and any substrings, otherwise
// the slice will only contain the line matched.
func FileContentCheck(path string, regex string) ([][]string, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		fd.Close()
	}()

	rdr := bufio.NewReader(fd)
	ret := make([][]string, 0)
	for {
		ln, err := rdr.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		mtch := re.FindStringSubmatch(ln)
		if len(mtch) > 0 {
			newmatch := make([]string, 0)
			newmatch = append(newmatch, mtch[0])
			for i := 1; i < len(mtch); i++ {
				newmatch = append(newmatch, mtch[i])
			}
			ret = append(ret, newmatch)
		}
	}

	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}

type SimpleFileLocator struct {
	executed bool
	root     string
	curDepth int
	maxDepth int
	matches  []string
}

func NewSimpleFileLocator() (ret SimpleFileLocator) {
	ret.root = "/"
	ret.maxDepth = 10
	ret.matches = make([]string, 0)
	return ret
}

func (s *SimpleFileLocator) Locate(target string, useRegexp bool) error {
	if s.executed {
		return fmt.Errorf("locator has already been executed")
	}
	s.executed = true
	return s.locateInner(target, useRegexp, "")
}

func (s *SimpleFileLocator) locateInner(target string, useRegexp bool, path string) error {
	var (
		spath string
		re    *regexp.Regexp
		err   error
	)

	// If processing this directory would result in us exceeding the
	// specified search depth, just ignore it.
	if (s.curDepth + 1) > s.maxDepth {
		return nil
	}

	if useRegexp {
		re, err = regexp.Compile(target)
		if err != nil {
			return err
		}
	}

	s.curDepth++
	defer func() {
		s.curDepth--
	}()

	if path == "" {
		spath = s.root
	} else {
		spath = path
	}
	dirents, err := ioutil.ReadDir(spath)
	if err != nil {
		// If we encounter an error while reading a directory, just
		// ignore it and keep going until we are finished.
		return nil
	}
	for _, x := range dirents {
		fname := filepath.Join(spath, x.Name())
		if x.IsDir() {
			s.locateInner(target, useRegexp, fname)
		} else if x.Mode().IsRegular() {
			if !useRegexp {
				if x.Name() == target {
					s.matches = append(s.matches, fname)
				}
			} else {
				if re.MatchString(x.Name()) {
					s.matches = append(s.matches, fname)
				}
			}
		}
	}
	return nil
}

func buildResults(fpres FPResult, r *modules.Result) (buf []byte, err error) {
	r.Success = true
	if len(fpres.Matches) > 0 {
		r.FoundAnything = true
	}
	r.Elements = fpres
	buf, err = json.Marshal(r)
	return
}

type Runner struct {
	Parameters Parameters
	Results    modules.Result
}

func (r Runner) Run(in io.Reader) (resStr string) {
	defer func() {
		if e := recover(); e != nil {
			r.Results.Errors = append(r.Results.Errors, fmt.Sprintf("%v", e))
			r.Results.Success = false
			err, _ := json.Marshal(r.Results)
			resStr = string(err)
			return
		}
	}()

	runtime.GOMAXPROCS(1)

	err := modules.ReadInputParameters(in, &r.Parameters)
	if err != nil {
		panic(err)
	}

	err = r.ValidateParameters()
	if err != nil {
		panic(err)
	}

	if r.Parameters.TemplateMode {
		fpres, err := executeTemplate(r.Parameters.TemplateParams.Name, r.Parameters.SearchDepth, r.Parameters.SearchRoot)
		if err != nil {
			panic(err)
		}
		buf, err := buildResults(fpres, &r.Results)
		if err != nil {
			panic(err)
		}
		resStr = string(buf)
		return
	}

	panic("no function specified")
	return
}

func (r Runner) ValidateParameters() (err error) {
	p := r.Parameters
	if !p.TemplateMode {
		return fmt.Errorf("currently template mode must be enabled")
	}

	// If template mode is in use, make sure a valid template has been
	// specified.
	if fp := getTemplateFingerprint(p.TemplateParams.Name); fp == nil {
		return fmt.Errorf("invalid template %v", p.TemplateParams.Name)
	}

	if p.SearchDepth <= 0 {
		return fmt.Errorf("invalid search depth")
	}

	return
}

type Parameters struct {
	// Enable or disable template mode in the module. When using template
	// mode, hardcoded fingerprints are used. In the future, this will
	// support being set to false to allow the client to supply more
	// advanced parameters allowing for more flexible scanning.
	//
	// To do this we need to figure out a way to be able to specify
	// flexible fingerprints from the mig command line side, but prevent
	// the use of fingerprints that could return data from more sensitive
	// areas of the file system (e.g., /etc/shadow, etc).
	TemplateMode   bool           `json:"templatemode"`
	TemplateParams TemplateParams `json:"template"`

	SearchDepth int    `json:"depth"`
	SearchRoot  string `json:"root"`
}

type TemplateParams struct {
	Name string `json:"name"`
}

func newParameters() *Parameters {
	return &Parameters{SearchDepth: 10}
}

func (r Runner) PrintResults(result modules.Result, foundOnly bool) (prints []string, err error) {
	var elem FPResult

	err = result.GetElements(&elem)
	if err != nil {
		panic(err)
	}

	for _, x := range elem.Matches {
		for _, y := range x.Entries {
			resStr := fmt.Sprintf("pkgprint name=%v root=%v entry=%v", elem.Name, x.Root, y.Match)
			prints = append(prints, resStr)
		}
	}
	if !foundOnly {
		for _, we := range result.Errors {
			prints = append(prints, we)
		}
	}

	return
}
