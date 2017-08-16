package process

import (
	"bufio"
	"fmt"
	"github.com/mozilla/masche/common"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type proc uint

func (p proc) Pid() uint {
	return uint(p)
}

func (p proc) Name() (name string, softerrors []error, harderror error) {
	exePath := filepath.Join("/proc", fmt.Sprintf("%d", p.Pid()), "exe")
	name, err := filepath.EvalSymlinks(exePath)

	if err != nil {
		// If the exe link doesn't take us to the real path of the binary of the process maybe it's not present anymore
		// or the process didn't started from a file. We mimic this ps(1) trick and take the name form
		// /proc/<pid>/status in that case.

		statusPath := filepath.Join("/proc", fmt.Sprintf("%d", p.Pid()), "status")
		statusFile, err := os.Open(statusPath)
		if err != nil {
			return name, nil, err
		}

		r := bufio.NewReader(statusFile)
		for line, _, err := r.ReadLine(); err != io.EOF; line, _, err = r.ReadLine() {
			if err != nil {
				return name, nil, err
			}

			namePrefix := "Name:"
			if strings.HasPrefix(string(line), namePrefix) {
				name := strings.Trim(string(line[len(namePrefix):]), " \t")

				// We add the square brackets to be consistent with ps(1) output.
				return "[" + name + "]", nil, nil
			}
		}

		return name, nil, fmt.Errorf("No name found for pid %v", p.Pid())
	}

	return name, nil, err
}

func (p proc) Close() (softerrors []error, harderror error) {
	return nil, nil
}

func (p proc) Handle() uintptr {
	return uintptr(p)
}

func getAllPids() (pids []uint, softerrors []error, harderror error) {
	files, err := ioutil.ReadDir("/proc/")
	if err != nil {
		return nil, nil, err
	}

	pids = make([]uint, 0)

	for _, f := range files {
		pid, err := strconv.Atoi(f.Name())
		if err != nil {
			continue
		}
		pids = append(pids, uint(pid))
	}

	return pids, nil, nil
}

func openFromPid(pid uint) (p Process, softerrors []error, harderror error) {
	// Check if we have permissions to read the process memory
	memPath := common.MemFilePathFromPid(pid)
	memFile, err := os.Open(memPath)
	if err != nil {
		harderror = fmt.Errorf("Permission denied to access memory of process %v", pid)
		return
	}
	defer memFile.Close()

	return proc(pid), nil, nil
}
