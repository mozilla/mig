// This is an example program that shows the usage of the memsearch package.
//
// With this program you can:
//   - Search for a string in the memory of a process with a given PID
//   - Print an arbitrary amount of bytes from the process memory.
package main

import (
	"encoding/hex"
	"flag"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/mozilla/masche/memaccess"
	"github.com/mozilla/masche/memsearch"
	"github.com/mozilla/masche/process"
)

var (
	action = flag.String("action", "<nil>", "Action to perfom. One of: search, regexp-search, file-search, print")
	pid    = flag.Int("pid", 0, "Process id to analyze")
	addr   = flag.Int("addr", 0x0, "The initial address in the process address space to search/print")

	// print action flags
	size = flag.Int("n", 4, "Amount of bytes to print")

	// search action flags
	needle = flag.String("needle", "Find This!", "String to search for (interpreted as []byte)")

	// regexp-search action flags
	regexpString = flag.String("regexp", "regexp?", "Regexp to search for")

	// file-search action flags
	fileneedle = flag.String("fileneedle", "example.in", "Filename that contains hex-encoded needle (spaces are ignored)")
)

func logErrors(harderror error, softerrors []error) {
	if harderror != nil {
		log.Fatal(harderror)
	}
	for _, soft := range softerrors {
		log.Print(soft)
	}
}

func main() {
	flag.Parse()

	proc, harderror, softerrors := process.OpenFromPid(uint(*pid))
	logErrors(harderror, softerrors)

	switch *action {

	case "<nil>":
		log.Fatal("Missing action flag.")
	case "file-search":
		data, err := ioutil.ReadFile(*fileneedle)
		if err != nil {
			log.Fatal(err)
		}
		encoded := strings.Replace(strings.Replace(strings.TrimSpace(string(data)), " ", "", -1), "\n", "", -1)
		data, err = hex.DecodeString(encoded)
		if err != nil {
			log.Fatal(err)
		}
		found, address, harderror, softerrors := memsearch.FindBytesSequence(proc, uintptr(*addr), data)
		logErrors(harderror, softerrors)
		if found {
			log.Printf("Found in address: %x\n", address)
		}

	case "search":
		found, address, harderror, softerrors := memsearch.FindBytesSequence(proc, uintptr(*addr), []byte(*needle))
		logErrors(harderror, softerrors)
		if found {
			log.Printf("Found in address: %x\n", address)
		}

	case "regexp-search":
		r, err := regexp.Compile(*regexpString)
		if err != nil {
			log.Fatal(err)
		}

		found, address, harderror, softerrors := memsearch.FindRegexpMatch(proc, uintptr(*addr), r)
		logErrors(harderror, softerrors)
		if found {
			log.Printf("Found in address: %x\n", address)
		}

	case "print":
		buf := make([]byte, *size)
		harderror, softerrors = memaccess.CopyMemory(proc, uintptr(*addr), buf)
		logErrors(harderror, softerrors)
		log.Println(string(buf))

	default:
		log.Fatal("Unrecognized action ", *action)
	}
}
