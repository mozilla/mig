=================================
Mozilla InvestiGator: yara module
=================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The yara module provides the ability to scan systems the agent is running on
for objects which match provided yara rules. An investigator can send a list of
yara rules to the MIG agents along with an indication of what objects should be
scanned, and the agents will return any objects which matched and the rules that
matched against them.

Scanning is currently limited to files only at the moment.

Building MIG with Yara support
------------------------------
Yara support is not enabled by default and requires certain dependencies on the
build system to enable. Specifically, you will want the to make sure that the
`yara libraries <https://github.com/VirusTotal/yara>`_ are installed on the system
you are building MIG on.

To ensure that any systems with a yara enabled agent do not need to have the yara
library installed, MIG can be built with the yara library statically linked into
the MIG binary.

Fetch and install yara with the required options
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Download the yara tarball and compile it with the required options. You will need
a working c compiler in addition to automake and autoconf.

.. code::

    $ curl -OL https://github.com/VirusTotal/yara/archive/v3.5.0.tar.gz
    $ tar -zxvf v3.5.0.tar.gz
    $ cd yara-3.5.0
    $ ./bootstrap.sh
    $ ./configure --disable-shared --disable-magic --disable-cuckoo --without-crypto
    $ make
    $ sudo make install

From here the agent can be compiled with yara support. The yara module should be
enabled in `conf/available_modules.go` (or whatever you have the Makefile variable
AVAILMOD set to). Then the agent can be compiled with yara support.

.. code::

    $ make mig-agent WITHYARA=yes

The previous example applies to Linux. If you are building an OSX agent, you might
need a few extra environment variables to help locate things, such as:

.. code::

    $ env CPATH=/my/path/to/yara/include LIBRARY_PATH=/my/path/to/yara/lib make mig-agent WITHYARA=yes

This should result in a mig-agent with the yara library builtin, which will work when
deployed to hosts without libyara. Note that, as modules are used in other MIG components
such as the client tools, you will likely want to set WITHYARA=yes when building the
client tools as well.

`tools/standalone_install.sh` also includes yara support, so can be reviewed for some
hints on the build process.

Usage
-----
Two options must be provided to the yara module.

The `rules` should specify the path on your system containing the yara rules
you want to send to the agents.

The `files` option should be set to a string which is essentially the arguments
you would provide to the file module. See the help output of the file module for
more information.

The following example shows a set of rules being used to scan everything in /bin
and in /sbin on each agent system.

.. code::

    $ mig yara -t all -rules ./testrules.yara -files '-path /bin -path /sbin -name .'
