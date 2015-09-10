==========
MIG RUNNER
==========
:Author: Aaron Meihm <ameihm@mozilla.com>

.. sectnum::
.. contents:: Table of contents

MIG runner is a service that can be deployed to automatically schedule actions
to run within the MIG environment and retrieve/process the results.

The runner interacts directly with the MIG API in the same manner as a client
would. When an action is scheduled to run, the runner will deploy the action
and schedule a time to gather results (shortly after the action has expired).
Once the action has expired, the runner will retrieve results from the API
and store these results in the runner directory.

Runner configuration file
-------------------------

An example configuration file for use by mig-runner is shown below.

.. code::

        ; Sample MIG runner configuration file

        [runner]
        directory = "/home/mig-runner/runner" ; The path to the root runner directory
        checkdirectory = 30 ; How often to check runners/ for job changes

        [logging]
        mode = "stdout" ; stdout | file | syslog
        level = "debug"

        [client]
        clientconfpath = "default" ; Path to client conf, default for $HOME/.migrc

The `directory` option specifies the root directory that stores all the
mig-runner related control information. A typical runner directory may look
something like this.

.. code::

        runner/
        |
        + runners/
        | |
        | + job1/
        | + job2/
        |
        + plugins/

Job configuration
-----------------

Under each job directory, a file entity.cfg defines the parameters used to
run this job.

.. code::

        [configuration]
        schedule = "<cronexpr>"
        plugin = "<plugin name>"

The schedule option should be set to a cron style expression to note when
the job should be run.

The plugin is optional.  If set, the value will be interpreted as an
executable in the plugins directory. The results of the job will be piped
into stdin of this executable in JSON format (mig-runner ResultEntry). The
plugin can then parse and forward the data as needed.

The results are also written into a `results/` subdirectory under the
runner directory, using the action ID as a file name. This happens
regardless of any plugin configuration for the job.

In the job directory, the MIG action should should be launched should be
called `action.json`. The time validity and expiration fields will be
filled in by the runner process before dispatching the action to the
API.

Output plugins
--------------

The runner writes JSON output to stdin of any configured output plugin. This
is intended to provide flexibility, allowing plugins to be developed in
any language. If a plugin is being developed in go, the runner ResultEntry
type can be used to parse incoming data. In other languages the JSON can
be parsed as desired.

