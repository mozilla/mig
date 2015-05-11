package testutil

import (
	"mig/modules"
	"testing"
)

func CheckModuleRegistration(t *testing.T, module_name string) {
	registration, ok := modules.Available[module_name]
	if !ok {
		t.Fatalf("module %s not registered", module_name)
	}

	modRunner := registration.Runner()
	if _, ok := modRunner.(modules.Moduler); !ok {
		t.Fatalf("module %s registration function does not return a Moduler",
			module_name)
	}
}
