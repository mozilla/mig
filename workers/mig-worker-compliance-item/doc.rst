===================================================
Mozilla InvestiGator: MozDef Compliance Item Worker
===================================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The MozDef Compliance Item worker implements a data transformation process that
converts the results of compliance actions into compliance items, and publishes
them to a rabbitmq endpoint for consumption by mozdef.
This worker serves a very specific purpose in the verification of security
compliance performed byMozilla OpSec. It may not be very useful to anyone else.

Configuration
-------------

The configuration needs standard access to MIG's rabbitmq relay, configured in
the `[mq]` section. Access to MozDef's rabbitmq endpoint is configured in the
`[mozdef]` section. The `[api]` section must only contain the location of MIG's
API endpoint, used to build links to commands in compliance items. Standard
logging can be configured in `[logging]`.

.. code::

	[api]
		host = "https://api.mig.example.net"
	[mozdef]
		host = "mozdef.rabbitmq.example.net"
		port = 5671
		user = "migcomplianceworker"
		pass = "secretpassphrase"
		vhost = "mozdef"
		exchange = "eventtask"
		routingkey = "eventtask"
		usetls  = true
		cacertpath  = "/etc/certs/ca.crt"
		clientcertpath = "/etc/certs/migcomplianceworker.crt"
		clientkeypath  = "/etc/certs/migcomplianceworker.key"
		timeout = "10s"

	; https://github.com/ameihm0912/vmintgr
	[vmintgr]
		bin = "/opt/vmintgr/bin/grouptest"

	[mq]
		host = "relay.mig.example.net"
		port = 5671
		user = "worker"
		pass = "secretpassphrase"
		vhost = "mig"
		usetls  = true
		cacert  = "/etc/certs/ca.crt"
		tlscert = "/etc/certs/migworker.crt"
		tlskey  = "/etc/certs/migworker.key"
		timeout = "10s"
	[logging]
		mode = "syslog" ; stdout | file | syslog
		level = "info"
		host = "localhost"
		port = 514
		protocol = "udp"

Upstart
~~~~~~~

To manage this worker with upstart, use the configuration below, for example in
`/etc/init/mig-compliance-item-worker.conf`.

.. code::

	# Mozilla InvestiGator MozDef Compliance Item Worker

	description     "MIG MozDef Compliance Item Worker"

	start on filesystem or runlevel [2345]
	stop on runlevel [!2345]

	setuid mig
	limit nofile 640000 640000

	respawn
	respawn limit 10 5
	umask 022

	console none

	pre-start script
		test /opt/mig_compliance_item_worker || { stop; exit 0; }
	end script

	# Start
	exec /opt/mig_compliance_item_worker
