// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]

package netstat /* import "mig.ninja/mig/modules/netstat" */

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"syscall"
)

var SYS_SETNS = 308

type linuxNsCache struct {
	fd   int
	name string
}

func (n *linuxNsCache) setNamespace() error {
	defer syscall.Close(n.fd)
	_, _, errno := syscall.RawSyscall(uintptr(SYS_SETNS), uintptr(n.fd), 0, 0)
	if errno != 0 {
		return fmt.Errorf("setNamespace Linux errno %v", errno)
	}
	return nil
}

func (n *linuxNsCache) getName() string {
	return n.name
}

func namespacesSupported() bool {
	return true
}

func cacheNamespaces() ([]nsCache, error) {
	retns := make([]nsCache, 0)
	pents, err := ioutil.ReadDir("/proc")
	if err != nil {
		return retns, err
	}

	seen := make([]string, 0)
	for _, procdir := range pents {
		// We only want numeric PID directories
		_, err = strconv.Atoi(procdir.Name())
		if err != nil {
			continue
		}
		nsfile := path.Join("/proc", procdir.Name(), "ns", "net")
		fi, err := os.Lstat(nsfile)
		if err != nil {
			return retns, err
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		lname, err := os.Readlink(nsfile)
		if err != nil {
			// Don't treat this as an error; this can be the result of a TOCTOU
			// issue, just ignore it and move on.
			continue
		}
		found := false
		for _, check := range seen {
			if check == lname {
				found = true
				break
			}
		}
		if found {
			continue
		}

		fd, err := syscall.Open(nsfile, syscall.O_RDONLY, 0666)
		if err != nil {
			// Likewise here.
			continue
		}
		seen = append(seen, lname)
		newns := linuxNsCache{fd: fd, name: lname}
		retns = append(retns, &newns)
	}

	return retns, nil
}
