// This program can be used to check if any process is running a given dynamic library.
// The -r flag specifies a regexp over the filename of the library, for example:
// ./prueba -r="libc" will match all programs that have the libc loaded as a dynamic library.
package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"

	"github.com/mozilla/masche/listlibs"
	"github.com/mozilla/masche/process"
)

var rstr = flag.String("r", "", "library name regexp")

func main() {
	flag.Parse()

	r, err := regexp.Compile(*rstr)
	if err != nil {
		log.Fatal(err)
	}

	ps, hard, softs := process.OpenAll()
	if hard != nil {
		log.Fatal(hard)
	}
	defer process.CloseAll(ps)
	for _, e := range softs {
		log.Println(e)
	}

	matches, hard, softs := findProcWithLib(r, ps)
	if hard != nil {
		log.Fatal(hard)
	}
	for _, e := range softs {
		fmt.Println(e)
	}

	fmt.Printf("Processes matching: %s\n", *rstr)
	for p, libs := range matches {
		n, hard, _ := p.Name()
		if hard != nil {
			log.Fatal(hard)
		}
		fmt.Printf("[%d] %s\n", p.Pid(), n)
		for _, l := range libs {
			fmt.Printf("\t%s\n", l)
		}
	}

}

func findProcWithLib(r *regexp.Regexp, ps []process.Process) (matches map[process.Process][]string, harderror error, softerrors []error) {
	matches = make(map[process.Process][]string)
	softerrors = make([]error, 0)
	for _, p := range ps {
		libs, hard, softs := listlibs.GetMatchingLoadedLibraries(p, r)
		if hard != nil {
			return nil, hard, softerrors
		}
		softerrors = append(softerrors, softs...)
		if len(libs) != 0 {
			matches[p] = libs
		}
	}
	return matches, nil, softerrors
}
