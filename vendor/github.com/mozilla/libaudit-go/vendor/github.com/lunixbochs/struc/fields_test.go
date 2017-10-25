package struc

import (
	"bytes"
	"reflect"
	"testing"
)

var refVal = reflect.ValueOf(reference)

func TestFieldsParse(t *testing.T) {
	if _, err := parseFields(refVal); err != nil {
		t.Fatal(err)
	}
}

func TestFieldsString(t *testing.T) {
	fields, _ := parseFields(refVal)
	fields.String()
}

type sizefromStruct struct {
	Size1 uint `struc:"sizeof=Var1"`
	Var1  []byte
	Size2 int `struc:"sizeof=Var2"`
	Var2  []byte
}

func TestFieldsSizefrom(t *testing.T) {
	var test = sizefromStruct{
		Var1: []byte{1, 2, 3},
		Var2: []byte{4, 5, 6},
	}
	var buf bytes.Buffer
	err := Pack(&buf, &test)
	if err != nil {
		t.Fatal(err)
	}
	err = Unpack(&buf, &test)
	if err != nil {
		t.Fatal(err)
	}
}

type sizefromStructBad struct {
	Size1 string `struc:"sizeof=Var1"`
	Var1  []byte
}

func TestFieldsSizefromBad(t *testing.T) {
	var test = &sizefromStructBad{Var1: []byte{1, 2, 3}}
	var buf bytes.Buffer
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("failed to panic on bad sizeof type")
		}
	}()
	Pack(&buf, &test)
}
