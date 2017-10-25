package struc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

type BenchExample struct {
	Test    [5]byte
	A       int32
	B, C, D int16
	Test2   [4]byte
	Length  int32
}

type BenchStrucExample struct {
	Test    [5]byte `struc:"[5]byte"`
	A       int     `struc:"int32"`
	B, C, D int     `struc:"int16"`
	Test2   [4]byte `struc:"[4]byte"`
	Length  int     `struc:"int32,sizeof=Data"`
	Data    []byte
}

var benchRef = &BenchExample{
	[5]byte{1, 2, 3, 4, 5},
	1, 2, 3, 4,
	[4]byte{1, 2, 3, 4},
	8,
}

var eightBytes = []byte("8bytestr")

var benchStrucRef = &BenchStrucExample{
	[5]byte{1, 2, 3, 4, 5},
	1, 2, 3, 4,
	[4]byte{1, 2, 3, 4},
	8, eightBytes,
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := Pack(&buf, benchStrucRef)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdlibEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := binary.Write(&buf, binary.BigEndian, benchRef)
		if err != nil {
			b.Fatal(err)
		}
		_, err = buf.Write(eightBytes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManualEncode(b *testing.B) {
	order := binary.BigEndian
	s := benchStrucRef
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		tmp := make([]byte, 29)
		copy(tmp[0:5], s.Test[:])
		order.PutUint32(tmp[5:9], uint32(s.A))
		order.PutUint16(tmp[9:11], uint16(s.B))
		order.PutUint16(tmp[11:13], uint16(s.C))
		order.PutUint16(tmp[13:15], uint16(s.D))
		copy(tmp[15:19], s.Test2[:])
		order.PutUint32(tmp[19:23], uint32(s.Length))
		copy(tmp[23:], s.Data)
		_, err := buf.Write(tmp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	var out BenchStrucExample
	var buf bytes.Buffer
	if err := Pack(&buf, benchStrucRef); err != nil {
		b.Fatal(err)
	}
	bufBytes := buf.Bytes()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewReader(bufBytes)
		err := Unpack(buf, &out)
		if err != nil {
			b.Fatal(err)
		}
		out.Data = nil
	}
}

func BenchmarkStdlibDecode(b *testing.B) {
	var out BenchExample
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, *benchRef)
	_, err := buf.Write(eightBytes)
	if err != nil {
		b.Fatal(err)
	}
	bufBytes := buf.Bytes()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewReader(bufBytes)
		err := binary.Read(buf, binary.BigEndian, &out)
		if err != nil {
			b.Fatal(err)
		}
		tmp := make([]byte, out.Length)
		_, err = buf.Read(tmp)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManualDecode(b *testing.B) {
	var o BenchStrucExample
	var buf bytes.Buffer
	if err := Pack(&buf, benchStrucRef); err != nil {
		b.Fatal(err)
	}
	tmp := buf.Bytes()
	order := binary.BigEndian
	for i := 0; i < b.N; i++ {
		copy(o.Test[:], tmp[0:5])
		o.A = int(order.Uint32(tmp[5:9]))
		o.B = int(order.Uint16(tmp[9:11]))
		o.C = int(order.Uint16(tmp[11:13]))
		o.D = int(order.Uint16(tmp[13:15]))
		copy(o.Test2[:], tmp[15:19])
		o.Length = int(order.Uint32(tmp[19:23]))
		o.Data = make([]byte, o.Length)
		copy(o.Data, tmp[23:])
	}
}

func BenchmarkFullEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		if err := Pack(&buf, reference); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFullDecode(b *testing.B) {
	var out Example
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(referenceBytes)
		if err := Unpack(buf, &out); err != nil {
			b.Fatal(err)
		}
	}
}
