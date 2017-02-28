# libaudit in Go

libaudit-go is a go package for interfacing with Linux audit.

[![Build Status](https://travis-ci.org/mozilla/libaudit-go.svg?branch=master)](https://travis-ci.org/mozilla/libaudit-go)
[![Go Report Card](https://goreportcard.com/badge/mozilla/libaudit-go "Go Report Card")](https://goreportcard.com/report/mozilla/libaudit-go)

libaudit-go is a pure Go client library for interfacing with the Linux auditing framework. It provides functions
to interact with the auditing subsystems over Netlink, including controlling the rule set and obtaining/interpreting
incoming audit events.

libaudit-go can be used to build go applications which perform tasks similar to the standard Linux auditing daemon
`auditd`.

To get started see package documentation at [godoc](https://godoc.org/github.com/mozilla/libaudit-go).

For a simple example of usage, see the [auditprint](./auditprint/) tool included in this repository.

```bash
sudo service stop auditd
go get -u github.com/mozilla/libaudit-go
cd $GOPATH/src/github.com/mozilla/libaudit-go
go install github.com/mozilla/libaudit-go/auditprint
sudo $GOPATH/bin/auditprint testdata/rules.json
```

Some key functions are discussed in the overview section below.

## Overview

### General 

##### NewNetlinkConnection 

To use libaudit-go programs will need to initialize a new Netlink connection. `NewNetlinkConnection` can be used
to allocate a new `NetlinkConnection` type which can then be passed to other functions in the library.

```go
s, err := libaudit.NewNetlinkConnection()
if err != nil {
        fmt.Printf("NewNetlinkConnection: %v\n", err)
} 
defer s.Close()
```

`NetlinkConnection` provides a `Send` and `Receive` method to send and receive Netlink messages to the kernel,
however generally applications will use the various other functions included in libaudit-go and do not need to
call these functions directly.

##### GetAuditEvents

GetAuditEvents starts an audit event monitor in a go-routine and returns. Programs can call this function and
specify a callback function as an argument. When the audit event monitor receives a new event, this callback
function will be called with the parsed `AuditEvent` as an argument.

```go

func myCallback(msg *libaudit.AuditEvent, err error) {
        if err != nil {
            // An error occurred getting or parsing the audit event
            return
        }
	// Print the fields
        fmt.Println(msg.Data)
	// Print the raw event
        fmt.Println(msg.Raw)
}

libaudit.GetAuditEvents(s, myCallback)
```

##### GetRawAuditEvents

`GetRawAuditEvents` behaves in a similar manner to `GetAuditEvents`, however programs can use this function
to instead just retrieve raw audit events from the kernel as a string, instead of having libaudit-go parse
these audit events into an `AuditEvent` type.

### Audit Rules

Audit rules can be loaded into the kernel using libaudit-go, however the format differs from the common rule
set used by userspace tools such as auditctl/auditd.

libaudit-go rulesets are defined as a JSON document. See [rules.json](./testdata/rules.json) as an example.
The libaudit-go type which stores the rule set is `AuditRules`.

##### SetRules

`SetRules` can be used to load an audit rule set into the kernel. The function takes a marshalled `AuditRules`
type as an argument (slice of bytes), and converts the JSON based rule set into a set of audit rules suitable
for submission to the kernel.

The function then makes the required Netlink calls to clear the existing rule set and load the new rules.

```go
// Load all rules from a file
content, err := ioutil.ReadFile("audit.rules.json")
if err != nil {
        fmt.Printf("error: %v\n", err)
	os.Exit(1)
}

// Set audit rules
err = libaudit.SetRules(s, content)
if err != nil {
        fmt.Printf("error: %v\n", err)
        os.Exit(1)
}
```
