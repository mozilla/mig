// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"io/ioutil"
	"mig"
	"mig/pgp"
	"net/http"
	"strconv"

	"github.com/jvehent/cljs"
)

// getInvestigator takes an investigatorid and returns an investigator
func getInvestigator(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getInvestigator()"}.Debug()
	}()
	iid, err := strconv.ParseUint(request.URL.Query()["investigatorid"][0], 10, 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'investigatorid': '%v'", err)
		panic(err)
	}

	// retrieve the investigator
	var inv mig.Investigator
	if iid > 0 {
		inv, err = ctx.DB.InvestigatorByID(iid)
		if err != nil {
			if fmt.Sprintf("%v", err) == "Error while retrieving investigator: 'sql: no rows in result set'" {
				// not found, return 404
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Investigator ID '%.0f' not found", iid)})
				respond(404, resource, respWriter, request)
				return
			} else {
				panic(err)
			}
		}
	} else {
		// bad request, return 400
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: fmt.Sprintf("Invalid Investigator ID '%.0f'", iid)})
		respond(400, resource, respWriter, request)
		return
	}
	// store the results in the resource
	investigatorItem, err := investigatorToItem(inv)
	if err != nil {
		panic(err)
	}
	resource.AddItem(investigatorItem)
	respond(200, resource, respWriter, request)
}

// describeCreateInvestigator returns a resource that describes how to create an investigator
func describeCreateInvestigator(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving describeCreateInvestigator()"}.Debug()
	}()
	err = resource.SetTemplate(cljs.Template{
		Data: []cljs.Data{
			{Name: "name", Value: "investigator's full name", Prompt: "Investigator Name"},
			{Name: "publickey", Value: "armored GPG public key", Prompt: "Public Key"},
		},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request)
}

// createInvestigator creates an investigator into the database
func createInvestigator(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving createInvestigator()"}.Debug()
	}()
	var inv mig.Investigator
	err = request.ParseMultipartForm(20480)
	if err != nil {
		panic(err)
	}
	inv.Name = request.FormValue("name")
	if inv.Name == "" {
		panic("Investigator name must not be empty")
	}
	// publickey is stored in a multipart post form, extract it
	_, keyHeader, err := request.FormFile("publickey")
	if err != nil {
		panic(err)
	}
	keyReader, err := keyHeader.Open()
	if err != nil {
		panic(err)
	}
	inv.PublicKey, err = ioutil.ReadAll(keyReader)
	if err != nil {
		panic(err)
	}
	if len(inv.PublicKey) == 0 {
		panic("Investigator Public Key must not be empty")
	}
	// validate the public key and obtain a fingerprint from it
	inv.PGPFingerprint, err = pgp.LoadArmoredPubKey(inv.PublicKey)
	if err != nil {
		panic(err)
	}
	// create the investigator in database
	inv.ID, err = ctx.DB.InsertInvestigator(inv)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "Investigator created in database"}
	err = resource.AddItem(cljs.Item{
		Href: fmt.Sprintf("%s/investigator?investigatorid=%.0f", ctx.Server.BaseURL, inv.ID),
		Data: []cljs.Data{{Name: "Investigator ID " + fmt.Sprintf("%.0f", inv.ID), Value: inv}},
	})
	respond(201, resource, respWriter, request)
}

// describeUpdateInvestigator returns a resource that describes how to update the status of an investigator
func describeUpdateInvestigator(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving describeUpdateInvestigator()"}.Debug()
	}()
	err = resource.SetTemplate(cljs.Template{
		Data: []cljs.Data{
			{Name: "id", Value: "investigator id to update", Prompt: "Investigator ID"},
			{Name: "status", Value: "new status of investigator", Prompt: "Investigator Status"},
		},
	})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request)
}

// updateInvestigator updates the status of an investigator in database
func updateInvestigator(respWriter http.ResponseWriter, request *http.Request) {
	var err error
	opid := getOpID(request)
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			emsg := fmt.Sprintf("%v", e)
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: emsg}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: emsg})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving updateInvestigator()"}.Debug()
	}()
	var inv mig.Investigator
	err = request.ParseForm()
	if err != nil {
		panic(err)
	}
	iid := request.FormValue("id")
	if iid == "" {
		panic("Investigator ID must not be empty")
	}
	inv.ID, err = strconv.ParseUint(iid, 10, 4)
	if err != nil {
		panic(err)
	}
	inv.Status = request.FormValue("status")
	if inv.Status == "" {
		panic("Investigator status must not be empty")
	}
	// create the investigator in database
	err = ctx.DB.UpdateInvestigatorStatus(inv)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Investigator %.0f status changed to %s", inv.ID, inv.Status)}
	err = resource.AddItem(cljs.Item{
		Href: fmt.Sprintf("%s/investigator?investigatorid=%.0f", ctx.Server.BaseURL, inv.ID),
		Data: []cljs.Data{{Name: "Investigator ID " + fmt.Sprintf("%.0f", inv.ID), Value: inv}},
	})
	respond(200, resource, respWriter, request)
}

// investigatorToItem receives a command and returns an Item in Collection+JSON
func investigatorToItem(inv mig.Investigator) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/investigator?investigatorid=%.0f", ctx.Server.BaseURL, inv.ID)
	links := make([]cljs.Link, 0)
	link := cljs.Link{
		Rel:  "investigator history",
		Href: fmt.Sprintf("%s/search?type=action&investigatorid=%.0f&limit=100", ctx.Server.BaseURL, inv.ID),
	}
	links = append(links, link)
	item.Links = links
	item.Data = []cljs.Data{
		{Name: "investigator", Value: inv},
	}
	return
}
