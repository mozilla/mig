// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"encoding/json"
	"fmt"
	"mig.ninja/mig"
	"mig.ninja/mig/client"
	"os"
	"path"
	"time"
)

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
		err = os.MkdirAll(rdir, 0755)
		if err != nil {
			panic(err)
		}
	}
	return rdir, nil
}

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
	outpath = path.Join(outpath, fmt.Sprintf("%.0f", r.Action.ID))
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

func processResults() {
	mlog("results processing routine started")

	reslist := make([]mig.RunnerResult, 0)

	for {
		timeout := false
		select {
		case nr := <-ctx.Channels.Results:
			mlog("monitoring result for %v/%.0f", nr.EntityName, nr.Action.ID)
			reslist = append(reslist, nr)
		case <-time.After(time.Duration(5) * time.Second):
			timeout = true
		}
		if !timeout {
			// Only attempt action results fetch if we are idle
			continue
		}

		// See if any actions have expired, if so grab the results
		oldres := reslist
		reslist = reslist[:0]
		for _, x := range oldres {
			if time.Now().After(x.Action.ExpireAfter) {
				err := getResults(x)
				if err != nil {
					mlog("results error for %v: %v", x.EntityName, err)
				}
				continue
			}
			reslist = append(reslist, x)
		}
	}
}
