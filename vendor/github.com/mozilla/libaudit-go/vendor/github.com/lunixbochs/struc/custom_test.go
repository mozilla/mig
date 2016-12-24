package struc

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"
	"testing"
)

type Int3 uint32

func (i *Int3) Pack(p []byte, opt *Options) (int, error) {
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], uint32(*i))
	copy(p, tmp[1:])
	return 3, nil
}
func (i *Int3) Unpack(r io.Reader, length int, opt *Options) error {
	var tmp [4]byte
	if _, err := r.Read(tmp[1:]); err != nil {
		return err
	}
	*i = Int3(binary.BigEndian.Uint32(tmp[:]))
	return nil
}
func (i *Int3) Size(opt *Options) int {
	return 3
}
func (i *Int3) String() string {
	return strconv.FormatUint(uint64(*i), 10)
}

func TestCustom(t *testing.T) {
	var buf bytes.Buffer
	var i Int3 = 3
	if err := Pack(&buf, &i); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0, 0, 3}) {
		t.Fatal("error packing custom int")
	}
	var i2 Int3
	if err := Unpack(&buf, &i2); err != nil {
		t.Fatal(err)
	}
	if i2 != 3 {
		t.Fatal("error unpacking custom int")
	}
}

type Int3Struct struct {
	I Int3
}

func TestCustomStruct(t *testing.T) {
	var buf bytes.Buffer
	i := Int3Struct{3}
	if err := Pack(&buf, &i); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0, 0, 3}) {
		t.Fatal("error packing custom int struct")
	}
	var i2 Int3Struct
	if err := Unpack(&buf, &i2); err != nil {
		t.Fatal(err)
	}
	if i2.I != 3 {
		t.Fatal("error unpacking custom int struct")
	}
}

// TODO: slices of custom types don't work yet
/*
type Int3SliceStruct struct {
	I [2]Int3
}

func TestCustomSliceStruct(t *testing.T) {
	var buf bytes.Buffer
	i := Int3SliceStruct{[2]Int3{3, 4}}
	if err := Pack(&buf, &i); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0, 0, 3}) {
		t.Fatal("error packing custom int struct")
	}
	var i2 Int3SliceStruct
	if err := Unpack(&buf, &i2); err != nil {
		t.Fatal(err)
	}
	if i2.I[0] != 3 && i2.I[1] != 4 {
		t.Fatal("error unpacking custom int struct")
	}
}
*/
