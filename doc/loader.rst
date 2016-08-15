==========
MIG LOADER
==========

.. sectnum::
.. contents:: Table of contents

Overview
--------
The MIG loader is the component in MIG that can look after keeping things up
to date on systems, for example on workstations and laptops that are not
centrally managed. More specifically, this involves periodically checking if
new agent code is available, making sure it's signed and safe to install, and
handling automatic agent/configuration upgrades.

In environments with a mature configuration management setup (such as via
puppet or ansible) MIG loader may not be required. The configuration management
system can periodically push updates out to required systems.

In other environments, the loader can help keep things up to date. As an example
this could be in cases where you want to keep MIG agents up to date on workstations that
are not centrally managed. Even in cases with a configuration management system in
place, you may want to use MIG loader to provide more fine-grained control over
updates to agents.

Here we describe how the loader works, and provide an example on how it is setup
and used to provision and continuously upgrade MIG agent deployments.

Differences between deploying with mig-loader and without
---------------------------------------------------------
Traditionally with MIG, if you wanted the agent installed on a set of systems,
you would package the MIG agent up and install that agent package on the desired
systems. If you wanted to upgrade the agent, you would need to upgrade the agent
package installed on the systems.

With mig-loader, instead of installing the agent on the systems you want to run
the agent on, you would install only mig-loader. mig-loader is a small binary
intended to be run from a periodic system such as cron. mig-loader will then
look after fetching the agent and installing it if it does not exist on the system,
and will look after upgrading the agent automatically if you want to publish new
agent updates. The upgrades can be controlled by a MIG administrator through the
MIG API and console tools.

Key components
--------------
Several components within MIG are responsible for supporting loader related
deployment. Some are dedicated to this, others have additional functionality
that is used.

mig-loader
~~~~~~~~~~
mig-loader runs on an endpoint and is responsible for managing the mig-agent
that is running on that particular endpoint. It would run on a host you would
like the agent deployed on. mig-loader looks after periodically checking if new
agents are available by contacting the API, validating signatures on new agent
manifests, and installing new agent code and configuration data when available. It
also looks after management related actions such as restarting the agent if a new
version has been installed.

mig-api
~~~~~~~
The API is the control interface of MIG used by investigators to interact with
the platform. From the perspective of the loader, it serves a
couple specific purposes.

The mig-loader contacts the API periodically to see if updates are available, and
fetches these updates from the API.

In addition, the API is also the component administrators would communicate with
using mig-console to upload agent updates and manage the upgrade process.

mig-console
~~~~~~~~~~~
mig-console, which is commonly used to interact in general with the MIG system also
has some additional functionality related to use with a loader deployment, such as
providing MIG administrators the ability to upload new agent upgrades, and manage
clients that are deployed using mig-loader.

The MIG database
~~~~~~~~~~~~~~~~
The MIG database stores agent manifests (discussed later). The API fetches data from
the database to provide it to mig-loader instances requesting updates.

About manifest signatures
-------------------------
When the loader asks the API for the current version of the agent that should be
installed, the API will respond with a signed manifest. The manifest is signed by
MIG administrators when it is uploaded to the API (discussed later) using the
administrators PGP key. The loader is built with the PGP public key of the
administrators, which allows the loader to validate the manifest signature is
correct before it will attempt to install updates.

You can require any number of signatures. For example, you could deploy so a
loader will accept a manifest signed by one MIG administrator, or potentially to
provide additional security you can require the manifest be signed by 2 or more
different administrators.

There are two places this needs to be configured:

* The MIG API configuration file
* The loader built-in configuration

Decide on the number of signatures you wish to require, then edit ``/etc/mig/api.cfg``
and add the required option, for example to require 2 signatures:

::

    [manifest]
        requiredsignatures = 2

The configuration option required for the loader built-in config is discussed later
in the building mig-loader section.

Building mig-loader for your environment
----------------------------------------
If the loader is to be used, it needs to be built with some basic configuration
that indicates how it should operate. This is done by editing the built-in
configuration source file for the loader. Copy the default configuration to
another file for editing.

::

    $ cd conf
    $ cp mig-loader-conf.go.inc mig-loader-myenv.go.inc

