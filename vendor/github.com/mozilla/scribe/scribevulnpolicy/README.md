# scribevulnpolicy

scribevulnpolicy is a tool that can be used to generate scribe policies that
perform vulnerability scans of a given target platform.

The tool integrates with [clair](https://github.com/coreos/clair), more specifically
the database clair maintains and uses this as a source of vulnerability data
for various platforms. clair does an excellent job of keeping this database up to
date from the various distributions, and scribevulnpolicy then queries this to
create scan policies.

## Quickstart

### Get clair running

To get things up and running, first you will need a running instance of clair.
clair is available as a docker image which can be used for this purpose.

Follow the [instructions](https://github.com/coreos/clair/blob/master/README.md) to
get clair running with docker-compose.

### Modify docker-compose configuration to expose database

Modify `docker-compose.yml` so the Postgres database used by clair is accessible
to scribevulnpolicy. This can be done by just adding a new `ports` entry to
`docker-compose.yml` for the postgres container.

### Set environment variables

Set the required environment variables to access the Postgres instance.

```bash
export PGUSER=postgres
export PGPASSWORD=password
export PGHOST=127.0.0.1
export PGDATABASE=postgres
```

### Generate policy

The list of available platforms can be determined by running scribevulnpolicy
with the `-V` flag.

```bash
$ $GOPATH/bin/scribevulnpolicy -V
centos6
centos7
```

Not all platforms clair maintains vulnerability data for may be available in
scribevulnpolicy, some require support to be added.

Finally, the policy can be generated.

`$ $GOPATH/bin/scribevulnpolicy centos6 > centos6.json`

This policy can then be run directly on the system using `scribecmd`, or through
an integrated scanning tool such as [mig](https://github.com/mozilla/mig) where
it will return any identified vulnerabilities on the system.
