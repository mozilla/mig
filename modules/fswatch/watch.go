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
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Set to true to enable extra debugging info in agent log
var debugFSWatch = false

// Default alert suppression window
var alertSuppressWindow = 15 * time.Minute

// Interfaces with fsnotify to monitor the file system
type fsWatcher struct {
	watcher *fsnotify.Watcher

	sync.Mutex // This lock should be picked up before making any changes to watcher
}

// Initialize fsWatcher f
func (f *fsWatcher) initialize() (err error) {
	f.watcher, err = fsnotify.NewWatcher()
	return
}

// Add path to fsWatcher f and start monitoring it
func (f *fsWatcher) addMonitor(path string) error {
	f.Lock()
	defer f.Unlock()
	return f.watcher.Add(path)
}

// Remove path from fsWatcher f
func (f *fsWatcher) removeMonitor(path string) error {
	f.Lock()
	defer f.Unlock()
	return f.watcher.Remove(path)
}

var watcher fsWatcher

// Used by hasher, describes a request to hash a file
type hashRequest struct {
	outChan chan []byte // Response will be sent on this channel
	path    string
}

// hasher aggregates hash requests and throttles file system access
type hasher struct {
	inQueue chan hashRequest
}

var fshasher hasher

// Respond to a new hash request, calculates SHA256 and replies on indicated
// channel
func (h *hasher) respond(nr hashRequest) {
	var ret []byte
	if debugFSWatch {
		logChan <- fmt.Sprintf("hashing %v", nr.path)
	}
	fd, err := os.Open(nr.path)
	if err != nil {
		nr.outChan <- ret
		return
	}
	defer fd.Close()
	hash := sha256.New()
	buf := make([]byte, 4096)
	for {
		n, err := fd.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			nr.outChan <- ret
			return
		}
		if n > 0 {
			hash.Write(buf[:n])
		}
	}
	ret = hash.Sum(nil)
	nr.outChan <- ret
}

// Spawns new hasher, listens on input queue and replies to requests
func (h *hasher) initialize() {
	var (
		lasthash time.Time = time.Now()
		tindex   int       = 1
	)
	h.inQueue = make(chan hashRequest, 128)
	go func() {
		for {
			nr := <-h.inQueue
			if time.Now().Sub(lasthash) < time.Second {
				if tindex < 10 {
					tindex++
				}
				time.Sleep(time.Duration(tindex) * (time.Millisecond * 5))
			} else {
				tindex = 1
			}
			h.respond(nr)
			lasthash = time.Now()
		}
	}()
}

// A monitoring profile, contains a list of profile entries where each entry
// describes a monitoring path on the file system
type profile struct {
	entries []profileEntry
}

// Initialize the monitoring profile, prepares each entry in the profile
func (p *profile) initialize() (err error) {
	for i := range p.entries {
		err = p.entries[i].initialize()
		if err != nil {
			return
		}
	}
	return
}

func (p *profile) routeEvent(ev fsnotify.Event) {
	// Identify which entry is monitoring the object associated with this event
	// and route the event there
	for i := range p.entries {
		if p.entries[i].hasObject(ev.Name) {
			p.entries[i].events <- ev
			return
		}
	}
	// No explicit object was identified for the event, so it's likely its related
	// to a directory being monitored. Remove the actual name and attempt to locate
	// an object for it's parent directory.
	pdir := path.Dir(ev.Name)
	for i := range p.entries {
		if p.entries[i].hasObject(pdir) {
			p.entries[i].events <- ev
			return
		}
	}
	newAlert(ALERT_HIGH, "unhandled notification for %v", ev.Name)
}

// An entry within a profile; this describes a single point of monitoring, which depending
// on if it's a directory or a regular file, could contain a list of objects or a single
// object.
type profileEntry struct {
	path      string
	recursive bool

	events  chan fsnotify.Event
	objects []object

	sync.Mutex // Protects objects, ensure this lock is picked up when referencing objects
}

// Add object for path to profileEntry pe
func (pe *profileEntry) addObject(path string, objtype int) (err error) {
	pe.Lock()
	defer pe.Unlock()
	// Make sure this object does not already exist
	for _, x := range pe.objects {
		if x.path == path {
			return fmt.Errorf("object for path %v already exists", path)
		}
	}
	pe.objects = append(pe.objects, object{path: path, objtype: objtype, monitored: true})
	logChan <- fmt.Sprintf("adding monitoring for object %v", path)
	watcher.addMonitor(path)
	return
}

