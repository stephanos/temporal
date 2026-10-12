package native

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/record"
)

var _ backends.Provider = (*Provider)(nil)

func TestProviderDescriptionIsLazyAndDetached(t *testing.T) {
	provider := New(Config{ToolchainRoot: "/must-not-be-read"})
	capabilities := provider.Capabilities()
	if capabilities.Name != "native" || !capabilities.Simulation || !capabilities.ReadOnlyMounts || !capabilities.IOTranscript {
		t.Fatalf("native capabilities = %#v", capabilities)
	}
	capabilities.Commands[0] = "changed"
	if provider.Capabilities().Commands[0] == "changed" {
		t.Fatal("capabilities retain caller-owned slices")
	}
	description, err := provider.Describe(t.Context(), backends.Configuration{})
	if err != nil || !transcript.Default().MatchesRecorded(description.IOProfile.Name, string(description.IOProfile.ImplementationSHA256), string(description.IOProfile.InventorySHA256), description.IOProfile.Inventory) || len(description.SemanticProbes) == 0 {
		t.Fatalf("lazy description = %#v, %v", description, err)
	}
	for _, config := range []backends.Configuration{{IOProfile: "foreign"}, {ClockTick: "foreign"}, {RequiredSemanticProbes: []string{"unknown"}}, {RequiredSemanticProbes: []string{description.SemanticProbes[0], description.SemanticProbes[0]}}} {
		if _, err := provider.Describe(t.Context(), config); err == nil {
			t.Fatalf("invalid configuration accepted: %#v", config)
		}
	}
}

func TestProviderEvidenceProjectsOperationsAndRejectsInvalidSources(t *testing.T) {
	provider := New(Config{})
	encoded, err := transcript.EncodeTranscript([]transcript.Operation{{Ordinal: 0, Name: "os.open", Result: 2}})
	if err != nil {
		t.Fatal(err)
	}
	observed, err := provider.Evidence(t.Context(), backends.EvidenceRequest{Transcript: encoded})
	if err != nil || !reflect.DeepEqual(observed.Operations, []backends.OperationOutcome{{Name: "os.open", Result: 2}}) || observed.Coverage.Schema != record.SemanticCoverageSchema {
		t.Fatalf("projected evidence = %#v, %v", observed, err)
	}
	for _, request := range []backends.EvidenceRequest{{Transcript: encoded, Probes: []string{"unknown"}}, {Transcript: []byte{1}}, {Probes: []string{"unknown"}}, {RequiredSemanticProbes: []string{"unknown"}}} {
		if _, err := provider.Evidence(t.Context(), request); err == nil {
			t.Fatalf("invalid evidence accepted: %#v", request)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := provider.Evidence(ctx, backends.EvidenceRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled evidence error = %v", err)
	}
}

func TestProviderExecutionRejectsForeignProfileBeforeLaunch(t *testing.T) {
	provider := New(Config{SupervisorCommand: []string{"must-not-launch"}, BootstrapCommand: []string{"must-not-launch"}})
	_, err := provider.Execute(t.Context(), backends.ExecutionRequest{IO: &backends.IOCapability{Profile: record.IOProfile{Name: "foreign"}}})
	if err == nil || err.Error() != "execution I/O profile identity does not match the native backend" {
		t.Fatalf("foreign profile error = %v", err)
	}
}

func TestCoordinatorInputSeparatesNativeConfiguration(t *testing.T) {
	provider := New(Config{ToolchainRoot: "native-root", SupervisorCommand: []string{"native-supervisor"}, RunnerBuild: "native-build"})
	input, err := provider.coordinatorInput([]byte(`{"Target":{},"Parallel":7}`))
	if err != nil {
		t.Fatal(err)
	}
	config, shared, err := CoordinatorInput(input)
	if err != nil || config.ToolchainRoot != "native-root" || config.RunnerBuild != "native-build" || !reflect.DeepEqual(config.SupervisorCommand, []string{"native-supervisor"}) {
		t.Fatalf("native coordinator configuration = %#v, %v", config, err)
	}
	var options struct {
		Target   struct{ ToolchainRoot string }
		Parallel int
	}
	if err := json.Unmarshal(shared, &options); err != nil || options.Target.ToolchainRoot != "native-root" || options.Parallel != 7 || strings.Contains(string(shared), "SupervisorCommand") || strings.Contains(string(shared), "RunnerBuild") {
		t.Fatalf("shared coordinator options = %s, %v", shared, err)
	}
	for _, input := range [][]byte{[]byte(`null`), []byte(`{}`), []byte(`{} {}`), []byte(`{"Target":{},"SupervisorCommand":7,"RunnerBuild":"build"}`)} {
		if _, _, err := CoordinatorInput(input); err == nil {
			t.Fatalf("invalid native coordinator input accepted: %s", input)
		}
	}
}