Here you would indicate where the API is, include any tags (similar to agent tags)
that should be included with this loader type, and you would also build in any
PGP keys that should be used as part of validation of manifest signatures
by the loader. Manifests are signed by MIG administrators, so normally you will
place the PGP public keys of MIG administrators in the loader configuration.

An important value to set here is the number of signatures that must be present on
a manifest before the loader will accept it. This can be set by changing the value
of the REQUIREDSIGNATURES variable. For example, to set the loader to require 2
valid signatures be present in the manifest:

.. code:: go

    var REQUIREDSIGNATURES = 2

The configuration file also contains variables used in environment
discovery similar to those available for the agent. The agent and loader both use
the same environment discovery functions, and the environment is provided to the API
by the loader to help the API determine which manifest it should provide, so you can
target manifests at loader instances in the same way you would use the ``-t`` flag
to ``mig`` to target specific agents with actions.

Once complete, build the loader binary with your configuration file.

::

    $ make mig-loader LOADERCONF=conf/mig-loader-myenv.go.inc
    mkdir -p bin/linux/amd64
    if [ ! -r conf/mig-loader-myenv.go.inc ]; then echo "conf/mig-loader-myenv.go.inc configuration file does not exist" ; exit 1; fi
    # test if the loader configuration variable contains something different than the default value
    # and if so, replace the link to the default configuration with the provided configuration
    if [ conf/mig-loader-myenv.go.inc != "conf/mig-loader-conf.go.inc" ]; then rm mig-loader/configuration.go; cp conf/mig-loader-myenv.go.inc mig-loader/configuration.go; fi
    GOOS=linux GOARCH=amd64 GO15VENDOREXPERIMENT=1 go build  -o bin/linux/amd64/mig-loader -ldflags "-X mig.ninja/mig.Version=20160512-0.9fe5f23.dev" mig.ninja/mig/mig-loader

You will end up with a mig-loader binary in ``bin/linux/amd64`` you can copy into
your manifest when you create it in a later step.

Building an agent for your environment
--------------------------------------
See the agent documentation for information on building an agent. The steps will
be similar to that of the loader.

Creating manifests
------------------
**Note:** Since manifests contain compiled code, you will need a manifest per-platform
type you want to deploy to. This means you will need to build a different loader and agent
depending on the OS type (e.g., Linux, Darwin) and architecture. You will create a
different manifest for each one as well.

A manifest is an agent and set of configuration data you want to push out to
devices in your environment. The current components that can be inside a manifest
include:

* A compiled mig-agent
* A compiled mig-loader
* An agent configuration file (e.g., /etc/mig/mig-agent.cfg)
* The agent client certificate
* The agent client certificate private key
* The CA key the agent should use to validate connections to the relay

If a file is not present in a manifest, it will not be deployed with the loader. For
example, you may not want a configuration file to be part of the manifest if you
want to deploy agents with a built-in configuration.

To create a manifest, create a directory we will use to place the files we want
to be in the manifest. Copy the components into the directory you want to be part
of the manifest. The components must have specific file names representing their
function. The directory name can be anything.

============= =======================================
Filename      Component
------------- ---------------------------------------
mig-agent     The MIG agent binary you want to deploy
mig-loader    The MIG loader binary you want to deploy
configuration Agent configuration file
cacert        CA certificate
agentcert     Agent certificate to connect to relay
agentkey      Agent key to connect to relay
============= =======================================

When creating a manifest, you will likely end up with something like this.

::

    $ cd mig-manifest-int-linux
    $ ls
    agentcert  agentkey  cacert  configuration  mig-agent  mig-loader

To finish creating our manifest we will use, tar/compress the directory into
the manifest file we will upload to the API.

::

    $ tar -czvf mig-manifest-linux.tar.gz mig-manifest-int-linux
    mig-manifest-int-linux/
    mig-manifest-int-linux/mig-loader
    mig-manifest-int-linux/configuration
    mig-manifest-int-linux/mig-agent
    mig-manifest-int-linux/agentcert
    mig-manifest-int-linux/cacert
    mig-manifest-int-linux/agentkey

