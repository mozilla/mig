MIG: Mozilla InvestiGator
=========================
<img style="float: right" src="doc/.files/MIG-logo-CC-small.jpg" size="300px">

[![Build Status](https://travis-ci.org/mozilla/mig.svg?branch=master)](https://travis-ci.org/mozilla/mig)

MIG is Mozilla's platform for investigative surgery of remote endpoints.

Table of Contents (ToC)
========================

* [Quick Start w/ Docker](#quick-start-with-docker)
* [What is This?](#what-is-this)
* [Technology](#technology)
* [Discussion](#discussion)
* [Documentation](#documentation)
* [Testing](#testing)

Quick Start w/ Docker
---------------------

You can spin up a local-only MIG setup using docker. The container is not suitable for production use but lets you experiment with MIG quickly.

```bash
$ docker pull mozilla/mig
$ docker run -it mozilla/mig
```

Once inside the container, you can use the MIG tools to query a local agent, as such:

```bash
mig@5345268590c8:~$ mig file -t all -path /usr/bin -sha2 5c1956eba492b2c3fffd8d3e43324b5c477c22727385be226119f7ffc24aad3f
1 agents will be targeted. ctrl+c to cancel. launching in 5 4 3 2 1 GO
Following action ID 7978299359234.
 1 / 1 [=========================================================] 100.00% 0/s4s
100.0% done in 3.029105958s
1 sent, 1 done, 1 succeeded
ed11f485244a /usr/bin/wget [lastmodified:2016-07-05 15:32:42 +0000 UTC, mode:-rwxr-xr-x, size:419080] in search 's1'
1 agent has found results
```

To explore the capabilities of MIG, take a look at the [CheatSheet](https://github.com/mozilla/mig/blob/master/doc/cheatsheet.rst).

What is this?
-------------

MIG is composed of agents installed on all systems of an infrastructure that are
be queried in real-time to investigate the file-systems, network state, memory
or configuration of endpoints.

| Capability        | Linux | MacOS | Windows |
| ----------------- | ----- | ----- | ------- |
| file inspection   | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) |
| network inspection| ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) | (partial) |
| memory inspection | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) | ![check](doc/.files/check_mark_green.png) |
| vuln management   | ![check](doc/.files/check_mark_green.png) | (planned) | (planned) |
| log analysis      | (planned) | (planned) | (planned) |
| system auditing   | ![check](doc/.files/check_mark_green.png) | (planned) | (planned) |

Imagine it is 7am on a saturday morning, and someone just released a
critical vulnerability for your favorite PHP application. The vuln is already
exploited and security groups are releasing indicators of compromise (IOCs).
Your weekend isn't starting great, and the thought of manually inspecting
thousands of systems isn't making it any better.

MIG can help. The signature of the vulnerable PHP app (the md5 of a file, a regex,
or just a filename) can be searched for across all your systems using
the `file` module. Similarly, IOCs such as specific log entries, backdoor files
with md5 and sha1/2/3 hashes, IP addresses from botnets or byte
strings in processes memories can be investigated using MIG. Suddenly, your
weekend is looking a lot better. And with just a few commands, thousands of systems
will be remotely investigated to verify that you're not at risk.

![MIG command line demo](doc/.files/mig-cmd-demo.gif)

MIG agents are designed to be lightweight, secure, and easy to deploy so you can
ask your favorite sysadmins to add it to a base deployment without fear of
breaking the entire production network. All parameters are built into the agent
at compile time, including the list and ACLs of authorized investigators.
Security is enforced using PGP keys, and even if MIG's servers are compromised,
as long as our keys are safe on your investigator's laptop, no one will break
into the agents.

MIG is designed to be fast, and asynchronous. It uses AMQP to distribute actions
to endpoints, and relies on Go channels to prevent components from blocking.
Running actions and commands are stored in a Postgresql database and on disk cache,
such that the reliability of the platform doesn't depend on long-running processes.

Speed is a strong requirement. Most actions will only take a few hundreds
milliseconds to run on agents. Larger ones, for example when looking for a hash in
a big directory, should run in less than a minute or two. All in all, an
investigation usually completes in between 10 and 300 seconds.

Privacy and security are paramount. Agents never send raw data back to the
platform, but only reply to questions instead. All actions are signed by GPG
keys that are not stored in the platform, thus preventing a compromise from
taking over the entire infrastructure.

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

We also have a [public mailing list at list@mig.ninja](http://list.mig.ninja/listinfo/list).

Documentation
-------------
All documentation is available in the 'doc' directory and on http://mig.mozilla.org .
* [Concepts & Internal Components](doc/concepts.rst)
* [Installation & Configuration](doc/configuration.rst)

Testing
-------
Assuming you have a dedicated Ubuntu system (like a VM), you can use the
standalone installation script to deploy a test environment rapidly.
```bash
$ sudo apt-get install golang git

# must be >= 1.5
$ go version
go version go1.6.1 linux/amd64

$ export GOPATH=$HOME/go

$ mkdir $GOPATH

$ go get mig.ninja/mig

$ cd $GOPATH/src/mig.ninja/mig

$ bash tools/standalone_install.sh
```
This script will install all of the components MIG needs for a localhost only
installation. Follow instructions at the end of the script to convert it to a
real infrastructure, or read [Installation & Configuration](doc/configuration.rst).
