======================
Persistent MIG Modules
======================

.. sectnum::
.. contents:: Table of Contents

This document describes persistent modules and how they differ from standard
MIG modules.

Persistent modules are invoked when the agent starts, and run for the lifetime
of the agent. They are supervised by the main agent process, and are restarted
if a failure occurs or they are shut down. Persistent modules can be queried in
the same way an investigator would query using a standard module, however instead
of the module being invoked once for a point-in-time investigation, the persistent
module is queried. This can be useful to return collected statistics on an on-going
basis, or otherwise conduct investigations over a period of time. A persistent module
can be considered to keep state over the course of its lifetime, where as a standard
module returns results representing a given point in time.

Persistent modules are required to implement all the same functions and satisfy
the same interfaces as standard modules, but they also are required to implement
a few additional components.

For general information on modules, see the `module documentation`_. This builds off
that documentation and is specific to the implementation of persistent modules.

.. _`module documentation`: modules.rst

Module logic
============

Registration
------------

All MIG modules must satisfy the Runner interface. Persistent modules must satisfy
both the Runner interface, and the PersistRunner interface.

.. code:: go

	type PersistRunner interface {
		RunPersist(io.ReadCloser, io.WriteCloser)
	}

Initial execution by the agent
------------------------------

When the agent starts up, any persistent modules will be started. This is done by
spawning a supervisor goroutine which will start the modules by calling
``mig-agent`` with the ``-P`` flag to indicate which persistent module to run.

The main agent process starts the module up and supervisory related communication
occurs over a pipe. Writes to and reads from the running persistent module process
occur over a pipe. From the module side, Writes to and reads from the main agent
process occur over stdout and stdin respectively.

When the persistent module is started by the agent, the modules ``RunPersist()``
function is executed. This function does not return, and is typically responsible
for starting up any tasks the module wants to execute in goroutines. Following all
this, the module will enter the ``modules.DefaultPersistHandlers()`` function which
provides a consistent entry point to the handling of supervisor messages between the
agent and the running module.

Querying persistent modules
---------------------------

When the agent receives a request to query a persistent module, the general flow
can be described as follows.

::

	Agent			module -m			module -P
	+---+			+-------+			+-------+
	Parameters ------stdin->|					|
				| Run() -------------listener---------->|----->+
				|					|      | Persist request handler
				|					|      |
	<---------------stdout--|<-------------------listener-----------|<-----+

* The agent calls itself with the ``-m`` flag, and sends the parameters into stdin of the new process.
* The modules Run() method is called; this connects to the socket the persistent module is listening on, and submits the same parameter set.
* The running persistent module accepts the new connection, parses the parameters and returns a result.
* The ``-m`` query process reads the result from the socket, and returns it to the agent on stdout.

Each new request entering the persistent module is handled in a new go-routine in the
running module. Care must be taken inside this module that data structures are locked or protected
as concurrent operations can occur with multiple queries hitting the module at the same time.

Additional details
==================

For additional details and examples, the examplepersist module should be reviewed.
