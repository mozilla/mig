Mozilla Investigator (MIG)
=========================
<img style="float: right" src="doc/.files/MIG-logo-CC-small.jpg" size="300px" img src="image" width="35%">

*Identifying vulnerability in remote endpoints.*

[![Build Status](https://travis-ci.org/mozilla/mig.svg?branch=master)](https://travis-ci.org/mozilla/mig)

What is Mozilla Investigator?
-------------

Mozilla Investigator (MIG) is a platform for identifying vulnerability in remote endpoints. "Agents" installed throughout systems of an infrastructure answer queries regarding file systems, network states, memory, and endpoint configuration in real time. With MIG, users can obtain information from many endpoints of an infrastructure simultaneously, thus identfying risk and improving security operations.

In other words...MIG is an army of Sherlock Holmes's at your fingertips, my dear Watson!

When do I use MIG?
-------------

Suppose a critical vulnerability has just been released for your favorite PHP application. The vuln is already
being exploited and security groups are releasing indicators of compromise (IOCs). The thought of inspecting
thousands of systems manually isn't exactly appealing, is it?

**MIG can help!** 

MIG searches across all systems for the signature of the vulnerable PHP app (the md5 of a file, a regex,
or simply a filename) via the `file` module. MIG also investigates IOCs, including:

* specific log entries
* backdoor files with md5 and sha 1/2/3 hashes
* IP addresses from botnets
* byte strings in processes memories

With just a few commands, MIG users can investigate *thousands* of remote systems.

![MIG command line demo](doc/.files/mig-cmd-demo.gif)

Design and Capability
-------------

MIG agents are designed to be lightweight, secure, and easy to deploy. You can ask your favorite sysadmins to add agents to base deployment without fear of breaking the entire production network. All parameters are built into the agent at compile time, including the ACLs of authorized investigators. PGP keys bolster security. Even if MIG servers become compromised, *nobody* can access agents as long as keys are stored safely by the investigator.

MIG is also designed to be fast and asynchronous. It uses AMQP to distribute actions
to endpoints and relies on Go channels to prevent blocking components. The reliability of the platform is not dependent on long-running processes, as running actions and commands are stored in a PostgreSQL database and on disk cache.

Investigations generally complete in 10 to 300 seconds. Many actions require only milliseconds for agents to run. More demanding actions, like searching for a hash in a large directory, require a few minutes.

For MIG users, privacy and security are essential. Agents do NOT send raw data back to the
platform, and only answer queries. All actions are signed by GPG
keys that are NOT stored in the platform, thereby preventing infrastructure compromise.

| Capability        | Linux | MacOS | Windows |
| ----------------- | ----- | ----- | ------- |
| file inspection   | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) |
| network inspection| ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) | (partial) |
| memory inspection | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) |
| vuln management   | ![check](doc/.files/check_mark_green.png) | (planned) | (planned) |
| log analysis      | (planned) | (planned) | (planned) |
| system auditing   | ![check](doc/.files/check_mark_green.png) | (planned) | (planned) |

Quick Start with Docker
---------------------

You can explore a local-only MIG setup using Docker. Docker provides a single container environment with most MIG components available. *Note that this setup is not intended for comprehensive MIG usage.*

Pull from Docker Hub:

```bash
$ docker pull mozilla/mig
$ docker run -it mozilla/mig
```

Alternatively, if the the source is checked out in your GOPATH, build your own image:

```bash
$ cd $GOPATH/src/github.com/mozilla/mig
$ docker build -t mozilla/mig:latest .
$ docker run -it mozilla/mig
```

Use MIG inside the container to query a local agent:

```bash
mig@5345268590c8:~$ /go/bin/mig file -t all -path /usr/bin -sha2 5c1956eba492b2c3fffd8d3e43324b5c477c22727385be226119f7ffc24aad3f
1 agents will be targeted. ctrl+c to cancel. launching in 5 4 3 2 1 GO
Following action ID 7978299359234.
 1 / 1 [=========================================================] 100.00% 0/s4s
100.0% done in 3.029105958s
1 sent, 1 done, 1 succeeded
ed11f485244a /usr/bin/wget [lastmodified:2016-07-05 15:32:42 +0000 UTC, mode:-rwxr-xr-x, size:419080] in search 's1'
1 agent has found results
```

To further explore the capabilities of MIG, see the [CheatSheet](https://github.com/mozilla/mig/blob/master/doc/cheatsheet.rst).

Technology
----------
MIG is built in Go. It uses a REST API that receives signed JSON messages. The messages are distributed
to agents via RabbitMQ and stored in a PostgreSQL database.

MIG is:
* fast in distribution
* simple to deploy across platforms
* secured using OpenPGP
* focused on privacy (never retrieves raw data from endpoints)

Watch this 10 minute demonstration of the console interface:

[![MIG youtube video](http://img.youtube.com/vi/wJwj5YB6FFA/0.jpg)](http://www.youtube.com/watch?v=wJwj5YB6FFA)

Watch the MIG presentation at SANS DFIR Summit in Austin, TX:

[![MIG @ DFIR Summit 2015](http://img.youtube.com/vi/pLyKPf3VsxM/0.jpg)](http://www.youtube.com/watch?v=pLyKPf3VsxM)

Discussion
----------
Join **#mig** at [irc.mozilla.org](https://wiki.mozilla.org/IRC) (use a web
client such as [mibbit](https://chat.mibbit.com)).

Documentation
-------------
All MIG documentation is available in the 'doc' directory and at http://mig.mozilla.org .
* [Concepts & Internal Components](doc/concepts.rst)
* [Installation & Configuration](doc/configuration.rst)
