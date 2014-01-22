package pgp

import (
	"bytes"
	"code.google.com/p/go.crypto/openpgp"
	"fmt"
	"io"
)

// TransformArmoredPubKeyToKeyring takes a public PGP key in armored form and transforms
// it into a keyring that can be used in other openpgp's functions
func TransformArmoredPubKeyToKeyring(pubkey string) (keyring io.Reader, err error) {

	// Load PGP public key
	el, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(pubkey))
	if err != nil {
		return
	}
	if len(el) != 1 {
		err = fmt.Errorf("Public GPG Key contains %d entities, wanted 1\n", len(el))
		return
	}

	// convert entity into keyring
	var buf bytes.Buffer
	err = el[0].Serialize(&buf)
	if err != nil {
		return
	}

	keyring = bytes.NewReader(buf.Bytes())

	return
}
