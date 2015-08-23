/* Go module for Collection+JSON

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2014
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

// cljs provides a thread safe interface to the manipulation of API resources that follow
// the syntax of Collection+JSON from http://amundsen.com/media-types/collection/format/
//
// cljs provides grammar checking for version 1.0 of the Collection+JSON standard.
//
// Sample code:
//
//	import "github.com/jvehent/cljs"
//	...
//	resource := cljs.New(request.URL.Path)
//
//	// add a link
//	err := resource.AddLink(cljs.Link{
//		Rel:  "home",
//		Href: request.URL.Path,
//		Name: "home"})
//
//	// add an item, first define the data and links slices,
//	// then insert into the resource
//	data := []cljs.Data{
//		{
//			Name:   "bob",
//			Prompt: "bob's name",
//			Value:  "bob",
//		},
//	}
//	links := []cljs.Link{
//		{
//			Rel:  "user",
//			Href: "/api/user/bob",
//			Name: "bob's details",
//		},
//	}
//	err = resource.AddItem(cljs.Item{Href: "/api/bob", Data: data, Links: links})
//	if err != nil {
//		panic(err)
//	}
//
//	// set a template
//	templatedata := []cljs.Data{
//		{Name: "email", Value: "", Prompt: "Someone's email"},
//		{Name: "full-name", Value: "", Prompt: "Someone's full name"},
//	}
//	resource.SetTemplate(cljs.Template{Data: templatedata})
//
//	// set an error
//	resource.SetError(cljs.Error{
//		Code: "internal error code 273841",
//		Message: "somethind went wrong"})
//
//	// generate a response body, ready to send as a HTTP response
//	body, err := resource.Marshal()
//
//	responseWriter.Write(body)	// from net/http module
//
// Resource structure in pseudo Go:
//
//	Resource {
//		Collection map[string]interface{} {
//			Version: "1.0",
//			Href: "/api/",
//			Links: []Link,
//			Items: []Item,
//			Queries: []Query,
//			Template: Template,
//			Error: Error
//		}
//	}
//
// Example JSON Resource (from the Collection+JSON standard):
//
//    { "collection" :
//      {
//        "version" : "1.0",
//        "href" : "http://example.org/friends/",
//        "links" : [
//          {"rel" : "feed", "href" : "http://example.org/friends/rss"}
//        ],
//        "items" : [
//          {
//            "href" : "http://example.org/friends/jdoe",
//            "data" : [
//              {"name" : "full-name", "value" : "J. Doe", "prompt" : "Full Name"},
//              {"name" : "email", "value" : "jdoe@example.org", "prompt" : "Email"}
//            ],
//            "links" : [
//              {"rel" : "blog", "href" : "http://examples.org/blogs/jdoe", "prompt" : "Blog"},
//              {"rel" : "avatar", "href" : "http://examples.org/images/jdoe", "prompt" : "Avatar", "render" : "image"}
//            ]
//          },
//          {
//            "href" : "http://example.org/friends/msmith",
//            "data" : [
//              {"name" : "full-name", "value" : "M. Smith", "prompt" : "Full Name"},
//              {"name" : "email", "value" : "msmith@example.org", "prompt" : "Email"}
//            ],
//            "links" : [
//              {"rel" : "blog", "href" : "http://examples.org/blogs/msmith", "prompt" : "Blog"},
//              {"rel" : "avatar", "href" : "http://examples.org/images/msmith", "prompt" : "Avatar", "render" : "image"}
//            ]
//          },
//          {
//            "href" : "http://example.org/friends/rwilliams",
//            "data" : [
//              {"name" : "full-name", "value" : "R. Williams", "prompt" : "Full Name"},
//              {"name" : "email", "value" : "rwilliams@example.org", "prompt" : "Email"}
//            ],
//            "links" : [
//              {"rel" : "blog", "href" : "http://examples.org/blogs/rwilliams", "prompt" : "Blog"},
//              {"rel" : "avatar", "href" : "http://examples.org/images/rwilliams", "prompt" : "Avatar", "render" : "image"}
//            ]
//          }
//        ],
//        "queries" : [
//          {"rel" : "search", "href" : "http://example.org/friends/search", "prompt" : "Search",
//            "data" : [
//              {"name" : "search", "value" : ""}
//            ]
//          }
//        ],
//        "template" : {
//          "data" : [
//            {"name" : "full-name", "value" : "", "prompt" : "Full Name"},
//            {"name" : "email", "value" : "", "prompt" : "Email"},
//            {"name" : "blog", "value" : "", "prompt" : "Blog"},
//            {"name" : "avatar", "value" : "", "prompt" : "Avatar"}
//          ]
//        }
//      }
//    }
//
package cljs

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Collection+JSON uses the ContentType 'application/vnd.collection+json' in
// HTTP response. This must be set in the HTTP header.
// For example, using Go's 'net/http' module:
//    responseWriter.Header().Set("Content-Type", cljs.ContentType)
var ContentType = "application/vnd.collection+json"

// Resource is a top-level document returned by an API
type Resource struct {
	Collection Collection `json:"collection"`
	mutex      sync.Mutex
}

// Collection is a container for everything inside a resource
type Collection struct {
	Version  string   `json:"version"`
	Href     string   `json:"href"`
	Links    []Link   `json:"links,omitempty"`
	Items    []Item   `json:"items,omitempty"`
	Queries  []Query  `json:"queries,omitempty"`
	Template Template `json:"template,omitempty"`
	Error    Error    `json:"error,omitempty"`
}

// New initializes a Resource. It sets the version number. The location of the
// document being returned is inialized to the value passed in 'root', which
// should be set to the URL to the root of the API.
func New(root string) *Resource {
	var r Resource
	r.Collection.Version = "1.0"
	r.Collection.Href = root
	return &r
}

// Marshal validates the syntax of a Resource and returns its json encoded
// version in a byte array.
func (r Resource) Marshal() (rj []byte, err error) {
	err = r.Validate()
	if err != nil {
		err = fmt.Errorf("Resource marshalling failed with error '%v'", err)
		return
	}

	rj, err = json.Marshal(r)
	if err != nil {
		err = fmt.Errorf("Resource marshalling failed with error '%v'", err)
		return
	}
	return
}

// Validate makes sure that the Resource conforms to the standard syntax
func (r Resource) Validate() (err error) {
	if r.Collection.Version != "1.0" {
		return fmt.Errorf("wrong version. Must be '1.0'")
	}
	if r.Collection.Href == "" {
		return fmt.Errorf("'href' is empty. Must contains resource location")
	}
	for i, link := range r.Collection.Links {
		err = link.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate link %d: %v", i, err)
		}
	}
	for i, item := range r.Collection.Items {
		err = item.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate item %d: %v", i, err)
		}
	}
	for i, query := range r.Collection.Queries {
		err = query.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate query %d: %v", i, err)
		}
	}
	err = r.Collection.Template.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate template: %v", err)
	}
	err = r.Collection.Error.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate resource error: %v", err)
	}
	return
}
