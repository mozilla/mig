package struc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestFloat16(t *testing.T) {
	// test cases from https://en.wikipedia.org/wiki/Half-precision_floating-point_format#Half_precision_examples
	tests := []struct {
		B string
		F float64
	}{
		//s expnt significand
		{"0 01111 0000000000", 1},
		{"0 01111 0000000001", 1.0009765625},
		{"1 10000 0000000000", -2},
		{"0 11110 1111111111", 65504},
		// {"0 00001 0000000000", 0.0000610352},
		// {"0 00000 1111111111", 0.0000609756},
		// {"0 00000 0000000001", 0.0000000596046},
		{"0 00000 0000000000", 0},
		// {"1 00000 0000000000", -0},
		{"0 11111 0000000000", math.Inf(1)},
		{"1 11111 0000000000", math.Inf(-1)},
		{"0 01101 0101010101", 0.333251953125},
	}
	for _, test := range tests {
		var buf bytes.Buffer
		f := Float16(test.F)
		if err := Pack(&buf, &f); err != nil {
			t.Error("pack failed:", err)
			continue
		}
		bitval, _ := strconv.ParseUint(strings.Replace(test.B, " ", "", -1), 2, 16)
		tmp := binary.BigEndian.Uint16(buf.Bytes())
		if tmp != uint16(bitval) {
			t.Errorf("incorrect pack: %s != %016b (%f)", test.B, tmp, test.F)
			continue
		}
		var f2 Float16
		if err := Unpack(&buf, &f2); err != nil {
			t.Error("unpack failed:", err)
			continue
		}
		// let sprintf deal with (im)precision for me here
		if fmt.Sprintf("%f", f) != fmt.Sprintf("%f", f2) {
			t.Errorf("incorrect unpack: %016b %f != %f", bitval, f, f2)
		}
	}
}
