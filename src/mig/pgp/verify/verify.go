package verify

import (
	"bytes"
	"code.google.com/p/go.crypto/openpgp"
	"code.google.com/p/go.crypto/openpgp/armor"
	"fmt"
	"io"
	"strings"
)

// Verify() checks the validity of a signature for some data,
// and returns a boolean set to true if valid and an OpenPGP Entity
func Verify(data string, signature string, keyring io.Reader) (valid bool, entity *openpgp.Entity, err error) {
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

	// open the keyring
	ring, err := openpgp.ReadKeyRing(keyring)
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
