package struc

import (
	"bytes"
	"fmt"
	"testing"
)

var packableReference = []byte{
	1, 0, 2, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4, 5, 0, 6, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 8,
	9, 10, 11, 12, 13, 14, 15, 16,
	0, 17, 0, 18, 0, 19, 0, 20, 0, 21, 0, 22, 0, 23, 0, 24,
}

func TestPackable(t *testing.T) {
	var (
		buf bytes.Buffer

		i8  int8   = 1
		i16 int16  = 2
		i32 int32  = 3
		i64 int64  = 4
		u8  uint8  = 5
		u16 uint16 = 6
		u32 uint32 = 7
		u64 uint64 = 8

		u8a  = [8]uint8{9, 10, 11, 12, 13, 14, 15, 16}
		u16a = [8]uint16{17, 18, 19, 20, 21, 22, 23, 24}
	)
	// pack tests
	if err := Pack(&buf, i8); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, i16); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, i32); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, i64); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, u8); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, u16); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, u32); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, u64); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, u8a[:]); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&buf, u16a[:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), packableReference) {
		fmt.Println(buf.Bytes())
		fmt.Println(packableReference)
		t.Fatal("Packable Pack() did not match reference.")
	}
	// unpack tests
	i8 = 0
	i16 = 0
	i32 = 0
	i64 = 0
	u8 = 0
	u16 = 0
	u32 = 0
	u64 = 0
	if err := Unpack(&buf, &i8); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, &i16); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, &i32); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, &i64); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, &u8); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, &u16); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, &u32); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, &u64); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, u8a[:]); err != nil {
		t.Fatal(err)
	}
	if err := Unpack(&buf, u16a[:]); err != nil {
		t.Fatal(err)
	}
	// unpack checks
	if i8 != 1 || i16 != 2 || i32 != 3 || i64 != 4 {
		t.Fatal("Signed integer unpack failed.")
	}
	if u8 != 5 || u16 != 6 || u32 != 7 || u64 != 8 {
		t.Fatal("Unsigned integer unpack failed.")
	}
	for i := 0; i < 8; i++ {
		if u8a[i] != uint8(i+9) {
			t.Fatal("uint8 array unpack failed.")
		}
	}
	for i := 0; i < 8; i++ {
		if u16a[i] != uint16(i+17) {
			t.Fatal("uint16 array unpack failed.")
		}
	}
}
