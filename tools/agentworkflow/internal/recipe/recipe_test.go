package recipe

import (
	"reflect"
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
		if stage.Kind != kind || !reflect.DeepEqual(stage, resolved.Stages[index]) {
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

func TestNormalizePreservesModelsForAgentStages(t *testing.T) {
	for _, kind := range []Kind{Discover, Plan, Implement, Review, Repair} {
		t.Run(string(kind), func(t *testing.T) {
			workflow := Default()
			stage := workflow.Stage(kind)
			stage.Models = Models{"codex": "gpt-5.3-codex", "claude": "opus"}
			for index := range workflow.Stages {
				if workflow.Stages[index].Kind == kind {
					workflow.Stages[index] = stage
				}
			}

			resolved, err := Normalize(workflow)
			if err != nil {
				t.Fatal(err)
			}
			models := resolved.Stage(kind).Models
			if models["codex"] != "gpt-5.3-codex" || models["claude"] != "opus" {
				t.Fatalf("models = %#v", models)
			}
			stage.Models["codex"] = "changed"
			if resolved.Stage(kind).Models["codex"] != "gpt-5.3-codex" {
				t.Fatal("normalized workflow shares mutable model storage")
			}
		})
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
		"blank codex model":        func(workflow *Workflow) { workflow.Stages[0].Models = Models{"codex": " "} },
		"blank claude model":       func(workflow *Workflow) { workflow.Stages[0].Models = Models{"claude": "\t"} },
		"models on check":          func(workflow *Workflow) { workflow.Stages[3].Models = Models{} },
		"models on apply":          func(workflow *Workflow) { workflow.Stages[6].Models = Models{"claude": "opus"} },
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
