// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package scribe

import (
	"fmt"
)

type object struct {
	Object      string      `json:"object"`
	FileContent filecontent `json:"filecontent"`
	FileName    filename    `json:"filename"`
	Package     pkg         `json:"package"`
	Raw         raw         `json:"raw"`
	HasLine     hasline     `json:"hasline"`

	isChain  bool  // True if object is part of an import chain.
	prepared bool  // True if object has been prepared.
	err      error // The last error condition encountered during preparation.
}

type genericSource interface {
	prepare() error
	getCriteria() []evaluationCriteria
	isChain() bool
	expandVariables([]variable)
	validate(d *Document) error
	mergeCriteria([]evaluationCriteria)
	fireChains(*Document) ([]evaluationCriteria, error)
}

func (o *object) validate(d *Document) error {
	if len(o.Object) == 0 {
		return fmt.Errorf("an object in document has no identifier")
	}
	si := o.getSourceInterface()
	if si == nil {
		return fmt.Errorf("%v: no valid source interface", o.Object)
	}
	err := si.validate(d)
	if err != nil {
		return fmt.Errorf("%v: %v", o.Object, err)
	}
	return nil
}

func (o *object) markChain() {
	o.isChain = o.getSourceInterface().isChain()
}

func (o *object) getSourceInterface() genericSource {
	if o.Package.Name != "" {
		return &o.Package
	} else if o.FileContent.Path != "" {
		return &o.FileContent
	} else if o.FileName.Path != "" {
		return &o.FileName
	} else if len(o.Raw.Identifiers) > 0 {
		return &o.Raw
	} else if o.HasLine.Path != "" {
		return &o.HasLine
	}
	return nil
}

func (o *object) fireChains(d *Document) error {
	si := o.getSourceInterface()
	// We only fire chains on root object types, not on chain entries
	// themselves.
	if si.isChain() {
		return nil
	}
	// If the object already has encountered an error, don't bother
	// trying to execute chain entries for it.
	if o.err != nil {
		debugPrint("fireChains(): skipping failed object \"%v\"\n", o.Object)
		return nil
	}
	criteria, err := si.fireChains(d)
	if err != nil {
		o.err = err
		return err
	}
	if criteria != nil {
		si.mergeCriteria(criteria)
	}
	return nil
}

func (o *object) prepare(d *Document) error {
	if o.isChain {
		debugPrint("prepare(): skipping chain object \"%v\"\n", o.Object)
		return nil
	}
	if o.prepared {
		return nil
	}
	o.prepared = true

	p := o.getSourceInterface()
	if p == nil {
		o.err = fmt.Errorf("object has no valid interface")
		return o.err
	}
	p.expandVariables(d.Variables)
	err := p.prepare()
	if err != nil {
		o.err = err
		return err
	}
	return nil
}
