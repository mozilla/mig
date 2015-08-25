package testutil

import (
	"mig.ninja/mig/modules"
	"testing"
)

func CheckModuleRegistration(t *testing.T, module_name string) {
	mod, ok := modules.Available[module_name]
	if !ok {
		t.Fatalf("module %s not registered", module_name)
	}

	// test getting a run instance (just don't fail!)
	mod.NewRun()
}
