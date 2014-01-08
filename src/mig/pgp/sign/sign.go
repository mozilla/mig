package sign

import (
	"fmt"
	"strings"
	"unsafe"
)

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -lgpgme libmig_gpgme.a
#include <libmig_gpgme.h>
*/
import "C"

// Sign() signs a string with a key. The function uses a C library that
// calls gpgme, for compatibility with gpg-agent.
func Sign(data string, key string) (sig string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("pgpSign(): %v", e)
		}
	}()

	// convert to C variable
	ckey := C.CString(key)
	if ckey == nil {
		panic("Error converting key to CString type")
	}
	cdata := C.CString(data)
	if cdata == nil {
		panic("Error converting data to CString type")
	}

	// calculate signature
	csig := C.GPGME_Sign(cdata, ckey)
	if csig == nil {
		panic("Error computing signature")
	}

	// convert signature back to Go string
	sig = deArmor(C.GoString(csig))
	if sig == "" {
		panic("Error converting signature to GOString type")
	}

	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(cdata))

	return
}

// deArmor takes a multi line armored GPG signature, and turns it back
// into a single line signature (thank you, camlistore folks)
func deArmor(sig string) string {
	index1 := strings.Index(sig, "\n\n")
	index2 := strings.Index(sig, "\n-----")
	if index1 == -1 || index2 == -1 {
		panic("Failed to parse signature from gpg.")
	}
	inner := sig[index1+2 : index2]
	return strings.Replace(inner, "\n", "", -1)
}
