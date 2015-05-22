package testutil

import (
	"mig/modules"
	"testing"
)

func CheckModuleRegistration(t *testing.T, module_name string) {
	mod, ok := modules.Available[module_name]
	if !ok {
		t.Fatalf("module %s not registered", module_name)
	}

	execution := mod.NewRunner()
	if _, ok := execution.(modules.Runner); !ok {
		t.Fatalf("module %s registration function does not return a Runner",
			module_name)
	}
}
