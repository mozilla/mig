package common

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func MapsFilePathFromPid(pid uint) string {
	return filepath.Join("/proc", fmt.Sprintf("%d", pid), "maps")
}

func MemFilePathFromPid(pid uint) string {
	return filepath.Join("/proc", fmt.Sprintf("%d", pid), "mem")
}

//Parses the memory limits of a mapping as found in /proc/PID/maps
func ParseMapsFileMemoryLimits(limits string) (start uintptr, end uintptr, err error) {
	fields := strings.Split(limits, "-")
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("Invalid memory limits, it must have two hexa numbers separeted by a single -")
	}

	start64, err := strconv.ParseUint(fields[0], 16, 64)
	if err != nil {
		return 0, 0, err
	}
	start = uintptr(start64)

	end64, err := strconv.ParseUint(fields[1], 16, 64)
	if err != nil {
		return 0, 0, err
	}
	end = uintptr(end64)

	return
}

// splitMapsEntry splits a line of the maps files returning a slice with an element for each of its parts.
func SplitMapsFileEntry(entry string) []string {
	res := make([]string, 0, 6)
	for i := 0; i < 5; i++ {
		if strings.Index(entry, " ") != -1 {
			res = append(res, entry[0:strings.Index(entry, " ")])
			entry = entry[strings.Index(entry, " ")+1:]
		} else {
			res = append(res, entry, "")
			return res
		}
	}
	res = append(res, strings.TrimLeft(entry, " "))
	return res
}
