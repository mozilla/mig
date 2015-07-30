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
management (e.g., services) as it is easy to deploy newer package versions
to these devices. This approach does not work well with devices such as user
workstations, as typically they are not managed in the same way.

`mig-loader` is intended to act as bootstrapping function for the agent. The
idea is, rather than install an agent on a system, `mig-loader` will be
installed instead, and will manage the agent software on an on-going basis.

Operation
---------
As an example, the loader would operate as follows.

The loader binary would be installed on a client system using a package in the
same way the agent would be installed now. This package would also be
responsible for adding an operation to the scheduling facility on the system
(for example, cron, Windows Scheduler) to run `mig-loader` on a periodic basis
on the system.

When the loader is run by the scheduler, it will create a manifest based on
what MIG components/configuration files are present on the system. It is likely
this will be limited to the agent itself, and configuration files used by the
agent. This manifest will include information such as the SHA256 checksum of
any present MIG files.

Next, the loader will generate an agent environment structure. This will
include the same information the agent typically sends as part of it's
environment (e.g., the system hostname, operator identifier, others).

Once the loader has this information, it will request the current manifest
from the API by sending it's environment. The API will look at the environment,
and determine the current manifest for use on this particular system. It is
intended this information will be stored in the MIG database. Once the API
has made this determination, it will respond to the loader with what the
manifest should be.

When the loader recieves this manifest, it will compare it with the manifest
it generated using local information from the system. If there are mismatches,
the loader will make a request to the API to fetch the files which are not
current, stage the files on the file system, and eventually replace the files
with the new versions.

In cases where there are changes, the loader will need to look after initiating
the new agent code. This would for example involve spawning a new mig-agent
process, which in turn will result in the old agent being killed. In cases
where it is a new system (no current mig-agent) this would also cause the agent
to self-install itself in the service manager for the OS.

Loader updates
--------------
It is anticipated the loader will requires updates far less likely than the
agent itself. However, it may be useful to support the loader having the
ability to update itself in addition to the agent.

Cryptographic considerations
----------------------------
All stages of the process will need to be authenticated. This will include:

* Authenticating the loader with the API, to ensure manifest requests or file requests can only originate from valid sources for that manifest. This will likely involve requiring some form of secret on the loader system (e.g., a GPG key to use similar authentication as clients do with the API).
* Authentication of manifests and files sent from the API; this will involve validating signatures on the manifest/files/etc. This will likely involve the addition of a signing key in the API.

Other considerations
--------------
* Although the target platform is workstations, in the future this approach could be used for servers as well.
