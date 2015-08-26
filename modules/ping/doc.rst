=================================
Mozilla InvestiGator: Ping module
=================================
:Author: Sushant Dinesh <sushant.dinesh94@gmail.com>

.. sectnum::
.. contents:: Table of Contents

The ping module (PM) allows a user to check connection of an endpoint to another host. PM supports ICMP, TCP and UDP pings to a remote host. The response returned by the ping module is similar to ping utility in UNIX systems.

Usage
-----

PM supports ICMP, TCP and UDP pings. PM requires that destination is a valid ipv4, ipv6 address or a Fully Qualified Domain name (FQDN).

.. code:: json

  {
        "count": 3,
        "destination": "www.google.com",
        "destinationport": 80,
        "protocol": "tcp",
        "timeout": 5
  }

Parameters
~~~~~~~~~~~~

* **protocol:** Specifies the protocol to be used for the ping test. Valid protocols are icmp, tcp and udp.
* **destination**: ipv4, ipv6 address or FQDN of the destination host to be checked.
* **destinationport**: port number on the destination host to be checked for connectivity. this option is to be left blank when  protocol is icmp. For tcp and udp, the destination port defaults to 80 when not specified.
* **count**: Number of times the destination has to be pinged. optional. defaults to 3.
* **timeout**: Seconds to wait for response before timing out. optional. defaults to 5s.

Note on scans
~~~~~~~~~~~~~~~~~~~

* Selecting protocol as TCP performs a TCP Connect scan. This means that a full connection is established (and broken down) when a tcp ping is performed. This might leave records in the destination systems logs.
* A timeout on udp ping indicates that the port checked for **maybe** open. Hence the module returns reachable as true (and latency as timeout). However, if the port is closed, the module returns "Connection Refused" indicating that the port is closed.

Examples
--------

Basic ICMP ping
~~~~~~~~~~~~~~~

.. code::

	$ mig ping -t "name='somehost.example.net'" -show all -d 8.8.8.8
	somehost.example.net icmp ping of 8.8.8.8 (8.8.8.8) succeeded. Target is reachable.
	somehost.example.net ping #1 succeeded in 36ms
	somehost.example.net ping #2 succeeded in 21ms
	somehost.example.net ping #3 succeeded in 31ms
	somehost.example.net command success

Single TCP ping of twitter.com:443
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code::

	$ mig ping -t "name='somehost.example.net'" -show all -d twitter.com -dp 443 -p tcp -c 1 -t 5
	somehost.example.net tcp ping of twitter.com:443 (199.16.156.102) succeeded. Target is reachable.
	somehost.example.net ping #1 succeeded in 27ms
	somehost.example.net command success

UDP Ping of Google's DNS
~~~~~~~~~~~~~~~~~~~~~~~~

UDP ping is less deterministic because no response is returned from the target
if the ping succeeded. A lack of response is considered a success, and no
latency is returned.

.. code::

	$ mig ping -t "name='somehost.example.net'" -show all -d 8.8.8.8 -dp 53 -p udp -c 10 -t 5
	somehost.example.net udp ping of 8.8.8.8:53 (8.8.8.8) succeeded. Target is reachable.
	somehost.example.net ping #1 may have succeeded (no udp response)
	somehost.example.net ping #2 may have succeeded (no udp response)
	somehost.example.net ping #3 may have succeeded (no udp response)
	somehost.example.net ping #4 may have succeeded (no udp response)
	somehost.example.net ping #5 may have succeeded (no udp response)
	somehost.example.net ping #6 may have succeeded (no udp response)
	somehost.example.net ping #7 may have succeeded (no udp response)
	somehost.example.net ping #8 may have succeeded (no udp response)
	somehost.example.net ping #9 may have succeeded (no udp response)
	somehost.example.net ping #10 may have succeeded (no udp response)
	somehost.example.net command success

