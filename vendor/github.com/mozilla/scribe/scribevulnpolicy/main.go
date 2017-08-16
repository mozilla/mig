// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/mozilla/scribe"
	"os"
)

const (
	platformCentos6 = iota
	platformCentos7
)

// Our list of platforms we will support policy generation for, this maps the
// platform constants to clair namespace identifiers
type supportedPlatform struct {
	platformID       int
	name             string
	clairNamespace   string
	clairNamespaceID int // Populated upon query of the database
	releaseTest      func(supportedPlatform, *scribe.Document) (string, error)
	pkgNewest        func(string) bool
}

var supportedPlatforms = []supportedPlatform{
	{platformCentos6, "centos6", "centos:6", 0, centosReleaseTest, centosOnlyNewest},
	{platformCentos7, "centos7", "centos:7", 0, centosReleaseTest, centosOnlyNewest},
}

// Given a clair namespace, return the supportedPlatform entry for it if it is
// supported, otherwise return an error
func getPlatform(clairNamespace string) (ret supportedPlatform, err error) {
	for _, x := range supportedPlatforms {
		if clairNamespace == x.clairNamespace {
			ret = x
			return
		}
	}
	err = fmt.Errorf("platform %v not supported", clairNamespace)
	return
}

type vuln struct {
	name           string
	fixedInVersion string
	pkgName        string
	severity       string
	link           string
	description    string
}

// Describes global configuration
type config struct {
	Database struct {
		DBName     string
		DBHost     string
		DBUser     string
		DBPassword string
		DBPort     string
	}
}

var dbconn *sql.DB
var cfg config

// Set any configuration based on environment variables
func configFromEnv() error {
	envvar := os.Getenv("PGHOST")
	if envvar != "" {
		cfg.Database.DBHost = envvar
	}
	envvar = os.Getenv("PGUSER")
	if envvar != "" {
		cfg.Database.DBUser = envvar
	}
	envvar = os.Getenv("PGPASSWORD")
	if envvar != "" {
		cfg.Database.DBPassword = envvar
	}
	envvar = os.Getenv("PGDATABASE")
	if envvar != "" {
		cfg.Database.DBName = envvar
	}
	envvar = os.Getenv("PGPORT")
	if envvar != "" {
		cfg.Database.DBPort = envvar
	}
	return nil
}

func dbInit() (err error) {
	connstr := fmt.Sprintf("dbname=%v host=%v user=%v password=%v port=%v sslmode=disable",
		cfg.Database.DBName, cfg.Database.DBHost, cfg.Database.DBUser,
		cfg.Database.DBPassword, cfg.Database.DBPort)
	dbconn, err = sql.Open("postgres", connstr)
	return
}

// Generate a test identifier; this needs to be unique in the document. Here we
// just use a few elements from the vulnerability and platform and return an MD5
// digest.
func generateTestID(v vuln, p supportedPlatform) (string, error) {
	h := md5.New()
	h.Write([]byte(v.name))
	h.Write([]byte(p.name))
	h.Write([]byte(v.pkgName))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func generatePolicy(p string) error {
	var (
		platform supportedPlatform
		doc      scribe.Document
	)
	// First make sure this is a supported platform, and this will also get us the namespace ID
	platforms, err := dbGetSupportedPlatforms()
	if err != nil {
		return err
	}
	supported := false
	for _, x := range platforms {
		if x.name == p {
			supported = true
			platform = x
			break
		}
	}
	if !supported {
		return fmt.Errorf("platform %v not supported for policy generation", p)
	}

	// Add the release test which will be used as a dependency on all checks
	// in the final test document
	reltestid, err := platform.releaseTest(platform, &doc)
	if err != nil {
		return err
	}

	// Get all vulnerabilities for the platform from the database
	vulns, err := dbVulnsForPlatform(platform)
	if err != nil {
		return err
	}

	// Add a test for each vulnerability
	for _, x := range vulns {
		var (
			newtest scribe.Test
			newobj  scribe.Object
			objname string
		)

		// See if we already have an object in the document that references
		// the package we want to lookup, if so we don't need to add a second
		// one
		found := false
		objname = fmt.Sprintf("obj-package-%v", x.pkgName)
		for _, y := range doc.Objects {
			if y.Package.Name == x.pkgName {
				found = true
				break
			}
		}
		if !found {
			newobj.Object = objname
			newobj.Package.Name = x.pkgName
			newobj.Package.OnlyNewest = platform.pkgNewest(x.pkgName)
			doc.Objects = append(doc.Objects, newobj)
		}

		newtest.TestName = x.name
		newtest.Object = objname
		newtest.EVR.Value = x.fixedInVersion
		newtest.EVR.Operation = "<"
		newtest.If = append(newtest.If, reltestid)
		newtest.TestID, err = generateTestID(x, platform)
		if err != nil {
			return err
		}
		// Add some tags to the test we can use when we parse results
		pkgtag := scribe.TestTag{Key: "package", Value: x.pkgName}
		newtest.Tags = append(newtest.Tags, pkgtag)
		sevtag := scribe.TestTag{Key: "severity", Value: x.severity}
		newtest.Tags = append(newtest.Tags, sevtag)
		linktag := scribe.TestTag{Key: "link", Value: x.link}
		newtest.Tags = append(newtest.Tags, linktag)
		doc.Tests = append(doc.Tests, newtest)
	}

	// Finally, display the policy on stdout
	outbuf, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", string(outbuf))

	return nil
}

func main() {
	var (
		genPlatform  string
		showVersions bool
		err          error
	)
	flag.BoolVar(&showVersions, "V", false, "show distributions we can generate policies for and exit")
	flag.Parse()
	if len(flag.Args()) >= 1 {
		genPlatform = flag.Args()[0]
	}

	err = configFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config from environment: %v\n", err)
		os.Exit(1)
	}
	err = dbInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing database: %v\n", err)
		os.Exit(1)
	}

	if showVersions {
		platforms, err := dbGetSupportedPlatforms()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error retrieving platforms: %v\n", err)
			os.Exit(1)
		}
		for _, x := range platforms {
			fmt.Printf("%v\n", x.name)
		}
		os.Exit(0)
	}

	if genPlatform == "" {
		fmt.Fprintf(os.Stderr, "error: platform to generate policy for not specified\n")
		os.Exit(1)
	}
	err = generatePolicy(genPlatform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating policy: %v\n", err)
	}
}
