======================
MIG Agent Architecture
======================

.. sectnum::
.. contents:: Table of Contents

Initialization process
----------------------------
The agent tries to be as autonomous as possible. One of the goal is to ship
agents without requiring external provisioning tools, such as Chef or Puppet.
Therefore, the agent attempts to install itself as a service, and also supports
optional automatic upgrades via the `mig-loader`_ companion program.

.. _`mig-loader`: loader.rst

As a portable binary, the agent needs to detect the type of operating system
and init method that is used by an endpoint. Depending on the endpoint,
different initialization methods are used. The diagram below explains the
decision process followed by the agent.

.. image:: .files/mig-agent-initialization-process.png

Go does not provide support for running programs in the backgroud. On endpoints
that run upstart, systemd (linux) or launchd (darwin), this is not an issue
because the init daemon takes care of running the agent in the background,
rerouting its file descriptors and restarting on crash. On Windows and System-V,
however, the agent daemonizes by forking itself into `foreground` mode, and
re-forking itself on error (such as loss of connectivity to the relay).
On Windows and System-V, if the agent is killed, it will not be restarted
automatically.

Registration process
~~~~~~~~~~~~~~~~~~~~

The initialization process goes through several environment detection steps
which are used to select the proper init method. Once started, the agent will
send a heartbeat to the public relay, and also store that heartbeat in its
`run` directory. The location of the `run` directory is platform specific.

* windows: C:\Windows\
* darwin: /Library/Preferences/mig/
* linux: /var/run/mig/

Below is a sample heartbeat message from a linux agent stored in
`/var/run/mig/mig-agent.ok`.

.. code:: json

	{
		"destructiontime": "0001-01-01T00:00:00Z",
		"environment": {
			"arch": "amd64",
			"ident": "Red Hat Enterprise Linux Server release 6.5 (Santiago)",
			"init": "upstart"
		},
		"heartbeatts": "2014-07-31T14:00:20.00442837-07:00",
		"name": "someserver.example.net",
		"os": "linux",
		"pid": 26256,
		"queueloc": "linux.someserver.example.net.5hsa811oda",
		"starttime": "2014-07-30T21:34:48.525449401-07:00",
		"version": "201407310027+bcbdd94.prod"
	}

The agent sends information about the OS configuration and it's environment
to the scheduler periodically. This includes information like the hostname
of the system it is running on, IP addresses assigned, AWS instance related
information, and others. It's possible on an endpoint this changes while the
agent is running. For example, a new IP address could be assigned via DHCP.
The agent periodically checks the system; if changes to the environment
are detected the heartbeat message will automatically be updated to include
those changes. The frequency environment checks occur can be controlled
through the ``refreshenv`` configuration option in the agent configuration
file, or the ``REFRESHENV`` variable in the agent built-in configuration.

Check-In mode
~~~~~~~~~~~~~
In infrastructure where running the agent as a permanent process is not
acceptable, it is possible to run the agent as a cron job. By starting the
agent with the flag **-m agent-checkin**, the agent will connect to the
configured relay, retrieve and run outstanding commands, and exit after 10
seconds of inactivity.

Communication with modules
--------------------------

Upon processing of an action, the scheduler will retrieve a list of agents to
send the action to. One action is then derived into multiple commands and sent
to agents.

An agent receives a command from the scheduler on its personal AMQP queue (1).
It parses the command (2) and extracts all of the operations to perform.
Operations are passed to modules and executed in parallel (3). Rather than
maintaining a state of the running command, the agent create a goroutine and a
channel tasked with receiving the results from the modules. Each modules
published its results inside that channel (4). The result parsing goroutine
receives them, and when it has received all of them, populates the `results` (5)
array of the command with the results from each module, and send the command
back to the scheduler(6).

When the agent is done running the command, both the channel and the goroutine
are destroyed.

 ::

                 +-------+   [ - - - - - - A G E N T - - - - - - - - - - - - ]
                 |command|+---->(listener)
                 +-------+          |(2)
                   ^                V
                   |(1)         (parser)
                   |               +       [ m o d u l e s ]
    +---------+    |            (3)|----------> op1 +----------------+
    |SCHEDULER|+---+               |------------> op2 +--------------|
    |         |<---+               |--------------> op3 +------------|
    +---------+    |               +----------------> op4 +----------+
                   |                                                 V(4)
                   |(6)                                         (receiver)
                   |                                                 |
                   |                                                 V(5)
                   +                                             (publisher)
                 +-------+                                           /
                 |results|<-----------------------------------------'
                 +-------+

The command received by the agent is composed of a copy of the action described
previously, but signed with the private key of a trusted investigator. It also
contains additional parameters that are specific to the targetted agent, such as
command processing timestamps, name of the agent queue on the message broker,
action and command unique IDs and status and results of the command. Below is an
command derived from the root password checking action, and ran on the host named
'host1.example.net'.

