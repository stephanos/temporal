package recipe

import (
	"strings"
	"testing"
)

func TestDefaultAndNormalizePreserveCanonicalGuardedRecipe(t *testing.T) {
	workflow := Default()
	resolved, err := Normalize(workflow)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Stages) != len(order) {
		t.Fatalf("stages = %#v", resolved.Stages)
	}
	for index, kind := range order {
		stage := resolved.Stage(kind)
		if stage.Kind != kind || stage != resolved.Stages[index] {
			t.Fatalf("stage %d = %#v, lookup=%#v", index, resolved.Stages[index], stage)
		}
	}
	if resolved.Stage(Apply).Mode != "explicit" || resolved.Stage(Check).Prompt != "" {
		t.Fatalf("guarded stages = %#v", resolved.Stages)
	}
	resolved.Stages[0].Prompt = "changed"
	if Default().Stages[0].Prompt == "changed" {
		t.Fatal("default workflow shares mutable storage")
	}
}

func TestNormalizeRejectsEveryRecipeContractViolation(t *testing.T) {
	cases := map[string]func(*Workflow){
		"wrong count": func(workflow *Workflow) { workflow.Stages = workflow.Stages[:len(workflow.Stages)-1] },
		"wrong order": func(workflow *Workflow) {
			workflow.Stages[0], workflow.Stages[1] = workflow.Stages[1], workflow.Stages[0]
		},
		"missing prompt":           func(workflow *Workflow) { workflow.Stages[0].Prompt = "" },
		"check prompt":             func(workflow *Workflow) { workflow.Stages[3].Prompt = "skip integrity" },
		"plan review prompt":       func(workflow *Workflow) { workflow.Stages[1].ReviewPrompt = "" },
		"plan revision prompt":     func(workflow *Workflow) { workflow.Stages[1].RevisionPrompt = "" },
		"unexpected review prompt": func(workflow *Workflow) { workflow.Stages[0].ReviewPrompt = "extra" },
		"automatic apply":          func(workflow *Workflow) { workflow.Stages[6].Mode = "automatic" },
		"mode on non-apply":        func(workflow *Workflow) { workflow.Stages[0].Mode = "explicit" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			workflow := Default()
			mutate(&workflow)
			if _, err := Normalize(workflow); err == nil {
				t.Fatalf("invalid workflow was accepted: %#v", workflow)
			}
		})
	}
	if stage := (Workflow{}).Stage(Discover); stage.Kind != Discover || !strings.Contains(stage.Prompt, "project") {
		t.Fatalf("zero workflow lookup = %#v", stage)
	}
}
