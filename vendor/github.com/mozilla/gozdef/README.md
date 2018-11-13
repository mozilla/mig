gozdef - Go client to send events to MozDef
===========================================

http://godoc.org/github.com/jvehent/gozdef

```go
package main

import (
    "github.com/jvehent/gozdef"
    "log"
)

func main() {
    // initialize a publisher to mozdef's rabbitmq relay
    conf := gozdef.MqConf{
        Host:       "mozdef.rabbitmq.example.net",
        Port:       5671,
        User:       "gozdefclient",
        Pass:       "s3cr3tpassw0rd",
        Vhost:      "mozdef",
        Exchange:   "eventtask",
        RoutingKey: "eventtask",
        UseTLS:     true,
        CACertPath: "/etc/pki/CA/certs/ca.crt",
        Timeout:    "10s",
    }
    publisher, err := gozdef.InitAmqp(conf)
    if err != nil {
        log.Fatal(err)
    }

    // create a new event and set values in the fields
    ev, err := gozdef.NewEvent()
    if err != nil {
        log.Fatal(err)
    }
    ev.Category = "demo"
    ev.Source = "test client"
    ev.Summary = "tl;dr: everything's fine!"
    ev.Tags = append(ev.Tags, "gozdef")

    // add details to the event, these fields are completely customizable
    ev.Details = struct {
        SrcIP   string `json:"sourceipaddress"`
        DestIP  string `json:"destinationipaddress"`
        Offense string `json:"offense"`
        Blocked bool
    }{
        SrcIP:   "10.0.1.2",
        DestIP:  "192.168.1.5",
        Offense: "brute force",
        Blocked: true,
    }

    // set the event severity to INFO
    ev.Info()

    // publish to mozdef
    err = publisher.Send(ev)
    if err != nil {
        log.Fatal(err)
    }
}
```
