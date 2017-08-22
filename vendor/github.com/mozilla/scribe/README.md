# scribe

scribe is a host policy evaluator written in Go.

[![Build Status](https://travis-ci.org/mozilla/scribe.svg?branch=master)](https://travis-ci.org/mozilla/scribe)
[![Go Report Card](https://goreportcard.com/badge/mozilla/scribe "Go Report Card")](https://goreportcard.com/report/mozilla/scribe)

## Overview

scribe is a Go library and frontend used to evaluate policies on systems.
Policies are specified as a JSON or YAML document containing a series of tests, and
these tests return a status indicating if the test criteria matched or not.

Tests reference objects in the policy file. An object can be considered an abstraction
of some data from the system, for example a package version or the contents of a specific
file. The tests also specify criteria that will be applied to the referenced object. For example,
if an object returns a line from a given file, the test could indicate that the data must
match specific content. If the match succeeeds, the test returns true.

It is intended to perform functions such as:

* Identification of software versions that do not meet a specific requirement
* Evaluation of hardening criteria or other system security policies
* Any other functions involving extraction and analysis of host information

The software is designed to return only test status criteria, and meta-data
associated with the test. It runs directly on the system being evaluated, and
requires no data from the system to be returned to a central server for
additional processing.

It's primary purpose is integration with Mozilla MIG which allows
investigators to perform system evaluation by sending a policy to the MIG
agent for execution. It is also suited to executing policies as part of an
instance build and testing process, or periodically on an installed system.

## Usage

Scribe policies can be evaluated using the scribecmd command line tool, or alternatively the scribe
library can be included in another go application.

This example shows evaluation of a given policy file, where only tests that return
true are displayed in the results.

```bash
$ ./scribecmd -f mypolicy.json -T
```

scribecmd supports other runtime options, see the usage output for details.

## Vulnerability scanning

scribe can be used to perform vulnerability scanning directly on the system using a suitable
policy file. The library implements various criteria specifications such as
EVR (epoch/version/release) testing that can be used to determine if a given package
version is less than what is required.

scribevulnpolicy is a policy generator that integrates with [clair](https://github.com/coreos/clair)
for vulnerability data. This tool can be used to generate scribe vulnerability check
policies for supported platforms. For details on usage see the
[documentation for scribevulnpolicy](./scribevulnpolicy/README.md).

## Additional documentation

Additional documentation on the library is available at [godoc.org](https://godoc.org/github.com/mozilla/scribe/).
