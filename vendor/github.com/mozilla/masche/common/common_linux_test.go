package common

import (
	"testing"
)

func TestSplitMapsFileEntry(t *testing.T) {
	var entries = []string{
		"7fb8faf65000-7fb8faf66000 rw-p 00023000 08:01 922969                     /lib/x86_64-linux-gnu/ld-2.19.so",
		"7fb8faf65000-7fb8faf66000 rw-p 00023000 08:01 922969                     /lib/x86_64-linux-gnu/with spaces.so",
		"7fb8faf66000-7fb8faf67000 rw-p 00000000 00:00 0",
		"7fff231a6000-7fff231c7000 rw-p 00000000 00:00 0          [stack]",
		"7fff231a6000-7fff231c7000 rw-p 00000000 00:00 0                          [stack]",
	}

	var results = [][]string{
		[]string{"7fb8faf65000-7fb8faf66000", "rw-p", "00023000", "08:01", "922969",
			"/lib/x86_64-linux-gnu/ld-2.19.so"},
		[]string{"7fb8faf65000-7fb8faf66000", "rw-p", "00023000", "08:01", "922969",
			"/lib/x86_64-linux-gnu/with spaces.so"},
		[]string{"7fb8faf66000-7fb8faf67000", "rw-p", "00000000", "00:00", "0", ""},
		[]string{"7fff231a6000-7fff231c7000", "rw-p", "00000000", "00:00", "0", "[stack]"},
		[]string{"7fff231a6000-7fff231c7000", "rw-p", "00000000", "00:00", "0", "[stack]"},
	}

	for i, entry := range entries {
		splitted := SplitMapsFileEntry(entry)
		if !compareStringSlices(results[i], splitted) {
			t.Error("Error splitting map entry", entry, " - Expected:", results[i], " - Got: ", splitted)
		}
	}
}

func TestParseMapsFileMemoryLimits(t *testing.T) {
	var memLimits = []string{
		"7fb8faf65000-7fb8faf66000",
		"7fff231a6000-7fff231c7000",
	}

	var results = [][]uintptr{
		[]uintptr{0x7fb8faf65000, 0x7fb8faf66000},
		[]uintptr{0x7fff231a6000, 0x7fff231c7000},
	}

	for i, limits := range memLimits {
		start, end, err := ParseMapsFileMemoryLimits(limits)
		if err != nil {
			t.Fatal(err)
		}

		if results[i][0] != start {
			t.Error("expected ", results[i][0], " and got ", start)
		}

		if results[i][1] != end {
			t.Error("expected ", results[i][1], " and got ", end)
		}
	}

	var invalidMemoryLimits = []string{
		"a",
		"aa-",
		"-a",
		"NonAlpha-1",
		"1-NonAlpha",
		"1-1-1",
	}

	for _, limits := range invalidMemoryLimits {
		_, _, err := ParseMapsFileMemoryLimits(limits)
		if err == nil {
			t.Error("an error should have been returned when parsing ", limits)
		}
	}

}

func compareStringSlices(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
