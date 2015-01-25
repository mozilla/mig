// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"mig"
	"strings"
)

// initWatchers initializes the watcher flags for all the monitored directories
func initWatchers(watcher *fsnotify.Watcher, ctx Context) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("initWatchers() -> %v", e)
		}
		ctx.Channels.Log <- mig.Log{Desc: "leaving initWatchers()"}.Debug()
	}()

	err = watcher.WatchFlags(ctx.Directories.Action.New, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Action.New)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Action.New)}.Debug()

	err = watcher.WatchFlags(ctx.Directories.Command.InFlight, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Command.InFlight)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Command.InFlight)}.Debug()

	err = watcher.WatchFlags(ctx.Directories.Command.Returned, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Command.Returned)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Command.Returned)}.Debug()

	err = watcher.WatchFlags(ctx.Directories.Action.Done, fsnotify.FSN_CREATE)
	if err != nil {
		e := fmt.Errorf("%v '%s'", err, ctx.Directories.Action.Done)
		panic(e)
	}
	ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watcher.Watch(): %s", ctx.Directories.Action.Done)}.Debug()

	return
}

// watchDirectories calls specific function when a file appears in a watched directory
func watchDirectories(watcher *fsnotify.Watcher, ctx Context) {
	for {
		select {
		case ev := <-watcher.Event:
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watchDirectories(): %s", ev.String())}.Debug()

			// New file detected, but the file size might still be zero, because inotify wakes up before
			// the file is fully written. If that's the case, wait a little and hope that's enough to finish writing
			err := waitForFileOrDelete(ev.Name, 5)
			if err != nil {
				ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("error while reading '%s': %v", ev.Name, err)}.Err()
				goto nextfile
			}
			// Use the prefix of the filename to send it to the appropriate channel
			if strings.HasPrefix(ev.Name, ctx.Directories.Action.New) {
				ctx.Channels.NewAction <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.InFlight) {
				ctx.Channels.UpdateCommand <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Command.Returned) {
				ctx.Channels.CommandReturned <- ev.Name
			} else if strings.HasPrefix(ev.Name, ctx.Directories.Action.Done) {
				ctx.Channels.ActionDone <- ev.Name
			}
		case err := <-watcher.Error:
			// in case of error, raise an emergency
			ctx.Channels.Log <- mig.Log{Desc: fmt.Sprintf("watchDirectories(): %v", err)}.Emerg()
		}
	nextfile:
	}
}
