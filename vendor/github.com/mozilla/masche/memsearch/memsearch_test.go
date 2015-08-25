package memsearch

import (
	"github.com/mozilla/masche/process"
	"github.com/mozilla/masche/test"
	"regexp"
	"testing"
)

var needle []byte = []byte("Find This!")

var buffersToFind = [][]byte{
	[]byte{0xc, 0xa, 0xf, 0xe},
	[]byte{0xd, 0xe, 0xa, 0xd, 0xb, 0xe, 0xe, 0xf},
	[]byte{0xb, 0xe, 0xb, 0xe, 0xf, 0xe, 0x0},
}

var notPresent = []byte("this string should generate a list of bytes not present in the process")

var regexpToMatch = []string{
	"Un dia vi una vaca vestida de uniforme",
	"Un dia vi.*",
	"Un.*vestida de uniforme",
	"Un[a-z\\ ]*vestida de",
}

var regexpToNotMatch = []string{
	"Un dia vi dos vacas vestidas de uniforme",
	"Un dia vi.*sin uniforme",
}

func TestSearchInOtherProcess(t *testing.T) {
	cmd, err := test.LaunchTestCaseAndWaitForInitialization()
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	pid := uint(cmd.Process.Pid)
	proc, err, softerrors := process.OpenFromPid(pid)
	test.PrintSoftErrors(softerrors)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()

	for i, buf := range buffersToFind {
		found, _, err, softerrors := FindBytesSequence(proc, 0, buf)
		test.PrintSoftErrors(softerrors)
		if err != nil {
			t.Fatal(err)
		} else if !found {
			t.Fatalf("memoryGrep failed for case %d, the following buffer should be found: %+v", i, buf)
		}
	}

	// This must not be present
	found, _, err, softerrors := FindBytesSequence(proc, 0, notPresent)
	test.PrintSoftErrors(softerrors)
	if err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatalf("memoryGrep failed, it found a sequense of bytes that it shouldn't")
	}
}

func TestRegexpSearchInOtherProcess(t *testing.T) {
	cmd, err := test.LaunchTestCaseAndWaitForInitialization()
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	pid := uint(cmd.Process.Pid)
	proc, err, softerrors := process.OpenFromPid(pid)
	test.PrintSoftErrors(softerrors)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()

	for i, str := range regexpToMatch {
		r, err := regexp.Compile(str)
		if err != nil {
			t.Fatal(err)
		}

		found, _, err, softerrors := FindRegexpMatch(proc, 0, r)
		test.PrintSoftErrors(softerrors)
		if err != nil {
			t.Fatal(err)
		} else if !found {
			t.Fatalf("memoryGrep failed for case %d, the following regexp should be found: %s", i, str)
		}
	}

	// These must not match
	for i, str := range regexpToNotMatch {
		r, err := regexp.Compile(str)
		if err != nil {
			t.Fatal(err)
		}

		found, _, err, softerrors := FindRegexpMatch(proc, 0, r)
		test.PrintSoftErrors(softerrors)
		if err != nil {
			t.Fatal(err)
		} else if found {
			t.Fatalf("memoryGrep failed for case %d, the following regexp shouldnt be found: %s", i, str)
		}
	}
}
