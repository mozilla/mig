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

This document does not discuss the details around writing scribe tests, the
scribe project documentation should be reviewed for that. This document focuses
on usage of the scribe module within MIG and provides some examples.

Usage
-----
Document analysis mode can be used by specifying a document to analyze with
with `path` option. By default, all tests are returned with a result. To
return only tests that evaluate to true, the `onlytrue` option can be used.

By default, results are returned in line mode (one result per line). The
`human` flag can be used to output extended results, and the `json` flag
can be used to output each result as a JSON document.

Document analysis mode
~~~~~~~~~~~~~~~~~~~~~~
In document analysis mode, a JSON document is supplied containing a valid
scribe document.

A scribe document contains a series of objects and tests. Objects obtain
information from the system, and tests evaluate this information against
specified criteria. An object can return more than one candidate, for example
if multiple files are identified on a system that match certain criteria. In
this case, the test will evaluate each candidate, and return a result for
each one.

The following is a simple document example that validates OpenSSL is at least
version 1.0.1e. If the criteria in the test matches, it will return true.

.. code:: json

    {
        "objects": [
        {
            "object": "openssl-package",
            "package": {
                "name": "openssl"
            }
        }
        ],
        "tests": [
        {
            "test": "openssl test",
            "object": "openssl-package",
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
if the object identified more then one object on the system. In this case, the
evaluator will be applied to each object identifier. If at least one evaluation
is true, the master result for the test will be true.

A more advanced test, returning true if, in this example Django is identified
on the system and the version is less than 1.4.5, and /etc/testfile also exists
on the system.

.. code:: json

    {
        "objects": [
        {
            "object": "djangoinit",
            "filecontent": {
                "path": "/",
                "file": "__init__\\.py",
                "expression": "^VERSION = \\((\\S+), (\\S+), (\\S+),",
                "concat": "."
            }
        },
        {
            "object": "testfile",
            "filename": {
                "path": "/etc",
                "file": "(testfile)"
            }
        }
        ],
        "tests": [
        {
            "test": "django and test file",
            "object": "djangoinit",
            "evr": {
                "operation": "<",
                "value": "1.4.5"
            },
            "if": [ "testfile exists" ]
        },
        {
            "test": "testfile exists",
            "object": "testfile"
        }
        ]
    }

The module is designed to only return a true or a false for tests; file content
from the file system is never returned from the agent.
