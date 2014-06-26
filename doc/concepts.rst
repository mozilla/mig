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
* Investigators: humans who use clients to investigate things on agents

An investigator uses a client (such as the MIG Console) to communicate with
the API. The API interfaces with the Database and the Scheduler.
When an action is created by an investigator, the API receives it and writes
it into the spool of the scheduler (they share it via NFS). The scheduler picks
it up, creates one command per target agent, and sends those commands to the
relays (running RabbitMQ). Each agent is listening on its own queue on the relay.
The agents execute their commands, and return the results through the same
relays (same exchange, different queues). The scheduler writes the results into
the database, where the investigator can access them through the API.
The agents also use the relays to send heartbeat at regular intervals, such that
the scheduler always knows how many agents are alive at a given time.

The end-to-end workflow is:

 ::

    {investigator} -https-> {API} -nfs-> {Scheduler} -amqps-> {Relays} -amqps-> {Agents}
                                \           /
                              sql\         /sql
                                 {DATABASE}

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
			"family": "compliance",
			"ref": "syslowaudit1"
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
defined in the database schema and are used to track the action status.

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
            "minimumweight": 2,
            "investigators": {
                "Bob Kelso": {
                    "fingerprint": "E60892BB9BD...",
                    "weight": 2
                },
                "John Smith": {
                    "fingerprint": "9F759A1A0A3...",
                    "weight": 1
                }
            }
        }
    }

`investigators` contains a list of users with their PGP fingerprints, and their
weight, an integer that represents their access level.
When an agent receives an action that calls the filechecker module, it will
first verify the signatures of the action, and then validates that the signers
are authorized to perform the action. This is done by summing up the weights of
the signatures, and verifying that they equal or exceed the minimum required
weight.

Thus, in the example above, investigator John Smith cannot issue a filechecker
action alone. His weight of 1 doesn't satisfy the minimum weight of 2 required
by the filechecker permission. Therefore, John will need to ask investigator Bob
Kelso to sign his action as well. The weight of both investigators are then
added, giving a total of 3, which satisfies the minimum weight of 2.

This method gives ample flexibility to require multiple signatures on modules,
and ensure that one investigator cannot perform sensitive actions on remote
endpoints without the permissions of others.

The default permission `default` can be used as a default for all modules. It
has the following syntax:

.. code:: json

	{
		"default": {
			"minimumweight": 2,
			"investigators": { ... }
			]
		}
	}

The `default` permission is overridden by module specific permissions.

The ACL is currently applied to modules. In the future, ACL will have finer
control to authorize access to specific functions of modules. For example, an
investigator could be authorized to call the `regex` function of filechecker
module, but only in `/etc`. This functionality is not implemented yet.

Extracting PGP fingerprints from public keys
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

On Linux, the `gpg` command can easily display the fingerprint of a key using
`gpg --fingerprint <key id>`. For example:

.. code:: bash

	$ gpg --fingerprint jvehent@mozilla.com
	pub   2048R/3B763E8F 2013-04-30
		  Key fingerprint = E608 92BB 9BD8 9A69 F759  A1A0 A3D6 5217 3B76 3E8F
	uid                  Julien Vehent (personal) <julien@linuxwall.info>
	uid                  Julien Vehent (ulfr) <jvehent@mozilla.com>
	sub   2048R/8026F39F 2013-04-30


You should always verify the trustworthiness of a key before using it:

.. code:: bash

	$ gpg --list-sigs jvehent@mozilla.com
	pub   2048R/3B763E8F 2013-04-30
	uid                  Julien Vehent (personal) <julien@linuxwall.info>
	sig 3        3B763E8F 2013-06-23  Julien Vehent (personal) <julien@linuxwall.info>
	sig 3        28A860CE 2013-10-04  Curtis Koenig <ckoenig@mozilla.com>
	.....

We want to extract the fingerprint, and obtain a 40 characters hexadecimal
string that can used in permissions.

.. code:: bash

	$gpg --fingerprint --with-colons jvehent@mozilla.com |grep '^fpr'|cut -f 10 -d ':'
	E60892BB9BD89A69F759A1A0A3D652173B763E8F

Agent registration process
--------------------------

