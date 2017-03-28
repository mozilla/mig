package yara

import "testing"

func TestCompiler(t *testing.T) {
	c, _ := NewCompiler()
	if err := c.AddString(
		"rule test : tag1 { meta: author = \"Hilko Bengen\" strings: $a = \"abc\" fullword condition: $a }", "",
	); err != nil {
		t.Errorf("error: %s", err)
	}
	if err := c.AddString("xxx", ""); err == nil {
		t.Error("did not recognize error")
	} else {
		t.Logf("expected error: %s", err)
	}
}

func TestPanic(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Error("MustCompile with broken data did not panic")
		} else {
			t.Logf("Everything ok, MustCompile panicked: %v", err)
		}
	}()
	_ = MustCompile("asflkjkl", nil)
}

func TestWarnings(t *testing.T) {
	c, _ := NewCompiler()
	c.AddString("rule foo { bar }", "")
	if len(c.Errors) == 0 {
		t.Error()
	}
	t.Logf("Recorded Errors=%#v, Warnings=%#v", c.Errors, c.Warnings)
}
