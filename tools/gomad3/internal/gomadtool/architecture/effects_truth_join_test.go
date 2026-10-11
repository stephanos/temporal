package architecture

import (
	"go/types"
	"testing"
)

func TestEffectTruthJoinPreservesOnlyAgreedFacts(t *testing.T) {
	for _, left := range []int{-1, 0, 1} {
		for _, right := range []int{-1, 0, 1} {
			first, second := valueOf(types.Typ[types.Bool]), valueOf(types.Typ[types.Bool])
			first.truth, second.truth = left, right
			got := join(first, second).truth
			want := 0
			if left == right {
				want = left
			}
			if got != want {
				t.Fatalf("truth join(%d,%d)=%d, want %d", left, right, got, want)
			}
		}
	}
}

func TestEffectTruthJoinKeepsDirtyAlternativeBranch(t *testing.T) {
	program := effectResolverProgram(t, `package record
func effect(){}
func Root(){condition:=true;condition=false;if condition{}else{go effect()}}
`)
	a := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	for _, finding := range a.findings {
		if finding.Category == "host-effect" {
			return
		}
	}
	t.Fatal("mixed true/false assignment facts pruned the dirty else branch")
}

func TestEffectIfUsesEvaluatedConditionBeforeBodyMutation(t *testing.T) {
	program := effectResolverProgram(t, `package record
func effect(){}
func Root(){condition:=true;if condition{condition=false}else{go effect()}}
`)
	a := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	if len(a.findings) != 0 {
		t.Fatalf("body mutation changed the condition already evaluated for this if: %v", a.findings)
	}
}
