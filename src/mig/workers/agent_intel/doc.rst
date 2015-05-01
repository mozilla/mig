========================================
Mozilla InvestiGator: Agent Intel Worker
========================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The Agent Intel Worker in a separate program that listens for event about
agents that newly joined the platform, and create asset hints that are
published to MozDef. This worker serves a very specific purpose in the
collection of asset data performed by Mozilla OpSec. It may not be very useful
to anyone else.

Configuration
-------------

This worker retrieves agents hearbeats in the MIG Agent format from the MIG
Relay, transforms them into Asset Hints, and publishes them to some other
rabbitmq endpoint when MozDef will retrieve them.


.. code::

	; mozdef rabbitmq endpoint
	[mozdef]
		host = "mozdef.rabbitmq.example.net"
		port = 5671
		user = "migassetworker"
		pass = "secretpassphrase"
		vhost = "mozdef"
		exchange = "eventtask"
		routingkey = "eventtask"
		usetls  = true
		cacertpath  = "/etc/certs/ca.crt"
		clientcertpath = "/etc/certs/migassetworker.crt"
		clientkeypath  = "/etc/certs/migassetworker.key"
		timeout = "10s"

	; https://github.com/ameihm0912/vmintgr
	[vmintgr]
		bin = "/opt/vmintgr/bin/grouptest"

	; mig rabbitmq endpoint
	[mq]
		host = "hostname.mig.relay.example.net"
		port = 5671
		user = "migworker"
		pass = "somepassphrase"
		vhost = "mig"
		usetls  = true
		cacert  = "/path/to/ca.crt"
		tlscert = "/path/to/client.crt"
		tlskey  = "/path/to/client.key"
		timeout = "10s"

	[logging]
		mode = "syslog" ; stdout | file | syslog
		level = "info"  ; debug | info | warning | error | critical
		host = "localhost"
		port = 514
		protocol = "udp"

Upstart
~~~~~~~

.. code::

	# Mozilla InvestiGator Agent Intel Worker

	description     "MIG Agent Intel Worker"

	start on filesystem or runlevel [2345]
	stop on runlevel [!2345]

	setuid mig
	limit nofile 640000 640000

	respawn
	respawn limit 10 5
	umask 022

	console none

	pre-start script
		test /opt/mig_agent_intel_worker || { stop; exit 0; }
	end script

	# Start
	exec /opt/mig_agent_intel_worker
