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
        snstopic = ""
        outputmozdef = ""
        channelsize = 1024

The dispatch module supports HTTP POST of records it generates to a specified endpoint, or
publishing to an SNS topic.

If HTTP POST is desired, set the ``httpurl`` parameter in the module configuration file.

To use SNS, set ``snstopic`` to the **name** of the topic (not the ARN). This topic must
exist in the region the instance is executing in.

The ``channelsize`` parameter sets the size of the dispatch module input buffer. If the
input buffer is full (meaning the module cannot drain messages fast enough) messages will
be dropped and the agent log file will indicate the number of messages dropped in a given
time period. This value can be increased as desired, and defaults to 1024.
