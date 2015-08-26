// This program provides a functionality similar to what `pgrep` does un Linux:
// it lists all the processes whose name matches a given regexp.
package main

import (
	"flag"
	"fmt"
	"github.com/mozilla/masche/process"
	"log"
	"regexp"
)

var reg = flag.String("r", ".*", "Regular Expression to use.")

func main() {
	flag.Parse()

	r, err := regexp.Compile(*reg)
	if err != nil {
		log.Fatal(err)
	}

	ps, hard, soft := process.OpenByName(r)
	if hard != nil {
		log.Fatal(err)
	}
	if soft != nil {
		for _, err := range soft {
			log.Println(err)
		}
	}

	for _, p := range ps {
		name, hard, soft := p.Name()
		if hard != nil {
			log.Fatal(err)
		}
		if soft != nil {
			for _, err := range soft {
				log.Println(err)
			}
		}
		fmt.Printf("Process: %s\nPid: %d\n\n", name, p.Pid())
	}
}
