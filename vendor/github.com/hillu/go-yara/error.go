// Copyright © 2015 Hilko Bengen <bengen@hilluzination.de>. All rights reserved.
// Use of this source code is governed by the license that can be
// found in the LICENSE file.

package yara

// #include <yara.h>
import "C"
import (
	"errors"
	"fmt"
)

func newError(code C.int) error {
	if code == 0 {
		return nil
	}
	if str, ok := errorStrings[code]; ok {
		return errors.New(str)
	}
	return fmt.Errorf("unknown error %d", code)
}

// FIXME: This should be generated from yara/error.h
var errorStrings = map[C.int]string{
	C.ERROR_INSUFICIENT_MEMORY:           "insufficient memory",
	C.ERROR_COULD_NOT_ATTACH_TO_PROCESS:  "could not attach to process",
	C.ERROR_COULD_NOT_OPEN_FILE:          "could not open file",
	C.ERROR_COULD_NOT_MAP_FILE:           "could not map file",
	C.ERROR_INVALID_FILE:                 "invalid file",
	C.ERROR_CORRUPT_FILE:                 "corrupt file",
	C.ERROR_UNSUPPORTED_FILE_VERSION:     "unsupported file version",
	C.ERROR_INVALID_REGULAR_EXPRESSION:   "invalid regular expression",
	C.ERROR_INVALID_HEX_STRING:           "invalid hex string",
	C.ERROR_SYNTAX_ERROR:                 "syntax error",
	C.ERROR_LOOP_NESTING_LIMIT_EXCEEDED:  "loop nesting limit exceeded",
	C.ERROR_DUPLICATED_LOOP_IDENTIFIER:   "duplicated loop identifier",
	C.ERROR_DUPLICATED_IDENTIFIER:        "duplicated identifier",
	C.ERROR_DUPLICATED_TAG_IDENTIFIER:    "duplicated tag identifier",
	C.ERROR_DUPLICATED_META_IDENTIFIER:   "duplicated meta identifier",
	C.ERROR_DUPLICATED_STRING_IDENTIFIER: "duplicated string identifier",
	C.ERROR_UNREFERENCED_STRING:          "unreferenced string",
	C.ERROR_UNDEFINED_STRING:             "undefined string",
	C.ERROR_UNDEFINED_IDENTIFIER:         "undefined identifier",
	C.ERROR_MISPLACED_ANONYMOUS_STRING:   "misplaced anonymous string",
	C.ERROR_INCLUDES_CIRCULAR_REFERENCE:  "includes circular reference",
	C.ERROR_INCLUDE_DEPTH_EXCEEDED:       "include depth exceeded",
	C.ERROR_WRONG_TYPE:                   "wrong type",
	C.ERROR_EXEC_STACK_OVERFLOW:          "exec stack overflow",
	C.ERROR_SCAN_TIMEOUT:                 "scan timeout",
	C.ERROR_TOO_MANY_SCAN_THREADS:        "too many scan threads",
	C.ERROR_CALLBACK_ERROR:               "callback error",
	C.ERROR_INVALID_ARGUMENT:             "invalid argument",
	C.ERROR_TOO_MANY_MATCHES:             "too many matches",
	C.ERROR_INTERNAL_FATAL_ERROR:         "internal fatal error",
	C.ERROR_NESTED_FOR_OF_LOOP:           "nested for of loop",
	C.ERROR_INVALID_FIELD_NAME:           "invalid field name",
	C.ERROR_UNKNOWN_MODULE:               "unknown module",
	C.ERROR_NOT_A_STRUCTURE:              "not a structure",
	C.ERROR_NOT_INDEXABLE:                "not indexable",
	C.ERROR_NOT_A_FUNCTION:               "not a function",
	C.ERROR_INVALID_FORMAT:               "invalid format",
	C.ERROR_TOO_MANY_ARGUMENTS:           "too many arguments",
	C.ERROR_WRONG_ARGUMENTS:              "wrong arguments",
	C.ERROR_WRONG_RETURN_TYPE:            "wrong return type",
	C.ERROR_DUPLICATED_STRUCTURE_MEMBER:  "duplicated structure member",
}
