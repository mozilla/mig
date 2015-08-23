scribe
======

scribe is a host policy evaluator written in Go.

[![Build Status](https://travis-ci.org/mozilla/scribe.svg?branch=master)](https://travis-ci.org/mozilla/scribe)

Overview
--------
scribe is a Go library and frontend used to evaluate policies on systems.
Policies are specified as a JSON document containing a series of tests, and
these tests return a status indicating if the test criteria passed.

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

Additional documentation
------------------------
Additional documentation on the library is available at [godoc.org](https://godoc.org/github.com/mozilla/scribe/src/scribe).
