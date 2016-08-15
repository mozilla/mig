===========
MIG CONSOLE
===========

.. sectnum::
.. contents:: Table of contents

`mig-console` is a terminal-based client to the MIG API. It provides a complete
access to data stored in MIG and exposed by the API, in an interface that is
easy to use. Similar to the command line client, `mig-console` reads its
configuration from $HOME/.migrc and will prompt the user to create one if it
doesn't exists.

The base screen of the API presents a dashboard of the current state of the
infrastructure. It includes the number of agents and endpoints, broken down
by version of the agent ,and a list of the latest actions that have run.

Underneath the dashboard, the user is invited to interact in a prompt that
starts with **mig>**.

.. code:: bash

	$ mig-console -c ~/.migrc-lnw

	## ##                                     _.---._     .---.
	# # # /-\ ---||  |    /\         __...---' .---. '---'-.   '.
	#   #|   | / ||  |   /--\    .-''__.--' _.'( | )'.  '.  '._ :
	#   # \_/ ---| \_ \_/    \ .'__-'_ .--'' ._'---'_.-.  '.   '-'.
		 ###                         ~ -._ -._''---. -.    '-._   '.
		  # |\ |\    /---------|          ~ -.._ _ _ _ ..-_ '.  '-._''--.._
		  # | \| \  / |- |__ | |                       -~ -._  '-.  -. '-._''--.._.--''.
		 ###|  \  \/  ---__| | |                            ~ ~-.__     -._  '-.__   '. '.                                                                                                                                                                   :
			  #####                                               ~~ ~---...__ _    ._ .' '.
			  #      /\  --- /-\ |--|----                                    ~  ~--.....--~
			  # ### /--\  | |   ||-\  //
			  #####/    \ |  \_/ |  \//__
	+------
	| Agents & Endpoints summary:
	| * 2 online agents on 2 endpoints
	| * 0 idle agents on 0 endpoints
	| * 0 endpoints are running 2 or more agents
	| * 0 endpoints appeared over the last 7 days
	| * 0 endpoints disappeared over the last 7 days
	| * 0 endpoints have been flapping
	| Online agents by version:
	| * version 20150909+f2456f5.dev: 2 agents
	| Idle agents by version:
	|
	| Latest Actions:
	| ----  ID  ---- + ----         Name         ---- + -Sent- + ----    Date    ---- + ---- Investigators ----
	| 4999271350274    file -c /home/ulfr/.migrc-l...        2   2015-09-29T15:40:35Z   Julien Vehent
	| 4964811669519    file -c /home/ulfr/.migrc-l...        2   2015-09-23T13:37:16Z   Julien Vehent
	| 4964811669506    file -c /home/ulfr/.migrc-l...        2   2015-09-23T13:37:03Z   Julien Vehent
	| 4964764024853    file -c /home/ulfr/.migrc-l...        2   2015-09-23T13:25:26Z   Julien Vehent
	| 4964764024834    file -c /home/ulfr/.migrc-l...        2   2015-09-23T13:24:57Z   Julien Vehent
	| 4949328330767    file -c /home/ulfr/.migrc-l...        2   2015-09-20T19:59:39Z   Julien Vehent
	| 4949328330754    file -c /home/ulfr/.migrc-l...        2   2015-09-20T19:59:25Z   Julien Vehent
	| 4948324450316    file -c /home/ulfr/.migrc-l...        2   2015-09-20T19:45:51Z   Julien Vehent
	| 4948324450307    file -c /home/ulfr/.migrc-l...        2   2015-09-20T19:33:17Z   Julien Vehent
	| 4947944865794    file -c /home/ulfr/.migrc-l...        2   2015-09-20T14:07:36Z   Julien Vehent
	+------

	Connected to https://mig.example.net/api/v1/. Exit with ctrl+d. Type help for help.
	mig>

Entering **help** here, or on any other mode of the console, provides the user
with a list of available functionalities::

	mig> help
	The following orders are available:
	action <id>             enter interactive action reader mode for action <id>
	agent <id>              enter interactive agent reader mode for agent <id>
	create action           create a new action
	create investigator     create a new investigator, will prompt for name and public key
	command <id>            enter command reader mode for command <id>
	exit                    leave
	help                    show this help
	history <count>         print last <count> entries in history. count=10 by default.
	investigator <id>       enter interactive investigator management mode for investigator <id>
	query <uri>             send a raw query string, without the base url, to the api
	search <search>         perform a search. see "search help" for more information.
	showcfg                 display running configuration
	status                  display platform status: connected agents, latest actions, ...

