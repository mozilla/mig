===================================================
Mozilla InvestiGator: MozDef Compliance Item Plugin
===================================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The MozDef compliance item plugin converts results returned for compliance
actions from `mig-runner` into compliance items that are consumable by
MozDef. The plugin will then publish the compliance items into MozDef.

This plugin serves a very specific purpose in the verification of security
compliance performed by Mozilla Infosec. It may not be very useful to anyone else,
but may be useful as an example of creating a plugin for mig-runner.

This plugin is not intended to be run on its own, but instead is called at
the end of a scheduled job within `mig-runner`. See the documentation for
`mig-runner` for additional information. To use the plugin, it should be
placed in the runner `plugins/` directory, and the job configuration should
be set to call this plugin.

Configuration
-------------

Since mig-runner plugins do not support arguments, the configuration file for
the plugin is hardcoded and should be placed in
`/etc/mig/runner-compliance.conf`.

The configuration needs to indicate the URL to POST compliance event data to
MozDef, in addition to the URL the MIG API listens on. The MIG API URL is only
required to populate certain fields in the compliance item and is not actually
communicated with.

Optionally, the path to a wrapper for the `grouptest.py` tool that is part of
`vmintgr` can be specified in the configuration file. If this is present, this
will be called to associate system owner information with a compliance event.

.. code::

	[mig]
		api = "https://api.mig.example.net"
	[mozdef]
                url = "http://127.0.0.1:8080/custom/complianceitems"

	; https://github.com/ameihm0912/vmintgr
	[vmintgr]
		bin = "/opt/vmintgr/bin/grouptest"

As an example, the `grouptest` wrapper may look as follows.

.. code::

        #!/bin/sh
        
        if [ -z "$1" ]; then
                exit 1
        fi
        cd /opt/vmintgr/vmintgr
        ./grouptest.py -j $1
        exit 0


