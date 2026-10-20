package measure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	wasmbackend "go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

type stressAdmissionProvider struct {
	backends.Provider
	prepared bool
	stop     error
}

func (*stressAdmissionProvider) Capabilities() backends.Capabilities {
	return wasmbackend.CapabilitiesForProfile(wasi.CooperativeProfile)
}

func (*stressAdmissionProvider) Describe(context.Context, backends.Configuration) (backends.Description, error) {
	return backends.Description{IOProfile: record.BackendIOProfile(record.BackendMetadata{Name: wasmbackend.Name, Provenance: record.BackendPayload{SHA256: record.HashBytes([]byte("admission fixture"))}})}, nil
}

func (provider *stressAdmissionProvider) Prepare(context.Context, target.Spec) (target.Prepared, error) {
	provider.prepared = true
	return target.Prepared{}, provider.stop
}

func TestStressRuntimeRequestPassesRunnerAdmissionWithoutExecuting(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	helper := filepath.Join(root, "unused-helper")
	if err := os.WriteFile(helper, []byte("helper identity only; never executed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tools/gomad"), 0700); err != nil {
		t.Fatal(err)
	}
	scope := &runScope{runResources: &runResources{}, options: Options{Repo: root, Inputs: map[string]string{"GOMAD_WASM_TEST_HELPER": helper}}}
	ctx := context.WithValue(t.Context(), runKey{}, scope)
	request, err := stressRequest(ctx, root, "runtime")
	if err != nil {
		t.Fatal(err)
	}
	if request.Config.Clock.ReadStepNanos != 1000 || request.Runtime.Choice.Limit != 1<<20 {
		t.Fatal("runtime request no longer matches cooperativeModuleRequest qualification policy")
	}
	if request.Config.Limits.TranscriptBytes < record.DefaultIOTranscriptBytes {
		t.Fatal("runtime request is below Runner's transcript admission bound")
	}
	if err := record.ValidateIOTranscriptLimit(request.Config.Limits.TranscriptBytes); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []uint64{request.Runtime.Choice.Limit, stressChoiceTraceLimit} {
		if _, err := choice.DiagnosticLimit(limit); err != nil {
			t.Fatalf("choice capacity %d cannot record bounded diagnostics: %v", limit, err)
		}
	}
	spec, err := stressTarget(ctx, "runtime", request)
	if err != nil {
		t.Fatal(err)
	}
	stop := errors.New("admission passed; stop before source preparation")
	provider := &stressAdmissionProvider{stop: stop}
	_, err = stressRunnerCampaign(ctx, root, provider, spec, request)
	if !provider.prepared || !errors.Is(err, stop) {
		t.Fatalf("runtime request failed Runner admission before preparation: %v", err)
	}
}