Agent upgrade process
---------------------
MIG supports upgrading agents in the wild. The upgrade protocol is designed with
security in mind. The flow diagram below presents a high-level view:

 ::

	Investigator          Scheduler             Agent             NewAgent           FileServer
	+-----------+         +-------+             +---+             +------+           +--------+
		  |                   |                   |                   |                   |
		  |    1.initiate     |                   |                   |                   |
		  |------------------>|                   |                   |                   |
		  |                   |  2.send command   |                   |                   |
		  |                   |------------------>| 3.verify          |                   |
		  |                   |                   |--------+          |                   |
		  |                   |                   |        |          |                   |
		  |                   |                   |        |          |                   |
		  |                   |                   |<-------+          |                   |
		  |                   |                   |                   |                   |
		  |                   |                   |    4.download     |                   |
		  |                   |                   |-------------------------------------->|
		  |                   |                   |                   |                   |
		  |                   |                   | 5.checksum        |                   |
		  |                   |                   |--------+          |                   |
		  |                   |                   |        |          |                   |
		  |                   |                   |        |          |                   |
		  |                   |                   |<-------+          |                   |
		  |                   |                   |                   |                   |
		  |                   |                   |      6.exec       |                   |
		  |                   |                   |------------------>|                   |
		  |                   |  7.return own PID |                   |                   |
		  |                   |<------------------|                   |                   |
		  |                   |                   |                   |                   |
		  |                   |------+ 8.mark     |                   |                   |
		  |                   |      | agent as   |                   |                   |
		  |                   |      | upgraded   |                   |                   |
		  |                   |<-----+            |                   |                   |
		  |                   |                   |                   |                   |
		  |                   |    9.register     |                   |                   |
		  |                   |<--------------------------------------|                   |
		  |                   |                   |                   |                   |
		  |                   |------+10.find dup |                   |                   |
		  |                   |      |agents in   |                   |                   |
		  |                   |      |registrations                   |                   |
		  |                   |<-----+            |                   |                   |
		  |                   |                   |                   |                   |
		  |                   |    11.send command to kill PID old agt|                   |
		  |                   |-------------------------------------->|                   |
		  |                   |                   |                   |                   |
		  |                   |  12.acknowledge   |                   |                   |
		  |                   |<--------------------------------------|                   |

All upgrade operations are initiated by an investigator (1). The upgrade is
triggered by an action to the upgrade module with the following parameters:

.. code:: json

    "Operations": [
        {
            "Module": "upgrade",
            "Parameters": {
                "linux/amd64": {
                    "to_version": "16eb58b-201404021544",
                    "location": "http://localhost/mig/bin/linux/amd64/mig-agent",
                    "checksum": "31fccc576635a29e0a27bbf7416d4f32a0ebaee892475e14708641c0a3620b03"
                }
            }
        }
    ],

* Each OS family and architecture have their own parameters (ex: "linux/amd64",
  "darwin/amd64", "windows/386", ...). Then, in each OS/Arch group, we have:
* to_version is the version an agent should upgrade to
* location points to a HTTPS address that contains the agent binary
* checksum is a SHA256 hash of the agent binary to be verified after download

The parameters above are signed using a standard PGP action signature.

The upgrade action is forwarded to agents (2) like any other action. The action
signature is verified by the agent (3), and the upgrade module is called. The
module downloads the new binary (4), verifies the version and checksum (5) and
installs itself on the system.

Assuming everything checks in, the old agent executes the binary of the new
agent (6). At that point, two agents are running on the same machine, and the
rest of the protocol is designed to shut down the old agent, and clean up.

After executing the new agent, the old agent returns a successful result to the
scheduler, and includes its own PID in the results.
The new agent starts by registering with the scheduler (7). This tells the
scheduler that two agents are running on the same node, and one of them must
terminate. The scheduler sends a kill action to both agents with the PID of the
old agent (8). The kill action may be executed twice, but that doesn't matter.
When the scheduler receives the kill results (9), it sends a new action to check
for `mig-agent` processes (10). Only one should be found in the results (11),
and if that is the case, the scheduler tells the agent to remove the binary of
the old agent (12). When the agent returns (13), the upgrade protocol is done.

If the PID of the old agent lingers on the system, an error is logged for the
investigator to decide what to do next. The scheduler does not attempt to clean
up the situation.

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
