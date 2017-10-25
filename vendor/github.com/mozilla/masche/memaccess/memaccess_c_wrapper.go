// +build windows darwin

package memaccess

// #include "memaccess.h"
// #cgo CFLAGS: -std=c99
import "C"

import (
	"fmt"
	"github.com/mozilla/masche/cresponse"
	"github.com/mozilla/masche/process"
	"unsafe"
)

func nextReadableMemoryRegion(p process.Process, address uintptr) (region MemoryRegion, softerrors []error, harderror error) {
	var isAvailable C.bool
	var cRegion C.memory_region_t

	response := C.get_next_readable_memory_region(
		(C.process_handle_t)(p.Handle()),
		C.memory_address_t(address),
		&isAvailable,
		&cRegion)
	softerrors, harderror = cresponse.GetResponsesErrors(unsafe.Pointer(response))
	C.response_free(response)

	if harderror != nil || isAvailable == false {
		return NoRegionAvailable, softerrors, harderror
	}

	return MemoryRegion{uintptr(cRegion.start_address), uint(cRegion.length)}, softerrors, harderror
}

func copyMemory(p process.Process, address uintptr, buffer []byte) (softerrors []error, harderror error) {
	buf := unsafe.Pointer(&buffer[0])

	n := len(buffer)
	var bytesRead C.size_t
	resp := C.copy_process_memory(
		(C.process_handle_t)(p.Handle()),
		C.memory_address_t(address),
		C.size_t(n),
		buf,
		&bytesRead,
	)

	softerrors, harderror = cresponse.GetResponsesErrors(unsafe.Pointer(resp))
	C.response_free(resp)

	if harderror != nil {
		harderror = fmt.Errorf("Error while copying %d bytes starting at %x: %s", n, address, harderror.Error())
		return
	}

	if len(buffer) != int(bytesRead) {
		harderror = fmt.Errorf("Could not copy %d bytes starting at %x, copyed %d", len(buffer), address, bytesRead)
	}

	return
}
