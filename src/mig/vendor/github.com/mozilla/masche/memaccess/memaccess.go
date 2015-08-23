// This packages contains an interface for accessing other processes' memory.
package memaccess

import (
	"fmt"
	"github.com/mozilla/masche/process"
)

// MemoryRegion represents a region of readable contiguos memory of a process.
// No readable memory can be available right next to this region, it's maximal in its upper bound.
//
// NOTE: This region is not necessary equivalent to the OS's region, if any.
type MemoryRegion struct {
	Address uintptr
	Size    uint
}

func (m MemoryRegion) String() string {
	return fmt.Sprintf("MemoryRegion[%x-%x)", m.Address, m.Address+uintptr(m.Size))
}

// A centinel value indicating that there is no more regions available.
var NoRegionAvailable MemoryRegion

// NextReadableMemoryRegion returns a memory region containing address, or the next readable region after address in
// case addresss is not in a readable region.
//
// If there aren't more regions available the special value NoRegionAvailable is returned.
func NextReadableMemoryRegion(p process.Process, address uintptr) (region MemoryRegion, harderror error,
	softerrors []error) {
	return nextReadableMemoryRegion(p, address)
}

// CopyMemory fills the entire buffer with memory from the process starting in address (in the process address space).
// If there is not enough memory to read it returns a hard error. Note that this is not the only hard error it may
// return though.
func CopyMemory(p process.Process, address uintptr, buffer []byte) (harderror error, softerrors []error) {
	return copyMemory(p, address, buffer)
}

// This type represents a function used for walking through the memory, see WalkMemory for more details.
type WalkFunc func(address uintptr, buf []byte) (keepSearching bool)

// WalkMemory reads all the memory of a process starting at a given address reading upto bufSize bytes into a buffer,
// and calling walkFn with the buffer and the start address of the memory in the buffer. If walkFn returns false
// WalkMemory stop reading the memory.
//
// NOTE: It can call to walkFn with a smaller buffer when reading the last part of a memory region.
func WalkMemory(p process.Process, startAddress uintptr, bufSize uint, walkFn WalkFunc) (harderror error,
	softerrors []error) {

	var region MemoryRegion
	region, harderror, softerrors = NextReadableMemoryRegion(p, startAddress)
	if harderror != nil {
		return
	}

	// The first region can start befor startAddress. If that happens, it must contain it. In that case, we set the
	// region's Adrress to startAdress to behave as documented.
	if region.Address < startAddress {
		if region.Address+uintptr(region.Size) <= startAddress {
			harderror = fmt.Errorf("First memory region doesn't contain the startAddress. This is a bug.")
			return
		}

		region.Size -= uint(startAddress - region.Address)
		region.Address = startAddress
	}

	const max_retries int = 5

	buf := make([]byte, bufSize)
	retries := max_retries

	for region != NoRegionAvailable {

		keepWalking, addr, err, serrs := walkRegion(p, region, buf, walkFn)
		softerrors = append(softerrors, serrs...)

		if err != nil && retries > 0 {
			// An error occurred: retry using the nearest region to the address that failed.
			retries--
			region, harderror, serrs = NextReadableMemoryRegion(p, addr)
			softerrors = append(softerrors, serrs...)
			if harderror != nil {
				return
			}

			// if some chunk of this new region was already read we don't want to read it again.
			if region.Address < addr {
				region.Address = addr
			}

			continue
		} else if err != nil {
			// we have exceeded our retries, mark the error as soft error and keep going.
			softerrors = append(softerrors, fmt.Errorf("Retries exceeded on reading %d bytes starting at %x: %s",
				len(buf), addr, err.Error()))
		} else if !keepWalking {
			return
		}

		region, harderror, serrs = NextReadableMemoryRegion(p, region.Address+uintptr(region.Size))
		softerrors = append(softerrors, serrs...)
		if harderror != nil {
			return
		}
		retries = max_retries
	}
	return
}

