// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"fmt"
	"regexp"
)

// HasLine is used to perform tests against whether or not a file contains a given
// regular expression
type HasLine struct {
	Path       string `json:"path,omitempty" yaml:"path,omitempty"`
	File       string `json:"file,omitempty" yaml:"file,omitempty"`
	Expression string `json:"expression,omitempty" yaml:"expression,omitempty"`

	matches []haslineStatus
}

type haslineStatus struct {
	path  string
	found bool
}

func (h *HasLine) validate(d *Document) error {
	if len(h.Path) == 0 {
		return fmt.Errorf("hasline path must be set")
	}
	if len(h.File) == 0 {
		return fmt.Errorf("hasline file must be set")
	}
	_, err := regexp.Compile(h.File)
	if err != nil {
		return err
	}
	if len(h.Expression) == 0 {
		return fmt.Errorf("hasline expression must be set")
	}
	_, err = regexp.Compile(h.Expression)
	if err != nil {
		return err
	}
	return nil
}

func (h *HasLine) mergeCriteria(c []evaluationCriteria) {
}

func (h *HasLine) fireChains(d *Document) ([]evaluationCriteria, error) {
	return nil, nil
}

func (h *HasLine) isChain() bool {
	return false
}

func (h *HasLine) expandVariables(v []Variable) {
	h.Path = variableExpansion(v, h.Path)
	h.File = variableExpansion(v, h.File)
}

func (h *HasLine) getCriteria() (ret []evaluationCriteria) {
	for _, x := range h.matches {
		n := evaluationCriteria{}
		n.identifier = x.path
		n.testValue = fmt.Sprintf("%v", x.found)
		ret = append(ret, n)
	}
	return ret
}

func (h *HasLine) prepare() error {
	debugPrint("prepare(): analyzing file system, path %v, file \"%v\"\n", h.Path, h.File)

	sfl := newSimpleFileLocator()
	sfl.root = h.Path
	err := sfl.locate(h.File, true)
	if err != nil {
		return err
	}

	for _, x := range sfl.matches {
		m, err := fileContentCheck(x, h.Expression)
		// XXX These soft errors during preparation are ignored right
		// now, but they should probably be tracked somewhere.
		if err != nil {
			continue
		}
		ncm := haslineStatus{}
		ncm.path = x
		if m == nil || len(m) == 0 {
			debugPrint("prepare(): content not found in \"%v\"\n", x)
			ncm.found = false
		} else {
			debugPrint("prepare(): content found in \"%v\"\n", x)
			ncm.found = true
		}
		h.matches = append(h.matches, ncm)
	}

	return nil
}
