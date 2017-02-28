=====================================
Mozilla InvestiGator: dispatch module
=====================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The dispatch module is used to dispatch alerts to a remote system (e.g., an event
collector).

Usage
-----

Persistent modules have the ability to generate alerts. These alerts are sent from
the persistent module to the master agent process.

When the dispatch module is not running, the agent will simply write the alert in the
standard agent log.

If the dispatch module is active, the agent will instead forward the alert to the dispatch
module, and the module can manage buffering the alert and forwarding it on to the event
collection system configured in the module configuration.

Configuration
-------------

The dispatch module is configured using ``dispatch.cfg`` in the agent configuration directory,
``/etc/mig`` or ``c:\mig``.

.. code::

        [dispatch]
        httpurl = "https://api.to.post.to/event

Currently the dispatch module only supports HTTP POST of alert messages from modules to a
configured HTTP endpoint.

``httpurl`` can be used to configure a URL all alerts will be posted to.
