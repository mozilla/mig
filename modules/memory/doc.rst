===================================
Mozilla InvestiGator: Memory module
===================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The memory module (MM) allows an investigator to inspect the content of the
memory of running processes without impacting the stability of a system. MM
implements the `Masche <https://github.com/mozilla/masche>`_ cross-platform
memory scanning library.

Usage
-----

MM implements searches that are defined by a search label. A search can have a
number of search parameters and options, defined below. There is no maximum
number of searches that can be performed by a single invocation of MM. The
module optimize searches to only scan the memory of each process once, and for
each buffer scanned, runs all needed checks on it.

MM can filter processes on their name or their linked libraries. A standard way
to use the module is to set the `MatchAll` option to `true` and specify a name
and a content or byte string to search for. MM will first filter the processes
that match the name, a cheap check to perform, and because `MatchAll` is true,
will only peek inside the memory of selected processes.

Without `MatchAll`, all checks are ran on all processes, which can be very
costly on a system that has a large memory usage.

In JSON format, searches are defined as a json object where each search has a
label (key) and search parameters (value).

A search label is a string between 1 and 64 characters, composed of letter
[a-zA-z], numbers [0-9], underscore [_] or dashes [-].

.. code:: json

	{
		"searches": {
			"somesearchlabel": {
				"names": [
					"firefox"
				],
				"contents": [
					"some bogus content"
				]
			},
			"another_search": {
				"libraries": [
					"libssl"
				],
				"bytes": [
					"afd37df8b18462"
				],
				"options": {
					"matchall": true,
					"maxlength": 50000
				}
			}
        }
    }

Filters
~~~~~~~
Search filters can be used to locate a process on its name, libraries or
content. Filters can be applied in two ways: either `matchall` is set and all
filters must match on a given process to include it in the results, or `matchall`
is not set and filters are treated individually.

Note: all regular expressions used in search filters use the regexp syntax
provided by the Go language, which is very similar to Posix. A full description
of the syntax is available at http://golang.org/pkg/regexp/syntax/.

* **names**: an array of regular expressions that are applied to the full
  executable path of a process

* **libraries**: an array of regular expressions that are applied to the linked
  libraries of a process. This filter does not match on static binaries.

* **contents**: an array of regular expressions that are applied to the memory
  content of a process. Beware that the regexes are utf-8 and some processes may
  use non-utf8 encoding internally (java does that). Consider using a byte
  string to match unusual encoding.

* **bytes**: an array of hexadecimal bytes strings that are search for in the
  memory content of a process.

Options
~~~~~~~

Several options can be applied to a search:

* **matchall** indicates that within a given search, all search filters must
  match on one process for it to be included in the results. Being a boolean,
  `matchall` is not set by default. The MIG command line sets it automatically,
  the console does not.

* **offset** can be used to set a non-zero start address used to skip some
  memory regions when scanning a process. This is useful when scanning very
  large processes.

* **maxlength** can be used to stop the scanning of the memory of a process when
  X number of bytes have been read. This is useful when scanning very large
  processes.

* **logfailures** indicates whether MM should return detailed logs of memory
  walking failures. Failures happen all the time because processes have regions
  that are locked and cannot be read. The underlying Masche library does not
  attempt to force its way through unreadable memory regions by default, but
  skips and logs them instead.

Memory scanning algorithm
-------------------------

The memory of a process is read from `offset` until `maxlength` by chunks of 4kB
by default. If one of the search includes a byte string that's longer than 4kB,
the size of the buffer is increased to twice the size of the longest byte
string to accomodate it.

Memory is read sequentially, and the buffer is moved forward by half of its size
at each iteration, meaning that the memory of a given process is read twice in
the sliding buffer::

	v-offset										v-maxlength
	|----------process-memory------------------------------------------------|
	[--- buffer i=1 ---]
				[--- buffer i=2 ---]
							[--- buffer i=3 ---]
									[--- buffer i=4 ---]STOP

All searches that are currently active are ran on a copy of the buffer. A given
memory region is only ever read once, regardless of the number of searches being
performed.

Walking the memory stops either when all the memory has been read, when
`maxlength` is reached, or as soon as all search filters have matched once.

