# Libaudit in Golang
Golang package (lib) for Linux Audit

Libaudit-go is a pure Go client library for dealing directly with linux audit framework.
The idea is provide a replacement to the existing auditd daemon and its libraries.
Originally developed for [Audit Go Heka Plugin](https://github.com/mozilla/audit-go)

To get started see package documentation at [godoc](https://godoc.org/github.com/mozilla/libaudit-go).

See [main.go](https://github.com/mozilla/audit-go/blob/master/main.go) for an example implementation of the client using libaudit-go.

## Supported Methods (API)

### General 


##### NewNetlinkConnection 
Open a audit netlink socket connection
Similar to `audit_open`, `NewNetlinkConnection` creates a `NETLINK_AUDIT` socket for communication with the kernel part of the Linux Audit Subsystem.

It provide three methods

* Close 
* Send
* Receive

Example : 
```
s, err := libaudit.NewNetlinkConnection()

if err != nil {
    log.Println(err)
    log.Fatalln("Error while availing socket! Exiting!")
} 

defer s.Close()
```
Definations of Send and Receive are :

**Send**

``` 
func (s *NetlinkConnection) Send(request *NetlinkMessage) error 
```

**Receive**

``` 
func (s *NetlinkConnection) Receive(bytesize int, block int) ([]NetlinkMessage, error) 
```


##### GetAuditEvents

Starts an Audit event monitor in a go-routine.

```
func AuditGetEvents(s *NetlinkConnection, cb EventCallback, ec chan error, args ...interface{})
```

This function start a audit event monitor and accept a callback that is called on each audit event received from the Audit Subsysten.

Example:

```golang

func EventCallback(msg *libaudit.AuditEvent, ce chan error, args ...interface{}) {
	// print the info map
	log.Println(msg.Data)
	// print the raw event
	log.Println(msg.Raw)
}

// Go rutine to monitor events and call callback for each event fired
libaudit.GetAuditEvents(s, EventCallback, errchan)
```

The callback accept AuditEvent type variable as an argument. AuditEvent is defined as

```golang
type AuditEvent struct {
	Serial				int
	Timestamp			float64
	Type 				string
	Data 				map[string]string
	Raw 				string
}
```

##### AuditGetRawEvents

Starts an Audit event monitor which emits raw events in a go-routine

```golang
func GetRawAuditEvents(s *NetlinkConnection, cb RawEventCallback, ec chan error, args ...interface{})
```
Same as GetAuditEvents but accept a string type in callback instead of AuditEvent type.

Example -

``` golang
func RawEventCallback(msg string, ce chan error, args ...interface{}) {
	log.Println(msg)
}

// Go rutine to monitor events and feed raw events to the callback
libaudit.GetRawAuditEvents(s, RawEventCallback, errchan)
```

##### AuditIsEnabled

This function will return 0 if audit is not enabled and 1 if enabled, and -1 on error.

```
func AuditIsEnabled(s *NetlinkConnection) (state int, err error)
```

Example :

```
status, err := libaudit.AuditIsEnabled(s)
```

##### AuditRequestStatus

Not yet implemented


### Audit Set

##### AuditSetEnabled

Enable or disable auditing, 1 to enable and 0 to disable.

```
func AuditSetEnabled(s *NetlinkConnection) error
```

Example : 

```
err := libaudit.AuditSetEnabled(s, 1)
```



##### AuditSetRateLimit

Sets rate limit for audit messages from kernel
```
func AuditSetRateLimit(s *NetlinkConnection, limit int) error
```

This function set the maximum number of messages that the kernel will send per second.

Example:

```
err = libaudit.AuditSetRateLimit(s, 600)
```

##### AuditSetBacklogLimit

Sets backlog limit for audit messages from kernel

```
func AuditSetBacklogLimit(s *NetlinkConnection, limit int) error
```

This function sets the queue length for audit events awaiting transfer to the audit daemon


Example :
```
err = libaudit.AuditSetBacklogLimit(s, 420)
```

##### AuditSetPid

Set audit daemon process ID


```
func AuditSetPid(s *NetlinkConnection, pid uint32 ) error 
```

This function registers the given PID with kernel as the program for receiving audit messages.


Example :
```
err = libaudit.AuditSetPid(s, uint32(syscall.Getpid()))
```

##### AuditSetFailure

Not yet implemented


### Audit Rules

##### SetRules

Set audit rules from a configuration file

```
func SetRules(s *NetlinkConnection, content []byte) error
```

This function accept the json rules file as byte array and register rules with audit.
See [audit.rules.json](https://github.com/mozilla/audit-go/blob/master/audit.rules.json) for example

Example:

```golang
// Load all rules
content, err := ioutil.ReadFile("audit.rules.json")
if err != nil {
	log.Print("Error:", err)
	os.Exit(0)
}

// Set audit rules
err = libaudit.SetRules(s, content)
```


##### DeleteAllRules

Delete all audit rules.

```
func DeleteAllRules(s *NetlinkConnection) error
```
Example:

```golang
err := DeleteAllRules(s)
```


##### ListAllRules

ListAllRules lists all audit rules currently loaded in audit kernel in the same format as shown by auditctl utility.

```
func ListAllRules(s *NetlinkConnection) ([]string, error)
```
Example:

```golang
	rulesArray, err := libaudit.ListAllRules(s)
```