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
					"^root",
					"!^root:\\$6"
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
				"sha2": [
					"fff415292dc59cc99d43e70fd69347d09b9bd7a581f4d77b6ec0fa902ebaaec8"
				],
				"options": {
					"matchall": true,
					"maxdepth": 3,
					"decompress": true
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

While browsing a path, the module will follow symlinks if they are located
within the base search path. For example, if the base path is set to
'/sys/bus/usb/devices/' and a symlink is found pointing to '/sys/devices', the
symlink will **not** be followed because it points to a location outside of the
base search path.

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
  If the regex is prefixed with "!", it will return files that do not match the
  expression.

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
  If the regex is prefixed with "!", it will return files that do not have the
  content that matches the expression. ex: `!^root:\$6` will return files that
  do not contain the string "root:$6".

* **md5**: a md5 checksum

* **sha1**: a sha1 checksum

* **sha2**: a sha2 checksum (sha256/sha384/sha512 decided based on hash length)

* **sha3**: a sha3 checksum (sha3_224/sha3_256/sha3_384/sha3_512 decided based
  on hash length)

Search Options
~~~~~~~~~~~~~~

Several options can be applied to a search:

* **maxdepth** controls the maximum number of directories that can be traversed
  by a search. For example, is a search has path `/home`, and `maxdepth` is set
  to the value 3, the deepest directory that can be visited is
  `/home/dir1/dir2/dir3`.

* **matchall** indicates that within a given search, all search filters must
  match on one file for it to be included in the results. Being a boolean,
  `matchall` is not set by default, but the command line and the console set it
  when creating file searches. Use `matchany` to deactivate it. `matchall` has
  a strong impact on search performances. See "Search algorithm".

  Examples:
	* `-name vim -sha1 21345asd -matchall` -> (name=vim AND sha1=21345asd)
	* `-name vim -sha1 21345asd -matchany` -> (name=vim OR sha1=21345asd)

* **macroal** stands for "Match All Contents Regexes On All Lines". It's a boolean
  option that requires that all `content` regexes must match on all the lines of
  a file. By default, content regexes are applied at the file level and will
  return a match if one line matches one regex, and if another line matches another
  regex. When the `macroal` option is set, each line in the file must match all
  content regexes defined in a given search to return a match. It is set to not
  set by default.

  example: `-path /home -name authorized_keys -content "^((#.+)|(\s+)|...list of ssh keys...)$" -macroal`

  will list authorized_keys file that have contain either a comment, an empty
  line or one of the listed ssh keys. It will only return a file in the results
  if all the lines of the file match the regex.

* **mismatch=<filter>** inverts the results for the given filter. This can be used
  to list files that did not match a given check, instead of the default which
  returns files that match a check.

  For example, the following search will return files where all lines match the
  content regex:

  `mig file -path /home -name ^authorized_keys -content "^((#.+)|(\s+)|..1stkey..|..2ndkey..)$" -macroal`

  But this search cannot list files that fail to match the content regex, which
  could be useful if we're looking for a file that contains a rogue SSH key.
  The mismatch option can be applied to the content filter to achieve this:

  `mig file -path /home -name ^authorized_keys -content "^((#.+)|(\s+)|..1stkey..|..2ndkey..)$" -macroal -mismatch content`

  This search will locate all authorized_keys files and the inspect their
  content. The `macroal` flag indicates that all lines of a file must match the
  content regex. The `mismatch` flag inverses that logic, and thus if a least
  one line does not match the content regex, the file will be returned as a
  match.

  The `mismatch` option can be applied to all check types: name, size, mode,
  mtime, content, md5, sha1, sha2, ... It can be specified multiple times:

  example: `-path /usr -name "^vim$" -content "linux-x86-64\.so" -sha1 943633c85bb80d39532450decf1f723735313f1f -sha1 350ac204ac8084590b209c33f39f09986f0ba682 -mismatch=content -mismatch=sha1`

* **matchlimit** controls how many files can be returned by a single search.
  This safeguard prevents a single run of the file module from crashing before
  of the amount of results it is returning. The default value is 1,000, which is
  already significant. If you plan on returning more than 1,000 results in a
  single file search, you should probably consider breaking it down into smaller
  searches, or running the search locally instead of through MIG.

* **returnsha256** instructs the agent to return the SHA256 hash for any
  matched files. The client will display the hash with the file information
  in the result. As an example, this option can be used to do basic file
  integrity monitoring across actions.

* **decompress** tells the agent to decompress gzipped files prior to
  inspecting content or calculating hashes. Note that if the decompress flag
  is set for one search, all searches will involve a test for file
  decompression.

* **maxerrors** sets the maximum number of walking errors returned by the file
  module while searching a path. Walking errors can rapidly increase when
  scanning pseudo file systems like /proc, and limiting them to a sensible
  number reduces noise. The default is set to 30. A value of 0 removes the limit
  and will return all walking errors.

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
content regexes and hashes are evaluated next. This approach increases the speed
of a search because fileinfo filters are significantly faster than content
filters.

The case of content regex is particular, because evaluation of the file stops
at the first positive occurence of the regex in a file. This is meant to speed
up searches on large files that may match a large number of times. The `macroal`
flag changes this behavior by requiring that all lines must match the content
regexes. When `macroal` is set, content inspection reads the entire file.

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

Note on symbolic links
~~~~~~~~~~~~~~~~~~~~~~

FM does not follow directory links but will follow file links. Directory links
could lead FM to scan a path that is far out of its initial search scope, and
can also lead to loops. A warning will be stored in the results when a directory
link was encountered and not followed.
