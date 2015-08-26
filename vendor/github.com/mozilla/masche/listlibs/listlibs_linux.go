package listlibs

import (
	"bufio"
	"fmt"
	"github.com/mozilla/masche/common"
	"github.com/mozilla/masche/process"
	"os"
)

func listLoadedLibraries(p process.Process) (libraries []string, harderror error, softerrors []error) {

	mapsFile, harderror := os.Open(common.MapsFilePathFromPid(p.Pid()))
	if harderror != nil {
		return
	}
	defer mapsFile.Close()

	scanner := bufio.NewScanner(mapsFile)
	processName, harderror, softerrors := p.Name()
	if harderror != nil {
		return
	}

	libs := make([]string, 0, 10)
	for scanner.Scan() {
		line := scanner.Text()
		items := common.SplitMapsFileEntry(line)

		if len(items) != 6 {
			return libs, fmt.Errorf("Unrecognised maps line: %s", line), softerrors
		}

		path := items[5]
		if path == processName {
			continue
		}

		if path == "/dev/zero" || path == "/dev/zero (deleted)" {
			continue
		}

		if path == "" {
			continue
		}

		if path[0] == '[' {
			continue
		}

		if inSlice(path, libs) {
			continue
		}

		libs = append(libs, path)
	}

	return libs, nil, nil
}

func inSlice(s string, slice []string) bool {
	for _, s2 := range slice {
		if s == s2 {
			return true
		}
	}

	return false
}
