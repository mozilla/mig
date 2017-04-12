package yara

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if r, err := Compile(`rule test : tag1 { meta: author = "Hilko Bengen" strings: $a = "abc" fullword condition: $a }`, nil); err != nil {
		os.Exit(1)
	} else if err = r.Save("testrules.yac"); err != nil {
		os.Exit(1)
	}
	rc := m.Run()
	os.Remove("testrules.yac")
	os.Exit(rc)
}
