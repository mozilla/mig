package struc

import (
	"bytes"
	"testing"
)

func TestBadType(t *testing.T) {
	defer func() { recover() }()
	Type(-1).Size()
	t.Fatal("failed to panic for invalid Type.Size()")
}

func TestTypeString(t *testing.T) {
	if Pad.String() != "pad" {
		t.Fatal("type string representation failed")
	}
}

type sizeOffTest struct {
	Size Size_t
	Off  Off_t
}

func TestSizeOffTypes(t *testing.T) {
	bits := []int{8, 16, 32, 64}
	var buf bytes.Buffer
	test := &sizeOffTest{1, 2}
	for _, b := range bits {
		if err := PackWithOptions(&buf, test, &Options{PtrSize: b}); err != nil {
			t.Fatal(err)
		}
	}
	reference := []byte{
		1, 2,
		0, 1, 0, 2,
		0, 0, 0, 1, 0, 0, 0, 2,
		0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2,
	}
	if !bytes.Equal(reference, buf.Bytes()) {
		t.Errorf("reference != bytes: %v", reference, buf.Bytes())
	}
	reader := bytes.NewReader(buf.Bytes())
	for _, b := range bits {
		out := &sizeOffTest{}
		if err := UnpackWithOptions(reader, out, &Options{PtrSize: b}); err != nil {
			t.Fatal(err)
		}
		if out.Size != 1 || out.Off != 2 {
			t.Errorf("Size_t/Off_t mismatch: {%d, %d}\n%v", out.Size, out.Off, buf.Bytes())
		}
	}
}
