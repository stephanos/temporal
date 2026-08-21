package generate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChoiceImplementationIdentityBindsGeneratedAndInstrumentedInputs(t *testing.T) {
	inputs := choiceImplementationInputs{
		Schema:          []byte("schema"),
		CodecTemplate:   []byte("codec"),
		RuntimeTemplate: []byte("runtime-codec"),
		RuntimeOverlay:  []byte("runtime-instrumentation"),
		ToolchainPatch:  []byte("toolchain-patch"),
		HostTrace:       []byte("host-trace"),
		HostTape:        []byte("host-tape"),
	}
	want := choiceImplementationIdentity(inputs)
	if want == ([32]byte{}) {
		t.Fatal("choice implementation identity is empty")
	}
	mutations := [][]byte{inputs.Schema, inputs.CodecTemplate, inputs.RuntimeTemplate, inputs.RuntimeOverlay, inputs.ToolchainPatch, inputs.HostTrace, inputs.HostTape}
	for index, original := range mutations {
		changed := inputs
		value := append(bytes.Clone(original), '!')
		switch index {
		case 0:
			changed.Schema = value
		case 1:
			changed.CodecTemplate = value
		case 2:
			changed.RuntimeTemplate = value
		case 3:
			changed.RuntimeOverlay = value
		case 4:
			changed.ToolchainPatch = value
		case 5:
			changed.HostTrace = value
		case 6:
			changed.HostTape = value
		}
		if got := choiceImplementationIdentity(changed); got == want {
			t.Fatalf("identity did not bind input %d", index)
		}
	}
}

