package main

import (
	"encoding/json"
	"fmt"
	"mig"
	"os"

	"mig/modules/filechecker"
)

func main() {
	var a2 mig.Action
	a, err := mig.ActionFromFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	a2 = a
	a2.SyntaxVersion = 2
	for i, op := range a.Operations {
		if op.Module == "filechecker" {
			input, err := json.Marshal(op.Parameters)
			if err != nil {
				panic(err)
			}
			a2.Operations[i].Parameters = filechecker.ConvertParametersV1toV2(input)
		}
	}
	out, err := json.Marshal(a2)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", out)
}
