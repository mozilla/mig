// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"encoding/json"
	"fmt"
	"github.com/jvehent/cljs"
	"mig.ninja/mig"
	"net/http"
	"strconv"
)

// Locate the manifest record that should be served to a given loader ID. Note
// the agent environment is also passed in here so the API can update the loader
// entry with the last-known state of the agent.
func locateManifestFromLoader(loaderid float64, agt mig.Agent) (mig.ManifestRecord, error) {
	var ret mig.ManifestRecord
	err := ctx.DB.UpdateLoaderEntry(loaderid, agt)
	if err != nil {
		return ret, err
	}
	manifestid, err := ctx.DB.ManifestIDFromLoaderID(loaderid)
	if err != nil {
		return ret, err
	}
	ret, err = ctx.DB.GetManifestFromID(manifestid)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func signManifest(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving signManifest()"}.Debug()
	}()

	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received manifest sign request")}.Debug()

	manifestid, err := strconv.ParseFloat(request.FormValue("manifestid"), 64)
	if err != nil {
		panic(err)
	}
	sig := request.FormValue("signature")
	if sig == "" {
		panic("Invalid signature specified")
	}

	err = ctx.DB.ManifestAddSignature(manifestid, sig, getInvID(request))
	if err != nil {
		panic(err)
	}

	respond(200, resource, respWriter, request)
}

func getManifest(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getManifest()"}.Debug()
	}()
	mid, err := strconv.ParseFloat(request.URL.Query()["manifestid"][0], 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'manifestid': '%v'", err)
		panic(err)
	}

	var mr mig.ManifestRecord
	if mid > 0 {
		mr, err = ctx.DB.GetManifestFromID(mid)
		if err != nil {
			if fmt.Sprintf("%v", err) == "Error while retrieving manifest: 'sql: no rows in result set'" {
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Manifest ID '%.0f' not found", mid)})
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
			Message: fmt.Sprintf("Invalid Manifest ID '%.0f'", mid)})
		respond(400, resource, respWriter, request)
		return
	}
	mi, err := manifestRecordToItem(mr, ctx)
	if err != nil {
		panic(err)
	}
	resource.AddItem(mi)
	respond(200, resource, respWriter, request)
}

// API entry point used to request a file be sent to the loader from the API.
func getManifestFile(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getManifestFile()"}.Debug()
	}()
	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received manifest file request")}.Debug()

	var manifestParam mig.ManifestParameters
	err = json.Unmarshal([]byte(request.FormValue("parameters")), &manifestParam)
	if err != nil {
		panic(err)
	}
	err = manifestParam.ValidateFetch()
	if err != nil {
		panic(err)
	}

	// Validate the loader key
	loaderid, err := ctx.DB.GetLoaderEntryID(manifestParam.LoaderKey)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Loader request for entry %v", loaderid)}.Debug()

	// Update the loader entry with the parameters, and locate a valid manifest
	mf, err := locateManifestFromLoader(loaderid, manifestParam.AgentIdentifier)
	if err != nil {
		panic(err)
	}
	data, err := mf.ManifestObject(manifestParam.Object)
	if err != nil {
		panic(err)
	}
	fetchresp := mig.ManifestFetchResponse{}
	fetchresp.Data = data

	// Send the response to the loader
	err = resource.AddItem(cljs.Item{
		Href: request.URL.String(),
		Data: []cljs.Data{
			{
				Name:  "content",
				Value: fetchresp,
			},
		}})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request)
}

// This API entry point is used by the loader to request a manifest file that
// indicates the most current version of the agent to be used. The loader
// sends some basic information in the request parameters so the API can decide
// which manifest to send the loader, and authenticate the loader instance.
//
// If the key passed in the request is not valid, the request will be rejected.
func getAgentManifest(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(500, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAgentManifest()"}.Debug()
	}()
	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	var manifestParam mig.ManifestParameters
	err = json.Unmarshal([]byte(request.FormValue("parameters")), &manifestParam)
	if err != nil {
		panic(err)
	}
	err = manifestParam.Validate()
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received manifest request")}.Debug()

	// Validate the loader key
	loaderid, err := ctx.DB.GetLoaderEntryID(manifestParam.LoaderKey)
	if err != nil {
		panic(err)
	}
	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Loader request for entry %v", loaderid)}.Debug()

	// Update the loader entry with the parameters, and locate a valid manifest
	mf, err := locateManifestFromLoader(loaderid, manifestParam.AgentIdentifier)
	if err != nil {
		panic(err)
	}
	m, err := mf.ManifestResponse()
	if err != nil {
		panic(err)
	}

	// Send the manifest to the loader
	err = resource.AddItem(cljs.Item{
		Href: request.URL.String(),
		Data: []cljs.Data{
			{
				Name:  "manifest",
				Value: m,
			},
		}})
	if err != nil {
		panic(err)
	}
	respond(200, resource, respWriter, request)
}

func manifestRecordToItem(mr mig.ManifestRecord, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/manifest?manifestid=%.0f", ctx.Server.BaseURL, mr.ID)
	item.Data = []cljs.Data{
		{Name: "manifest", Value: mr},
	}
	return
}