For example, let's review action id 4999271350274::

	mig> action 4999271350274
	Entering action reader mode. Type exit or press ctrl+d to leave. help may help.
	Action: 'file -c /home/ulfr/.migrc-lnw -path /etc -name ^passwd -content julien '.
	Launched by 'Julien Vehent' on '2015-09-29 11:40:31.792627 -0400 EDT'.
	Status 'completed'.
	2 sent, 2 done, 2 succeeded
	action 274>

When entering the action reader mode, a short summary of the action is displayed
to the user. We can see that the action was launched by "Julien Vehent" on 2 agents
and that it completed. The prompt is changed to **action 274** to indicate that
we are now in action reader mode. Entering **help** here provides a list of
functionalities specific to that mode::

	The following orders are available:
	command <id>    jump to command reader mode for command <id>

	copy            enter action launcher mode using current action as template

	counters        display the counters of the action

	details         display the details of the action, including status & times

	exit            exit this mode (also works with ctrl+d)

	help            show this help

	investigators   print the list of investigators that signed the action

	json            show the json of the action

	list <show>     returns the list of commands with their status
					<show>: * set to "all" to get all results (default)
							* set to "found" to only display positive results
							* set to "notfound" for negative results
					list can be followed by a 'filter' pipe:
					ex: ls | grep server1.(dom1|dom2) | grep -v example.net

	r               refresh the action (get latest version from upstream)

	results <show> <render> display results of all commands
							<show>: * set to "all" to get all results (default)
									* set to "found" to only display positive results
									* set to "notfound" for negative results
							<render>: * set to "text" to print results in console (default)
									  * set to "map" to generate an open a google map

	times           show the various timestamps of the action

It is possible to review results by entering **results**. If we only want results
from agents that have found *something*, we can use **results found**. If we
want those results inside a Google Map, enter **results found map**.

The raw json of the action is available via **json**.

To get a list of agents on which the action ran, use **list all**. You can filter
the list using **list found** and **list notfound** to only get agents that have,
or have not, found something. This command also supports a very basic `grep`.

To continue, let's list the agents that have found *something*::

	action 274> list found
	..
	---- Command ID ----    ---- Agent Name & ID----
		   4999268991155    server1.example.net [4942082344151]
		   4999268991154    server2.example.net [4942082360682]
	2 agents have found things

We can inspect the command that ran on server1 by entering its ID::

	action 274> command 4999268991155
	Entering command reader mode. Type exit or press ctrl+d to leave. help may help.
	Command 4999268991155 ran on agent 'server1.example.net' based on action 'file -c /home/ulfr/.migrc-lnw -path /etc -name ^passwd -content julien '
	command 155>

As you can see, the console mode has changed from `action 274` to
`command 155`. This mode has its own set of functionalities that you
can explore via **help**.

Creating actions
----------------

The console provides a fine grained action generation interface. There
are two ways to create a new action:

1. From the **mig>** mode, create a new empty action using **create action**.

2. From the **action>** reader mode, when reviewing a previous action,
   copy it to a new action using **copy**.

Both methods enter the action launcher mode. Method 2 only enters this
mode with an preset action, rather than an empty one::

	mig> create action
	Entering action launcher with empty template
	Type exit or press ctrl+d to leave. help may help.
	launcher> help
	The following orders are available:
	addoperation <module>   append a new operation of type <module> to the action operations
	listagents              list agents targetted by an action
	deloperation <opnum>    remove operation numbered <opnum> from operations array, count starts at zero
	details                 display the action details
	exit                    exit this mode
	help                    show this help
	json <pack>             show the json of the action
	launch <nofollow>       launch the action. to return before completion, add "nofollow"
	load <path>             load an action from a file at <path>
	setname <name>          set the name of the action
	settarget <target>      set the target
	settimes <start> <stop> set the validity and expiration dates
	sign                    PGP sign the action
	times                   show the various timestamps of the action

Action parameters can be edited prior to launching it:

* **setname** sets the name field of the action to a new string::

	launcher> setname Test action that pings google.com

* **settarget** sets the target of the action. Targets can either be a targeting
  string, or a macro if defined in migrc. The target is evaluated
  right away, and a list of targeted agents can be obtained via **listagents**::

	launcher> settarget environment->>'os'='linux' and mode='daemon'
	2 agents will be targetted. To get the list, use 'listagents'

	launcher> listagents
	----    ID      ---- + ----         Name         -------
		   4942082360682   server1.example.net
		   4942082344151   server2.example.net

