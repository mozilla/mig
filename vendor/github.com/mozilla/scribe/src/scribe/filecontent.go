// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

type FileContent struct {
	Path       string `json:"path,omitempty"`
	File       string `json:"file,omitempty"`
	Expression string `json:"expression,omitempty"`
	Concat     string `json:"concat,omitempty"`

	ImportChain []string `json:"import-chain,omitempty"`

	matches []contentMatch
}

type contentMatch struct {
	path    string
	matches []matchLine
}

type matchLine struct {
	fullmatch string
	groups    []string
}

func (f *FileContent) validate(d *Document) error {
	if len(f.Path) == 0 {
		return fmt.Errorf("filecontent path must be set")
	}
	if len(f.File) == 0 {
		return fmt.Errorf("filecontent file must be set")
	}
	_, err := regexp.Compile(f.File)
	if err != nil {
		return err
	}
	if len(f.Expression) == 0 {
		return fmt.Errorf("filecontent expression must be set")
	}
	_, err = regexp.Compile(f.Expression)
	if err != nil {
		return err
	}
	err = validateChains(f.ImportChain, d)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileContent) fireChains(d *Document) ([]evaluationCriteria, error) {
	if len(f.ImportChain) == 0 {
		return nil, nil
	}
	debugPrint("fireChains(): firing chains for filecontent object\n")
	uids := make([]string, 0)
	for _, x := range f.matches {
		found := false
		for _, y := range uids {
			if x.path == y {
				found = true
				break
			}
		}
		if found {
			continue
		}
		uids = append(uids, x.path)
	}
	ret := make([]evaluationCriteria, 0)
	for _, x := range uids {
		varlist := make([]Variable, 0)
		debugPrint("fireChains(): run for \"%v\"\n", x)

		// Build our variable list for the filecontent chain import.
		dirent, _ := path.Split(x)
		newvar := Variable{Key: "chain_root", Value: dirent}
		varlist = append(varlist, newvar)

		// Execute each chain entry in order for each identifier.
		for _, y := range f.ImportChain {
			oc, _ := d.getObjectInterfaceCopy(y)
			oc.expandVariables(varlist)
			err := oc.prepare()
			if err != nil {
				return nil, err
			}
			criteria, err := oc.fireChains(d)
			if err != nil {
				return nil, err
			}
			if criteria != nil {
				oc.mergeCriteria(criteria)
			}

			// Extract the criteria. Rewrite the identifier based
			// on what identifier was used for the chain.
			excri := oc.getCriteria()
			for _, z := range excri {
				z.identifier = x
				ret = append(ret, z)
			}
		}
	}
	return ret, nil
}

func (f *FileContent) mergeCriteria(c []evaluationCriteria) {
	for _, x := range c {
		nml := matchLine{}
		nml.groups = make([]string, 0)
		nml.groups = append(nml.groups, x.testValue)
		ncm := contentMatch{}
		ncm.path = x.identifier
		ncm.matches = append(ncm.matches, nml)
		f.matches = append(f.matches, ncm)
	}
}

func (f *FileContent) isChain() bool {
	if hasChainVariables(f.Path) {
		return true
	}
	return false
}

func (f *FileContent) expandVariables(v []Variable) {
	f.Path = variableExpansion(v, f.Path)
	f.File = variableExpansion(v, f.File)
}

func (f *FileContent) getCriteria() (ret []evaluationCriteria) {
	for _, x := range f.matches {
		for _, y := range x.matches {
			for _, z := range y.groups {
				n := evaluationCriteria{}
				n.identifier = x.path
				n.testValue = z
				ret = append(ret, n)
			}
		}
	}
	if len(f.Concat) != 0 {
		return criteriaConcat(ret, f.Concat)
	}
	return ret
}

func (f *FileContent) prepare() error {
	debugPrint("prepare(): analyzing file system, path %v, file \"%v\"\n", f.Path, f.File)

	sfl := newSimpleFileLocator()
	sfl.root = f.Path
	err := sfl.locate(f.File, true)
	if err != nil {
		return err
	}

	for _, x := range sfl.matches {
		m, err := fileContentCheck(x, f.Expression)
		// XXX These soft errors during preparation are ignored right
		// now, but they should probably be tracked somewhere.
		if err != nil {
			continue
		}
		if m == nil || len(m) == 0 {
			continue
		}

		ncm := contentMatch{}
		ncm.path = x
		ncm.matches = m
		f.matches = append(f.matches, ncm)
		debugPrint("prepare(): content matches in %v\n", ncm.path)
		for _, i := range ncm.matches {
			debugPrint("prepare(): full match: \"%v\"\n", i.fullmatch)
			for j := range i.groups {
				debugPrint("prepare(): group %v: \"%v\"\n", j, i.groups[j])
			}
		}
	}

	return nil
}

type simpleFileLocator struct {
	executed bool
	root     string
	curDepth int
	maxDepth int
	matches  []string
	locator  func(string, bool, string, int) ([]string, error)
}

func newSimpleFileLocator() (ret simpleFileLocator) {
	// XXX This needs to be fixed to work with Windows.
	ret.root = "/"
	ret.maxDepth = 10
	ret.matches = make([]string, 0)
	if sRuntime.fileLocator != nil {
		ret.locator = sRuntime.fileLocator
	}
	return ret
}

func (s *simpleFileLocator) locate(target string, useRegexp bool) error {
	if s.executed {
		return fmt.Errorf("locator has already been executed")
	}
	s.executed = true
	if s.locator != nil {
		buf, err := s.locator(target, useRegexp, s.root, s.maxDepth)
		if err != nil {
			return err
		}
		s.matches = buf
		return nil
	}
	return s.locateInner(target, useRegexp, "")
}

func (s *simpleFileLocator) symFollowIsRegular(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if fi.Mode().IsRegular() {
		return true, nil
	}
	return false, nil
}

func (s *simpleFileLocator) locateInner(target string, useRegexp bool, path string) error {
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
			err = s.locateInner(target, useRegexp, fname)
			if err != nil {
				return err
			}
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
		} else if (x.Mode() & os.ModeSymlink) > 0 {
			isregsym, err := s.symFollowIsRegular(fname)
			if err != nil {
				// Ignore these errors and continue searching
				return nil
			}
			if isregsym {
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
	}
	return nil
}

func fileContentCheck(path string, regex string) ([]matchLine, error) {
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
	ret := make([]matchLine, 0)
	for {
		// XXX Ignore potential partial reads (prefix) here, for lines
		// with excessive length we will just treat it as multiple
		// lines
		buf, _, err := rdr.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
		ln := string(buf)
		mtch := re.FindStringSubmatch(ln)
		if len(mtch) > 0 {
			newmatch := matchLine{}
			newmatch.groups = make([]string, 0)
			newmatch.fullmatch = mtch[0]
			for i := 1; i < len(mtch); i++ {
				newmatch.groups = append(newmatch.groups, mtch[i])
			}
			ret = append(ret, newmatch)
		}
	}

	if len(ret) == 0 {
		return nil, nil
	}
	return ret, nil
}
