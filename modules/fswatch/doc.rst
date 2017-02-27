====================================
Mozilla InvestiGator: fswatch module
====================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The fswatch module is a persistent module that monitors the system the
agent is running on for changes to certain files and directories. The
module uses `fsnotify <https://github.com/fsnotify/fsnotify>`_ to
receive indications writes are occuring in monitored paths, and if a file
is modified the module compares it to previously known signatures for the
file. An alert/log is generated if any changes occur.

Usage
-----
This module is a persistent module. If the module is enabled it actively runs
on behalf of the agent, and an investigator does not need to initiate a query.
To use the module it needs to be enabled in the agent and configured.

The module can queried using ``mig`` or ``mig-console`` to check its health, and
it simple returns a small JSON document if it is operating normally.

.. code:: json
    {
        "ok": true
    }

Enable and configure module
~~~~~~~~~~~~~~~~~~~~~~~~~~~
Make sure you do not have ``nopersistmods`` set to true in the MIG agent
configuration, if this is set persistent modules will not be started by the
agent.

In ``available_modules.go``, make sure the module is enabled when the agent is
built. See the agent related configuration details for additional information.

Lastly, a policy must be specified to indicate what the module should monitor. A
default exists for each platform that is used by the module is no configuration
file is present on the system. To use a custom configuration, ``/etc/mig/fswatch.cfg``
should be created.

.. code::

    [paths]
    path = /home/user/testfile
    path = /home/user/testdir
    path = recursive:/bin
    path = recursive:/sbin

Each path item specifies a path on the file system to monitor. If ``recursive:``
prefixes the path, subdirectories in the path will also be monitored, otherwise
just the root path is monitored. Paths can reference individual files, or
directories. The ``recursive:`` option does not apply to regular files.