.. code:: json

	{
	  "id": 1.427392971126604e+18,
	  "action": { ... SIGNED COPY OF THE ACTION ... },
	  "agent": {
		"id": 1.4271760437936648e+18,
		"name": "host1.example.net",
		"queueloc": "linux.host1.example.net.981alsd19aos1984",
		"mode": "daemon",
		"version": "20150324+0d0f88c.prod"
	  },
	  "status": "success",
	  "results": [
		{
		  "foundanything": true,
		  "success": true,
		  "elements": {
			"root_passwd_hashed_or_disabled": [
			  {
				"file": "/etc/shadow",
				"fileinfo": {
				  "lastmodified": "2015-02-07 01:51:07.17850601 +0000 UTC",
				  "mode": "----------",
				  "size": 1684
				},
				"search": {
				  "contents": [
					"root:(\\*|!|\\$(1|2a|5|6)\\$).+"
				  ],
				  "options": {
					"matchall": false,
					"matchlimit": 0,
					"maxdepth": 0
				  },
				  "paths": [
					"/etc"
				  ]
				}
			  }
			]
		  },
		  "statistics": {
			"exectime": "2.017849ms",
			"filescount": 1,
			"openfailed": 0,
			"totalhits": 1
		  },
		  "errors": null
		}
	  ],
	  "starttime": "2015-03-26T18:02:51.126605Z",
	  "finishtime": "2015-03-26T18:03:00.671232Z"
	}

The results of the command show that the file '/etc/shadow' has matched, and
thus "FoundAnything" returned "True".

The invocation of the file module has completed successfully, which is
represented by **results->0->success=true**. In our example, there is only one
operation in the **action->operations** array, so only one result is present.
When multiple operations are performed, each has its results listed in a
corresponding entry of the results array (operations[0] is in results[0],
operations[1] in results[1], etc...).

Finally, the agent has performed all operations in the operations array
successfully, and returned **status=success**. Had a failure happened on the
agent, the returned status would be one of "failed", "timeout" or "cancelled".

Command expiration & timeouts
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

To prevent abuse of resources, agents will kill long-running modules after a
given period of time. That timeout is hardcoded in the agent configuration
at compile time and defaults to 5 minutes.

.. code:: go

	// timeout after which a module run is killed
	var MODULETIMEOUT time.Duration = 300 * time.Second

That timeout represents the **maximum** execution time of a single operation. If
an action contains 3 operations, each operation gets its own timeout. But because
operations run in parallel in the agent, the maximum runtime of an action should
be very close to the value of MODULETIMEOUT.

In a typical deployment, it is safe to increase MODULETIMEOUT to allow for
longer operations. A value of 20 minutes is usual. Make sure to fine tune this
to your environment, and get the approval of your ops team because mig-agent
may end up consuming resources (but never more than 50% of the cpu available on
a system).

Oftentimes, an investigator will want a timeout that is much shorter than the value
of MODULETIMEOUT. In the MIG command line, the flag `-e` controls the
expiration. It defaults to 5 minutes but can be set to 30 seconds for simple
investigations. When that happens, the agent will calculate an appropriate expiration
for the operations being run. If the expiration set on the action is set to 30 seconds,
the agent will kill operations that run for more than 30 seconds.

If the expiration is larger than the value of MODULETIMEOUT (for example, 2
hours), then MODULETIMEOUT is used. Setting a long expiration may be useful to
allow agents that only check in periodically to pick up actions long after they
are launched.

Agent/Modules message format
~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The agent accepts different classes of inputs on stdin, as one-line JSON objects.
The most common one is the ``parameters`` class, but it could also receive a
``stop`` input that indicates that the module should stop its execution immediately.
The format of module input messages is defined by ``modules.Message``.

.. code:: go

	// Message defines the input messages received by modules.
	type Message struct {
		Class      string      // represent the type of message being passed to the module
		Parameters interface{} // for `parameters` class, this interface contains the module parameters
	}

	const (
		MsgClassParameters string = "parameters"
		MsgClassStop       string = "stop"
	)

When the agent receives a command to pass to a module for execution, it
extracts the operation parameters from ``Command.Action.Operations[N].Parameters``
and copies them into ``Message.Parameters``. It then sets ``Message.Class`` to
``modules.MsgClassParameters``, marshals the struct into JSON, and pass the
resulting ``[]byte`` to the module as an IO stream.

Agent upgrade process via mig-loader
------------------------------------
MIG supports upgrading agents in the wild through the use of the companion
program mig-loader. Using mig-loader is optional; you don't need to use
mig-loader in your environment if you want to upgrade agents yourself.

The following is a high level diagram of how the loader interacts with the
API and the agent during the upgrade process. Note this diagram focuses on
the agent being upgraded, but it could be any file in the manifest such as
the certificates, agent configuration, or loader. In all cases changes to
anything will result in a respawn of any running agent by the loader.

::

        /------ Endpoint ---------\
        Agent                Loader              API
        +---+                +----+             +--+
        |                    |                     |
        |                    | 1. request manifest |
        |                    |-------------------->|------+
        |                    |                     |      | 2. update loader
        | 3. valid  +--------|                     |      | record in database
        | manifest  |        |                     |<-----+
        | sig?      +------->|                     |
        |                    |                     |
        | 4. does   +--------|                     |
        | current   |        |                     |
        | agent     |        |                     |
        | match?    +------->|                     |
        |                    |                     |
        |                    | 5. fetch new agent  |
        |                    |    or other files   |
        |                    |    from manifest    |
        |                    |    that dont match  |
        |                    |-------------------->|
        |                    |                     |
        | 6. stage  +--------|                     |
        | agent on  |        |                     |
        | disk      +------->|                     |
        |                    |                     |
        | 7. agent  +--------|                     |
        | SHA256    |        |                     |
        | matches   |        |                     |
        | manifest? +------->|                     |
        |                    |                     |
        |  8. install agent  |                     |
        |<-------------------|                     |
        |                    |                     |
        |  9. stop old agent |                     |
        |<-------------------|                     |
        |                    |                     |
        | 10. start new      |                     |
        |<-------------------|                     |
        |                    |                     |

For more information on how MIG loader can be used see the relevant
documentation in `MIG LOADER`_.

.. _`MIG LOADER`: loader.rst
