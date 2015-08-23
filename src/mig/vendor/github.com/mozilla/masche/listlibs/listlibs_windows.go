package listlibs

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/mozilla/masche/process"
)

// #cgo CFLAGS: -std=c99
// #cgo CFLAGS: -DPSAPI_VERSION=1
// #cgo LDFLAGS: -lpsapi
// #include "listlibs_windows.h"
import "C"

func listLoadedLibraries(p process.Process) (libraries []string, harderror error, softerrors []error) {
	r := C.getModules(C.process_handle_t(p.Handle()))
	defer C.EnumProcessModulesResponse_Free(r)
	if r.error != 0 {
		return nil, fmt.Errorf("getModules failed with error: %d", r.error), nil
	}
	mods := make([]string, r.length)
	// We use this to access C arrays without doing manual pointer arithmetic.
	cmods := *(*[]C.ModuleInfo)(unsafe.Pointer(
		&reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(r.modules)),
			Len:  int(r.length),
			Cap:  int(r.length)}))
	for i, _ := range mods {
		mods[i] = C.GoString(cmods[i].filename)
	}
	return mods, nil, nil
}
