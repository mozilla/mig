// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"fmt"
	"path"
	"regexp"
)

type filename struct {
	Path string `json:"path"`
	File string `json:"file"`

	matches []nameMatch
}

type nameMatch struct {
	path  string
	match string
}

func (f *filename) isChain() bool {
	return false
}

func (f *filename) fireChains(d *Document) ([]evaluationCriteria, error) {
	return nil, nil
}

func (f *filename) mergeCriteria(c []evaluationCriteria) {
}

func (f *filename) validate(d *Document) error {
	if len(f.Path) == 0 {
		return fmt.Errorf("filename path must be set")
	}
	if len(f.File) == 0 {
		return fmt.Errorf("filename file must be set")
	}
	return nil
}

func (f *filename) expandVariables(v []variable) {
	f.Path = variableExpansion(v, f.Path)
}

func (f *filename) getCriteria() (ret []evaluationCriteria) {
	for _, x := range f.matches {
		n := evaluationCriteria{}
		n.identifier = x.path
		n.testValue = x.match
		ret = append(ret, n)
	}
	return ret
}

func (f *filename) prepare() error {
	debugPrint("prepare(): analyzing file system, path %v, file \"%v\"\n", f.Path, f.File)

	sfl := newSimpleFileLocator()
	sfl.root = f.Path
	err := sfl.locate(f.File, true)
	if err != nil {
		return err
	}

	re, err := regexp.Compile(f.File)
	if err != nil {
		return err
	}

	for _, x := range sfl.matches {
		_, testFilename := path.Split(x)
		mtch := re.FindStringSubmatch(testFilename)
		if len(mtch) < 2 {
			continue
		}
		nnm := nameMatch{}
		nnm.path = x
		nnm.match = mtch[1]
		f.matches = append(f.matches, nnm)
	}

	return nil
}
