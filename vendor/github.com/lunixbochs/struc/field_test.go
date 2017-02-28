package struc

import (
	"bytes"
	"testing"
)

type badFloat struct {
	BadFloat int `struc:"float64"`
}

func TestBadFloatField(t *testing.T) {
	buf := bytes.NewReader([]byte("00000000"))
	err := Unpack(buf, &badFloat{})
	if err == nil {
		t.Fatal("failed to error on bad float unpack")
	}
}

type emptyLengthField struct {
	Strlen int `struc:"sizeof=Str"`
	Str    []byte
}

func TestEmptyLengthField(t *testing.T) {
	var buf bytes.Buffer
	s := &emptyLengthField{0, []byte("test")}
	o := &emptyLengthField{}
	if err := Pack(&buf, s); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, o); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s.Str, o.Str) {
		t.Fatal("empty length field encode failed")
	}
}

type fixedSlicePad struct {
	Field []byte `struc:"[4]byte"`
}

func TestFixedSlicePad(t *testing.T) {
	var buf bytes.Buffer
	ref := []byte{0, 0, 0, 0}
	s := &fixedSlicePad{}
	if err := Pack(&buf, s); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), ref) {
		t.Fatal("implicit fixed slice pack failed")
	}
	if err := Unpack(&buf, s); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s.Field, ref) {
		t.Fatal("implicit fixed slice unpack failed")
	}
}

type sliceCap struct {
	Len   int `struc:"sizeof=Field"`
	Field []byte
}

func TestSliceCap(t *testing.T) {
	var buf bytes.Buffer
	tmp := &sliceCap{0, []byte("1234")}
	if err := Pack(&buf, tmp); err != nil {
		t.Fatal(err)
	}
	tmp.Field = make([]byte, 0, 4)
	if err := Unpack(&buf, tmp); err != nil {
		t.Fatal(err)
	}
}
