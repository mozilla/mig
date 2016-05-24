====================================
Mozilla InvestiGator: Package module
====================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The package module (pkg) supports querying of the OS package manager
using the package query functionality of `scribe <https://github.com/mozilla/scribe>`_.
It can be used to return a list of packages installed on an agent system
that match supplied regular expressions.

Usage
-----

Package query mode can be used by specifying one or more `name` options for
the pkg module. In this mode, the module will query the OS package manager
on the agent system, identifying any packages matching the regular expression
argument to `name`, and returning those packages along with their versions. In
addition to this, the agent will also return what type of package was identified.

It is also possible to optionally filter on the version, so the agent only returns
packages which also match the given version. This can be done by supplying a
regular expression to match against the package version as a `version` option.

Currently the following package managers are supported for query on an agent system.

* **RPM**: RPM based package managers, for Red Hat, CentOS, etc
* **dpkg**: DPKG based package managers, for Debian, Ubuntu, etc

Packages managed in other package managers will not be returned by the module.
