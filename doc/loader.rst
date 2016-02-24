==========
MIG LOADER
==========

.. sectnum::
.. contents:: Table of contents

**Note**: mig-loader is a work in progress component and is not complete. This
document is intended to summarize the concept and planned approach, and will be
updated once additional progress is made.

Overview
--------
In normal scenarios, the MIG agent is deployed using some form of package, for
example RPM or deb. When the agent needs to be updated, the older agent package
is removed from the system and a new package is installed.

This approach works well for systems under some form of configuration
management (e.g., servers) as it is easy to deploy newer package versions
to these devices. This approach does not work well with devices such as user
workstations, as typically they are not managed in the same way.

mig-loader is intended to act as bootstrapping function for the agent. The
idea is, rather than install an agent on a system, mig-loader will be
installed instead, and will manage the agent software on an on-going basis,
looking after updates.

Using mig-loader
----------------
The following high level steps are part of deploying and using mig-loader:

* Configure the loader and build it
* Create a manifest containing the agent and other support files, and make it avialable through the API
* Install the loader on systems you wish to run the agent on

From this point on, the loader will automatically fetch the agent and manage it.

Building the loader and configuration
-------------------------------------
If the loader is to be used, it needs to be built with some basic configuration
that indicates how it should operate. This is done by editing the built-in
configuration source file for the loader (``mig-loader/configuration.go``).

Here you would indicate where the API is, include any tags (similar to agent tags)
that should be included with this loader type, and you would also build in any
GPG keys that should be used as part of validation of manifest signatures
by the loader. The configuration file also contains variables used in environment
discovery similar to those available for the agent. The agent and loader both use
the same environment discovery functions, and the environment is provided to the API
by the loader to help the API determine which manifest it should provide to the loader.

OSX specific notes
~~~~~~~~~~~~~~~~~~
The loader can be packaged up for OSX after it has been compiled using the
``osx-loader-pkg`` target. This will build a standalone OSX installer, that when
run will prompt the user for a loader registration key, and configure the
loader to run periodically on the target system.

The installer creates a launchd job which runs periodically. In a typical scenario
once the loader has run successfully once on the target system, you would have
2 launchd jobs related to mig; the loader job and the agent itself.

Windows specific notes
~~~~~~~~~~~~~~~~~~~~~~
Support for Windows has not yet been added.

Linux specific notes
~~~~~~~~~~~~~~~~~~~~
The loader functions on Linux but no automated installation is available,
it can be configured to run with a few steps. The loader binary needs to be
installed, and the loader registration key for this instance needs to be in
``/etc/mig/mig-loader.key``. Following this, the loader just needs to be
setup in cron to run periodically as root.

Operation
---------
The general operating methodology is as follows:

* Once the loader is installed, it periodically requests manifests from the API
* The loader validates the signature on the manifests using the built-in GPG keys
* The loader compares these manifests with MIG related files installed
* The loader fetches any files from the API indicated in the manifest that do not match what is installed on the system
* The loader looks after restarting the MIG agent if needed

Loader authentication
~~~~~~~~~~~~~~~~~~~~~
The loader authenticates against the API differently than the MIG client
applications. Specifically, the client applications use GPG signatures as part
of authentication, where the loader uses registration keys (API keys) to
authenticate with the API. These keys uniquely identify a loader instance, and
only permit access to manifest related API endpoints.

Each loader that is provisioning an agent should have it's own unique
registration key. These would typically be provided to a user along with
the installation package.

Each loader must have an entry in the loaders table in the database. When the
loader configures MIG on a target system, it will also use the registration
key along with the loader name indicated in the MIG database as the AMQP
relay credentials. The loader learns the AMQP user component as it is included
in part of the response sent to the loader from the API.

To prevent a loader instance from future API access, the entry can be removed
from the MIG database, and from the RabbitMQ user database.

Manifest management
-------------------
Manifests represent a set of files that a loader can request and deploy to
the target system. Manifests are stored in the MIG database. To control
which manifest a loader will receive, targetting strings are used similar to
how specific agents would be targetted with an action using MIG client
applications.

The mig loader generates an environment string using the same code the agent
uses.

When the manifest is created, the targetting string is associated with it.
Once the manifest is active, any loader manifest requests will recieve the
manifest which matches loader using the targetting string. The most recent
manifest that is active that matches is sent.

Adding and managing manifests
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
Manifests can be managed using the API. The manifest reader can be used to
modify and sign an existing manifest. The create manifest function can be
used to send a new manifest to the API for storage in the database.

Loader updates
--------------
It is anticipated the loader will requires updates far less likely than the
agent itself. However, it may be useful to support the loader having the
ability to update itself in addition to the agent.
