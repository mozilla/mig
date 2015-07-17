===================================
Mozilla InvestiGator: scribe module
===================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The scribe module provides host-based analysis based on a JSON document
containing a series of tests. The module is based on the scribe engine;
scribe can be found `here <https://github.com/mozilla/scribe>`_.

The scribe module is intended to help support:

* Executing policy checks on systems, for example as part of using MIG for vulnerability management
* Execute more advanced file content tests involving dependencies

In addition to supporting on-host analysis using scribe documents, the
module can also be used to query elements of the host system such as the
package manager database for installed package information.

This document does not discuss the details around writing scribe tests, the
scribe project documentation should be reviewed for that. This document focuses
on usage of the scribe module within MIG and provides some examples.

Usage
-----
The scribe module supports two execution modes, scribe document analysis
and package query mode. To use document analysis mode, the `path` option can
be given to the module along with the path to a scribe document. To use
package query mode, the `pkgmatch` option is supplied with a regular expression
specifying which packages to return information for from the agent.

Document analysis mode
~~~~~~~~~~~~~~~~~~~~~~
In document analysis mode, a JSON document is supplied containing a valid
scribe document.

A scribe document contains a series of tests. Tests are typically comprised
of two elements - a test source, which details a source of information from
the host system, and an evaluator which specifies criteria that must match
for the test to be true.

The following is a simple document example that validates OpenSSL is at least
version 1.0.1e. If this test fails, it will return true.

.. code:: json

    {
        "tests": [
        {
            "name": "openssl test",
            "identifier": "test-0001",
            "package": {
                "name": "openssl"
            },
            "evr": {
                "operation": "<",
                "value": "1.0.1e"
            }
        }
        ]
    }

Passing this to the module will return the test status.

::

    1 agents will be targeted. ctrl+c to cancel. launching in 5 4 3 2 1 GO
    Following action ID 4580457251059.status=inflight.
    - 100.0% done in 3.16738436s
    1 sent, 1 done, 1 succeeded
    ubuntu-dev master [false] name:"openssl test" hastrue:false error:""
    ubuntu-dev sub [false] name:"openssl test" identifier:"openssl"
    1 agent has found results

In this case, the test returns false. The master result for the test indicates
false, as the sub result was false. A single test can have multiple sub-results
if the source matched more then one object on the system. For example, multiple
versions of the same package are installed, multiple files are identified that
match file criteria, and so on. In this case, the evaluator will be applied to
each object identifier. If at least one evaluation is true, the master result
for the test will be true.

A more advanced test, returning true if, in this example Django is identified
on the system and the version is less than 1.4.5, and /etc/testfile also exists
on the system.

.. code:: json

    {
        "tests": [
        {
            "name": "djangoinit",
            "identifier": "test-0001",
            "filecontent": {
                "path": "/",
                "file": "__init__\\.py",
                "expression": "^VERSION = \\((\\S+), (\\S+), (\\S+),"
            }
        },
        {
            "name": "modifier",
            "identifier": "test-0002",
            "modifier": {
                "concat": {
                    "operator": "."
                },
                "sources": [
                { "name": "djangoinit", "select": "all" }
                ]
            },
            "evr": {
                "operation": "<",
                "value": "1.4.5"
            },
            "if": [ "test file exists" ]
        },
        {
            "name": "testfile",
            "aliases": [ "test file exists" ],
            "identifier": "test-0003",
            "filename": {
                "path": "/etc",
                "file": "(testfile)"
            }
        }
        ]
    }

The module is designed to only return a true or a false for tests; file content
from the file system is never returned from the agent.

Package query mode
~~~~~~~~~~~~~~~~~~
Package query mode can be used by specifying a `pkgmatch` argument to the
scribe module with a regular expression. Any packages identified on the
agent system will be returned along with their version details. This mode
does not do any test analysis, but instead queries directly into the
scribe library to access the package management interface.

Currently the following package managers are supported for query on an agent system.

* **RPM**: RPM based package managers, for Red Hat, CentOS, etc
* **dpkg**: DPKG based package managers, for Debian, Ubuntu, etc

Packages managed in other package managers will not be returned by the module.

