package memaccess

import (
	"bufio"
	"fmt"
	"github.com/mozilla/masche/common"
	"github.com/mozilla/masche/process"
	"os"
)

func nextReadableMemoryRegion(p process.Process, address uintptr) (region MemoryRegion, softerrors []error,
	harderror error) {

	mapsFile, harderror := os.Open(common.MapsFilePathFromPid(p.Pid()))
	if harderror != nil {
		return
	}
	defer mapsFile.Close()

	region = MemoryRegion{}
	scanner := bufio.NewScanner(mapsFile)

	for scanner.Scan() {
		line := scanner.Text()
		items := common.SplitMapsFileEntry(line)

		if len(items) != 6 {
			return region, softerrors, fmt.Errorf("Unrecognised maps line: %s", line)
		}

		start, end, err := common.ParseMapsFileMemoryLimits(items[0])
		if err != nil {
			return region, softerrors, err
		}

		if end <= address {
			continue
		}

		// Skip vsyscall as it can't be read. It's a special page mapped by the kernel to accelerate some syscalls.
		if items[5] == "[vsyscall]" {
			continue
		}

		// Check if memory is unreadable
		if items[1][0] == '-' {

			// If we were already reading a region this will just finish it. We only report the softerror when we
			// were actually trying to read it.
			if region.Address != 0 {
				return region, softerrors, nil
			}

			softerrors = append(softerrors, fmt.Errorf("Unreadable memory %s", items[0]))
			continue
		}

		size := uint(end - start)

		// Begenning of a region
		if region.Address == 0 {
			region = MemoryRegion{Address: start, Size: size}
			continue
		}

		// Continuation of a region
		if region.Address+uintptr(region.Size) == start {
			region.Size += size
			continue
		}

		// This map is outside the current region, so we are ready
		return region, softerrors, nil
	}

	// No region left
	if err := scanner.Err(); err != nil {
		return NoRegionAvailable, softerrors, err
	}

	// The last map was a valid region, so it was not closed by an invalid/non-contiguous one and we have to return it
	if region.Address > 0 {
		return region, softerrors, harderror
	}

	return NoRegionAvailable, softerrors, nil
}

func copyMemory(p process.Process, address uintptr, buffer []byte) (softerrors []error, harderror error) {
	mem, harderror := os.Open(common.MemFilePathFromPid(p.Pid()))

	if harderror != nil {
		harderror := fmt.Errorf("Error while reading %d bytes starting at %x: %s", len(buffer), address, harderror)
		return softerrors, harderror
	}
	defer mem.Close()

	bytesRead, harderror := mem.ReadAt(buffer, int64(address))
	if harderror != nil {
		harderror := fmt.Errorf("Error while reading %d bytes starting at %x: %s", len(buffer), address, harderror)
		return softerrors, harderror
	}

	if bytesRead != len(buffer) {
		return softerrors, fmt.Errorf("Could not read the entire buffer")
	}

	return softerrors, nil
}
