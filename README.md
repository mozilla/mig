= Mozilla Investigator =
== Messenging ==
An `Action` is the top-level objects. Actions are created by MIG users,
and processed by a scheduler. The internal representation of an Action
is a `mig.Action`. A scheduler takes one `mig.Action` and generates one or
multiple `mig.Command` from it. `mig.Command` are sent to agents, executed,
and returned to the scheduler with a `mig.Command.Result`.

Both `mig.Action` and `mig.Commands` are stored in the database in JSON format.

Action example, as written by a MIG user.

    {
        "Name":     "testaction",
        "Target":   "linux",
        "Check":    "filechecker",
        "Arguments": {
            "test1": {
                "Path": "/etc/passwd",
                "Type": "contains",
                "Value": "^ulfr"
            },
            "test2": {
                "Path": "/etc/group",
                "Type": "contains",
                "Value": "^puppet"
            },
            "test3": {
                "Path": "/opt/agent",
                "Type": "sha256",
                "Value": "128611f7c30c7f5a0cd7ba9d0a02fa891204179b49f0da66b5c96114474309c9"
            },
            "test4": {
                "Path": "/usr/lib",
                "Type": "md5",
                "Value": "451bbb983d522af59204417b0af15fd9"
            }
        },
        "RunDate":  "immediate",
        "Expiration":  "30m"
    }

Internal representation of an Actio is a `mig.Action`. The scheduler adds
a `mig.Action.ID` that is unique to each launched Action. A list of command
IDs generated from the Action is also kept in `mig.Action.CommandIDs`.

    {
        "Name": "testaction",
        "ID": 5941327413322435544,
        "Expiration": "30m",
        "RunDate": "immediate",
        "Target": "linux",
        "Check": "filechecker",
        "Arguments": {
            "test1": {
                "Path": "/etc/passwd",
                "Type": "contains",
                "Value": "^ulfr"
            },
            "test2": {
                "Path": "/etc/group",
                "Type": "contains",
                "Value": "^puppet"
            },
            "test3": {
                "Path": "/opt/agent",
                "Type": "sha256",
                "Value": "128611f7c30c7f5a0cd7ba9d0a02fa891204179b49f0da66b5c96114474309c9"
            },
            "test4": {
                "Path": "/usr/lib",
                "Type": "md5",
                "Value": "451bbb983d522af59204417b0af15fd9"
            }
        },
        "CommandIDs": [
            5941327413072766544,
            5941327411825102210,
            5941327413098967745
        ]
    }

The `mig.Action` above generates 3 individual `mig.Command` that are sent to
their respective agents. Here's an example of `mig.Command` sent to `agt1`

    {
        "AgentName": "agt1",
        "AgentQueueLoc": "linux.agt1",
        "ID": 5941327413072766544,
        "Action": {
            "Name": "testaction",
            "ID": 5941327413322435544,
            "Expiration": "30m",
            "RunDate": "immediate",
            "Target": "linux",
            "Check": "filechecker",
            "CommandIDs": null,
            "Arguments": {
                "test1": {
                    "Path": "/etc/passwd",
                    "Type": "contains",
                    "Value": "^ulfr"
                },
                "test2": {
                    "Path": "/etc/group",
                    "Type": "contains",
                    "Value": "^puppet"
                },
                "test3": {
                    "Path": "/opt/agent",
                    "Type": "sha256",
                    "Value": "128611f7c30c7f5a0cd7ba9d0a02fa891204179b49f0da66b5c96114474309c9"
                },
                "test4": {
                    "Path": "/usr/lib",
                    "Type": "md5",
                    "Value": "451bbb983d522af59204417b0af15fd9"
                }
            }
        },
        "Results": {
            "test1": {
                "Files": [
                    "/etc/passwd"
                ],
                "MatchCount": 1,
                "TestedFiles": 1
            },
            "test2": {
                "Files": [
                    "/etc/group"
                ],
                "MatchCount": 1,
                "TestedFiles": 1
            },
            "test3": {
                "Files": null,
                "MatchCount": 0,
                "TestedFiles": 1
            },
            "test4": {
                "Files": [
                    "/usr/lib/libperl.so.5.14.2"
                ],
                "MatchCount": 1,
                "TestedFiles": 4406
            }
        }
    }

`mig.Command` contains a full copy of the `mig.Action` that generated it.
It also has its own ID, such that the tuple `mig.Command.Action.ID`-`mig.Command.ID`
can be used to uniquely identify an action and a command.

As soon as a Command is sent over the network, it is stored in the database.

== Database ==
Data is stored in MongoDB. Actions and Commands have their own separate collections.

The query below retrieves a list of agents that have executed a specific action ID,
and returned one positive result on the check called MOZSYSCOMPLOWENCCRED.
```
> var commands = db.commands.find(
... {
... 'action.id': 5952800863268821556,
... 'results.MOZSYSCOMPLOWENCCRED.MatchCount': 1
... }
... );
```
The results are stored in the `commands` variable.
The list of agents can be printed in mongo shell directly:
```
> commands.forEach(function(command){print(command.agentname);});
```
See MongoDB reference documentation for a full explanation of the query language.