Creating a new manifest in the API
----------------------------------
Next we need to send our new manifest to the API, so it is available to be
fetched by loader instances we are running. This is accomplished using
mig-console.

Permission to manage manifests must be set on your investigator to create
manifests. Ensure the PermManifests permission has been applied; you can
validate this by looking at the details for your investigator in ``mig-console``,
and can use the ``setperms`` command if needed.

The ``create manifest`` command is used to create the new manifest.

::

    mig> create manifest
    Entering manifest creation mode.
    Please provide the name of the new manifest
    name> a new manifest
    Name: 'a new manifest'
    Please provide loader targeting string for manifest.
    target> env#>>'{os}'='linux'
    Target: 'env#>>'{os}'='linux''
    Please enter path to new manifest archive content.
    contentpath> /home/myuser/mig-manifest-linux.tar.gz
    {
      "id": 0,
      "name": "a new manifest",
      "content": "...",
      "timestamp": "0001-01-01T00:00:00Z",
      "status": "staged",
      "target": "env#\u003e\u003e'{os}'='linux'",
      "signatures": null
    }
    create manifest? (y/n)> y
    Manifest successfully created
    mig>

The name can be any value you want to use. The target string is important. This
tells the API which systems should receive this manifest. In this case, we
indicate this manifest should be sent to all Linux systems from which the loader
is requesting agent code for. Any valid agent targetting string can be used here,
which can allow for more detailed deployment criteria for a given manifest.

The last value we provide is the manifest file created in the previous step. Note
the status shown for the manifest is ``staged``. For a manifest to become ``active`` and
available, it must be signed by a prerequisite number of MIG administrators. These
signatures are what is used by mig-loader to validate the manifest is authentic
before deploying it on an endpoint.

::

    mig> search manifest where manifestid=34
    Searching manifest after 2011-11-05T20:03:51Z and before 2020-11-17T20:03:51Z, limited to 100 results
    - ID - + ----      Name      ---- + -- Status -- + -------------- Target -------- + ---- Timestamp ---
        34   a new manifest             staged         env#>>'{os}'='linux'             2016-05-12T19:56:20Z
    mig> manifest 34
    Entering manifest reader mode. Type exit or press ctrl+d to leave. help may help.
    Manifest: 'a new manifest'.
    Status 'staged'.
    manifest 34> sign
    Manifest signature has been accepted
    manifest 34>

Now that the manifest is signed, you can validate this. If still in the manifest
reader, reload the manifest with ``r`` and use the ``json`` command to show the
manifest details. If the required number of signatures are present, it will be listed
as active and will now be available to be fetched by loader instances. mig-loader
instances will always receive the newest active manifest that matches the targetting
string specified in the manifest.

The ``entry`` command can be used to show the SHA256 sums of files in the manifest. If
you want to disable a manifest, the ``disable`` command can be used. The ``reset`` command
can be used to remove any existing signatures from a manifest and mark it as staged.

Creating new loader instances
-----------------------------
When mig-loader runs on an endpoint and connects to the API to see if updates are
available and fetch files, it must be authenticated. This authentication occurs by
sending a loader key to the API, which should be unique per endpoint loader instance.
The loader key is essentially an API token. In this example, we will create a new
loader instance for a Linux system, so we can deploy the manifest we just created
to that system.

Permission to manage loaders must be set on your investigator to create
loaders. Ensure the PermLoaders permission has been applied; you can
validate this by looking at the details for your investigator in ``mig-console``,
and can use the ``setperms`` command if needed.

::

    mig> create loader
    Entering loader creation mode.
    Please provide the name of the new entry
    name> corbomite.internal
    Name: 'corbomite.internal'
    Provide expected environment target string, or enter for none
    expectenv> tags#>>'{operator}'='myorg'
    Generating loader prefix...
    Generating loader key...
    {
      "id": 0,
      "name": "corbomite.internal",
      "prefix": "qqLwjje7",
      "key": "BNbZUenzBaucYKgK6ubkz0yqDZ7k4kNX",
      "agentname": "",
      "lastseen": "0001-01-01T00:00:00Z",
      "enabled": false
      "expectenv": "tags#\u003e\u003e'{operator}'='myorg'"
    }
    
    Loader key including prefix to supply to client will be "qqLwjje7BNbZUenzBaucYKgK6ubkz0yqDZ7k4kNX"
    create loader entry? (y/n)> y
    New entry successfully created but is disabled
    mig>

