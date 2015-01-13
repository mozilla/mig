================================
Mozilla InvestiGator Cheat Sheet
================================
:Author: Guillaume Destuynder <kang@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

This is a list of common operations you may want to run with MIG.

File/Filechecker module operations
==================================

- Find if files in /etc/cron.d contain "mysql://" on hosts "*buildbot*"

.. code:: bash

    mig file -t "os='linux' AND name like '%buildbot%'" -path /etc/cron.d/ -content "mysql://"

- Find if file /etc/passwd has been modified in the past 2 days

.. code:: bash

    mig file -t "os='linux'" -path /etc/passwd -mtime <2d

- Find endpoints with high uptime

.. code:: bash

    mig file -t "os='linux' OR os='darwin'" -path /proc/uptime -content "^[5-9]{1}[0-9]{7,}\\."

- Find endpoints running process "/sbin/auditd"

.. code:: bash

    mig file -t "os='linux'" -path /proc/ -name cmdline -content "/sbin/auditd"
