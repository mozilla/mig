// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package fswatch /* import "mig.ninja/mig/modules/fswatch" */

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"
)

// fsWatchProfile specifies which paths to monitor on the file system, as a
// list of fsWatchProfileEntry types
type fsWatchProfile struct {
	entries []fsWatchProfileEntry
}

// fsWatchProfileEntry describes a path to monitor on the file system. path
// can reference either a file or a directory. In the case of a directory,
// the first level contents of that directory will be read and objects will
// be populated with these vales. In the case of just a file, objects will
// contain a single entry for the file path.
type fsWatchProfileEntry struct {
	path    string
	objects []fsWatchObject
}

// Collect all entries in a directory indicated by a profile entry, note that we do
// not do recursion here and only read the first directory level
func (f *fsWatchProfileEntry) collectDirectory() error {
	dirents, err := ioutil.ReadDir(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			if len(f.objects) != 0 {
				newAlert(ALERT_MEDIUM, "monitored directory %v disappeared", f.path)
				// Since it is gone, also remove any objects referenced by it
				f.objects = nil
			}
			return nil
		}
		return err
	}

	oldobjs := f.objects[:]
	foundents := make([]string, 0)

	for _, fname := range dirents {
		fpath := path.Join(f.path, fname.Name())
		finfo, err := os.Stat(fpath)
		if err != nil {
			if os.IsNotExist(err) {
				// We saw the entry in ReadDir but it's gone now, note this and
				// continue
				newAlert(ALERT_MEDIUM, "path %v disappeared from %v during directory refresh",
					fpath, f.path)
				continue
			}
			return err
		}
		// Only monitor regular files in a directory, anything else ignore
		if !finfo.Mode().IsRegular() {
			continue
		}
		foundents = append(foundents, fpath)
	}

	f.objects = nil
	for i := range oldobjs {
		var objval *fsWatchObject
		for _, x := range foundents {
			if oldobjs[i].path == x {
				objval = &oldobjs[i]
				break
			}
		}
		if objval != nil { // Existed in oldobjs and foundents, retain the entry
			f.objects = append(f.objects, *objval)
		} else {
			newAlert(ALERT_MEDIUM, "monitored path %v disappeared", oldobjs[i].path)
		}
	}
	// Add any new entries
	for _, x := range foundents {
		found := false
		for _, y := range f.objects {
			if y.path == x {
				found = true
				break
			}
		}
		if found {
			continue
		}
		f.objects = append(f.objects, fsWatchObject{x, nil, nil})
		logChan <- fmt.Sprintf("fswatch added %v from directory", x)
	}

	return nil
}

// Return true if a fsWatchProfileEntry f has an object entry for path
func (f *fsWatchProfileEntry) hasObject(path string) bool {
	for _, x := range f.objects {
		if x.path == path {
			return true
		}
	}
	return false
}

// Do hash comparisons of all objects in fsWatchProfileEntry f
func (f *fsWatchProfileEntry) hashcheck() error {
	for i := range f.objects {
		err := f.objects[i].hash()
		if err != nil {
			return err
		}
	}
	return nil
}

// Scan all entries in the profile, and populate the object list for each entry
func (f *fsWatchProfileEntry) refresh() error {
	finfo, err := os.Stat(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			if len(f.objects) != 0 {
				newAlert(ALERT_MEDIUM, "monitored path %v disappeared", f.path)
				// Since it is gone, also remove any objects referenced by it
				f.objects = nil
			}
			return nil
		}
		return err
	}
	if finfo.Mode().IsDir() {
		return f.collectDirectory()
	} else if finfo.Mode().IsRegular() {
		if !f.hasObject(f.path) {
			f.objects = append(f.objects, fsWatchObject{f.path, nil, nil})
			logChan <- fmt.Sprintf("fswatch added %v", f.path)
		}
	} else {
		return fmt.Errorf("fswatch entry is not a directory or regular file")
	}
	return nil
}

// fsWatchObject describes an individual object identified in a profile entry
type fsWatchObject struct {
	path     string // object path
	previous []byte // hash previously calculated
	current  []byte // hash currently calculated
}

// Hash object f, setting previous to current and recalculating current
func (f *fsWatchObject) hash() error {
	fd, err := os.Open(f.path)
	if err != nil {
		return err
	}
	defer fd.Close()
	f.previous = f.current
	h := sha256.New()
	buf := make([]byte, 4096)
	for {
		n, err := fd.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if n > 0 {
			h.Write(buf[:n])
		}
	}
	f.current = h.Sum(nil)
	if f.previous == nil {
		return nil
	}
	if bytes.Compare(f.previous, f.current) != 0 {
		newAlert(ALERT_HIGH, "signature for %v changed: %x -> %x", f.path,
			f.previous, f.current)
	}
	return nil
}

// Primary function that identifies all entries in our watch lists, hashes
// identified objects, and creates alert messages
func fsWatchRefreshEntries(profile *fsWatchProfile) (err error) {
	for i := range profile.entries {
		err = profile.entries[i].refresh()
		if err != nil {
			return
		}
		err = profile.entries[i].hashcheck()
		if err != nil {
			return
		}
	}
	return
}

// Main entry routine for file system monitor
func fsWatch(cfg config) {
	var err error
	profile := localFsWatchProfile

	sdur := 5 * time.Minute
	if cfg.FSWatch.Interval != "" {
		sdur, err = time.ParseDuration(cfg.FSWatch.Interval)
		if err != nil {
			handlerErrChan <- err
			return
		}
	}
	logChan <- fmt.Sprintf("fswatch interval set to %v", sdur)

	// If custom paths have been indicated in the config file, override the
	// local profile
	if len(cfg.FSWatchPaths.Path) != 0 {
		logChan <- "fswatch using monitoring paths set it configuration file"
		profile.entries = nil
		for _, x := range cfg.FSWatchPaths.Path {
			newent := fsWatchProfileEntry{
				path:    x,
				objects: nil,
			}
			profile.entries = append(profile.entries, newent)
			logChan <- fmt.Sprintf("fswatch watching: %v", x)
		}
	} else {
		logChan <- "fswatch using built-in monitoring paths"
		for _, x := range profile.entries {
			logChan <- fmt.Sprintf("fswatch watching: %v", x.path)
		}
	}

	for {
		err = fsWatchRefreshEntries(&profile)
		if err != nil {
			handlerErrChan <- err
			return
		}
		time.Sleep(sdur)
	}
}
