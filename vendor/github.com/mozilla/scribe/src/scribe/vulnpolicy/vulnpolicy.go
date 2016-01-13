// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package vulnpolicy

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"scribe"
)

type Policy struct {
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

// This structure represents intermediate vulnerability information that can
// be converted into scribe tests for execution.
//
// The ID value should be unique to a given issue, and consistent in the sense
// if the policy is recreated from the same source data, the ID should remain
// the same.
type Vulnerability struct {
	OS       string   `json:"os,omitempty"`
	Release  string   `json:"release,omitempty"`
	Package  string   `json:"package,omitempty"`
	Version  string   `json:"version,omitempty"`
	ID       string   `json:"id,omitempty"` // Globally unique test identifier
	Metadata Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
	Description string   `json:"description"`
	CVE         []string `json:"cve"`
	CVSS        string   `json:"cvss"`
	Category    string   `json:"category"`
}

var testcntr int

func getReleaseTest(doc *scribe.Document, vuln Vulnerability) (string, error) {
	if vuln.OS == "ubuntu" {
		return ubuntuGetReleaseTest(doc, vuln)
	} else if (vuln.OS == "redhat") || (vuln.OS == "centos") {
		return redhatGetReleaseTest(doc, vuln)
	}
	return "", fmt.Errorf("unable to create release definition")
}

func getReleasePackage(vuln Vulnerability) (string, string) {
	if vuln.OS == "ubuntu" {
		return ubuntuGetReleasePackage(vuln)
	}
	return vuln.Package, ""
}

func getTestID(vuln Vulnerability) (string, error) {
	if len(vuln.ID) == 0 || len(vuln.OS) == 0 ||
		len(vuln.Release) == 0 {
		return "", fmt.Errorf("test for %v missing ID, OS, or Release", vuln.Package)
	}
	h := md5.New()
	h.Write([]byte(vuln.ID))
	h.Write([]byte(vuln.OS))
	h.Write([]byte(vuln.Release))
	h.Write([]byte(vuln.Package))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func addTest(doc *scribe.Document, vuln Vulnerability) error {
	// Get the release definition for the test, if it's missing from
	// the document it will be added
	reltestid, err := getReleaseTest(doc, vuln)
	if err != nil {
		return err
	}

	// See if we already have an object definition for the package, if
	// not add it
	objid := ""
	for _, x := range doc.Objects {
		if x.Package.Name == vuln.Package {
			objid = x.Object
			break
		}
	}
	if objid == "" {
		objid = fmt.Sprintf("obj-package-%v", vuln.Package)
		obj := scribe.Object{}
		obj.Object = objid
		obj.Package.Name, obj.Package.CollectMatch = getReleasePackage(vuln)
		doc.Objects = append(doc.Objects, obj)
	}

	test := scribe.Test{}
	testidstr, err := getTestID(vuln)
	if err != nil {
		return err
	}
	// Build a more descriptive name for this test to override the test ID
	// in command output
	test.TestName = fmt.Sprintf("test-%v-%v-%v-%v", vuln.OS, vuln.Release, vuln.Package, testcntr)
	test.TestID = testidstr
	test.Description = vuln.Metadata.Description
	test.Object = objid
	test.EVR.Value = vuln.Version
	test.EVR.Operation = "<"
	test.If = append(test.If, reltestid)
	// Include all listed CVEs as a tag in the test
	cvelist := scribe.TestTag{Key: "cve"}
	var cveval string
	for _, x := range vuln.Metadata.CVE {
		if cveval != "" {
			cveval += ","
		}
		cveval += x
	}
	cvelist.Value = cveval
	test.Tags = append(test.Tags, cvelist)
	// Include CVSS if available
	if vuln.Metadata.CVSS != "" {
		test.Tags = append(test.Tags, scribe.TestTag{Key: "cvss", Value: vuln.Metadata.CVSS})
	}
	if vuln.Metadata.Category != "" {
		test.Tags = append(test.Tags, scribe.TestTag{Key: "category", Value: vuln.Metadata.Category})
	}
	doc.Tests = append(doc.Tests, test)
	testcntr++

	return nil
}

func DocumentFromPolicy(buf []byte) (ret scribe.Document, err error) {
	policy := Policy{}
	err = json.Unmarshal(buf, &policy)
	if err != nil {
		return
	}

	// Create a test for each vulnerability that is listed in the
	// policy. Create depedency release tests in the document as
	// well as we go.
	testcntr = 0
	for _, x := range policy.Vulnerabilities {
		err = addTest(&ret, x)
		if err != nil {
			return
		}
	}
	return
}