func TestReadSimulationModelSchemaPinsBoundedOperationVocabulary(t *testing.T) {
	definition, err := readSimulationModelSchema(filepath.Join("..", "..", "..", "simulation", "schema", "modelwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 1 || definition.Profile != "gomadv3.simulation-model/v1" || definition.TransportMagic != "GOMADPM\x03" || definition.Limits.FrameBytes != 128<<20 || definition.Limits.StringBytes != 4096 || definition.Limits.DataBytes != 64<<20 || definition.Limits.Entries != 100_000 || definition.Limits.NodeBytes != 256 || definition.Limits.ErrorBytes != 4096 {
		t.Fatalf("simulation model protocol = %+v", definition)
	}
	if definition.Models.Network != 1 || definition.Models.Volume != 2 || definition.NetworkOperations.Listen != 1 || definition.NetworkOperations.ConnSetWriteDeadline != 13 || definition.VolumeOperations.Resolve != 1 || definition.VolumeOperations.MappingClose != 28 {
		t.Fatalf("simulation model operation vocabulary = network %+v volume %+v", definition.NetworkOperations, definition.VolumeOperations)
	}
}

func TestReadLiveCapabilitySchemaPinsClosedProtocol(t *testing.T) {
	definition, err := readLiveCapabilitySchema(filepath.Join("..", "..", "..", "target", "internal", "livecap", "livecap.json"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 2 || definition.Schema != "gomadv3.live-capability-manifest/v2" || definition.Symbol != "runtime.gomadCapabilities" || definition.GuardSymbol != "runtime.gomadCapabilityGuard" {
		t.Fatalf("live capability identity = version %d schema %q symbol %q", definition.Version, definition.Schema, definition.Symbol)
	}
	if definition.Header.Magic != "GOMADCAPABILITY\x00" || definition.Header.Bytes != 112 {
		t.Fatalf("live capability header = magic %q bytes %d", definition.Header.Magic, definition.Header.Bytes)
	}
	if definition.Limits.PayloadBytes != 16<<20 || definition.Limits.Facts != 100_000 || definition.Limits.StringBytes != 4<<10 || definition.Limits.OwnerFacts != 4_096 {
		t.Fatalf("live capability limits = %+v", definition.Limits)
	}
	wantKinds := []string{"boundary", "capability", "foreign", "guard", "linkname"}
	if strings.Join(definition.FactKinds, ",") != strings.Join(wantKinds, ",") {
		t.Fatalf("live capability fact kinds = %v, want %v", definition.FactKinds, wantKinds)
	}
	wantDispositions := []string{"denied", "guarded", "modeled", "pack"}
	if strings.Join(definition.Dispositions, ",") != strings.Join(wantDispositions, ",") {
		t.Fatalf("live capability dispositions = %v, want %v", definition.Dispositions, wantDispositions)
	}
	if strings.Join(definition.GuardExemptions, ",") != "syscall.Clearenv,syscall.Environ,syscall.Errno.Error,syscall.Errno.Is,syscall.Errno.Temporary,syscall.Errno.Timeout,syscall.Getenv,syscall.Setenv,syscall.Unsetenv,syscall.Write" {
		t.Fatalf("live capability guard exemptions = %v", definition.GuardExemptions)
	}
	wantForbidden := []string{"os/exec", "os/signal", "os/user", "plugin", "runtime/cgo", "syscall"}
	if strings.Join(definition.ForbiddenImports, ",") != strings.Join(wantForbidden, ",") || strings.Join(definition.ForbiddenPrefixes, ",") != "golang.org/x/sys" {
		t.Fatalf("live capability forbidden vocabulary = imports %v prefixes %v", definition.ForbiddenImports, definition.ForbiddenPrefixes)
	}
}

func TestLiveCapabilityUniverseIdentityBindsBoundaryAndForbiddenVocabulary(t *testing.T) {
	definition := liveCapabilitySchema{
		Version: 1, Schema: "gomadv3.live-capability-manifest/v1",
		ForbiddenImports: []string{"syscall"}, ForbiddenPrefixes: []string{"golang.org/x/sys"},
	}
	want, err := liveCapabilityUniverseIdentity(definition, "sha256:"+strings.Repeat("1", 64))
	if err != nil {
		t.Fatal(err)
	}
	changed := definition
	changed.ForbiddenImports = []string{"os/exec", "syscall"}
	got, err := liveCapabilityUniverseIdentity(changed, "sha256:"+strings.Repeat("1", 64))
	if err != nil {
		t.Fatal(err)
	}
	if got == want {
		t.Fatal("capability universe identity omitted forbidden imports")
	}
	changed = definition
	changed.GuardExemptions = []string{"syscall.rsaAlignOf"}
	got, err = liveCapabilityUniverseIdentity(changed, "sha256:"+strings.Repeat("1", 64))
	if err != nil {
		t.Fatal(err)
	}
	if got == want {
		t.Fatal("capability universe identity omitted guard exemptions")
	}
	got, err = liveCapabilityUniverseIdentity(definition, "sha256:"+strings.Repeat("2", 64))
	if err != nil {
		t.Fatal(err)
	}
	if got == want {
		t.Fatal("capability universe identity omitted boundary manifest")
	}
}

func TestLiveCapabilityImplementationIdentityBindsProducerAndValidatorInputs(t *testing.T) {
	inputs := liveCapabilityImplementationInputs{
		Schema: []byte("schema"), CodecTemplate: []byte("codec"), CompilerEmitter: []byte("compiler"), LinkerProjector: []byte("linker"),
		Encoder: []byte("encoder"), GuardFlag: []byte("guard-flag"), GuardSource: []byte("guard-source"), RuntimeGuard: []byte("runtime-guard"),
		InterceptionSource: []byte("interception"), BoundaryTable: []byte("boundary"), HostValidator: []byte("validator"), ProjectionContract: []byte("projection"),
	}
	want := liveCapabilityImplementationIdentity(inputs)
	if want == ([32]byte{}) {
		t.Fatal("live capability implementation identity is empty")
	}
	mutations := [][]byte{
		inputs.Schema, inputs.CodecTemplate, inputs.CompilerEmitter, inputs.LinkerProjector, inputs.Encoder,
		inputs.GuardFlag, inputs.GuardSource, inputs.RuntimeGuard, inputs.InterceptionSource, inputs.BoundaryTable, inputs.HostValidator, inputs.ProjectionContract,
	}
	for index, original := range mutations {
		changed := inputs
		value := append(bytes.Clone(original), '!')
		switch index {
		case 0:
			changed.Schema = value
		case 1:
			changed.CodecTemplate = value
		case 2:
			changed.CompilerEmitter = value
		case 3:
			changed.LinkerProjector = value
		case 4:
			changed.Encoder = value
		case 5:
			changed.GuardFlag = value
		case 6:
			changed.GuardSource = value
		case 7:
			changed.RuntimeGuard = value
		case 8:
			changed.InterceptionSource = value
		case 9:
			changed.BoundaryTable = value
		case 10:
			changed.HostValidator = value
		case 11:
			changed.ProjectionContract = value
		}
		if got := liveCapabilityImplementationIdentity(changed); got == want {
			t.Fatalf("identity did not bind input %d", index)
		}
	}
}

func TestRunGeneratesAndChecksEveryEndpoint(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{
		"toolchain/runtime/go1.26.4.patch",
		"toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadcap.go",
		"toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadguard.go",
		"toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go",
		"toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go",
		"toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go",
		"toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go",
		"choice/tape.go",
		"choice/trace.go",
		"toolchain/runtime/overlay/src/runtime/gomad.go",
		"deterministicio/boundary/manifest.json",
		"deterministicio/boundary_generated.go",
		"choice/schema/choicewire.go.tmpl",
		"choice/schema/choicewire.json",
		"choice/schema/choicewire_runtime.go.tmpl",
		"choice/schema/choicewire_test.go.tmpl",
		"deterministicio/schema/iowire.go.tmpl",
		"deterministicio/schema/iowire.json",
		"deterministicio/schema/iowire_test.go.tmpl",
		"simulation/schema/modelwire.go.tmpl",
		"simulation/schema/modelwire.json",
		"simulation/schema/modelwire_test.go.tmpl",
		"simulation/schema/modeltransport.go.tmpl",
		"target/internal/livecap/livecap.go",
		"target/internal/livecap/livecap.go.tmpl",
		"target/internal/livecap/livecap.json",
		"target/internal/livecap/project.go",
	} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "..", relative))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := GenerateProtocols(root, false); err != nil {
		t.Fatal(err)
	}
	if err := GenerateProtocols(root, true); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{
		"choice/internal/wire/wire_generated.go",
		"choice/internal/wire/wire_generated_test.go",
		"deterministicio/internal/wire/wire_generated.go",
		"deterministicio/internal/wire/wire_generated_test.go",
		"toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go",
		"toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated_test.go",
		"toolchain/runtime/overlay/src/runtime/gomad_choicewire_generated.go",
		"toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go",
		"toolchain/runtime/overlay/src/internal/gomadwire/wire_generated_test.go",
		"toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go",
		"toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated_test.go",
		"runner/internal/execution/simulation_model_wire_generated.go",
		"toolchain/runtime/overlay/src/internal/gomadsim/model_transport_generated.go",
		"target/internal/livecap/protocol_generated.go",
		"toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("generated endpoint %q: %v", relative, err)
		}
	}
	for _, relative := range []string{"choice/internal/wire/wire_generated.go", "deterministicio/internal/wire/wire_generated.go", "toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go", "target/internal/livecap/protocol_generated.go"} {
		stale := filepath.Join(root, filepath.FromSlash(relative))
		current, err := os.ReadFile(stale)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stale, append(current, []byte("stale")...), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := GenerateProtocols(root, true); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("GenerateProtocols(check) after changing %q error = %v", relative, err)
		}
		if err := GenerateProtocols(root, false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadSchemaRejectsTrailingData(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "deterministicio", "schema", "iowire.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "iowire.json")
	contents = append(contents, []byte("\n{}\n")...)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSchema(path); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("readSchema() error = %v", err)
	}
}
