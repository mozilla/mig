================================
Mozilla InvestiGator Cheat Sheet
================================
:Author: Guillaume Destuynder <kang@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

This is a list of common operations you may want to run with MIG.

File module
-----------

You can find detailled documentation by running `mig file help` or in the
online doc at `doc/module_file.html`.

.. _`doc/module_file.html`: http://mig.mozilla.org/doc/module_file.html

Find files in /etc/cron.d that contain "mysql://" on hosts "*buildbot*"
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This is a simple file content check that looks into all the files contained in
`/etc/cron.d` for a string that matched `mysql://`.

.. code:: bash

    mig file -t "queueloc LIKE 'linux.%' AND name LIKE '%buildbot%'" -path /etc/cron.d/ -content "mysql://"

Find files /etc/passwd that have been modified in the past 2 days
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The `mtime` check of the file module matches against the last modified
timestamp of a file.

.. code:: bash

    mig file -t "queueloc LIKE 'linux.%'" -path /etc/passwd -mtime <2d

Find endpoints with high uptime
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

On Linux and MacOS, the uptime of a host is kept in `/proc/uptime`. We can
apply a regex on that file to list hosts with an uptime larger or lower than
any amount.

Note the search target that uses postgres's regex format `~*`.

.. code:: bash

    mig file -t "queueloc ~* '^(linux|darwin).%'" -path /proc/uptime -content "^[5-9]{1}[0-9]{7,}\\."

Find endpoints running process "/sbin/auditd"
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Here, the '^' in the content regex is important to prevent mig from listing
itself while searching for the command line.

.. code:: bash

    mig file -t "queueloc LIKE 'linux.%'" -path /proc/ -name cmdline -content "^/sbin/auditd"

Netstat module
--------------

You can find detailled documentation by running `mig netstat help`.

Searching for a fraudulent IP
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Given an ip 1.2.3.4 associated with fraudulent traffic, we can use the netstat
module to verify that the IP isn't currently connected to any endpoint.

.. code:: bash

	mig netstat -ci 1.2.3.4

`-ci` stands for connected IP, and accepts an IP or a CIDR, in v4 or v6.

Locating a device by its mac address
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

MIG `netstat` can be used to find endpoints that have a given mac address in
their arp tables, which helps geographically locating an endpoint.

.. code:: bash

	mig netstat -nm 8c:70:5a:c8:be:50

`-nm` stands for neighbor mac and takes a regex (ex: `^8c:70:[0-9a-f]`).
