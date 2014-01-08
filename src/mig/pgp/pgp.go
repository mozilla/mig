package pgp

import (
	"bytes"
	"code.google.com/p/go.crypto/openpgp"
	"code.google.com/p/go.crypto/openpgp/armor"
	"fmt"
	"os"
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

// Verify() checks the validity of a signature for some data,
// and returns a boolean set to true if valid and an OpenPGP Entity
func Verify(data string, signature string) (valid bool, entity *openpgp.Entity, err error) {
	valid = false

	// re-armor signature and transform into io.Reader
	sigReader := strings.NewReader(reArmor(signature))

	// decode armor
	sigBlock, err := armor.Decode(sigReader)
	if err != nil {
		panic(err)
	}
	if sigBlock.Type != "PGP SIGNATURE" {
		err = fmt.Errorf("Wrong signature type '%s'", sigBlock.Type)
		panic(err)
	}

	// convert to io.Reader
	srcReader := strings.NewReader(data)

	// verify the signature and get the signer back
	ringFile, err := os.Open("/home/ulfr/.gnupg/secring.gpg")
	if err != nil {
		panic(err)
	}
	defer ringFile.Close()

	ring, err := openpgp.ReadKeyRing(ringFile)
	if err != nil {
		panic(err)
	}

	entity, err = openpgp.CheckDetachedSignature(ring, srcReader, sigBlock.Body)
	if err != nil {
		panic(err)
	}

	// we passed, signature is valid
	valid = true

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

// reArmor takes a single line armor and turns it back into an PGP-style
// multi-line armored string (thank you, camlistore folks)
func reArmor(line string) string {
	lastEq := strings.LastIndex(line, "=")
	if lastEq == -1 {
		return ""
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "-----BEGIN PGP SIGNATURE-----\n\n")
	payload := line[0:lastEq]
	crc := line[lastEq:]
	for len(payload) > 0 {
		chunkLen := len(payload)
		if chunkLen > 60 {
			chunkLen = 60
		}
		fmt.Fprintf(buf, "%s\n", payload[0:chunkLen])
		payload = payload[chunkLen:]
	}
	fmt.Fprintf(buf, "%s\n-----BEGIN PGP SIGNATURE-----\n", crc)
	return buf.String()
}
