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

