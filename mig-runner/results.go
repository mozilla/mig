// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
	"os"
	"path"
	"time"
)

// Given the name of a scheduled job, retrieve the path that should be used
// to store the results for this job.
func getResultsStoragePath(nm string) (rdir string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getResultsStoragePath() -> %v", e)
		}
	}()
	tstamp := time.Now().UTC().Format("20060102")
	rdir = path.Join(ctx.Runner.RunDirectory, nm, "results", tstamp)

	_, err = os.Stat(rdir)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		err = os.MkdirAll(rdir, 0700)
		if err != nil {
			panic(err)
		}
	}
	return rdir, nil
}

// For mig.RunnerResult r, get the results associated with this job from the
// API.
func getResults(r mig.RunnerResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getResults() -> %v", e)
		}
	}()

	mlog("fetching results for %v/%.0f", r.EntityName, r.Action.ID)

	cli, err := client.NewClient(ctx.ClientConf, "mig-runner-results")
	if err != nil {
		panic(err)
	}
	results, err := cli.FetchActionResults(r.Action)
	if err != nil {
		panic(err)
	}
	mlog("%v/%.0f: fetched results, %v commands returned", r.EntityName, r.Action.ID, len(results))

	// Store the results
	outpath, err := getResultsStoragePath(r.EntityName)
	if err != nil {
		panic(err)
	}
	outpath = path.Join(outpath, fmt.Sprintf("%.0f.json", r.Action.ID))
	fd, err := os.Create(outpath)
	if err != nil {
		panic(err)
	}
	defer fd.Close()

	r.Commands = results
	buf, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	_, err = fd.Write(buf)
	if err != nil {
		panic(err)
	}
	mlog("wrote results for %.0f to %v", r.Action.ID, outpath)

	// If a plugin has been configured on the result set, call the
	// plugin on the data.
	if r.UsePlugin != "" {
		err = runPlugin(r)
		if err != nil {
			panic(err)
		}
	} else {
		mlog("no output plugin for %.0f, skipping plugin processing", r.Action.ID)
	}

	return nil
}

// Determine the path that should be used to store the in-flight action
// information for a given job.
func flightPath(rr mig.RunnerResult) string {
	aid := fmt.Sprintf("%.0f.json", rr.Action.ID)
	return path.Join(ctx.Runner.RunDirectory, rr.EntityName, "inflight", aid)
}

// Cache action in-flight information on the file system.
func actionInFlight(rr mig.RunnerResult) error {
	fpath := flightPath(rr)
	dn, _ := path.Split(fpath)
	err := os.MkdirAll(dn, 0700)
	if err != nil {
		return err
	}
	fd, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer fd.Close()
	buf, err := json.Marshal(rr)
	if err != nil {
		return err
	}
	_, err = fd.Write(buf)
	if err != nil {
		return err
	}
	mlog("%v: action marked as inflight: %v", rr.EntityName, fpath)
	return nil
}

// Remove action in-flight information from the file system when complete.
func actionComplete(rr mig.RunnerResult) error {
	fpath := flightPath(rr)
	err := os.Remove(fpath)
	if err != nil {
		return err
	}
	mlog("%v: action marked as complete, removed %v", rr.EntityName, fpath)
	return nil
}

// Scan the runner spool directory looking for actions that are known to be
// in-flight that we have not loaded. For example, this can occur if the runner
// is restarted after a job has been submitted to the API, but before the
// results are returned. This will load the in-flight action data from the
// file system so the runner is aware of the job and will obtain the results
// when required.
func scanInFlight(reslist []mig.RunnerResult) ([]mig.RunnerResult, error) {
	rdir := ctx.Runner.RunDirectory
	rents, err := ioutil.ReadDir(rdir)
	if err != nil {
		return reslist, err
	}
	for _, x := range rents {
		if !x.IsDir() {
			continue
		}
		fdir := path.Join(rdir, x.Name(), "inflight")
		fents, err := ioutil.ReadDir(fdir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return reslist, err
		}
		for _, y := range fents {
			fpath := path.Join(fdir, y.Name())
			fd, err := os.Open(fpath)
			if err != nil {
				return reslist, err
			}
			buf, err := ioutil.ReadAll(fd)
			if err != nil {
				fd.Close()
				return reslist, err
			}
			var rr mig.RunnerResult
			err = json.Unmarshal(buf, &rr)
			found := false
			for _, x := range reslist {
				if x.Action.ID == rr.Action.ID {
					found = true
				}
			}
			if !found {
				// This is an action we do not know about,
				// potentially because the runner was restarted
				// after action dispath before the results were
				// fetched.
				mlog("appending unknown cached action %.0f", rr.Action.ID)
				reslist = append(reslist, rr)
			}
			fd.Close()
		}
	}

	return reslist, nil
}

// Results processing routine. Keeps track of known submitted actions and
// retrieves results / runs plugins as results become available.
func processResults() {
	mlog("results processing routine started")

	var reslist []mig.RunnerResult

	for {
		timeout := false
		select {
		case nr := <-ctx.Channels.Results:
			mlog("monitoring result for %v/%.0f", nr.EntityName, nr.Action.ID)
			err := actionInFlight(nr)
			if err != nil {
				mlog("%v: unable to mark in flight: %v", nr.EntityName, err)
				continue
			}
			reslist = append(reslist, nr)
		case <-time.After(time.Duration(5) * time.Second):
			timeout = true
		}
		if !timeout {
			// Only attempt action results fetch if we are idle
			continue
		}

		var err error
		reslist, err = scanInFlight(reslist)
		if err != nil {
			mlog("error scanning for cached inflight requests: %v", err)
		}

		resDelay := ctx.Client.DelayResults

		// See if any actions have expired, if so grab the results
		oldres := reslist
		reslist = reslist[:0]
		for _, x := range oldres {
			extime := x.Action.ExpireAfter
			if resDelay != "" {
				d, err := time.ParseDuration(resDelay)
				if err != nil {
					mlog("results error for %v: %v", x.EntityName, err)
					mlog("%v: ignoring specified results delay", x.EntityName)
				} else {
					extime = extime.Add(d)
				}
			}
			if time.Now().After(extime) {
				err := getResults(x)
				if err != nil {
					mlog("results error for %v: %v", x.EntityName, err)
				}
				err = actionComplete(x)
				if err != nil {
					// This is fatal; if this happens it
					// will result in the results being
					// fetched again.
					panic(err)
				}
				continue
			}
			reslist = append(reslist, x)
		}
	}
}