// Remove object for path from profileEntry pe
func (pe *profileEntry) removeObject(path string) (err error) {
	pe.Lock()
	monitored := false
	for i := range pe.objects {
		if pe.objects[i].path == path {
			monitored = pe.objects[i].monitored
			pe.objects = append(pe.objects[:i], pe.objects[i+1:]...)
			break
		}
	}
	if !monitored {
		pe.Unlock()
		return nil
	}
	logChan <- fmt.Sprintf("removing monitoring for object %v", path)
	watcher.removeMonitor(path)
	if len(pe.objects) == 0 {
		// If the length is zero, that means the root monitored object was
		// removed. Spawn a routine here to periodically check it if is
		// recreated and reinitialize it if we see it.
		logChan <- fmt.Sprintf("last object in %v removed, periodically checking for existence", pe.path)
		close(pe.events)
		pe.events = nil
		pe.Unlock()
		go func() {
			for {
				_, errl := os.Stat(pe.path)
				if errl == nil {
					err = pe.initialize()
					if err != nil {
						handlerErrChan <- err
					}
					return
				}
				time.Sleep(time.Second * 5)
			}
		}()
		return nil
	}
	pe.Unlock()
	return nil
}

// Return true if profileEntry pe contains an object for path
func (pe *profileEntry) hasObject(path string) bool {
	pe.Lock()
	defer pe.Unlock()
	for _, x := range pe.objects {
		if x.path == path {
			return true
		}
	}
	return false
}

// Recursively processes the directory in the profile entry, identifying any
// subdirectories and adding them as objects
func (pe *profileEntry) processDirectory() (err error) {
	var paths []string
	wf := func(p string, finfo os.FileInfo, lerr error) error {
		if !finfo.Mode().IsDir() {
			return nil
		}
		paths = append(paths, p)
		return nil
	}
	err = filepath.Walk(pe.path, wf)
	if err != nil {
		return err
	}
	for _, x := range paths {
		err = pe.addObject(x, TYPE_DIRECTORY)
		if err != nil {
			return err
		}
	}
	return
}

// Update the current hash for an object in the profile entry; ensure the entry
// lock has been acquired before calling this function
func (pe *profileEntry) updateObjectHash(path string, newh []byte) error {
	var (
		objidx int
		found  bool
	)
	for i := range pe.objects {
		if pe.objects[i].path == path {
			objidx = i
			found = true
			break
		}
	}
	if !found {
		logChan <- fmt.Sprintf("warning: received hash reply for untracked object %v", path)
		return nil
	}
	pe.objects[objidx].previous = pe.objects[objidx].current
	pe.objects[objidx].current = newh
	if debugFSWatch {
		logChan <- fmt.Sprintf("hash update: %v %x -> %x", path,
			pe.objects[objidx].previous, pe.objects[objidx].current)
	}
	if len(pe.objects[objidx].previous) == 0 {
		// No previous hash, nothing to do here
		return nil
	}
	// Compare our new hash to the previously generated one
	if bytes.Compare(pe.objects[objidx].previous, pe.objects[objidx].current) != 0 {
		pe.objects[objidx].alert()
	}
	return nil
}

// Initialize a given entry in the profile; following return of this function
// monitoring is active for the given entry
func (pe *profileEntry) initialize() (err error) {
	if pe.events == nil {
		pe.events = make(chan fsnotify.Event, 0)
	}
	finfo, err := os.Stat(pe.path)
	if err != nil { // Paths in the profile must exist
		// XXX Right now this is fatal, and if a path specified in the
		// profile does not exist the module will error out and exit. It
		// might be better to just ignore this and periodically test to
		// see if the path exists in the future.
		return err
	}
	if finfo.Mode().IsRegular() {
		err = pe.addObject(pe.path, TYPE_FILE)
		if err != nil {
			return err
		}
	} else if finfo.Mode().IsDir() && !pe.recursive {
		err = pe.addObject(pe.path, TYPE_DIRECTORY)
		if err != nil {
			return err
		}
	} else if finfo.Mode().IsDir() && pe.recursive {
		// If it's a directory and recursive monitoring is set, we will want
		// to identify any subdirectories here we also want to monitor.
		err = pe.processDirectory()
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("file type of entry path %v cannot be monitored", pe.path)
	}
	go func() {
		// Spawn a routine to create some initial records for the entry; for files we
		// just need to generate a hash request, for any directories we will do an initial
		// walk and generate requests as we go
		pe.Lock()
		objcopy := pe.objects
		for _, x := range objcopy {
			var nr hashRequest
			if x.objtype == TYPE_FILE {
				nr.path = x.path
				nr.outChan = make(chan []byte, 1)
				fshasher.inQueue <- nr
				nhash := <-nr.outChan
				pe.updateObjectHash(x.path, nhash)
			} else {
				dirents, _ := ioutil.ReadDir(x.path)
				for _, y := range dirents {
					if !y.Mode().IsRegular() {
						continue
					}
					nr.path = path.Join(x.path, y.Name())
					// Add the identifier object to the object list, but we don't
					// use addObject here as we don't require monitoring since it is
					// covered by the directory monitoring
					pe.objects = append(pe.objects, object{path: nr.path, objtype: TYPE_FILE})
					nr.outChan = make(chan []byte, 1)
					fshasher.inQueue <- nr
					nhash := <-nr.outChan
					pe.updateObjectHash(nr.path, nhash)
				}
			}
		}
		pe.Unlock()
	}()
	go func() {
		for {
			ev, ok := <-pe.events
			if !ok {
				// Something went wrong in the event channel, possibly because
				// the root monitored object was removed; if this is the case
				// just return, otherwise we will generate an error
				pe.Lock()
				if len(pe.objects) != 0 {
					handlerErrChan <- fmt.Errorf("profile entry event channel closed")
				}
				pe.Unlock()
				return
			}
			err = pe.handleEvent(ev)
			if err != nil {
				handlerErrChan <- err
				return
			}
		}
	}()
	return
}

