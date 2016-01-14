# MIG Agent Sandboxing

## Overview

This is the MIG Sandbox Project repository. As the name implies, it is a sandbox for the MIG Agent modules.

The implementation is written in Go, in order to be fully compatible with MIG.  
Functionality is achieved by applying [seccomp](https://www.kernel.org/doc/Documentation/prctl/seccomp_filter.txt) filters (Linux) and constructing sandbox profiles for each module to define behavior through whitelisting syscalls.

## Dependencies

The following requirements must be met in order to sandbox MIG:
* Go v1.5
* MIG
* libseccomp v2
* [libseccomp go bindings](https://github.com/seccomp/libseccomp-golang)

## Links

[Official MIG Repository](https://github.com/mozilla/mig)  
[Mozilla Wiki Page](https://wiki.mozilla.org/Security/Automation/Winter_Of_Security_2015/MIG_Agent_Sanboxing)
