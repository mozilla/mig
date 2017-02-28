package struc

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

type Nested struct {
	Test2 int `struc:"int8"`
}

type Example struct {
	Pad    []byte `struc:"[5]pad"`        // 00 00 00 00 00
	I8f    int    `struc:"int8"`          // 01
	I16f   int    `struc:"int16"`         // 00 02
	I32f   int    `struc:"int32"`         // 00 00 00 03
	I64f   int    `struc:"int64"`         // 00 00 00 00 00 00 00 04
	U8f    int    `struc:"uint8,little"`  // 05
	U16f   int    `struc:"uint16,little"` // 06 00
	U32f   int    `struc:"uint32,little"` // 07 00 00 00
	U64f   int    `struc:"uint64,little"` // 08 00 00 00 00 00 00 00
	Boolf  int    `struc:"bool"`          // 01
	Byte4f []byte `struc:"[4]byte"`       // "abcd"

	I8     int8    // 09
	I16    int16   // 00 0a
	I32    int32   // 00 00 00 0b
	I64    int64   // 00 00 00 00 00 00 00 0c
	U8     uint8   `struc:"little"` // 0d
	U16    uint16  `struc:"little"` // 0e 00
	U32    uint32  `struc:"little"` // 0f 00 00 00
	U64    uint64  `struc:"little"` // 10 00 00 00 00 00 00 00
	BoolT  bool    // 01
	BoolF  bool    // 00
	Byte4  [4]byte // "efgh"
	Float1 float32 // 41 a0 00 00
	Float2 float64 // 41 35 00 00 00 00 00 00

	Size int    `struc:"sizeof=Str,little"` // 0a 00 00 00
	Str  string `struc:"[]byte"`            // "ijklmnopqr"
	Strb string `struc:"[4]byte"`           // "stuv"

	Size2 int    `struc:"uint8,sizeof=Str2"` // 04
	Str2  string // "1234"

	Size3 int    `struc:"uint8,sizeof=Bstr"` // 04
	Bstr  []byte // "5678"

	Nested  Nested  // 00 00 00 01
	NestedP *Nested // 00 00 00 02
	TestP64 *int    `struc:"int64"` // 00 00 00 05

	NestedSize int      `struc:"sizeof=NestedA"` // 00 00 00 02
	NestedA    []Nested // [00 00 00 03, 00 00 00 04]

	Skip int `struc:"skip"`

	CustomTypeSize    Int3   `struc:"sizeof=CustomTypeSizeArr"` // 00 00 00 04
	CustomTypeSizeArr []byte // "ABCD"
}

var five = 5

var reference = &Example{
	nil,
	1, 2, 3, 4, 5, 6, 7, 8, 0, []byte{'a', 'b', 'c', 'd'},
	9, 10, 11, 12, 13, 14, 15, 16, true, false, [4]byte{'e', 'f', 'g', 'h'},
	20, 21,
	10, "ijklmnopqr", "stuv",
	4, "1234",
	4, []byte("5678"),
	Nested{1}, &Nested{2}, &five,
	6, []Nested{{3}, {4}, {5}, {6}, {7}, {8}},
	0,
	Int3(4), []byte("ABCD"),
}

var referenceBytes = []byte{
	0, 0, 0, 0, 0, // pad(5)
	1, 0, 2, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4, // fake int8-int64(1-4)
	5, 6, 0, 7, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0, // fake little-endian uint8-uint64(5-8)
	0,                  // fake bool(0)
	'a', 'b', 'c', 'd', // fake [4]byte

	9, 0, 10, 0, 0, 0, 11, 0, 0, 0, 0, 0, 0, 0, 12, // real int8-int64(9-12)
	13, 14, 0, 15, 0, 0, 0, 16, 0, 0, 0, 0, 0, 0, 0, // real little-endian uint8-uint64(13-16)
	1, 0, // real bool(1), bool(0)
	'e', 'f', 'g', 'h', // real [4]byte
	65, 160, 0, 0, // real float32(20)
	64, 53, 0, 0, 0, 0, 0, 0, // real float64(21)

	10, 0, 0, 0, // little-endian int32(10) sizeof=Str
	'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', // Str
	's', 't', 'u', 'v', // fake string([4]byte)
	04, '1', '2', '3', '4', // real string
	04, '5', '6', '7', '8', // fake []byte(string)

	1, 2, // Nested{1}, Nested{2}
	0, 0, 0, 0, 0, 0, 0, 5, // &five

	0, 0, 0, 6, // int32(6)
	3, 4, 5, 6, 7, 8, // [Nested{3}, ...Nested{8}]

	0, 0, 4, 'A', 'B', 'C', 'D', // Int3(4), []byte("ABCD")
}

func TestCodec(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, reference); err != nil {
		t.Fatal(err)
	}
	out := &Example{}
	if err := Unpack(&buf, out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reference, out) {
		t.Fatal("encode/decode failed")
	}
}

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, reference); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), referenceBytes) {
		t.Fatal("encode failed")
	}
}

func TestDecode(t *testing.T) {
	buf := bytes.NewReader(referenceBytes)
	out := &Example{}
	if err := Unpack(buf, out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reference, out) {
		t.Fatal("decode failed")
	}
}

func TestSizeof(t *testing.T) {
	size, err := Sizeof(reference)
	if err != nil {
		t.Fatal(err)
	}
	if size != len(referenceBytes) {
		t.Fatal("sizeof failed")
	}
}

type ExampleEndian struct {
	T int `struc:"int16,big"`
}

func TestEndianSwap(t *testing.T) {
	var buf bytes.Buffer
	big := &ExampleEndian{1}
	if err := PackWithOrder(&buf, big, binary.BigEndian); err != nil {
		t.Fatal(err)
	}
	little := &ExampleEndian{}
	if err := UnpackWithOrder(&buf, little, binary.LittleEndian); err != nil {
		t.Fatal(err)
	}
	if little.T != 256 {
		t.Fatal("big -> little conversion failed")
	}
}

func TestNilValue(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, nil); err == nil {
		t.Fatal("failed throw error for bad struct value")
	}
	if err := Unpack(&buf, nil); err == nil {
		t.Fatal("failed throw error for bad struct value")
	}
	if _, err := Sizeof(nil); err == nil {
		t.Fatal("failed to throw error for bad struct value")
	}
}

type sliceUnderrun struct {
	Str string   `struc:"[10]byte"`
	Arr []uint16 `struc:"[10]uint16"`
}

func TestSliceUnderrun(t *testing.T) {
	var buf bytes.Buffer
	v := sliceUnderrun{
		Str: "foo",
		Arr: []uint16{1, 2, 3},
	}
	if err := Pack(&buf, &v); err != nil {
		t.Fatal(err)
	}
}
