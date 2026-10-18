package measurement

import (
	"slices"
	"strings"
	"testing"
)

func TestFlagsAndCommandsAreDerivedFromInputs(t *testing.T) {
	for variable, flag := range map[string]string{
		"GOMAD_WASM_PROFILE_ROOT":               "profile-root",
		"GOMAD_STOCK_GO":                        "stock-go",
		"GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES": "features",
	} {
		if got := Flag(variable); got != flag {
			t.Fatalf("Flag(%s) = %q, want %q", variable, got, flag)
		}
	}
	command := Command("go", "/repo", "runtime-cost-profile", map[string]string{"GOMAD_WASM_PROFILE_ROOT": "/root", "GOMAD_STOCK_GO": "", "PATH": "/bin"})
	want := []string{"go", "run", "./internal/backends/wasm/cmd/wasmmeasure", "runtime-cost-profile", "--repo", "/repo", "--profile-root", "/root"}
	if !slices.Equal(command, want) {
		t.Fatalf("command %q", command)
	}
	record := RerunRecord("go", "/repo", "runtime-cost-profile", map[string]string{"GOMAD_WASM_PROFILE_ROOT": "/a b", "PATH": "/bin"}, true)
	shell, _ := record["command"].(string)
	if !strings.HasSuffix(shell, "--profile-root '/a b'") || record["environment"].(map[string]string)["PATH"] != "/bin" {
		t.Fatalf("rerun record %v", record)
	}
	for _, name := range Names() {
		inputs := Inputs(name)
		if len(inputs) == 0 || !inputs[0].Required {
			t.Fatalf("%s lists no required input first", name)
		}
		seen := map[string]bool{}
		for _, input := range inputs {
			if !strings.HasPrefix(input.Variable, "GOMAD_") || seen[Flag(input.Variable)] || Flag(input.Variable) == "repo" || Flag(input.Variable) == "go" {
				t.Fatalf("%s input %s has a reserved or duplicate flag", name, input.Variable)
			}
			seen[Flag(input.Variable)] = true
		}
	}
}
