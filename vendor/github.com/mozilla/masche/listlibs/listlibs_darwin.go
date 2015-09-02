package listlibs

// #cgo CFLAGS: -std=c99
// #include "listlibs_darwin.h"
import "C"

import (
	"github.com/mozilla/masche/cresponse"
	"github.com/mozilla/masche/process"
	"reflect"
	"unsafe"
)

func listLoadedLibraries(p process.Process) (libraries []string, harderror error, softerrors []error) {
	var ptr uintptr
	var sizeT C.size_t
	clibs := (***C.char)(C.malloc(C.size_t(unsafe.Sizeof(ptr))))
	count := (*C.size_t)(C.malloc(C.size_t(unsafe.Sizeof(sizeT))))
	defer C.free(unsafe.Pointer(clibs))
	defer C.free(unsafe.Pointer(count))

	response := C.list_loaded_libraries((C.process_handle_t)(p.Handle()), clibs, count)
	defer C.free_loaded_libraries_list(*clibs, *count)
	harderror, softerrors = cresponse.GetResponsesErrors(unsafe.Pointer(response))
	C.response_free(response)

	if harderror != nil {
		return
	}

	libraries = make([]string, 0, *count)
	clibsSlice := *(*[]*C.char)(unsafe.Pointer(
		&reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(*clibs)),
			Len:  int(*count),
			Cap:  int(*count)}))

	processName, harderror, softs := p.Name()
	if harderror != nil {
		return
	}
	softerrors = append(softerrors, softs...)

	for i, _ := range clibsSlice {
		if clibsSlice[i] == nil {
			continue
		}

		str := C.GoString(clibsSlice[i])
		if str == processName {
			continue
		}
		libraries = append(libraries, str)
	}

	return
}
