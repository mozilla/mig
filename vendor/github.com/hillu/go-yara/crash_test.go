package yara

import (
	"fmt"
	"runtime"
	"testing"
)

// Making a copy of Compiler struct should not cause a crash.
func TestCompilerFinalizer(t *testing.T) {
	var c Compiler
	func() {
		fmt.Println("Create compiler")
		c1, _ := NewCompiler()
		c = *c1
	}()
	fmt.Println("Trigger GC")
	runtime.GC()
	fmt.Println("Trigger Gosched")
	runtime.Gosched()
	fmt.Println("Manually call destructure on copy")
	c.Destroy()
	t.Log("Did not crash due to yr_*_destroy() being called twice. Yay.")
}

// Making a copy of Rules struct should not cause a crash.
func TestRulesFinalizer(t *testing.T) {
	var r Rules
	func() {
		fmt.Println("Create rules")
		r1, _ := Compile("rule test { condition: true }", nil)
		r = *r1
	}()
	fmt.Println("Trigger GC")
	runtime.GC()
	fmt.Println("Trigger Gosched")
	runtime.Gosched()
	fmt.Println("Manually call destructure on copy")
	r.Destroy()
	t.Log("Did not crash due to yr_*_destroy() being called twice. Yay.")
}
