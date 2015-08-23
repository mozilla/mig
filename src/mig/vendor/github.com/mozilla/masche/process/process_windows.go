package process

// #cgo CFLAGS: -std=c99
// #cgo CFLAGS: -DPSAPI_VERSION=1
// #cgo LDFLAGS: -lpsapi
// #include "process.h"
// #include "process_windows.h"
import "C"

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/mozilla/masche/cresponse"
)

func (p process) Name() (name string, harderror error, softerrors []error) {
	var cname uintptr
	r := C.GetProcessName(p.hndl, (**C.char)(unsafe.Pointer(&cname)))

	harderror, softerrors = cresponse.GetResponsesErrors(unsafe.Pointer(r))
	C.response_free(r)
	if harderror == nil {
		name = C.GoString((*C.char)(unsafe.Pointer(cname)))
		C.free(unsafe.Pointer(cname))
	}
	return
}

func getAllPids() (pids []uint, harderror error, softerrors []error) {
	r := C.getAllPids()
	defer C.EnumProcessesResponse_Free(r)
	if r.error != 0 {
		return nil, fmt.Errorf("getAllPids failed with error %d", r.error), nil
	}

	pids = make([]uint, 0, r.length)
	// We use this to access C arrays without doing manual pointer arithmetic.
	cpids := *(*[]C.DWORD)(unsafe.Pointer(
		&reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(r.pids)),
			Len:  int(r.length),
			Cap:  int(r.length)}))
	for i, _ := range cpids {
		pid := uint(cpids[i])
		// pids 0 and 4 are reserved in windows.
		if pid == 0 || pid == 4 {
			continue
		}
		pids = append(pids, pid)
	}

	return pids, nil, nil
}
