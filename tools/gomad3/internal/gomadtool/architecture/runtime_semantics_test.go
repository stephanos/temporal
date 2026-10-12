package architecture

import "testing"

func TestRuntimeSemanticsPreparationOwner(t *testing.T) {
	const module = "go.temporal.io/server/tools/gomad3"
	if got := Owner(module, module+"/runtimesemantics"); got != "runtimesemantics" {
		t.Fatalf("runtime-input preparation owner = %q", got)
	}
	for _, test := range []struct {
		from, to string
		allowed  bool
	}{
		{"toolchain", "runtimesemantics", true},
		{"developer", "runtimesemantics", true},
		{"runtimesemantics", "hostfs", true},
		{"runtimesemantics", "toolchain", false},
		{"runtimesemantics", "runner", false},
		{"world", "runtimesemantics", false},
	} {
		if got := OwnerMayImport(module, test.from, test.to, module+"/"+test.from, module+"/"+test.to); got != test.allowed {
			t.Errorf("%s -> %s allowed = %v, want %v", test.from, test.to, got, test.allowed)
		}
	}
}
