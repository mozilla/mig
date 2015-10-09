# Welcome to the Mozilla InvestiGator project!

MIG is an open source Mozilla project. We welcome all contributions and will
help you get started and submit your first patch. The best place to get your
questions answered is in the `#mig` channel on
[irc.mozilla.org](https://wiki.mozilla.org/IRC).

If you're new to Mozilla, you may find the [Contributing to the Mozilla
codebase](https://developer.mozilla.org/en-US/docs/Introduction) page on MDN
full of interesting information.

## Coding guidelines

* Commits must be prefixed with one of the following tag:
    - `[doc]` for commits that update documentation
    - `[minor]`, `[medium]`, `[major]` for regular commits, pick the level that
      represents the importance of your commit the best.
    - `[(level)/bug]` for commits that fix a bug

* If your commit is linked to a github issue, please add a reference
  to it in the commit message.
  ex: `[minor] fix bad counter on agent stats, fixes #125123`

* Pull requests must represent the final state of your commit. If you spent two
  weeks working on code, please only submit the commits that represent the
  latest version of your code at the moment of the submission. While keeping a
  history of your personal progress may be interesting to you, it makes reviews
  longer and more difficult. `git rebase` is your friend here.

* Inform your reviewer that you have corrected something in a subsequent commit,
  to help the reviewer follow up. For example:
    - reviewer: `please check len(string) prior to calling string[10:15]`
    - committer: `fixed in 452ced1c`

* All Go code must be formatted using `gofmt`.

* External dependencies must be vendored, the source of the repository must be
  added into the Makefile under `go_vendor_dependencies`. Your patch must
  include the vendored packages in a separate commit.

* When writing modules, follow the documentation at [Modules
  writing](http://mig.mozilla.org/doc/modules.rst.html) and check that:
    - tests have been added that cover the core functionalities of the module
    - documentation has been written into modules/<modulename>/doc.rst
  Please don't pick a cute or clever name for your module. Pick a name that is
  as explicit as possible. For example, a module that inspects files on the file
  system would be called `file`. A module that lists packages is called `pkg`.

* In modules, you can set GOMAXPROCS to 1, or you can leave it unset and it will
  be defined somewhere else. Never set GOMAXPROCS to something greater than 1,
  we care about the impact the agent has on the cpu usage of its host.

* While we do not enforce line size or function length, please be mindful of the
  readability of your code. A single function that is longer than 100 lines, or
  larger than 120 columns should probably be rewritten into smaller and cleaner
  functions.

## Licensing

All contributions to MIG must be placed under the Mozilla Public License Version
2.0. We don't ask you to sign anything, but we expect everyone to follow these
guidelines: [Commit Access Requirements](https://www.mozilla.org/en-US/about/governance/policies/commit/requirements/)