The name can be any value you want, but usually you will want something describing
the system or in the case of a workstation something describing the user of the
device. Here we just used the hostname.

If desired, an expected environment value can be set on the instance. If set, this target string
must match desired parts of the environment the loader is sending, if it does not the request will
be rejected.

The key including prefix is the API key that
will need to be configured in mig-loader on that system to allow it to authenticate as
this loader instance.

The new loader is created in a disabled state. Lets enable it so that it can be
used.

::

    mig> search loader where loadername=%corb%
    Searching loader after 2011-11-05T20:22:49Z and before 2020-11-17T20:22:49Z, limited to 100 results
    - ID - + ----      Name      ---- + ----   Agent Name   ---- + -- Enabled - + -- Last Used ---
        12   corbomite.internal         unset                      false          2016-05-12T20:16:30Z
    mig> loader 12
    Entering loader reader mode. Type exit or press ctrl+d to leave. help may help.
    Loader: 'corbomite.internal'.
    Status 'false'.
    loader 12> enable
    Loader has been enabled
    reloaded
    loader 12>

Note the agent name is unset as it has not been used yet. Once mig-loader connects
and authenticates as this loader instance, it will be populated with the hostname of
the device.

Initial provisioning of mig-loader
----------------------------------
At this point, we have:

* Our manifest created, and available via the API
* A loader instance created, that will be used by our test instance for updates

Next, we want to provision mig-loader to our test device. mig-loader needs to be
installed once on the system we want to keep the agent updated on. Once it has been
installed, it will continuously keep itself and the agent up to date on the system
based on the manifests you are using.

You can use the same loader package for all similar devices in your environment if
you want to. For example, in an environment with OSX and Linux devices, the simplest
possible loader configuration would have 2 active manifests at any given time, with
2 loader packages, and a number of loader instances configured (one per device).

Most of the time, you will provision the initial loader installation on the system
by installing a package containing ``mig-loader``. The test client system is Ubuntu
based, so first we make a loader package using our loader configuration.

::

    $ make deb-loader LOADERCONF=conf/mig-loader-myenv.go.inc
    mkdir -p bin/linux/amd64
    if [ ! -r conf/mig-loader-myenv.go.inc ]; then echo "conf/mig-loader-myenv.go.inc configuration file does not exist" ; exit 1; fi
    # test if the loader configuration variable contains something different than the default value
    # and if so, replace the link to the default configuration with the provided configuration
    if [ conf/mig-loader-myenv.go.inc != "conf/mig-loader-conf.go.inc" ]; then rm mig-loader/configuration.go; cp conf/mig-loader-myenv.go.inc mig-loader/configuration.go; fi
    GOOS=linux GOARCH=amd64 GO15VENDOREXPERIMENT=1 go build  -o bin/linux/amd64/mig-loader -ldflags "-X mig.ninja/mig.Version=20160516-0.8ba7319.dev" mig.ninja/mig/mig-loader
    rm -fr tmp
    install -s -D -m 0755 bin/linux/amd64/mig-loader tmp/sbin/mig-loader
    install -D -m 0644 LICENSE tmp/usr/share/doc/mig-loader/copyright
    mkdir -p tmp/var/lib/mig
    mkdir -p tmp/etc/mig
    fpm -C tmp -n mig-loader --license GPL --vendor mozilla \
        --description "Mozilla InvestiGator Agent Loader\nAgent loader binary" \
        -m "Mozilla <noreply@mozilla.com>" --url http://mig.mozilla.org \
        --architecture x86_64 -v 20160516-0.8ba7319.dev \
        -s dir -t deb .
    Debian packaging tools generally labels all files in /etc as config files, as mandated by policy, so fpm defaults to this behavior for deb packages. You can disable this default behavior with --deb-no-default-config-files flag {:level=>:warn}
    Created package {:path=>"mig-loader_20160516-0.8ba7319.dev_amd64.deb"}

