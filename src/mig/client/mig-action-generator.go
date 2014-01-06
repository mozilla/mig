package main
import (
	"bytes"
	"code.google.com/p/go.crypto/openpgp"
	"code.google.com/p/go.crypto/openpgp/armor"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"mig"
	"mig/modules/filechecker"
	"os"
	"strings"
	"time"
	"unsafe"
)

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -lgpgme libmig_gpgme.a
#include <libmig_gpgme.h>
*/
import "C"

func main() {

	var Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Mozilla InvestiGator Action Generator\n" +
			"usage: %s -k=<key id> (-i <input file)\n\n" +
			"Command line to generate and sign MIG Actions.\n" +
			"The resulting actions are display on stdout.\n\n" +
			"Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}

	// command line options
	var key = flag.String("k", "key identifier", "Key identifier used to sign the action (ex: B75C2346)")
	var file = flag.String("i", "/path/to/file", "Load action from file")
	flag.Parse()

	// We need a key, if none is set on the command line, fail
	if *key == "key identifier" {
		Usage()
		os.Exit(-1)
	}

	var ea mig.ExtendedAction
	var err error
	if *file != "/path/to/file" {
		// get action from local json file
		ea, err = getActionFromFile(*file)
	} else {
		//interactive mode
		ea, err = getActionFromTerminal()
	}
	if err != nil {
		panic(err)
	}

	// Compute a GPG signature of the action
	sig, err := signAction(ea, *key)
	if err != nil {
		panic(err)
	}

	// transform sig into json array
	asig := strings.Split(sig, "\n")
	for _, sigcomp := range asig {
		ea.Signature = append(ea.Signature, sigcomp)
	}
	ea.SignatureDate = time.Now().UTC()
	jsonAction, err := json.Marshal(ea)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", jsonAction)

	// Verify the GPG signature
	err = verifySignature(ea)
	if err != nil {
		panic(err)
	}

}

func getActionFromFile(path string) (ea mig.ExtendedAction, err error) {
	fd, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(fd, &ea)
	if err != nil {
		panic(err)
	}
	return
}

func getActionFromTerminal() (ea mig.ExtendedAction, err error) {
	err = nil
	fmt.Print("Action name> ")
	_, err = fmt.Scanln(&ea.Action.Name)
	if err != nil {
		panic(err)
	}
	fmt.Print("Action Target> ")
	_, err = fmt.Scanln(&ea.Action.Target)
	if err != nil {
		panic(err)
	}
	fmt.Print("Action Check> ")
	_, err = fmt.Scanln(&ea.Action.Check)
	if err != nil {
		panic(err)
	}
	fmt.Print("Action Expiration> ")
	var expiration string
	_, err = fmt.Scanln(&expiration)
	if err != nil {
		panic(err)
	}
	ea.Action.ScheduledDate = time.Now().UTC()
	period, err := time.ParseDuration(expiration)
	if err != nil {
		log.Fatal(err)
	}
	ea.Action.ExpirationDate = time.Now().UTC().Add(period)

	var checkArgs string
	switch ea.Action.Check {
	default:
		fmt.Print("Unknown check type, supply JSON arguments> ")
		_, err := fmt.Scanln(&checkArgs)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal([]byte(checkArgs), ea.Action.Arguments)
		if err != nil {
			panic(err)
		}
	case "filechecker":
		fmt.Println("Filechecker module parameters")
		var name string
		var fcargs filechecker.FileCheck
		fmt.Print("Check Name> ")
		_, err := fmt.Scanln(&name)
		if err != nil {
			panic(err)
		}
		fmt.Print("Filechecker Type> ")
		_, err = fmt.Scanln(&fcargs.Type)
		if err != nil {
			panic(err)
		}
		fmt.Print("File Path> ")
		_, err = fmt.Scanln(&fcargs.Path)
		if err != nil {
			panic(err)
		}
		fmt.Print("Check Value> ")
		_, err = fmt.Scanln(&fcargs.Value)
		if err != nil {
			panic(err)
		}
		fc := make(map[string]filechecker.FileCheck)
		fc[name] = fcargs
		ea.Action.Arguments = fc
	}
	return
}

// signAction signs a string with a key. The function uses a C library that
// calls gpgme, for compatibility with gpg-agent.
func signAction(ea mig.ExtendedAction, key string) (sig string, err error) {
	// prepare string for signature
	srcStr := prepDataToSign(ea)

	// convert to C variable
	ckey := C.CString(key)
	cstr := C.CString(srcStr)

	// calculate signature
	csig := C.MIGSign(ckey, cstr)

	// convert signature back to Go string
	sig = C.GoString(csig)

	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(cstr))

	return
}

// verifySignature checks the validity of an armored signature
func verifySignature(ea mig.ExtendedAction) error {
	var signature string

	// extract armored signature to string
	for _, s := range ea.Signature {
		signature = fmt.Sprintf("%s\n%s", signature, s)
	}

	// transform string into io.Reader
	sigReader := bytes.NewBufferString(signature)

	// decode armor
	block, err := armor.Decode(sigReader)
	if err != nil {
		panic(err)
	}
	if block.Type != "PGP SIGNATURE" {
		log.Fatal("Wrong signature type", block.Type)
	}

	// get the source data
	srcStr := prepDataToSign(ea)

	// convert to io.Reader
	srcReader := bytes.NewBufferString(srcStr)

	// verify the signature and get the signer back
	pubringFile, err := os.Open("/home/ulfr/.gnupg/pubring.gpg")
	if err != nil {
		panic(err)
	}
	defer pubringFile.Close()
	pubring, err := openpgp.ReadKeyRing(pubringFile)
	if err != nil {
		panic(err)
	}
	signer, err := openpgp.CheckArmoredDetachedSignature(pubring, srcReader, sigReader)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Signature verified. Signer=")
	for _, ident := range signer.Identities {
		fmt.Printf("'%s', ", ident.UserId.Email)
	}
	fmt.Printf("\n")
	return nil
}

// prepDataToSign concatenates Action components into a string
func prepDataToSign(ea mig.ExtendedAction) (str string) {
	str = ea.Action.Name
	str += ea.Action.Target
	str += ea.Action.Check
	str += ea.Action.ScheduledDate.String()
	str += ea.Action.ExpirationDate.String()
	return
}
