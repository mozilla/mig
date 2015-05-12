====================================
Mozilla InvestiGator: Package module
====================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The package module (pkg) is intended to provide capabilities allowing
inspection of software deployed on an agent system. The pkg module is
primarily implemented through the use of the `mozoval <https://github.com/mozilla/mozoval>`_
OVAL processing library.

The module currently supports two functions:

* Execution of OVAL definitions, performing vulnerability analysis functions
* Matching against OS package managers, allowing inspection of deployed software versions

Usage
-----

OVAL processing mode
~~~~~~~~~~~~~~~~~~~~
In OVAL processing mode, the module parses and executes an OVAL XML definition
file supplied by the client. The OVAL definitions are transmitted to the agent
in a compressed base64 encoded format. The agent then decodes this data, and
pushes the OVAL definitions through the mozoval processor. The result of definitions
are then returned to the investigator, indicating a true or false status.

To use OVAL processing mode, the `oval` option is supplied on the command line,
giving a properly formatted OVAL definition file as an argument.

The `includefalse` parameter can be specified if false evaluations should be
returned to the investigator. By default. only OVAL definitions that have
evaluated to true are returned.

The mozoval library supports the use of concurrent goroutines for definition
evaluation. By default, only a single routine is used to evaluate definitions
by the module. To increase this, the `concurrency` flag can be used. Note that
care should be taken in increasing this value above 1, as it can result in
more significant processor utilization on the agent system.

Package query mode
~~~~~~~~~~~~~~~~~~
Package query mode can be used by specifying one or more `name` options for
the pkg module. In this mode, the module will query the OS package manager
on the agent system, identifying any packages matching the regular expression
argument to `name`, and returning those packages along with their versions. In
addition to this, the agent will also return what type of package was identified.

Currently the following package managers are supported for query on an agent system.

* **RPM**: RPM based package managers, for Red Hat, CentOS, etc
* **dpkg**: DPKG based package managers, for Debian, Ubuntu, etc

Packages managed in other package managers will not be returned by the module.

Notes on OVAL execution
-----------------------
Note that the mozoval library does not implement the full OVAL reference or
specification. Instead this library is focused on a limited subset of OVAL
evaluations, designed to support patching and vulnerability identification.

The library may not support certain functions or parameters that are supported
in OVAL; validation should be done on definition files being used to ensure they
are supported. mozoval is under development, and is intended to support additional
functionality in the future.

