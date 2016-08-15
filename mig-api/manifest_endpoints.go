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
func locateManifestFromLoader(loaderid float64, agt mig.Agent) (mr mig.ManifestRecord, err error) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("locateManifestFromLoader(): %v", e)}
			err = fmt.Errorf("Unable to locate valid manifest for this loader entry")
		}
	}()
	err = ctx.DB.UpdateLoaderEntry(loaderid, agt)
	if err != nil {
		panic(err)
	}
	// Confirm the submitted environment matches any expected environment set on the
	// loader entry. This check is intended to prevent a malicious loader process from
	// submitting a forged environment to obtain a different manifest.
	err = ctx.DB.CompareLoaderExpectEnv(loaderid)
	if err != nil {
		panic(err)
	}
	// If the check was successful, determine which manifest to send
	manifestid, err := ctx.DB.ManifestIDFromLoaderID(loaderid)
	if err != nil {
		panic(err)
	}
	mr, err = ctx.DB.GetManifestFromID(manifestid)
	if err != nil {
		panic(err)
	}
	return
}

// Manipulate the status of an existing manifest
func statusManifest(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving statusManifest()"}.Debug()
	}()

	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received manifest status change request")}.Debug()

	manifestid, err := strconv.ParseFloat(request.FormValue("manifestid"), 64)
	if err != nil {
		panic(err)
	}
	sts := request.FormValue("status")
	// A manifest can only be marked as staged, or disabled. Once a
	// manifest has been disabled, it's status can no longer be changed.
	if sts == "staged" {
		err = ctx.DB.ManifestClearSignatures(manifestid)
		if err != nil {
			panic(err)
		}
		err = ctx.DB.ManifestUpdateStatus(manifestid, ctx.Manifest.RequiredSignatures)
		if err != nil {
			panic(err)
		}
	} else if sts == "disabled" {
		err = ctx.DB.ManifestDisable(manifestid)
		if err != nil {
			panic(err)
		}
	} else {
		panic("Invalid status specified, must be disabled or staged")
	}

	respond(http.StatusOK, resource, respWriter, request)
}

// Request to sign an existing manifest
func signManifest(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
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

	err = ctx.DB.ManifestAddSignature(manifestid, sig, getInvID(request),
		ctx.Manifest.RequiredSignatures)
	if err != nil {
		panic(err)
	}

	respond(http.StatusOK, resource, respWriter, request)
}

// Add a new manifest record
func newManifest(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving newManifest()"}.Debug()
	}()

	// Since this is a file upload, override the limit on the request body
	// and set a reasonable maximum.
	request.Body = http.MaxBytesReader(respWriter, request.Body, 20971520)

	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received new manifest request")}.Debug()

	mrstr := request.FormValue("manifest")
	if mrstr == "" {
		panic("no manifest record specified in form")
	}
	var mr mig.ManifestRecord
	err = json.Unmarshal([]byte(mrstr), &mr)
	if err != nil {
		panic(err)
	}
	err = mr.Validate()
	if err != nil {
		panic(err)
	}
	err = ctx.DB.ManifestAdd(mr)
	if err != nil {
		panic(err)
	}

	respond(http.StatusCreated, resource, respWriter, request)
}

// Return information describing an existing manifest
func getManifest(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
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
				respond(http.StatusNotFound, resource, respWriter, request)
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
		respond(http.StatusBadRequest, resource, respWriter, request)
		return
	}
	mi, err := manifestRecordToItem(mr, ctx)
	if err != nil {
		panic(err)
	}
	resource.AddItem(mi)
	respond(http.StatusOK, resource, respWriter, request)
}

