================
MIG command line
================

.. sectnum::
.. contents:: Table of contents

The MIG command line is a terminal-based client interface that provides fast
interaction with the MIG API to launch actions and receive results. Its goal is
to be simple and fast to use for basic investigations. Unlike the mig-console,
it does not provide access to previous actions, search or investigators
management.

To install the MIG command line, use `go get mig.ninja/mig/client/mig`. This
command will place a binary called `mig` under $GOPATH/bin/mig. On first run, it
will invite you to create a configuration:

.. code::

	$ mig file -path /etc -name "^passwd$" -content "jvehent"
	no configuration file found at /home/jvehent/.migrc
	would you like to generate a new configuration file at /home/jvehent/.migrc? Y/n> Y
	found key 'Julien Vehent (ulfr) <jvehent@mozilla.com>' with fingerprint 'E60892BB9BD89A69F759A1A0A3D652173B763E8F'.
	use this key? Y/n> Y
	using key E60892BB9BD89A69F759A1A0A3D652173B763E8F
	what is the location of the API? (ex: https://mig.example.net/api/v1/) > https://api.mig.example.net/api/v1/
	MIG client configuration generated at /home/jvehent/.migrc

	1226 agents will be targeted. ctrl+c to cancel. launching in 5 4....
	
Subsequent runs of `mig` will read the configuration from `$HOME/.migrc` directly.

For more examples on how to use the mig command line, see the `cheatsheet
<cheatsheet.rst.html>`_.
