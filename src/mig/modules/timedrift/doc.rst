======================================
Mozilla InvestiGator: TimeDrift module
======================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The timedrift module evaluates the current time on a given endpoint against the
time retrieved from a list of NTP servers. If the -drift parameter is passed,
the module checks that the endpoint's time is within or without the drift
window.

When evaluating drift, the module returns FoundAnything=true for endpoints that
have drifted beyond the accepted value, and for which the local time is out of
sync compared with NTP servers.

Usage
-----

timedrift can be called with empty parameters, and then only returns the local
time of the target endpoint. When called with a drift parameter, NTP
connections are established to evaluated the local time against network time.

.. code:: json

	{
	  "module": "timedrift",
	  "parameters": {
		"drift": "5s"
	  }
	}

Examples
--------

Endpoint non-compliant with a 10 millisecond drift
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Evaluating a 10ms drift is not useful, because the latency between the
endpoint and the NTP servers is most likely greater than 10ms, and
endpoint would always fail that test. But it illustrates the output
from an endpoint that has drifted beyond the acceptable value.

.. code::

    $ mig timedrift -t "name='somehost.example.net'" -show all -drift 10ms 2>/dev/null
    stat: execution time 252.902127ms
    somehost.example.net local time is 2015-03-14T13:26:27.441740604-04:00
    somehost.example.net local time is out of sync from NTP servers
    somehost.example.net Local time is ahead of ntp host 0.pool.ntp.org by 17.731324ms
    somehost.example.net Local time is ahead of ntp host 1.pool.ntp.org by 16.542859ms
    somehost.example.net Local time is ahead of ntp host 2.pool.ntp.org by 20.853337ms
    somehost.example.net Local time is ahead of ntp host 3.pool.ntp.org by 33.743419ms
    somehost.example.net stat: 0.pool.ntp.org responded in 44.26132ms with time 2015-03-14 17:26:27.473289999 +0000 UTC. local time drifts by 17.731324ms
    somehost.example.net stat: 1.pool.ntp.org responded in 38.263502ms with time 2015-03-14 17:26:27.520487097 +0000 UTC. local time drifts by 16.542859ms
    somehost.example.net stat: 2.pool.ntp.org responded in 46.682002ms with time 2015-03-14 17:26:27.576307275 +0000 UTC. local time drifts by 20.853337ms
    somehost.example.net stat: 3.pool.ntp.org responded in 83.492232ms with time 2015-03-14 17:26:27.660943187 +0000 UTC. local time drifts by 33.743419ms
    somehost.example.net command success

Endpoint compliant with a 5 seconds drift
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code::

    $ mig timedrift -t "name='somehost.example.net'" -show all -drift 5s 2>/dev/null
    stat: execution time 1.76047894s
    somehost.example.net local time is 2015-03-14T13:26:10.764244879-04:00
    somehost.example.net local time is within acceptable drift from NTP servers
    somehost.example.net stat: 0.pool.ntp.org responded in 44.574857ms with time 2015-03-14 17:26:10.996919999 +0000 UTC. local time drifts by 17.557879ms
    somehost.example.net stat: 1.pool.ntp.org responded in 38.52106ms with time 2015-03-14 17:26:12.139883892 +0000 UTC. local time drifts by 16.917595ms
    somehost.example.net stat: 2.pool.ntp.org responded in 46.79544ms with time 2015-03-14 17:26:12.38555022 +0000 UTC. local time drifts by 20.839501ms
    somehost.example.net stat: 3.pool.ntp.org responded in 82.798078ms with time 2015-03-14 17:26:12.490975185 +0000 UTC. local time drifts by 33.808416ms
    somehost.example.net command success

Get localtime from endpoint
~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. code::

    $ mig timedrift -t "name='somehost.example.net'" 2>/dev/null
    somehost.example.net local time is 2015-03-14T13:32:24.226318523-04:00
