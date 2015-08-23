CLJS: Collection+JSON in Go
===========================

cljs is a module that facilitates the generation of Collection+JSON resources in
a REST API. It is designed to be thread-safe, and have solid grammar validation.

```go

package main

import (
    "fmt"
    "github.com/jvehent/cljs"
)

func main() {
    resource := cljs.New("/root/url/of/api")

    // add a link
    err := resource.AddLink(cljs.Link{
        Rel:  "home",
        Href: "/api/home",
        Name: "home"})
    if err != nil {
        panic(err)
    }

    // add an item, first define the data and links slices,
    // then insert into the resource
    data := []cljs.Data{
        {
            Name:   "bob",
            Prompt: "bob's name",
            Value:  "bob",
        },
    }
    links := []cljs.Link{
        {
            Rel:  "user",
            Href: "/api/user/bob",
            Name: "bob's details",
        },
    }
    err = resource.AddItem(cljs.Item{
        Href:  "/api/bob",
        Data:  data,
        Links: links})
    if err != nil {
        panic(err)
    }

    // set a template
    templatedata := []cljs.Data{
        {Name: "email", Value: "", Prompt: "Someone's email"},
        {Name: "full-name", Value: "", Prompt: "Someone's full name"},
    }
    resource.SetTemplate(cljs.Template{Data: templatedata})

    // set an error
    resource.SetError(cljs.Error{
        Code:    "internal error code 273841",
        Message: "somethind went wrong"})

    // generate a response body, ready to send as a HTTP response
    body, err := resource.Marshal()
    if err != nil {
        panic(err)
    }

    // return a response using net/http
    // responseWriter.Write(body)

    fmt.Printf("%s", body)
}
```

Run the code above with `go run example.go` and you will obtain the following
output:

```json
{ "collection": {
        "error": {
            "code": "internal error code 273841",
            "message": "somethind went wrong"
        },
        "href": "/root/url/of/api",
        "items": [
            {
                "data": [
                    {
                        "name": "bob",
                        "prompt": "bob's name",
                        "value": "bob"
                    }
                ],
                "href": "/api/bob",
                "links": [
                    {
                        "href": "/api/user/bob",
                        "name": "bob's details",
                        "rel": "user"
                    }
                ]
            }
        ],
        "links": [
            {
                "href": "/api/home",
                "name": "home",
                "rel": "home"
            }
        ],
        "template": {
            "data": [
                {
                    "name": "email",
                    "prompt": "Someone's email",
                    "value": ""
                },
                {
                    "name": "full-name",
                    "prompt": "Someone's full name",
                    "value": ""
                }
            ]
        },
        "version": "1.0"
    }
}
```

