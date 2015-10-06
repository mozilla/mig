==========
MIG RUNNER
==========

.. sectnum::
.. contents:: Table of contents

MIG runner is a service that can be deployed to automatically schedule actions
to run within the MIG environment and retrieve/process the results.

The runner interacts directly with the MIG API in the same manner as a client
would. When an action is scheduled to run, the runner will deploy the action
and schedule a time to gather results (shortly after the action has expired).
Once the action has expired, the runner will retrieve results from the API
and store these results in the runner directory. The runner can also send the
results from MIG to an external program for automatic parsing or formatting,
for example to create events for MozDef and send them.

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
        delayresults = "30s"; Duration after action expiry to fetch results

If the GPG key used by mig-runner is protected by a passphrase, the
`passphrase` option can be included under the `client` section. If this is
specified this passphrase will be used to access the private key.

The `delayresults` value is optional. If not set, the runner will attempt
to fetch action results when the action has expired. If this is set to a
duration string value, the runner will wait the specified duration after
action expiry before fetching results (for example to ensure all results
are written to the database by the scheduler).

The `checkdirectory` option specifies the number of seconds that elapse
between scans of the runner directory for job changes. The runner will
automatically add or remove jobs as new jobs are added in the spool
directory. When the runner identifies changes to a job, this will be
indicated in the log file as the old job configuration will be removed
and the new configuration installed.

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
into stdin of this executable in JSON format (mig.RunnerResult). The
plugin can then parse and forward the data as needed.

Optionally the `expiry` setting can be set to a go Duration string to use
for action expiry (for example 10m for 10 minutes). If this is not set
in the job configuration, a default of 5 minutes will be used.

Optionally the `sendonly` setting can be set to true, which will result in
the runner only dispatching the action, but not attempting to retrieve
any results. As an example, this can be used to deploy actions that would
be processed by MIG workers, rather than retrieved and processed on the
runner side.

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
any language. If a plugin is being developed in go, the mig.RunnerResult
type can be used to parse incoming data. In other languages the JSON can
be parsed as desired.