// Given a manifest ID, return the list of known loaders which match the
// targeting string
func manifestLoaders(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving manifestLoaders()"}.Debug()
	}()
	mid, err := strconv.ParseFloat(request.URL.Query()["manifestid"][0], 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'manifestid': '%v'", err)
		panic(err)
	}

	if mid > 0 {
		_, err = ctx.DB.GetManifestFromID(mid)
		if err != nil {
			if fmt.Sprintf("%v", err) == "Error while retrieving manifest: 'sql: no rows in result set'" {
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Manifest ID '%.0f' not found", mid)})
				respond(http.StatusNotFound, resource, respWriter, request)
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
		respond(http.StatusBadRequest, resource, respWriter, request)
		return
	}
	ldrs, err := ctx.DB.AllLoadersFromManifestID(mid)
	if err != nil {
		// If we fail here, it could be because no matching loaders were
		// found or the where clause specified withi the manifest is
		// invalid. Just return a generic error message here but also
		// log the actual error.
		ctx.Channels.Log <- mig.Log{OpID: opid,
			Desc: fmt.Sprintf("Error selecting loaders from manifest %v: %v", mid, err)}
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: fmt.Sprintf("No matching loaders for manifest '%.0f'", mid)})
		respond(http.StatusNotFound, resource, respWriter, request)
		return
	}
	for _, ldr := range ldrs {
		item, err := loaderEntryToItem(ldr, ctx)
		if err != nil {
			panic(err)
		}
		resource.AddItem(item)
	}
	respond(http.StatusOK, resource, respWriter, request)
}

// API entry point used to request a file be sent to the loader from the API.
// This would typically be called from a loader after it has received a
// manifest and determined updates to file system objects are required.
func getManifestFile(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
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

	loaderid := getLoaderID(request)
	if loaderid == 0 {
		panic("Request has no valid loader ID")
	}
	ctx.Channels.Log <- mig.Log{OpID: opid,
		Desc: fmt.Sprintf("Loader request from %v for %v",
			loaderid, manifestParam.Object)}.Debug()

	// Update the loader entry with the parameters, and locate a valid manifest
	mf, err := locateManifestFromLoader(loaderid, manifestParam.AgentIdentifier)
	if err != nil {
		panic(err)
	}
	data, err := mf.ManifestObject(manifestParam.Object)
	if err != nil {
		panic(err)
	}
	fetchresp := mig.ManifestFetchResponse{Data: data}

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
	respond(http.StatusOK, resource, respWriter, request)
}

// This API entry point is used by the loader to request a manifest file that
// indicates the most current version of the agent to be used. The loader
// sends some basic information in the request parameters so the API can decide
// which manifest to send the loader.
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
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getAgentManifest()"}.Debug()
	}()
	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "Received manifest request"}.Debug()

	var manifestParam mig.ManifestParameters
	err = json.Unmarshal([]byte(request.FormValue("parameters")), &manifestParam)
	if err != nil {
		panic(err)
	}
	err = manifestParam.Validate()
	if err != nil {
		panic(err)
	}

	loaderid := getLoaderID(request)
	if loaderid == 0 {
		panic("Request has no valid loader ID")
	}

	// Update the loader entry with the parameters, and locate a valid manifest
	mf, err := locateManifestFromLoader(loaderid, manifestParam.AgentIdentifier)
	if err != nil {
		panic(err)
	}
	m, err := mf.ManifestResponse()
	if err != nil {
		panic(err)
	}

	// Include the loader ID with the response
	m.LoaderName, err = ctx.DB.GetLoaderName(loaderid)
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
	respond(http.StatusOK, resource, respWriter, request)
}

// Return information describing an existing loader entry
func getLoader(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving getLoader()"}.Debug()
	}()
	lid, err := strconv.ParseFloat(request.URL.Query()["loaderid"][0], 64)
	if err != nil {
		err = fmt.Errorf("Wrong parameters 'loaderid': '%v'", err)
		panic(err)
	}

	var le mig.LoaderEntry
	if lid > 0 {
		le, err = ctx.DB.GetLoaderFromID(lid)
		if err != nil {
			if fmt.Sprintf("%v", err) == "Error while retrieving loader: 'sql: no rows in result set'" {
				resource.SetError(cljs.Error{
					Code:    fmt.Sprintf("%.0f", opid),
					Message: fmt.Sprintf("Loader ID '%.0f' not found", lid)})
				respond(http.StatusNotFound, resource, respWriter, request)
				return
			} else {
				panic(err)
			}
		}
	} else {
		// bad request, return 400
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: fmt.Sprintf("Invalid Loader ID '%.0f'", lid)})
		respond(http.StatusBadRequest, resource, respWriter, request)
		return
	}
	li, err := loaderEntryToItem(le, ctx)
	if err != nil {
		panic(err)
	}
	resource.AddItem(li)
	respond(http.StatusOK, resource, respWriter, request)
}

