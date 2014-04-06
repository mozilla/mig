===================================================
Mozilla InvestiGator Concepts & Internal Components
===================================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

MIG is a platform to perform remote forensic on endpoints. It is composed of:

* Agent: a program that runs on endpoints and receives commands to run locally
  from the MIG platform. Commands are ran by agents using modules, such as
  'filechecker'.
* Scheduler: a router and processor that receives orders and forward them to
  agents
* API: an interface to the MIG platform used by investigators
* Clients: clients are used by investigators to interact with the MIG platform
  via the API
* Queue: a message queueing daemon that passes messages between the scheduler
  and the agents
* Database: a storage backend used by the scheduler and the api

Below is a high-level view of the different components:

 ::

    ( )               signed actions
    \|/  +------+  -----------------------> +-------+
     |   |client|    responses              | A P I |
    / \  +------+ <-----------------------  +-----+-+       +--------+
    investigator                                  +-------->|  data  |
                                                            |        |
                                              action/command|--------|
                                                            |        |
                                                  +-------->|  base  |
                                                  |         |        |
                      signed commands     +-------+---+     +--------+
                                          |           |
                      +++++--------------+| SCHEDULER |
                      |||||               |           |
                      vvvvv               +-----------+
                    +-------+                  ^^^^^
                    |       |                  |||||
                    |message|+-----------------+++++
                    |-------|     command responses
                    |broker |
                    |       |
                    +-------+
                    ^^    ^ ^
                    ||    | |
       +------------+|    | +-----------------+
       |           +-+    +--+                |
       |           |         |                |
    +--+--+     +--+--+    +-+---+          +-+---+
    |agent|     |agent|    |agent|  .....   |agent|
    +-----+     +-----+    +-----+          +-----+

Actions and Commands are messages passed between the differents components.

Actions and Commands
--------------------

Actions
~~~~~~~

Actions are JSON files created by investigator to perform tasks on agents.

For example, an investigator who wants to verify than root passwords are hashed
and salted on linux systems, would use the following action:

.. code:: json

	{
		"name": "Compliance check for Auditd",
		"description": {
			"author": "Julien Vehent",
			"email": "ulfr@mozilla.com",
			"url": "https://some_example_url/with_details",
			"revision": 201402071200
		},
		"target": "linux",
		"threat": {
			"level": "info",
			"family": "compliance"
		},
		"operations": [
			{
				"module": "filechecker",
				"parameters": {
					"/etc/shadow": {
						"regex": {
							"root password strongly hashed and salted": [
								"root:\\$(2a|5|6)\\$"
							]
						}
					}
				}
			}
		],
		"syntaxversion": 1
	}

The parameters are:

* Name: a string that represents the action.
* Target: a search string that will be used by the scheduler to find the agents
  the action will run on.
* Description and Threat: additional fields to describe the action
* Operations: an array of operations, each operation calls a module with a set
  of parameters. The parameters syntax are specific to the module.
* SyntaxVersion: indicator of the action format used. Should be set to 1.

Upon generation, additional fields are appended to the action:

* PGPSignatures: all of the parameters above are concatenated into a string and
  signed with the investigator's private GPG key. The signature is part of the
  action, and used by agents to verify that an action comes from a trusted
  investigator. `PGPSignatures` is an array that contains one or more signature
  from authorized investigators.
* ValidFrom and ExpireAt: two dates that constrains the validity of the action
  to a UTC time window.

Actions files are submitted to the API or the Scheduler directly. The PGP
Signatures are always verified by the agents, and can optionally be verified by
other components along the way.
Additional attributes are added to the action by the scheduler. Those are
defined as `ExtendedAction` and are used to track the action status.

Commands
~~~~~~~~

Upon processing of an Action, the scheduler will retrieve a list of agents to
send the action to. One action is then derived into Commands. A command contains an
action plus additional parameters that are specific to the target agent, such as
command processing timestamps, name of the agent queue on the message broker,
Action and Command unique IDs, status and results of the command. Below is an
example of the previous action ran against the agent named
'myserver1234.test.example.net'.

.. code:: json

	{
		"action":        { ... signed copy of action ... }
		"agentname":     "myserver1234.test.example.net",
		"agentqueueloc": "linux.myserver1234.test.example.net.55tjippis7s4t",
		"finishtime":    "2014-02-10T15:28:34.687949847Z",
		"id":            5978792535962156489,
		"results": [
			{
				"elements": {
					"/etc/shadow": {
						"regex": {
							"root password strongly hashed and salted": {
								"root:\\$(2a|5|6)\\$": {
									"Filecount": 1,
									"Files": {},
									"Matchcount": 0
								}
							}
						}
					}
				},
				"extra": {
					"statistics": {
						"checkcount": 1,
						"checksmatch": 0,
						"exectime": "183.237us",
						"filescount": 1,
						"openfailed": 0,
						"totalhits": 0,
						"uniquefiles": 0
					}
				},
				"foundanything": false
			}
		],
		"starttime": "2014-02-10T15:28:34.118926659Z",
		"status": "succeeded"
	}


