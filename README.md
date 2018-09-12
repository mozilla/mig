Mozilla Investigator (MIG)
=========================
<img style="float: right" src="doc/.files/MIG-logo-CC-small.jpg" size="300px" img src="image" width="35%">

*Investigating vulnerability of remote endpoints.*

[![Build Status](https://travis-ci.org/mozilla/mig.svg?branch=master)](https://travis-ci.org/mozilla/mig)

What is Mozilla Investigator?
-------------

Mozilla Investigator (MIG) is a platform for identifying vulnerability of remote endpoints. "Agents" in all systems of an infrastructure facilitate queries of file systems, network states, memory, and endpoint configuration in real time. Using MIG, users can obtain information from many endpoints in an infrastructure simultaneously, thus identfying risk and improving security operations.

In other words...it's an army of Sherlock Holmes's, my dear Watson!

Suppose a critical vulnerability is released for your favorite PHP application. The vuln is already
being exploited and security groups are releasing indicators of compromise (IOCs). The thought of manually inspecting
thousands of systems isn't exactly exciting, is it?

MIG can help! 

The signature of the vulnerable PHP app (the md5 of a file, a regex,
or simply a filename) can be searched for across all systems via
the `file` module. Similarly, IOCs can be investigated using MIG, including:

* specific log entries
* backdoor files with md5 and sha 1/2/3 hashes
* IP addresses from botnets
* byte strings in processes memories

With just a few simple commands, MIG users investigate thousands of remote systems to identify risk.

![MIG command line demo](doc/.files/mig-cmd-demo.gif)

MIG agents are designed to be lightweight, secure, and easy to deploy, so you can
ask your favorite sysadmins to add agents to base deployment without fear of
breaking the entire production network. All parameters are built into the agent
at compile time, including the list and ACLs of authorized investigators.
Security is enforced using PGP keys. Even if MIG's servers are compromised, *nobody* can break into the agents
if our keys are kept safe on the investigator's laptop.

MIG is designed to be fast and asynchronous. It uses AMQP to distribute actions
to endpoints, and relies on Go channels to prevent components from blocking. The reliability of the platform doesn't depend on long-running processes, as running actions and commands are stored in a PostgreSQL database and on disk cache.

Most actions will only require milliseconds to run on agents. Larger actions, like searching for a hash in
a big directory, will require a few minutes. Generally, an
investigation completes in 10 to 300 seconds.

Privacy and security are paramount. Agents do NOT send raw data back to the
platform, and only reply to queries. All actions are signed by GPG
keys that are NOT stored in the platform, thus preventing compromised infrastructure.

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

You can explore a local-only MIG setup using Docker. Docker provides a single container environment with most MIG components available. Note the container is not intended for full MIG usage.

Pulling from Docker Hub:

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

Inside the container, use MIG to query a local agent:

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

To further explore the capabilities of MIG, examine the [CheatSheet](https://github.com/mozilla/mig/blob/master/doc/cheatsheet.rst).

Technology
----------
MIG is built in Go and uses a REST API that receives signed JSON messages distributed
to agents via RabbitMQ and stored in a Postgres database.

It is:
* Massively Distributed means Fast.
* Simple to deploy and Cross-Platform.
* Secured using OpenPGP.
* Respectful of privacy by never retrieving raw data from endpoints.

Check out this 10 minutes video for a more general presentation and a demo of
the console interface.

[![MIG youtube video](http://img.youtube.com/vi/wJwj5YB6FFA/0.jpg)](http://www.youtube.com/watch?v=wJwj5YB6FFA)

MIG was recently presented at the SANS DFIR Summit in Austin, Tx. You can watch the recording below:

[![MIG @ DFIR Summit 2015](http://img.youtube.com/vi/pLyKPf3VsxM/0.jpg)](http://www.youtube.com/watch?v=pLyKPf3VsxM)

Discussion
----------
Join **#mig** on [irc.mozilla.org](https://wiki.mozilla.org/IRC) (use a web
client such as [mibbit](https://chat.mibbit.com)).

Documentation
-------------
All documentation is available in the 'doc' directory and on http://mig.mozilla.org .
* [Concepts & Internal Components](doc/concepts.rst)
* [Installation & Configuration](doc/configuration.rst)
