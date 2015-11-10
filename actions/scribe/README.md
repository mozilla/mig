# Scribe policies for vulnerability management

This folder contains [Scribe](https://github.com/mozilla/scribe/) policy files
generated from the [OpenVAS NVT](http://www.openvas.org/openvas-nvt-feed.html)
datasets. The data is provided under the GPLv2 license.

* `rhsa-2015.json` is generated from RHSA data for Red Hat systems in the NVT database.
* `usn-2015.json` is generated from USN data for Ubuntu systems in the NVT database.

Run them like this:
```bash
$ mig scribe -t "environment->>'ident' LIKE 'Red%'" -path rhsa-2015.json -onlytrue -human
```

```bash
$ mig scribe -t "environment->>'ident' LIKE 'Ubuntu%'" -path usn-2015.json -onlytrue -human
```
