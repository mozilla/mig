package struc

import (
	"bytes"
	"reflect"
	"testing"
)

func parseTest(data interface{}) error {
	_, err := parseFields(reflect.ValueOf(data))
	return err
}

type empty struct{}

func TestEmptyStruc(t *testing.T) {
	if err := parseTest(&empty{}); err == nil {
		t.Fatal("failed to error on empty struct")
	}
}

type chanStruct struct {
	Test chan int
}

func TestChanError(t *testing.T) {
	if err := parseTest(&chanStruct{}); err == nil {
		// TODO: should probably ignore channel fields
		t.Fatal("failed to error on struct containing channel")
	}
}

type badSizeof struct {
	Size int `struc:"sizeof=Bad"`
}

func TestBadSizeof(t *testing.T) {
	if err := parseTest(&badSizeof{}); err == nil {
		t.Fatal("failed to error on missing Sizeof target")
	}
}

type missingSize struct {
	Test []byte
}

func TestMissingSize(t *testing.T) {
	if err := parseTest(&missingSize{}); err == nil {
		t.Fatal("failed to error on missing field size")
	}
}

type badNested struct {
	Empty empty
}

func TestNestedParseError(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, &badNested{}); err == nil {
		t.Fatal("failed to error on bad nested struct")
	}
}