This package will contain the mig-loader binary built with our configuration, which contains the
API URL the loader should use and the PGP keys that will be used to validate incoming manifests. Next
the package can be installed on the system we want to run the agent on.

Loader configuration and initial run
------------------------------------
The loader should be setup to run periodically on the system. This ensures the device periodically
checks for updates, and installs new agent code when required. The periodic job configuration depends
on the operating system the loader is installed on. For Linux based devices, typically ``mig-loader``
would be setup to run as root via a cron entry, or if cron is not on the system using a systemd
timer. On Darwin, the installer automatically creates an interval based launchd job to run the loader.

Put the loader key for this instance into ``/etc/mig/mig-loader.key``. This should contain the key
we used to create the loader instance on a single line.

We can run ``/sbin/mig-loader`` manually on the system now.

::

    # /sbin/mig-loader
    logging routine started
    Ident is Ubuntu 15.10 wily
    Init is upstart
    leaving findOSInfo()
    Found local address 10.0.0.18/24
    Found public ip 10.0.0.18
    AWS metadata service not found, skipping fetch
    initialized local bundle information
    mig-agent /sbin/mig-agent -> not found
    mig-loader /sbin/mig-loader -> 40d83204825421c82379b65b8c7077fd110a4af5391acfc8052e568d0830af26
    configuration /etc/mig/mig-agent.cfg -> not found
    agentcert /etc/mig/agent.crt -> not found
    agentkey /etc/mig/agent.key -> not found
    cacert /etc/mig/ca.crt -> not found
    requesting manifest from https://my.mig.api.url:1664/api/v1/manifest/agent/
    1 valid signatures in manifest
    comparing mig-agent /sbin/mig-agent
    we have not found
    they have d3bc2fdbd42404f2df9472d8de900889f8755d12041cda7f65fa7ba99e3eeda3
    refreshing mig-agent
    fetching file from https://my.mig.api.url:1664/api/v1/manifest/fetch/
    validating staged file signature
    renaming existing file
    installing staged file
    comparing mig-loader /sbin/mig-loader
    we have 40d83204825421c82379b65b8c7077fd110a4af5391acfc8052e568d0830af26
    they have 3d584ad090c556234ad6148006ab0dcd693ab9f99c386413a8597034420384dc
    refreshing mig-loader
    fetching file from https://my.mig.api.url:1664/api/v1/manifest/fetch/
    validating staged file signature
    renaming existing file
    installing staged file
    comparing configuration /etc/mig/mig-agent.cfg
    we have not found
    they have d51a2e9d955aaca94e88159ad6235cbaccf9680f0d8e82dcee0f2f0f0df83038
    refreshing configuration
    fetching file from https://my.mig.api.url:1664/api/v1/manifest/fetch/
    validating staged file signature
    renaming existing file
    installing staged file
    comparing agentcert /etc/mig/agent.crt
    we have not found
    they have 017525f2f851311e9b0e26a139252c13b186a6507206cbd0dcc1ca35258b9566
    refreshing agentcert
    fetching file from https://my.mig.api.url:1664/api/v1/manifest/fetch/
    validating staged file signature
    renaming existing file
    installing staged file
    comparing agentkey /etc/mig/agent.key
    we have not found
    they have 88df8f032916dfa0ae6c4778fd2aa2084c1aac017aab70f7d4bc6f4327c5c24c
    refreshing agentkey
    fetching file from https://my.mig.api.url:1664/api/v1/manifest/fetch/
    validating staged file signature
    renaming existing file
    installing staged file
    comparing cacert /etc/mig/ca.crt
    we have not found
    they have 215394a591db4dbf2bbbb17a4d45b5bc6d335d15a7d2c42876d4b27f8269bda9
    refreshing cacert
    fetching file from https://my.mig.api.url:1664/api/v1/manifest/fetch/
    validating staged file signature
    renaming existing file
    installing staged file
    running triggers due to modification
    terminateAgent() -> exit status 1 (ignored)

By running the loader manually you can validate it has connectivity. We should now have an
agent running on the system. Future invocations of mig-loader by the periodic job will
keep the agent and associated files up to date, and look after restarting the agent when
required.

