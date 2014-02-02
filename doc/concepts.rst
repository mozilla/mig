===================================================
Mozilla InvestiGator Concepts & Internal Components
===================================================
:Author: Julien Vehent <jvehent@mozilla.com>

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
        "Name":     "policy_system_rootpasswd_hash",
        "Target":   "linux",
        "Order":    "filechecker",
        "ScheduledDate":    "2013-12-01T12:00:00.0Z",
        "ExpirationDate":   "2013-12-01T13:00:00.0Z",
        "Arguments": {
            "root_password_hashed_and_salted" : {
                "Path": "/etc/shadow",
                "Type": "contains",
                "Value": "root:\\$(1|2a|5|6)\\$"
            }
        },
        "PGPSignature": "iQIcBA.....2cV=SFKc",
        "PGPSignatureDate": "2013-12-01T12:05:35.379828065Z"
    }

The parameters are:

* Name: a string that represents the action.
* Target: a search string that will be used by the scheduler to find the agents
  the action will run on.
* Order: the type of action, that typically matches a module name on the agent
* ScheduledDate and ExpirationDate: give a time window for the action to run
* Arguments: parameters that are passed to the agent's module. Their definition
  depends on the module. In the case of the filechecker module, arguments
  contains a list of checks defined by Path, Type and Value. Other modules have
  different syntaxes.
* PGPSignature: all of the parameters above are concatenated into a string and
  signed with the investigator's private GPG key. The signature is part of the
  action, and used by agents to verify that an action comes from a trusted
  investigator.
* PGPSignatureDate: is the date of the PGP signature, used as a timestamp of
  the action creation.

Actions files are submitted to the API or the Scheduler directly. Both
components will verify the PGPSignature before queueing the action for
execution. Additional, internal, attributes are then added to the action for
processing inside the MIG Platform. Those are defined as ExtendedAction.

Commands
~~~~~~~~

Upon processing of an Action, the scheduler will retrieve a list of agents to
send the action to. Action are then derived into Commands. A command contains an
action plus additional parameters that are specific to the target agent, such as
command processing timestamps, name of the agent queue on the message broker,
Action and Command unique IDs, status and results of the command. Below is an
example of the previous action ran against the agent named "fedbox":

.. code:: json

    {

        "ID": 5974340862284208059,
        "AgentName": "fedbox",
        "AgentQueueLoc": "linux.fedbox.55pvb3lm4a34e",
        "StartTime": "2013-12-01T12:33:48.887892346Z",
        "FinishTime": "2013-12-01T12:33:48.906556518Z",
        "Action": {
            "ID": 5974340861480881809,
            "Name": "policy_system_rootpasswd_hash",
            "Target": "linux",
            "Order": "filechecker",
            "ScheduledDate":    "2013-12-01T12:00:00.0Z",
            "ExpirationDate":   "2013-12-01T13:00:00.0Z",
            "Arguments": {
                "root_password_hashed_and_salted": {
                    "Path": "/etc/shadow",
                    "Type": "contains",
                    "Value": "root:\\$(1|2a|5|6)\\$"
                }
            },
            "PGPSignature": "iQIcBA.....2cV=SFKc",
            "PGPSignatureDate": "2013-12-01T12:05:35.379828065Z"
        },
        "Results": {
            "root_password_hashed_and_salted": {
                "Files": [
                    "/etc/shadow"
                ],
                "MatchCount": 1,
                "TestedFiles": 1
            }
        },
        "Status": "succeeded"
    }


As you can see, the action parameters are copied verbatim into the command, and
passed to the agent. The scheduler added IDs for both the command and the
action, because one action will spawn multiple commands. The results of the
command show that the file '/etc/shadow' positively matched exactly one time.
The command as succeeded. Had a failure happened on the agent, the scheduler
would have been notified and the status would be one of "succeeded", "failed",
"timeout" or "cancelled".

Agent registration process
--------------------------

Agent upgrade process
---------------------
