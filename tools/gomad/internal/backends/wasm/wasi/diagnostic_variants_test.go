package wasi

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiagnosticRunRootPreservesEarlierInvocation(t *testing.T) {
	base := t.TempDir()
	first, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(first, "seed0-guest.firefox.json")
	if err := os.WriteFile(marker, []byte("first profile"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Dir(first) != base || filepath.Dir(second) != base {
		t.Fatalf("invocations share their evidence directory: first=%s second=%s", first, second)
	}
	file, err := os.OpenFile(filepath.Join(second, "seed0-guest.firefox.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("fresh invocation cannot create its profile", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "first profile" {
		t.Fatal("earlier profile changed", err)
	}
}

func TestDiagnosticVariantEvidenceMasksOnlyEnvironmentOutputs(t *testing.T) {
	baseline, variant := newTestEnvironment(t, testConfig()), newTestEnvironment(t, testConfig())
	variant.config.Environment = append(variant.config.Environment, "GOGC=200")
	for _, environment := range []*Environment{baseline, variant} {
		call(t, environment, "environ_sizes_get", `{}`)
		call(t, environment, "environ_get", `{}`)
		call(t, environment, "args_get", `{}`)
	}
	before := Result{Transcript: baseline.Transcript()}
	after := Result{Transcript: variant.Transcript()}
	unmasked, err := projectVariantEvidence(after, false)
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectVariantEvidence(before, false)
	if err != nil || reflect.DeepEqual(want, unmasked) {
		t.Fatal("unmasked environment change was lost", err)
	}
	want, err = projectVariantEvidence(before, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, true)
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("GC environment was not masked", err)
	}
	for _, replacement := range []struct{ old, new string }{
		{`"args_get"`, `"sched_yield"`},
		{`"errno":0`, `"errno":1`},
		{`"id":1`, `"id":9`},
	} {
		changed := after
		changed.Transcript = bytes.Replace(after.Transcript, []byte(replacement.old), []byte(replacement.new), 1)
		projection, err := projectVariantEvidence(changed, true)
		if err == nil && reflect.DeepEqual(want, projection) {
			t.Fatalf("mask hid non-environment change: %+v", replacement)
		}
	}
}

func TestDiagnosticVariantEvidenceSeparatesIdentities(t *testing.T) {
	code := uint32(0)
	before := Result{Termination: "exit", ExitCode: &code, Stdout: []byte("out"), Stderr: []byte("err"), Transcript: []byte{}, NetworkTranscript: []byte("network"), EnvironmentIdentity: "baseline", VirtualTimeElapsedNanos: 12}
	before.ChoiceTrace.Bytes = []byte("choice")
	after := before
	after.EnvironmentIdentity, after.FuelRemaining = "variant", 99
	baseRequest := Request{ModuleSHA256: "base-module", HelperSHA256: "base-helper", Engine: PinnedEngine()}
	variantRequest := Request{ModuleSHA256: "variant-module", HelperSHA256: "variant-helper", Engine: PinnedEngine(), diagnosticConfigurationRevision: PinnedEngine().Configuration + "-fuel-off"}
	want, err := projectVariantEvidence(before, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, false)
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("identity or fuel accounting entered behavior equality", err)
	}
	if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(variantRequest, after)) {
		t.Fatal("variant identities were not reported separately")
	}
	for _, field := range []string{"module_sha256", "helper_sha256", "environment_identity", "engine_revision"} {
		t.Run(field, func(t *testing.T) {
			request, result := baseRequest, before
			switch field {
			case "module_sha256":
				request.ModuleSHA256 = "other-module"
			case "helper_sha256":
				request.HelperSHA256 = "other-helper"
			case "environment_identity":
				result.EnvironmentIdentity = "other-environment"
			case "engine_revision":
				request.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-fuel-off"
			}
			if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(request, result)) {
				t.Fatal("changed identity was omitted")
			}
			projection, err := projectVariantEvidence(result, false)
			if err != nil || !reflect.DeepEqual(want, projection) {
				t.Fatal("identity change altered behavior equality", err)
			}
		})
	}
	for name, mutate := range map[string]func(*Result){
		"choice":  func(r *Result) { r.ChoiceTrace.Bytes = []byte("other") },
		"wasi":    func(r *Result) { r.Transcript = []byte("other") },
		"network": func(r *Result) { r.NetworkTranscript = []byte("other") },
		"stdout":  func(r *Result) { r.Stdout = []byte("other") },
		"stderr":  func(r *Result) { r.Stderr = []byte("other") },
		"outcome": func(r *Result) { r.Termination = "trap" },
		"exit":    func(r *Result) { other := uint32(1); r.ExitCode = &other },
		"time":    func(r *Result) { r.VirtualTimeElapsedNanos++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := before
			mutate(&changed)
			projection, err := projectVariantEvidence(changed, false)
			if err == nil && reflect.DeepEqual(want, projection) {
				t.Fatal("behavior change was hidden")
			}
		})
	}
}

func TestDiagnosticVariantRecommendationRequiresReliableGain(t *testing.T) {
	for _, test := range []struct {
		name                string
		baseline, candidate []uint64
		failed, identical   bool
		want                string
	}{
		{"gain", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, true, "adopt"},
		{"spread", []uint64{90, 99, 100, 101, 110}, []uint64{80, 89, 90, 91, 100}, false, true, "investigate"},
		{"slower", []uint64{98, 99, 100, 101, 102}, []uint64{108, 109, 110, 111, 112}, false, true, "reject"},
		{"mismatch", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, false, "reject"},
		{"failed", []uint64{98, 99, 100, 101, 102}, nil, true, false, "reject"},
		{"missing", []uint64{100}, nil, false, true, "investigate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := variantRecommendation(test.baseline, test.candidate, test.failed, test.identical); got != test.want {
				t.Fatalf("recommendation=%s want=%s", got, test.want)
			}
		})
	}
}