// Update expect values on a loader entry
func expectLoader(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving expectLoader()"}.Debug()
	}()

	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received loader expect change request")}.Debug()

	loaderid, err := strconv.ParseFloat(request.FormValue("loaderid"), 64)
	if err != nil {
		panic(err)
	}
	eval := request.FormValue("expectenv")
	err = ctx.DB.LoaderUpdateExpect(loaderid, eval)
	if err != nil {
		panic(err)
	}

	respond(http.StatusOK, resource, respWriter, request)
}

// Enable or disable a loader entry
func statusLoader(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving statusLoader()"}.Debug()
	}()

	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received loader status change request")}.Debug()

	loaderid, err := strconv.ParseFloat(request.FormValue("loaderid"), 64)
	if err != nil {
		panic(err)
	}
	sts := request.FormValue("status")
	var setval bool
	if sts == "enabled" {
		setval = true
	}
	err = ctx.DB.LoaderUpdateStatus(loaderid, setval)
	if err != nil {
		panic(err)
	}

	respond(http.StatusOK, resource, respWriter, request)
}

// Change key set on a loader entry
func keyLoader(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving keyLoader()"}.Debug()
	}()

	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received loader key change request")}.Debug()

	loaderid, err := strconv.ParseFloat(request.FormValue("loaderid"), 64)
	if err != nil {
		panic(err)
	}
	lkey := request.FormValue("loaderkey")
	if lkey == "" {
		// bad request, return 400
		resource.SetError(cljs.Error{
			Code:    fmt.Sprintf("%.0f", opid),
			Message: "Invalid key specified"})
		respond(http.StatusBadRequest, resource, respWriter, request)
		return
	}
	err = mig.ValidateLoaderKey(lkey)
	if err != nil {
		panic(err)
	}
	hashkey, salt, err := hashLoaderKey(lkey, nil)
	if err != nil {
		panic(err)
	}
	err = ctx.DB.LoaderUpdateKey(loaderid, hashkey, salt)
	if err != nil {
		panic(err)
	}

	respond(http.StatusOK, resource, respWriter, request)
}

// Add a new loader entry
func newLoader(respWriter http.ResponseWriter, request *http.Request) {
	loc := fmt.Sprintf("%s%s", ctx.Server.Host, request.URL.String())
	opid := getOpID(request)
	resource := cljs.New(loc)
	defer func() {
		if e := recover(); e != nil {
			ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("%v", e)}.Err()
			resource.SetError(cljs.Error{Code: fmt.Sprintf("%.0f", opid), Message: fmt.Sprintf("%v", e)})
			respond(http.StatusInternalServerError, resource, respWriter, request)
		}
		ctx.Channels.Log <- mig.Log{OpID: opid, Desc: "leaving newLoader()"}.Debug()
	}()

	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	ctx.Channels.Log <- mig.Log{OpID: opid, Desc: fmt.Sprintf("Received new loader request")}.Debug()

	lestr := request.FormValue("loader")
	if lestr == "" {
		panic("no loader entry specified in form")
	}
	var le mig.LoaderEntry
	err = json.Unmarshal([]byte(lestr), &le)
	if err != nil {
		panic(err)
	}
	err = le.Validate()
	if err != nil {
		panic(err)
	}
	// Hash the loader key to provide it to LoaderAdd
	hkey, salt, err := hashLoaderKey(le.Key, nil)
	if err != nil {
		panic(err)
	}
	err = ctx.DB.LoaderAdd(le, hkey, salt)
	if err != nil {
		panic(err)
	}

	respond(http.StatusCreated, resource, respWriter, request)
}

func manifestRecordToItem(mr mig.ManifestRecord, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/manifest?manifestid=%.0f", ctx.Server.BaseURL, mr.ID)
	item.Data = []cljs.Data{
		{Name: "manifest", Value: mr},
	}
	return
}

func loaderEntryToItem(ldr mig.LoaderEntry, ctx Context) (item cljs.Item, err error) {
	item.Href = fmt.Sprintf("%s/loader?loaderid=%.0f", ctx.Server.BaseURL, ldr.ID)
	item.Data = []cljs.Data{
		{Name: "loader", Value: ldr},
	}
	return
}