* **settimes** defines the validity period of the action using
  <start> and <stop> parameters. <start> can be set to "now", and <stop>
  can be a duration relative to "now"::

	launcher> settimes now +10m
	launcher> times
	Valid from   '2015-10-06 13:09:36.189134664 +0000 UTC' until '2015-10-06 13:20:36.189134664 +0000 UTC'
	Started on   '0001-01-01 00:00:00 +0000 UTC'
	Last updated '0001-01-01 00:00:00 +0000 UTC'
	Finished on  '0001-01-01 00:00:00 +0000 UTC'

* **addoperation <module>** is used to add a new operation to the action.
  <module> must be a know MIG module, such as `file`, `netstat`, `memory`,
  `scribe`, `ping`, `timedrift`, etc... When adding an operation, the console
  enters a new module-specific mode that takes the operation parameters. For
  example, this is how to add a ping operation that sends 5 TCP requests to
  google.com::

	launcher> addoperation ping
	Ping module checks connectivity between an endpoint and a remote host. It supports
	icmp, tcp and udp ping. See doc at http://mig.mozilla.org/doc/module_ping.html

	d <ip/fqdn>     Destination Address can be ipv4, ipv6 or FQDN
					example: d www.mozilla.org
							 d 63.245.217.105

	dp <port>       For TCP and UDP, specifies the port to test connectivity to
					example: dp 53

	p <protocol>    Protocol to use for the ping. This can be "icmp", "tcp" or "udp"
					example: p udp

	c <count>       Number of ping/connection attempts. Defaults to 3.
					example: c 5

	t <timeout>     Connection timeout in seconds. Defaults to 5.
					example: t 10
	ping> p tcp
	ping> d google.com
	ping> c 5
	ping> done
	Inserting ping operation with parameters:
	{
	  "module": "ping",
	  "parameters": {
		"destination": google.com",
		"protocol": "tcp",
		"count": 5
	  }
	}

* **deloperation** removes an operation from the action operations list.
  Use **json** to visualize the list, and remove an operation using its
  position in the list, starting with zero::

	launcher> json
	{
	  "id": 0,
	  "name": "Test action that pings google.com",
	  "target": "environment-\u003e\u003e'os'='linux' and mode='daemon'",
	  "description": {},
	  "threat": {},
	  "validfrom": "2015-10-06T13:09:36.189134664Z",
	  "expireafter": "2015-10-06T13:20:36.189134664Z",
	  "operations": [
		{
		  "module": "ping",
		  "parameters": {
			"destination": "google.com",
			"protocol": "tcp",
			"count": 5
		  }
		}
	  ],
	  "pgpsignatures": null,
	  "starttime": "0001-01-01T00:00:00Z",
	  "finishtime": "0001-01-01T00:00:00Z",
	  "lastupdatetime": "0001-01-01T00:00:00Z",
	  "counters": {}
	}

	launcher> deloperation 0

	launcher> json
	{
	  "id": 0,
	  "name": "Test action that pings google.com",
	  "target": "environment-\u003e\u003e'os'='linux' and mode='daemon'",
	  "description": {},
	  "threat": {},
	  "validfrom": "2015-10-06T13:09:36.189134664Z",
	  "expireafter": "2015-10-06T13:20:36.189134664Z",
	  "operations": [],
	  "pgpsignatures": null,
	  "starttime": "0001-01-01T00:00:00Z",
	  "finishtime": "0001-01-01T00:00:00Z",
	  "lastupdatetime": "0001-01-01T00:00:00Z",
	  "counters": {}
	}

When ready to launch the action, type **launch**. The console will enter
follower mode and print the progress of the action. When completed, the
console will directly enter action reader mode, where you can type **results**
to view the results::

	launcher> launch
	Action 'Test action that pings google.com' successfully launched with ID '5033038708749' on target 'environment->>'os'='linux' and mode='daemon''
	Following action ID 5033038708749.status=inflight...50%.status=completed
	- 100.0% done in 10.004244071s
	2 sent, 2 done, 2 succeeded

	Entering action reader mode. Type exit or press ctrl+d to leave. help may help.
	Action: 'Test action that pings google.com'.
	Launched by 'Julien Vehent' on '2015-10-06 09:12:31.608546 -0400 EDT'.
	Status 'completed'.
	2 sent, 2 done, 2 succeeded
	action 749> results
	server2.example.net  ping of google.com failed. Target is no reachable.
	server2.example.net command success
	server1.example.net  ping of google.com failed. Target is no reachable.
	server1.example.net command success
	2 agents have all results

Searching
---------

