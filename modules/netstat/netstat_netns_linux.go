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

func namespacesSupported() bool {
	return true
}

func setNamespace(fd int) error {
	defer syscall.Close(fd)
	_, _, errno := syscall.RawSyscall(uintptr(SYS_SETNS), uintptr(fd), 0, 0)
	if errno != 0 {
		return fmt.Errorf("setNamespace Linux errno %v", errno)
	}
	return nil
}

func cacheNamespaces() ([]int, error) {
	retfd := make([]int, 0)
	pents, err := ioutil.ReadDir("/proc")
	if err != nil {
		return retfd, err
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
			return retfd, err
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
		retfd = append(retfd, fd)
	}

	return retfd, nil
}
