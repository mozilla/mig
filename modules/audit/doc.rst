====================================
Mozilla InvestiGator: audit module
====================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The audit module can be used to read kernel audit messages (e.g., Linux audit) and either
write them to the agent's log file, or more typically dispatch the audit trail off the system
using the dispatch module.

Currently only Linux is supported by the audit module.

Auditing support for Linux is implemented using `libaudit-go <https://github.com/mozilla/libaudit-go>`_.

Usage
-----

The audit module is a persistent module. When the agent is configured with the audit module, it
will spawn the audit module as a subprocess and initialize auditing on the platform. The audit
module will enable audit, load the audit rules indicated in the configuration into the kernel, and
begin collecting/parsing audit data from the kernel.

If the dispatch module is also loaded with the agent, any audit messages will be sent to the
dispatch module where they can be transmitted to an event collection system. Otherwise, the agent
will simply log the JSON formatted audit messages in the agent log.

Configuration
-------------

The audit module is configured using ``audit.cfg`` in the agent configuration directory, ``/etc/mig``.

.. code::

        [audit]
        rulespath = /etc/mig/audit.rules.json
        ratelimit = 500
        backloglimit = 16384
        includeraw = no

``rulespath`` indicates the path to load audit rules into the kernel from. Note that this is not a
standard audit configuration, but a JSON based rule set as is used in
`libaudit-go <https://github.com/mozilla/libaudit-go>`_.

``ratelimit`` and ``backloglimit`` can be used to configure the Linux auditing rate and back log
limits. If respective defaults of 500 and 16384 will be used.

``includeraw`` causes the raw audit message to be included with the parsed audit fields in the output.