Return to the base mode using ctrl+d or **exit**::

	command 155> exit
	exit
	action 274> exit
	exit
	mig>

When in **mig>** mode, you can use the **search** functionality to
explore passed actions and commands, or find investigators. Details
of how to run searches can be obtains via **mig> search help**.

Searches can be slow if your dataset is very large, so make sure to
use time windows and limits to help speed up a search::

	mig> search action where investigatorname=%vehent% and agentname=server1% and after=2015-09-01T00:00:00Z and before=2015-10-01T00:00:00Z
	Searching action after 2015-09-01T00:00:00Z and before 2015-10-01T00:00:00Z, limited to 100 results
	----- ID ----- + --------   Action Name ------- + ----------- Target  ---------- + ---- Investigators ---- + - Sent - + - Status - + --- Last Updated ---
	4999271350274    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-29T15:40:35Z
	4964811669519    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-23T13:37:16Z
	4964811669506    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-23T13:37:03Z
	4964764024853    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   inflight     2015-09-23T13:25:26Z
	4964764024834    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   inflight     2015-09-23T13:24:57Z
	4949328330767    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T19:59:39Z
	4949328330754    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T19:59:25Z
	4948324450316    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T19:45:51Z
	4948324450307    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T19:33:17Z
	4947944865794    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T14:07:36Z
	4947909869570    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T13:58:41Z
	4947901022223    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T13:56:42Z
	4947901022210    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T13:56:26Z
	4947890798596    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    2   completed    2015-09-20T13:55:02Z
	4885615083769    timedrift -c /home/ulfr/.mi...   status='online' AND mode='d...   Julien Vehent                    3   completed    2015-09-09T17:02:56Z
	4885615083755    pkg -c /home/ulfr/.migrc-ln...   status='online' AND mode='d...   Julien Vehent                    3   completed    2015-09-09T17:01:33Z
	4885615083739    memory -c /home/ulfr/.migrc...   status='online' AND mode='d...   Julien Vehent                    3   completed    2015-09-09T17:01:04Z
	4885615083724    file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   Julien Vehent                    3   completed    2015-09-09T16:58:00Z

Managing investigators
----------------------

The console can be used to create new investigators, active and
disable them, and well as review their activity.

To review the activity of an investigator, find its ID using the
search command and enter investigator mode::

	mig> search investigator where name=%vehent%
	Searching investigator after 2011-03-31T13:16:55Z and before 2020-04-12T13:16:55Z, limited to 100 results
	- ID - + ----         Name         ---- + --- Status ---
		 2   Julien Vehent                    active

	mig> investigator 2
	Entering investigator mode. Type exit or press ctrl+d to leave. help may help.
	Investigator 2 named 'Julien Vehent'

	inv 2> help
	The following orders are available:
	details                 print the details of the investigator
	exit                    exit this mode
	help                    show this help
	lastactions <limit>     print the last actions ran by the investigator. limit=10 by default.
	pubkey                  show the armored public key of the investigator
	r                       refresh the investigator (get latest version from upstream)
	setstatus <status>      changes the status of the investigator to <status> (can be 'active' or 'disabled')

The command **lastactions** will print the latest activity of this investigator::

	inv 2> lastactions
	----- ID ----- + --------    Action Name ------- + ----------- Target   ---------- + ----    Date    ---- +  -- Status --
	5033038708749     Test action that pings goog...   environment->>'os'='linux' ...   2015-10-06T09:12:31-04:00    completed
	4999271350274     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-29T11:40:31-04:00    completed
	4964811669519     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-23T09:37:12-04:00    completed
	4964811669506     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-23T09:36:59-04:00    completed
	4964764024853     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-23T09:25:22-04:00    inflight
	4964764024834     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-23T09:24:52-04:00    inflight
	4949328330767     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-20T15:59:35-04:00    completed
	4949328330754     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-20T15:59:21-04:00    completed
	4948324450316     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-20T15:45:47-04:00    completed
	4948324450307     file -c /home/ulfr/.migrc-l...   status='online' AND mode='d...   2015-09-20T15:33:13-04:00    completed

To disable him, use **setstatus disabled**. Disabled investigators are no longer
allowed to send investigations via the API::

	inv 2> setstatus disabled
	Investigator status set to disabled

	inv 2> details
	Investigator ID 2
	name     Julien Vehent
	status   disabled
	key id   E60892BB9BD89A69F759A1A0A3D652173B763E8F
	created  2015-09-09 09:53:28.989481 -0400 EDT
	modified 2015-09-09 09:53:28.989481 -0400 EDT

	inv 2> setstatus active
	Investigator status set to active