// This function walks through a single memory region calling walkFunc with a given buffer. It always fills as much of
// the buffer as possible before calling walkFunc, but it never calls it with overlaped memory sections.
//
// If the buffer cannot be filled a hard error is returned with the starting address of the chunk of memory that could
// not be read. If no harderror is returned errorAddress must be ignored.
//
// If any of the calls to walkFn returns false, this function inmediatly returns, with keepWalking set to false and no
// hard error.
func walkRegion(p process.Process, region MemoryRegion, buf []byte, walkFn WalkFunc) (keepWalking bool,
	errorAddress uintptr, harderror error, softerrors []error) {
	softerrors = make([]error, 0)
	keepWalking = true
	remainingBytes := uintptr(region.Size)
	for addr := region.Address; remainingBytes > 0; {
		if remainingBytes < uintptr(len(buf)) {
			buf = buf[:remainingBytes]
		}

		err, serrs := CopyMemory(p, addr, buf)
		softerrors = append(softerrors, serrs...)

		if err != nil {
			harderror = err
			errorAddress = addr
			return
		}

		keepWalking = walkFn(addr, buf)
		if !keepWalking {
			return
		}

		addr += uintptr(len(buf))
		remainingBytes -= uintptr(len(buf))
	}

	return
}

// Thiw function works as WalkMemory, except that it reads overlapped bytes. It first calls walkFn with a full buffer,
// then advances just half of the buffer size, and calls it again.
// As with WalkRegion, the buffer can be smaller at the end of a region.
// NOTE: It doesn't work with odd bufSize.
func SlidingWalkMemory(p process.Process, startAddress uintptr, bufSize uint, walkFn WalkFunc) (
	harderror error, softerrors []error) {

	if bufSize%2 != 0 {
		return fmt.Errorf("SlidingWalkMemory doesn't support odd bufferSizes"), softerrors
	}

	buffer := make([]byte, bufSize)
	halfBufferSize := bufSize / 2
	currentBufferStartsAt := uintptr(0)
	bufferedBytes := uint(0)
	harderror, softerrors = WalkMemory(p, startAddress, halfBufferSize,
		func(address uintptr, currentBuffer []byte) (keepSearching bool) {

			fromAnotherRegion := currentBufferStartsAt+uintptr(bufferedBytes) < address && currentBufferStartsAt != 0

			if fromAnotherRegion {
				if bufferedBytes > 0 && bufferedBytes < bufSize {
					// Call walkFn with buffer because if was starting a region and as it's not complete it hasn't been
					// sent to walkFn
					if !walkFn(currentBufferStartsAt, buffer[:bufferedBytes]) {
						return false
					}
				}

				bufferedBytes = 0
			}

			if bufferedBytes == 0 {
				copy(buffer, currentBuffer)
				currentBufferStartsAt = address

				// If the currentBuffer is smaller the region has finished
				if uint(len(currentBuffer)) != halfBufferSize {
					if !walkFn(currentBufferStartsAt, buffer[:len(currentBuffer)]) {
						return false
					}
				} else {
					bufferedBytes = halfBufferSize
				}

				return true
			}

			if bufferedBytes == bufSize {
				copy(buffer, buffer[:halfBufferSize])
				currentBufferStartsAt += uintptr(halfBufferSize)
			}

			copy(buffer[halfBufferSize:], currentBuffer)
			if !walkFn(currentBufferStartsAt, buffer[:halfBufferSize+uint(len(currentBuffer))]) {
				return false
			}

			// If the currentBuffer is smaller the region has finished
			if uint(len(currentBuffer)) != halfBufferSize {
				bufferedBytes = 0
			} else {
				bufferedBytes = bufSize
			}

			return true
		})

	// If we only have half buffer filled we haven't called walkFn yet with it
	if bufferedBytes == halfBufferSize {
		walkFn(currentBufferStartsAt, buffer[:halfBufferSize])
	}

	return
}
