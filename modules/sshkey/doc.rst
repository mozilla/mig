===================================
Mozilla InvestiGator: sshkey module
===================================
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The sshkey module is designed to scan agent file systems and return fingerprints
for any keys found on the host. The module will identify private and public keys,
returning the fingerprint and path for each, and in addition will also scan
`authorized_keys` files and return the fingerprint for each key noted in these files.

The module can be used to gather information to locate a key matching a specific
fingerprint, or to gather information on any inter-system relationships by correlating
returned authorized key fingerprints with private/public key fingerprints.

For example, using this module you can correlate keys on systems (e.g., system keys)
reported as private/public key fingerprints, with the hosts they have access to (reported
as authorized key file fingerprints).

For RSA/DSA private keys which are encrypted, the module will note the existance of the
key but can not return the fingerprint. However, if an adjacent public key is present the
fingerprint will be returned here.

This module makes use of the file module for file system analysis as a dependency.

Usage
-----
The module can be run with no arguments in which case the defaults are used. The default
for Linux and Darwin is to scan /root and /home for SSH keys, to a maximum depth of 8
directories. The default for Windows is the same depth, but to scan c:\Users. These
defaults can be overridden with the `path` and `maxdepth` options to the module.