The results of the command show that the file '/etc/shadow' has not matched,
and thus "FoundAnything" returned "false.
While the result is negative, the command itself has succeeded. Had a failure
happened on the agent, the scheduler would have been notified and the status
would be one of "failed", "timeout" or "cancelled".

Access Control Lists
--------------------

Not all keys can perform all actions. The scheduler, for example, sometimes need
to issue specific actions to agents (such as during the upgrade protocol) but
shouldn't be able to perform more dangerous actions. This is enforced by
an Access Control List, or ACL, stored on the agents. An ACL describes who can
access what function of which module. It can be used to require multiple
signatures on specific actions, and limit the list of investigators allowed to
perform an action.

An ACL is composed of permissions, which are JSON documents hardwired into
the agent configuration. In the future, MIG will dynamically ship permissions
to agents.

Below is an example of a permission for the `filechecker` module:

.. code:: json

	{
		"filechecker": {
			"requiredsignatures": 1,
			"authoritativesigners": [
				"E60892BB9BD89A69F759A1A0A3D652173B763E8F"
			]
		}
	}

`authoritativesigners` contains the PGP fingerprint of the public key of an
investigator. When an agent receives an action that calls the filechecker
module, it will first verify the signature of the action, and then validates
that the signer is authorized to perform the action.

The default permission `default` can be used as a default for all modules. It
has the following syntax:

.. code:: json

	{
		"default": {
			"requiredsignatures": 1,
			"authoritativesigners": [
				"E60892BB9BD...",
				"9F759A1A0A3...",
				"A69F759A1A0..."
			]
		}
	}

The `default` permission is overridden by module specific permissions.

If a module requires multiple signatures, the `nonauthoritativesigners`
attribute can be used to list investigators that can sign, but which signature
isn't sufficient to launch the action. In addition, the attribute
`requiredauthoritativesigners` controls how many signatures from
`authoritativesigners` are required. If `requiredauthoritativesigners` is set to
0, and `requiredsignatures` is set to 2, then two `nonauthoritativesigners` can
sign and launch an action using this module without the approval of an
`authoritativesigners`, as shown below in the firewall permission.

.. code:: json

	{
		"firewall": {
			"requiredsignatures": 2,
			"requiredauthoritativesigners": 0
			"authoritativesigners": [
				"E60892BB9BD...",
				"9F759A1A0A3...",
				"A69F759A1A0..."
			],
			"nonauthoritativesigners": [
				"2FC05413E11...",
				"8AD5956347F..."
			]
		}
	}

The ACL is currently applied to modules. In the future, ACL will have finer
control to authorize access to specific functions of modules. For example, an
investigator could be authorized to call the `regex` function of filechecker
module, but only in `/etc`. This functionality is not implemented yet.

Extracting PGP fingerprints from public keys
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

On Linux, the `gpg` command can easily display the fingerprint of a key using
`gpg --fingerprint <key id`. For example:

.. code:: bash

	$ gpg --fingerprint jvehent@mozilla.com
	pub   2048R/3B763E8F 2013-04-30
		  Key fingerprint = E608 92BB 9BD8 9A69 F759  A1A0 A3D6 5217 3B76 3E8F
	uid                  Julien Vehent (personal) <julien@linuxwall.info>
	uid                  Julien Vehent (ulfr) <jvehent@mozilla.com>
	sub   2048R/8026F39F 2013-04-30

We want to extract the fingerprint and remove all spaces, in order to obtain a
40 characters hexadecimal string that can used in permissions.

.. code:: bash

	$ gpg --fingerprint jvehent@mozilla.com | grep fingerprint \
	| cut -d '=' -f 2 | sed  's/ //g'
	E60892BB9BD89A69F759A1A0A3D652173B763E8F

Agent registration process
--------------------------

Agent upgrade process
---------------------

Agent command execution flow
----------------------------

An agent receives a command from the scheduler on its personal AMQP queue (1).
It parses the command (2) and extracts all of the operations to perform.
Operations are passed to modules and executed asynchronously (3). Rather than
maintaining a state of the running command, the agent create a goroutine and a
channel tasked with receiving the results from the modules. Each modules
published its results inside that channel (4). The result parsing goroutine
receives them, and when it has received all of them, builds a response (5)
that is sent back to the scheduler(6).

When the agent is done running the command, both the channel and the goroutine
are destroyed.

 ::

             +-------+   [ - - - - - - A G E N T - - - - - - - - - - - - ]
             |command|+---->(listener)
             +-------+          |(2)
               ^                V
               |(1)         (parser)
               |               +       [ m o d u l e s ]
    +-----+    |            (3)|----------> op1 +----------------+
    |SCHED|+---+               |------------> op2 +--------------|
    | ULER|<---+               |--------------> op3 +------------|
    +-----+    |               +----------------> op4 +----------+
               |                                                 V(4)
               |(6)                                         (receiver)
               |                                                 |
               |                                                 V(5)
               +                                             (sender)
             +-------+                                           /
             |results|<-----------------------------------------'
             +-------+
