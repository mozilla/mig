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

You can find detailled documentation by running `mig file help` or in the
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

Netstat module
--------------

You can find detailled documentation by running `mig netstat help`.

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
