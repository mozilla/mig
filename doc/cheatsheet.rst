================================
Mozilla InvestiGator Cheat Sheet
================================

.. sectnum::
.. contents:: Table of Contents

This is a list of common operations you may want to run with MIG.

All examples use the MIG command line cli. You can run the examples on your
local machine by specifying `-t local`. The `local` target invokes MIG modules
in the cli instead of calling mig-agent like a normal investigation would.

File module
-----------

You can find detailed documentation by running `mig file help` or in the
online doc at `doc/module_file.html`_.

.. _`doc/module_file.html`: http://mig.mozilla.org/doc/module_file.html

Find files in /etc/cron.d that contain "mysql://" on hosts "*buildbot*"
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This is a simple file content check that looks into all the files contained in
`/etc/cron.d` for a string that matched `mysql://`.

.. code:: bash

    mig file -t "environment->>'os'='linux' AND name LIKE '%buildbot%'" -path /etc/cron.d/ -content "mysql://"

Find files /etc/passwd that have been modified in the past 2 days
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The `mtime` check of the file module matches against the last modified
timestamp of a file.

.. code:: bash

    mig file -t "environment->>'os'='linux'" -path /etc/passwd -mtime <2d

Find endpoints with high uptime
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

On Linux and MacOS, the uptime of a host is kept in `/proc/uptime`. We can
apply a regex on that file to list hosts with an uptime larger or lower than
any amount.

Note the search target that uses postgres's regex format `~*`.

.. code:: bash

    mig file -t "environment->>'os' IN ('linux', 'darwin')" -path /proc/uptime -content "^[5-9]{1}[0-9]{7,}\\."

Find endpoints running process "/sbin/auditd"
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Here, the '^' in the content regex is important to prevent mig from listing
itself while searching for the command line.

.. code:: bash

    mig file -path /proc/ -name cmdline -content "^/sbin/auditd"

Another option, if using '^' is not possible, is to enclose one of the letter
of the process name into brackets:

.. code:: bash

	$ mig file -t "tags->>'operator'='IT'" -path /proc -name "^cmdline$" -maxdepth 2 -content "[a]rcsight"

Find which machines have a specific USB device connected
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

In this example, we'll look for the CryptoStick USB device (vendor:product 20a0:4107).
You can find more device id's with the command `lsusb`.

.. code:: bash

	mig file -matchany -path /sys/devices/ -name "^uevent$" -content "PRODUCT=20a0/4107"

Find "authorized_keys" files with unknown pubkeys
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

If you know which keys should be present in an authorized_keys file, the `file`
module can be used to find file that have extra, unknown, keys.

The first thing needed is a regex with the list of valid public keys. The regex
will also accept any line that starts with a comment character `#` or empty
lines.

One important thing to note is that public keys are base64 encoded and contain
slashes "/" and pluses "+" that conflict with Go's regex format. Those need to
be escaped prior to being passed to MIG.

.. code:: bash

	echo $PUBKEY | sed "s;\/;\\\/;g" | sed "s;\+;\\\+;g"

A valid pubkey regex could be:

.. code:: bash

	"^((#.+)|(\s+)?|(ssh-rsa AAAAB3NznoMzq\+2r2Vx2bhFWMU3Uuid 1061157)|(ssh-rsa AAYWH\+0XAASw== ffxbld_rsa))$"

We can require that this regex must match **every** line of a file using the
`-macroal` parameter, which stand for "Match All Content Regexes On All Lines".

Then, using the `-mismatch content` option, we can ask the file module to return
the files that **don't** conform to the regex. The combination of the content
regex, the `macroal` option and the `-mismatch content` option together will
return files that have unknown keys.

.. code:: bash

	mig file -path /home -path /root -name "^authorized_keys" \
	-content "^((#.+)|(\s+)?|(ssh-rsa AAAAB3NznoMzq\+2r2Vx2bhFWMU3Uuid 1061157)|(ssh-rsa AAYWH\+0XAASw== ffxbld_rsa))$" \
	-macroal -mismatch content

Find if a user is currently running any process (or is connected)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

If you know the UID of a user, you can check if he has any process running.
Additionally, this means that you can find out if he's connected as well, with
the same command.  In this example `1663` is the UID of the user we're looking
for.

.. code:: bash

        mig file -path /proc/ -maxdepth 2 -name "^status$" -content "^Uid:\s+(1664)\s+"

Netstat module
--------------

You can find detailed documentation by running `mig netstat help`.

