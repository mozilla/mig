MIG: Mozilla InvestiGator
=========================

MIG is OpSec's platform for investigative surgery of remote endpoints. It's a
platform that allows investigators to send actions to pools of agents, and check
for indicator of compromision, verify the state of a configuration, block an
account, create a firewall rule or update a blacklist.

For example: an investigator launches an action to search for an apache module
that matches a given md5 value. MIG will register the action, find all the
relevant targets and send commands to each target with the detail of the
action. Each agent then individually run the action using built-in modules
locally, and sends the results back to the MIG platform.

Agents are designed to be lightweight, secure, and easy to deploy. All
parameters are built into the agent at compile time, include the list of
investigator's public keys. The agent binary is statically compiled for a target
platform and can be shipped without any external dependency.

MIG is designed to be fast, and asynchronous. It uses AMQP to distribute actions
to endpoints, and relies on Go channels to prevent components from blocking.
Running actions and commands are stored on disk cache, and don't rely on running
processes for reliability.

Speed is a strong requirement. Most actions will only take a few hundreds
milliseconds to run. Larger ones, for example when looking for a hash in a large
directory, should run in less than a minute.

Check out this 6 minutes presentation for background:

[![MIG youtube video](http://img.youtube.com/vi/uwrJ6Mtc4S0/0.jpg)](http://www.youtube.com/watch?v=uwrJ6Mtc4S0)

Goals
-----

* Query a pool of endpoints to verify the presence of a specific indicators
  (similar to IOC, but we use a different format)
* Provide response mechanisms to lock down compromised endpoints
* Periodically verify endpoint's compliance with the Security Policies

Features
--------
* Provide strong authentication of investigators
    * Actions must have a valid GPG signature, each investigator has a different
      key, for tracking.
* Provide a way to inspect remote systems for indicators of compromise (IOC).
  At the moment, this is limited to :
    * file by name
    * file content by regex
    * file hashes: md5, sha1, sha256, sha384, sha512, sha3_224,sha3_256,
      sha3_384, sha3_512
* Protect data security, investigate without intruding:
    * Raw data must not be readily available to investigators

Todo list:
* More agent modules
    * low level devices (memory, file system blocks, network cards)
    * established connections
    * firewall rules
    * lots more ...
* Provide response mechanisms, including:
    * dynamic firewall rules additions & removal
    * system password changes
    * process execution (execve) & destruction (kill)
* Input/Output IOCs, Yara, ... through the API
* Output results in standard format for alerting

Documentation
-------------
All documentation is available in the 'doc' directory.
