package architecture

import "testing"

func TestEffectNotKeepsActuallyReachableDirtyBranches(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"literal-false", "func Root(){if !false{go effect()}}"},
		{"literal-true-else", "func Root(){if !true{}else{go effect()}}"},
		{"alias", "func Root(){value:=false;alias:=value;if !alias{go effect()}}"},
		{"source-return", "func False()bool{return false};func Root(){if !False(){go effect()}}"},
		{"unknown", "func Root(value bool){if !value{go effect()}}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\nfunc effect(){}\n"+test.source)
			a := effectResolverAnalysis(program)
			root := program.Functions["example.invalid/resolver/record.Root"]
			a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			for _, finding := range a.findings {
				if finding.Category == "host-effect" {
					return
				}
			}
			t.Fatal("logical negation pruned an actually reachable dirty branch")
		})
	}
}

func TestEffectNotInvertsOnlyReturnedTruth(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         int
	}{
		{"false", "func Root()bool{return !false}", 1},
		{"true", "func Root()bool{return !true}", -1},
		{"unknown", "func Root(value bool)bool{return !value}", 0},
		{"operand-unchanged", "func Root()bool{value:=false;_ = !value;return value}", -1},
		{"alias-unchanged", "func Root()bool{value:=false;alias:=value;_ = !alias;return value}", -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\n"+test.source)
			a := effectResolverAnalysis(program)
			root := program.Functions["example.invalid/resolver/record.Root"]
			value := a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			if value == nil || value.truth != test.want {
				t.Fatalf("negation/source operand truth: %v, want %d", value, test.want)
			}
		})
	}
}