Searching for a fraudulent IP
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Given an ip 1.2.3.4 associated with fraudulent traffic, we can use the netstat
module to verify that the IP isn't currently connected to any endpoint.

.. code:: bash

	mig netstat -ci 1.2.3.4

`-ci` stands for connected IP, and accepts an IP or a CIDR, in v4 or v6.

Locating a device by its mac address
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

MIG `netstat` can be used to find endpoints that have a given mac address in
their arp tables, which helps geographically locating an endpoint.

.. code:: bash

	mig netstat -nm 8c:70:5a:c8:be:50

`-nm` stands for neighbor mac and takes a regex (ex: `^8c:70:[0-9a-f]`).

Listing endpoints that have active connections to the Internet
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The search below tells the `netstat` module to capture all connections with one
IP in a public CIDR. The list of CIDR is rather long, because it avoid private
CIDR (the netstat module doesn't have an `exclude` option).

.. code:: bash

	mig netstat -e 60s	-ci 1.0.0.0/8		-ci 2.0.0.0/7		-ci 4.0.0.0/6	-ci 8.0.0.0/7 \
	-ci 11.0.0.0/8		-ci 12.0.0.0/6		-ci 16.0.0.0/4		-ci 32.0.0.0/3	-ci 64.0.0.0/3 \
	-ci 96.0.0.0/4		-ci 112.0.0.0/5		-ci 120.0.0.0/6		-ci 124.0.0.0/7	-ci 126.0.0.0/8 \
	-ci 128.0.0.0/3		-ci 160.0.0.0/5		-ci 168.0.0.0/6		-ci 172.0.0.0/12 \
	-ci 172.32.0.0/11	-ci 172.64.0.0/10	-ci 172.128.0.0/9	-ci 173.0.0.0/8 \
	-ci 174.0.0.0/7		-ci 176.0.0.0/4		-ci 192.0.0.0/9		-ci 192.128.0.0/11 \
	-ci 192.160.0.0/13	-ci 192.169.0.0/16	-ci 192.170.0.0/15	-ci 192.172.0.0/14 \
	-ci 192.176.0.0/12	-ci 192.192.0.0/10	-ci 193.0.0.0/8		-ci 194.0.0.0/7 \
	-ci 196.0.0.0/6		-ci 200.0.0.0/5		-ci 208.0.0.0/4

Ping module
-----------

Test web connectivity to google
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Testing reachability of google.com over HTTP can be done using the ping module.

.. code:: bash

	$ mig ping -t "name LIKE '%phx1%'" -d google.com -dp 80 -p tcp

List endpoints that cannot ping a destination
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Need to find which endpoints cannot connect to some destination? ICMP Ping is a
pretty good way to get that data. Make sure to adapt the `show` parameter to
list endpoints that have failed the ping.

.. code:: bash

	$ mig ping -t "name LIKE '%scl3%'" -show notfound -d 10.22.75.57 -p icmp

Timedrift module
----------------

The timedrift module is fairly basic: it retrieves localtime and compares it to
NTP time if asked to check for drift. As such, it only takes a single parameter
to evaluate drift from network time.

.. code:: bash

	$ mig timedrift -drift 60s
	1402 agents will be targeted. ctrl+c to cancel. launching in 5 4 3 2 1 GO
	Following action ID 1428420741979034880.
	status=inflight...55% ...66% ...67% ......89% ..89% ...89% ......90% ..90% ......90% ...90% ..90% ...^Cstop following action. agents may still be running. printing available results:
	host1.dc2.example.net local time is 2015-04-07T15:35:00.768951216Z
	host1.dc2.example.net local time is out of sync from NTP servers
	host1.dc2.example.net Local time is ahead of ntp host time.nist.gov by 3m2.660981781s
	1 agents have found results

Advanced targetting
-------------------

MIG can use complex queries to target specific agents. The following examples
outline some of the capabilities. At the core, the `target` parameter is just a
WHERE condition executed against the agent table of the MIG database, so if you
know the DB schema, you can craft any targetting you want.

.. code::

	mig=> \d agents
					 Table "public.agents"
		 Column      |           Type           | Modifiers 
	-----------------+--------------------------+-----------
	 id              | numeric                  | not null
	 name            | character varying(2048)  | not null
	 queueloc        | character varying(2048)  | not null
	 mode            | character varying(2048)  | not null
	 version         | character varying(2048)  | not null
	 pid             | integer                  | not null
	 status          | character varying(255)   | 
	 environment     | json                     | 
	 tags            | json                     |
	 starttime       | timestamp with time zone | not null
	 destructiontime | timestamp with time zone | 
	 heartbeattime   | timestamp with time zone | not null

* **id** is the numerical unique ID of the agent
* **name** is a string containing the agent hostname (fqdn)
* **queueloc** is the name of the agent queue on rabbitmq
* **mode** is either `daemon` or `checkin` and represents the mode the agent
  runs as
* **version** is the agent version in the form `<YYYY-MM-DD>-<commit hash>`
* **pid** is the PID of the agent's main process
* **status** is one of `online`, `idle` or `offline`
* **environment** is a JSON document that contains information about the
  system the agent runs on. See below.
* **tags** is a JSON document that contains specific tags defined by the MIG
  platform administrator. This can be used to identify the business unit an
  agent runs on, or anything that helps targetting. It need to be defined at
  agent's compile time.
* **starttime**, **heartbeattime** and **destructiontime** are timestamps

Environments
~~~~~~~~~~~~

During startup, the agent retrieves some amount of information about the
host it runs on. That information is stored in the `environment` column
of the agent table, and can be used to target specific agents. Below is a
typical environment set by a Linux agent:

.. code:: json

	{
		"init": "upstart",
		"ident": "Debian testing-updates sid",
		"os": "linux",
		"arch": "amd64",
		"isproxied": false,
		"addresses": [
			"172.21.0.2/20",
			"172.21.0.3/20",
			"fe80::56ee:75ff:fe4b:d625/64",
			"fe80::3602:86ff:fe2b:6fdd/64"
		],
		"publicip": "172.21.0.2"
	}

Using `Postgres's JSON`_ querying support, we can build targets using specific
fields of the environment columns. For example, this is how we target Linux
systems only:

.. _`Postgres's JSON`: http://www.postgresql.org/docs/9.4/static/datatype-json.html

.. code:: bash

	$ mig file -t "environment->>'os'='linux'" ...

mig-agent-search
~~~~~~~~~~~~~~~~

`mig-agent-search` is a small client that lists agents based on a query. It is
useful to test target queries before using them live. You can obtain it via `go
get mig.ninja/mig/client/mig-agent-search`.

.. code:: bash

	$ mig-agent-search "tags->>'operator'='opsec' AND environment->>'os'='linux' AND mode='daemon' AND status='online' AND name like 'mig-api%'"                                                                                  
	name; id; status; version; mode; os; arch; pid; starttime; heartbeattime; operator; ident; publicip; addresses
	"mig-api3.use1.opsec.mozilla.com"; "4892412351434"; "online"; "20150910+3cf667c.prod"; "daemon"; "linux"; "amd64"; "20024"; "2015-09-10T19:00:05Z"; "2015-09-10T21:17:05Z"; "opsec"; "Ubuntu 14.04 trusty"; "52.1.207.252"; "[172.19.1.171/26 fe80::c6d:44ff:fead:edd9/64]"
	"mig-api4.use1.opsec.mozilla.com"; "4892412350962"; "online"; "20150910+3cf667c.prod"; "daemon"; "linux"; "amd64"; "17967"; "2015-09-10T19:00:03Z"; "2015-09-10T21:18:03Z"; "opsec"; "Ubuntu 14.04 trusty"; "52.1.207.252"; "[172.19.1.13/26 fe80::107e:4fff:fe5c:97e5/64]"


Target agents that found results in a previous action
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Useful to run a second action on the agents that returned positive results in a
first one. The query is a bit complex because it uses Postgres JSON array
processing.

Given an action with ID 12345 that was run and returned results, we want to run
a new action on the agents that matched action 12345. To do so, use the target
that follows:

.. code:: bash

	mig file -t "id IN ( \
		SELECT agentid FROM commands, json_array_elements(commands.results) AS r \
		WHERE commands.actionid = 12345 AND r#>>'{foundanything}' = 'true')" \
	-path /etc/passwd -content "^spongebob"

The subquery select command results for action 12345 and return the ID of
agents that have at least one `foundanything` set to true. Since command
results are an array, and each entry of the array contains a foundanything
value, the query iterates through each entry of the array using postgres's
`json_array_elements` function.

Directly invoking the mig-agent
-------------------------------

In order to test queries locally, you may want to run them directly against a local agent.
The agent takes input parameters from a JSON action file or alternatively from stdin.

For example, to match a md5 of inside of /usr/bin, you could run:

.. code:: bash

        mig-agent -m file -d <<<
        '{"class":"parameters","parameters":{"searches":{"s1":{"paths":["/usr/bin"],"md5":["cf4eb543a119e87cb112785e2b62ccd0"]}}}}'
        ; echo
