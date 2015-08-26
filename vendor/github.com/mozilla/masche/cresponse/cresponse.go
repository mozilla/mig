package cresponse

// #include "response.h"
// #cgo CFLAGS: -std=c99
import "C"

import (
	"fmt"
	"reflect"
	"unsafe"
)

// CError is the Go represnentation of responce.h's error_t.
type CError struct {
	number      int
	description string
}

func (err CError) Error() string {
	return fmt.Sprintf("System error number %d: %s", err.number, err.description)
}

// GetGetResponsesErrors returns the Go representation of the errors present in a C.reponse_t.
//
// NOTE: cgo types are private to each module, so exporting a function that expects a *C.response_t doesn't make sense,
// so we export a function with an unsafe.Pointer and we cast it internally.
func GetResponsesErrors(responsePointer unsafe.Pointer) (harderror error, softerrors []error) {
	response := (*C.response_t)(responsePointer)
	if response.fatal_error != nil && int(response.fatal_error.error_number) != 0 {
		harderror = cErrorFromErrorT(*response.fatal_error)
	} else {
		harderror = nil
	}

	softerrorsCount := int(response.soft_errors_count)
	softerrors = make([]error, 0, softerrorsCount)

	cSoftErrorsHeader := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(response.soft_errors)),
		Len:  softerrorsCount,
		Cap:  softerrorsCount,
	}
	cSoftErrors := *(*[]C.error_t)(unsafe.Pointer(&cSoftErrorsHeader))

	for _, cErr := range cSoftErrors {
		softerrors = append(softerrors, cErrorFromErrorT(cErr))
	}

	return
}

func cErrorFromErrorT(err C.error_t) CError {
	return CError{
		number:      int(err.error_number),
		description: C.GoString(err.description),
	}
}