Creating investigators
~~~~~~~~~~~~~~~~~~~~~~

To create a new investigator, go back to **mig>** mode and type
**create investigator**. The console will prompt the name of the new
investigator, if additional permissions should be set, and as well 
the location of her public PGP key.

By default unless specific investigators will be created with no
additional permissions. Answering yes to the permission related
questions grant the investigator additional access to API functionality.

You can either provide a local path to the public key file on disk,
on provide a fingerprint in the format "0x<40 char sha1 hash>". When
a fingerprint is provided, the console will attempt to retrieve the
key from the keyserver `gpg.mozilla.org`::

        mig> create investigator
        Entering investigator creation mode. Please provide the full name
        and the public key of the new investigator.
        name> Bob Kelso
        Name: 'Bob Kelso'
        With no additional permissions, the investigator will be permitted
        access to run investigations. Answer yes to any of the following to add
        additional permissions to the investigator.
        
        If this is the first investigator being added, you should make this
        investigator an admin.
        Allow investigator to manage users (admin)? (yes/no)> no
        Investigator will not have administrative permissions
        Allow investigator to manage loaders? (yes/no)> no
        Investigator will not have loader management permissions
        Allow investigator to manage manifests? (yes/no)> no
        Investigator will not have manifest management permissions
        Please provide a public key. You can either provide a local path to the
        armored public key file, or a full length PGP fingerprint.
        example:
        pubkey> 0x716CFA6BA8EBB21E860AE231645090E64367737B
        pubkey> 0x716CFA6BA8EBB21E860AE231645090E64367737B
        retrieving public key from http://gpg.mozilla.org
        -----BEGIN PGP PUBLIC KEY BLOCK-----
        Version: SKS 1.1.5
        Comment: Hostname: keyserver.mozilla.org
        
        mQENBEbv+5sBCADNHPvUIajRoxb/qylLrzwm9e+9sB8R/jhY4gxOzGZRDHECPvNeTUd9eogV
        n24rQDTWowkE+t9sW7vlD3TUWdBEAhXEpDZBfzlTBWIzEb1m3hwPOQM10ZNX6jPS1WlGsfoE
        LsUC0HmFTtOx4b5os9mIYbsjsDWd/JZjn0yUIv4eb28+fle6BkbgqIotLW4d1gTrxVlFc3be
        m+4OqimQ/v2LZDV+uObEkbh4UvmTtOCCx8zAOyZohPmICUbmJBc8KWWhzLOo8b9ns/GqP41q
        /9IuTQDXP2GUAKXzBKSdQiNzJP8Skfu4tPyGsGJSErprPC9t43HPPUgfeW9/sfuaw+vnABEB
        AAG0JUp1bGllbiBWRUhFTlQgPGp1bGllbkBsaW51eHdhbGwuaW5mbz6JATYEEwECACAFAkbv
        +5sCGy8GCwkIBwMCBBUCCAMEFgIDAQIeAQIXgAAKCRBkUJDmQ2dze5U9CADK4Z6X02TP9afJ
        AyWF32zM3UdMksJ/F2wuo2HBHT0iOomw4ecNzHyO1P5BTglm5LC5ZZrV+Dx6Jve75JiSDTSD
        V3AhpR+M83rw8YKkeUrbTvfsy3+qhB7HYNIbCKT0lgAAL05SmDnYwYMQIV+p3T0F8BgGhkGT
        vHdKLhzEhKNOMaMCwCd1SsiBepA976oBUp9h5Vt6TEFyG6hCcFP90DFjNlK17yMNjbdrgyd6
        FkGEePKK3RhaLxcPShAgCnYzYYMABLYu1ow5AxxtEJTBHkJFkIzE9XM/lmyekYhWfX/Q4jpn
        1aiiVlf2klZlFSFy/TXuuQH1JO3YlKo9uHjlvSQb
        =+Ru1
        -----END PGP PUBLIC KEY BLOCK-----
        
        create investigator? (y/n)> y
        Investigator 'Bob Kelso' successfully created with ID 4
        
        mig> investigator 4
        Entering investigator mode. Type exit or press ctrl+d to leave. help may help.
        Investigator 4 named 'Bob Kelso'
        
        inv 4> details
        Investigator ID 4
        name        Bob Kelso
        status      active
        permissions 0 (Investigator only)
        key id      716CFA6BA8EBB21E860AE231645090E64367737B
        created     2015-10-06 09:23:14.473307 -0400 EDT
        modified    2015-10-06 09:23:14.473307 -0400 EDT

The new investigator now has access to the API.