// Handles a file system watcher event coming in for this profile entry, this
// function is responsible for updating the object list as needed, and creating
// hash requests for observed changes
func (pe *profileEntry) handleEvent(ev fsnotify.Event) error {
	if ev.Op == fsnotify.Create && pe.recursive {
		// If this is a create event, and recursive monitoring is set for the
		// entry, determine if a directory was created and if so we will start
		// monitoring it.
		finfo, err := os.Stat(ev.Name)
		if err != nil {
			// We couldn't stat the newly created directory, don't treat this
			// as fatal but generate an alert
			newAlert(ALERT_MEDIUM, "unable to stat new path %v", ev.Name)
			return nil
		}
		if finfo.Mode().IsDir() {
			return pe.addObject(ev.Name, TYPE_DIRECTORY)
		} else if finfo.Mode().IsRegular() {
			// If it was a regular file creation, add it as an untracked object
			pe.Lock()
			pe.objects = append(pe.objects, object{path: ev.Name, objtype: TYPE_FILE})
			pe.Unlock()
		}
	} else if ev.Op == fsnotify.Remove {
		// An object was removed, determine if it is in our object list and
		// if so remove it there as well.
		if pe.hasObject(ev.Name) {
			newAlert(ALERT_LOW, "path %v removed", ev.Name)
			return pe.removeObject(ev.Name)
		}
	}
	if ev.Op == fsnotify.Create || ev.Op == fsnotify.Write {
		// A change occurred, we will create a new async request to hash
		// the file
		go func() {
			// Introduce a randomized delay before we generate the hash request,
			// intended to provide some time for any remaining writes to the object
			// to complete
			time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
			rch := make(chan []byte, 1)
			nr := hashRequest{
				path:    ev.Name,
				outChan: rch,
			}
			fshasher.inQueue <- nr
			nhash := <-rch
			pe.Lock()
			pe.updateObjectHash(ev.Name, nhash)
			pe.Unlock()
		}()
	}
	return nil
}

const (
	_ = iota
	TYPE_FILE
	TYPE_DIRECTORY
)

// Describes an object being monitoried
type object struct {
	path      string    // Path
	objtype   int       // TYPE_FILE, TYPE_DIRECTORY
	monitored bool      // Object is monitored by fsnotify
	previous  []byte    // Previously calculated hash
	current   []byte    // Latest hash
	lastAlert time.Time // Last time we alerted for this object
}

func (o *object) alert() {
	// Determine when we last alerted for this object; if it has been within
	// alertSuppressWindow we dont generate one
	if !o.lastAlert.IsZero() {
		if time.Now().Sub(o.lastAlert) <= alertSuppressWindow {
			if debugFSWatch {
				logChan <- fmt.Sprintf("suppressed alert for %v", o.path)
			}
			return
		}
	}
	newAlert(ALERT_CRITICAL, "%v signature changed %x -> %x", o.path,
		o.previous, o.current)
	o.lastAlert = time.Now()
}

// Main entry routine for file system monitor
func fsWatch(cfg config) {
	var (
		err          error
		localprofile profile
	)
	fshasher.initialize()
	err = watcher.initialize()
	if err != nil {
		handlerErrChan <- err
		return
	}
	if len(cfg.Paths.Path) == 0 {
		localprofile = localFsWatchProfile
	} else {
		// Construct a profile to use based on the paths in the configuration
		for _, x := range cfg.Paths.Path {
			var pent profileEntry
			if strings.HasPrefix(x, "recursive:") {
				pent.path = x[10:]
				pent.recursive = true
			} else {
				pent.path = x
			}
			localprofile.entries = append(localprofile.entries, pent)
		}
	}
	err = localprofile.initialize()
	if err != nil {
		handlerErrChan <- err
		return
	}
	for {
		select {
		case ev, ok := <-watcher.watcher.Events:
			if !ok {
				handlerErrChan <- fmt.Errorf("watcher event channel closed")
				return
			}
			if debugFSWatch {
				logChan <- fmt.Sprintf("fsnotify: %+v", ev)
			}
			localprofile.routeEvent(ev)
		case err := <-watcher.watcher.Errors:
			// If we get some sort of error from the watcher, just kill the
			// module
			handlerErrChan <- err
			return
		}
	}
}
