=================================
Mozilla InvestiGator: File module
=================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

The file module (FM) provides a basic tools to inspect a file system. It is
inspired by "find" on Unix, and implements a subset of its functionalities
with a focus on speed of execution.

Usage
-----

FM implements searches that are defined by a search label. A search can have a
number of search parameters and options, defined below. There is no maximum
number of searches that can be performed by a single invocation of FM. However,
heavy invocations are frowned upon, because the MIG Agent will most likely kill
modules that run for more than 5 minutes (configurable).

In JSON format, searches are defined as a json object where each search has a
label (key) and search parameters (value).

A search label is a string between 1 and 64 characters, composed of letter
[a-zA-z], numbers [0-9], underscore [_] or dashes [-].

A search must have at least one search path.

.. code:: json

	{
		"searches": {
			"somesearchlabel": {
				"paths": [
					"/etc/shadow"
				],
				"contents": [
					"^root"
				]
			},
			"another_search": {
				"paths": [
					"/usr"
				],
				"sizes": [
					"<371m"
				],
				"modes": [
					"^-r-xr-x--"
				]
				"sha256": [
					"fff415292dc59cc99d43e70fd69347d09b9bd7a581f4d77b6ec0fa902ebaaec8"
				],
				"options": {
					"matchall": true,
					"maxdepth": 3
				}
			}
        }
    }

Search Paths
~~~~~~~~~~~~

A search can have an unlimited number of search paths. Each path is treated as
a string. No path expansion or regular expression is permitted in a path string.

A path can indicate a directory or a file. In the case of a directory, FM will
enter the directory structure recursively until its end is reached, or until
`maxdepth` is exceeded.

For each path defined in a search, all search filters will be evaluated.

Search Filters
~~~~~~~~~~~~~~

Search filters can be used to locate a file on its metadata (fileinfo) or
content. Filters can be applied in two ways: either `matchall` is set and all
filters must match on a given file to include it in the results, or `matchall`
is not set and filters are treated individually.

Note: all regular expressions used in search filters use the regexp syntax
provided by the Go language, which is very similar to Posix. A full description
of the syntax is available at http://golang.org/pkg/regexp/syntax/.

Metadata filters:

* **name**: a regular expression that is applying on the base name of a file.

* **size**: a size filter indicates whether we want files that are larger or
  smaller than a given size. The syntax uses a prefix `<` or `>` to indicate
  smaller than and greater than. The file size is assumed to be in bytes, and
  multipliers can be provided as suffix: `k`, `m`, `g` and `t` for kilobytes,
  megabytes, gigabytes and terabytes. For example, the filter `<10m` will match
  on files that have a size inferior than 10 megabytes. When `matchall` is set,
  several size filters can provide an efficient way to bound the search to a
  given file size window.

* **mode**: mode filters on both the type and permission of a file. The filter
  uses a regular expression that applies on the stringified filemode returned by
  Go. The mode string first contains the type of the file, followed by the
  permissions of the file.
  For example, a regular file with 640 permissions would return `-rw-r-----`
  and a regular expression on that string can be used to match the file.
  If the file has special attributes, such as setuid or sticky bits, those are
  prepended to the mode string: `gtrwx--x--x`. The meaning of each letter is
  defined in the Go documentation at http://golang.org/pkg/os/#FileMode.

* **mtime**: mtime filters on the modification time of a file. It takes a
  period parameter that checks if the file has been modified since a given
  perior, or before a given period. For example, the mtime filter `<90d` will
  match of files that have been modified over the last nighty days, while the
  filter `>5h` will match modified more than 5 hours ago.
  The mtime syntax takes a prefix `<` or `>`, a integer that represents the
  period, and a suffix `d`, `h` or `m` for days, hours and minutes.

Content filters:

* **content**: a regular expression that matches against the content of the
  file. Inspection stops at the first occurence of the regular expression that
  matches on the file.

* **md5**: a md5 checksum

* **sha1**: a sha1 checksum

* **sha256**: a sha256 checksum

* **sha384**: a sha384 checksum

* **sha512**: a sha512 checksum

* **sha3_224**: a sha3_224 checksum

* **sha3_256**: a sha3_256 checksum

* **sha3_384**: a sha3_384 checksum

* **sha3_512**: a sha3_512 checksum

Search Options
~~~~~~~~~~~~~~

Several options can be applied to a search:

* **maxdepth** controls the maximum number of directories that can be traversed
  by a search. For example, is a search has path `/home`, and `maxdepth` is set
  to the value 3, the deepest directory that can be visited is
  `/home/dir1/dir2/dir3`.

* **matchall** indicates that within a given search, all search filters must
  match on one file for it to be included in the results. Being a boolean,
  `matchall` is not set by default. The MIG command line sets it automatically,
  the console does not.

* **matchlimit** controls how many files can be returned by a single search.
  This safeguard prevents a single run of the file module from crashing before
  of the amount of results it is returning. The default value is 1,000, which is
  already significant. If you plan on returning more than 1,000 results in a
  single file search, you should probably consider breaking it down into smaller
  searches, or running the search locally instead of through MIG.

Search algorithm
----------------

FM traverse a directory tree starting from a root path and until no search are
longer active. FM traverses a given path only once, regardless of the number of
searches that are being performed. When FM enters a directory, it activates
searches that apply to the directory, and deactivates the ones that don't.
As soon as no searches are active, FM either tries another root path, or exits.

Inside a given directory, FM evaluates all files one by one. The filters on
fileinfo are first applied: name, size, mode and mtime. If the matchall option
is set, and at least one of the fileinfo filter does not match, the file is
discarded. If matchall is not set, or if all fileinfo filters match, the
filters on file content are then evaluated: content regex and checksums.

The case of content regex is particular, because evaluation of the file stops
at the first positive occurence of the regex in a file. This is meant to speed
up searches on large files that may match a large number of times.

Once all searches are deactivated, FM builds a result object from the internal
checks results. For each search, each file that matched is included once. If
the search was set to `matchall`, the search parameters are not included in the
results (we now that all of them must have matched). If `matchall` was not set,
then each file returns the list of checks that matched it. It is thus possible
to have, in one same search, a file match of a file size filter, and another
one match on a sha256 checksum.

Search activation & deactivation
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

While processing the directory structure, FM compares the current path with the
search paths of each search. A single search can have multiple paths, and if
one of them matches the current path, the search is activated.

For example, if the current path is `/var/lib/postgres`, and a search has a
path set to `/var`, the search will be activated for the current directory.

Unless the value of `maxdepth` indicates that the search should not go beyond a
certain number of subdirectories, and that number is reached. In which case,
the search is deactivated.
