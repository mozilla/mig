MIG standalone installation
===========================

The MIG standalone installation script `standalone_install.sh` can be used to
install a fully functional version of MIG on a system. The script is primarily
tested and used on Ubuntu hosts (e.g. Ubuntu 14) but may execute correctly on
other distributions.

To use the script, run `bash standalone_install.sh` from the `tools/` directory
in the MIG distribution.

Agent development on standalone installation
--------------------------------------------

A deployment using standalone installation is useful for development. During
the standalone installation process, configuration files are generated that are
used by various MIG processes. This occurs under `tools/mig/`, which is a new
checkout of MIG created by the installer. This means, if you were to rebuild
the agent using the checkout containing `standalone_install.sh`, the agent will
fail to connect as it has invalid credentials.

To work around this, when building a new agent from your MIG directory,
instruct the build process to use the configuration that was created
during standalone install so it works with the rest of your system. This
can be done using a make command as follows:

`make mig-agent AGTCONF=tools/mig/conf/mig-agent-conf.go BUILDENV=demo`
