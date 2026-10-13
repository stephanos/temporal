package protocol

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
	definition, err := readSimulationModelSchema(filepath.Join("..", "..", "..", "..", "simulation", "schema", "modelwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 1 || definition.Profile != "gomad3.simulation-model/v1" || definition.TransportMagic != "GOMADPM\x03" || definition.Limits.FrameBytes != 128<<20 || definition.Limits.StringBytes != 4096 || definition.Limits.DataBytes != 64<<20 || definition.Limits.Entries != 100_000 || definition.Limits.NodeBytes != 256 || definition.Limits.ErrorBytes != 4096 {
		t.Fatalf("simulation model protocol = %+v", definition)
	}
	if definition.Models.Network != 1 || definition.Models.Volume != 2 || definition.NetworkOperations.Listen != 1 || definition.NetworkOperations.ConnSetWriteDeadline != 13 || definition.VolumeOperations.Resolve != 1 || definition.VolumeOperations.MappingClose != 28 {
		t.Fatalf("simulation model operation vocabulary = network %+v volume %+v", definition.NetworkOperations, definition.VolumeOperations)
	}
}

func TestReadWorldSnapshotSchemaPinsBoundedTransitionVocabulary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshotwire.json")
	contents := []byte(`{
  "version": 1,
  "profile": "gomad3.world-compressed-snapshot/v1",
  "magic": "GMWSNAP1",
  "limits": {"depth": 4, "enum_bytes": 32},
  "layouts": {
    "snapshot": ["schema-version", "config.seed", "config.max-requests", "config.max-events", "config.max-queued-events", "config.max-transitions", "config.max-payload-bytes", "config.max-string-bytes", "now", "next-request-id", "next-event-id", "next-transition", "payload-bytes", "replay.cursor", "replay.expected", "transcript-digest", "state-digest"],
    "request": ["kind", "resource.adapter", "resource.kind", "resource.key", "priority", "payload"],
    "readiness": ["request-id", "at", "kind", "payload", "equivalence-class"],
    "transition": ["sequence", "previous-digest", "digest"],
    "quiescence": ["kind", "before", "after", "deliveries", "blocked"],
    "delivery": ["request-id", "event-id", "at", "kind", "payload"]
  },
  "transitions": [
    {"kind": "register", "tag": 1, "fields": ["request", "request-id"]},
    {"kind": "ready", "tag": 2, "fields": ["readiness", "event-id"]},
    {"kind": "cancel", "tag": 3, "fields": ["request-id", "result.request-id", "result.event-id", "status"]},
    {"kind": "quiesce", "tag": 4, "fields": ["result"]}
  ]
}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	definition, err := readWorldSnapshotSchema(path)
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 1 || definition.Profile != "gomad3.world-compressed-snapshot/v1" || definition.Magic != "GMWSNAP1" || definition.Limits.Depth != 4 || definition.Limits.EnumBytes != 32 {
		t.Fatalf("world snapshot wire identity = %+v", definition)
	}
	if strings.Join(definition.Layouts.Snapshot, ",") != "schema-version,config.seed,config.max-requests,config.max-events,config.max-queued-events,config.max-transitions,config.max-payload-bytes,config.max-string-bytes,now,next-request-id,next-event-id,next-transition,payload-bytes,replay.cursor,replay.expected,transcript-digest,state-digest" ||
		strings.Join(definition.Layouts.Request, ",") != "kind,resource.adapter,resource.kind,resource.key,priority,payload" ||
		strings.Join(definition.Layouts.Readiness, ",") != "request-id,at,kind,payload,equivalence-class" ||
		strings.Join(definition.Layouts.Transition, ",") != "sequence,previous-digest,digest" ||
		strings.Join(definition.Layouts.Quiescence, ",") != "kind,before,after,deliveries,blocked" ||
		strings.Join(definition.Layouts.Delivery, ",") != "request-id,event-id,at,kind,payload" {
		t.Fatalf("world snapshot field layouts = %+v", definition.Layouts)
	}
	if len(definition.Transitions) != 4 || definition.Transitions[0].Kind != "register" || definition.Transitions[0].Tag != 1 || definition.Transitions[1].Kind != "ready" || definition.Transitions[1].Tag != 2 || definition.Transitions[2].Kind != "cancel" || definition.Transitions[2].Tag != 3 || definition.Transitions[3].Kind != "quiesce" || definition.Transitions[3].Tag != 4 {
		t.Fatalf("world snapshot transition vocabulary = %+v", definition.Transitions)
	}
}

func TestWorldSnapshotTemplateFollowsSchemaFieldAndTagOrder(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	contents, err := os.ReadFile(filepath.Join(root, "world", "schema", "snapshotwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	contents = bytes.Replace(contents,
		[]byte(`"request": ["kind", "resource.adapter"`),
		[]byte(`"request": ["resource.adapter", "kind"`), 1)
	path := filepath.Join(t.TempDir(), "snapshotwire.json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	definition, err := readWorldSnapshotSchema(path)
	if err != nil {
		t.Fatal(err)
	}
	definition.Transitions[0], definition.Transitions[1] = definition.Transitions[1], definition.Transitions[0]
	generated, err := generate(filepath.Join(root, "world", "schema", "snapshotwire.go.tmpl"), worldSnapshotTemplateData{Package: "world", Schema: definition})
	if err != nil {
		t.Fatal(err)
	}
	adapter := bytes.Index(generated, []byte("w.text(request.Resource.Adapter)"))
	kind := bytes.Index(generated, []byte("w.text(request.Kind)"))
	if adapter < 0 || kind < 0 || adapter >= kind {
		t.Fatalf("generated request field order did not follow schema: adapter=%d kind=%d", adapter, kind)
	}
	ready := bytes.Index(generated, []byte(`case "ready":`))
	register := bytes.Index(generated, []byte(`case "register":`))
	if ready < 0 || register < 0 || ready >= register {
		t.Fatalf("generated transition order did not follow schema: ready=%d register=%d", ready, register)
	}
}

func TestReadLiveCapabilitySchemaPinsClosedProtocol(t *testing.T) {
	definition, err := readLiveCapabilitySchema(filepath.Join("..", "..", "..", "..", "backends", "native", "build", "internal", "livecap", "livecap.json"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 2 || definition.Schema != "gomad3.live-capability-manifest/v2" || definition.Symbol != "runtime.gomadCapabilities" || definition.GuardSymbol != "runtime.gomadCapabilityGuard" {
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
		Version: 1, Schema: "gomad3.live-capability-manifest/v1",
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
		"backends/native/toolchain/runtime/go1.27.1.patch",
		"backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadcap.go",
		"backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadguard.go",
		"backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go",
		"backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go",
		"backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go",
		"backends/native/toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go",
		"choice/tape.go",
		"choice/trace.go",
		"backends/native/toolchain/runtime/overlay/src/runtime/gomad.go",
		"runtimesemantics/gomad_semantics.go.txt",
		"backends/native/transcript/boundary/manifest.json",
		"backends/native/transcript/boundary_generated.go",
		"choice/schema/choicewire.go.tmpl",
		"choice/schema/choicewire.json",
		"choice/schema/choicewire_runtime.go.tmpl",
		"choice/schema/choicewire_test.go.tmpl",
		"backends/native/transcript/schema/iowire.go.tmpl",
		"backends/native/transcript/schema/iowire.json",
		"backends/native/transcript/schema/iowire_runtime.go.tmpl",
		"backends/native/transcript/schema/iowire_test.go.tmpl",
		"simulation/schema/modelwire.go.tmpl",
		"simulation/schema/modelwire.json",
		"simulation/schema/modelwire_test.go.tmpl",
		"simulation/schema/modeltransport.go.tmpl",
		"simulation/schema/timewire.json",
		"simulation/schema/timewire_host.go.tmpl",
		"simulation/schema/timewire_host_test.go.tmpl",
		"simulation/schema/timewire_runtime.go.tmpl",
		"simulation/schema/timewire_runtime_export_test.go.tmpl",
		"simulation/schema/timewire_runtime_test.go.tmpl",
		"world/schema/snapshotwire.go.tmpl",
		"world/schema/snapshotwire.json",
		"backends/native/build/internal/livecap/livecap.go",
		"backends/native/build/internal/livecap/livecap.go.tmpl",
		"backends/native/build/internal/livecap/livecap.json",
		"backends/native/build/internal/livecap/project.go",
	} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", relative))
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
		"internal/iowire/wire_generated.go",
		"internal/iowire/wire_generated_test.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated_test.go",
		"runtimesemantics/gomad_choicewire_generated.go.txt",
		"backends/native/toolchain/runtime/overlay/src/runtime/gomad_iowire_generated.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated_test.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated_test.go",
		"backends/native/execution/simulation_model_wire_generated.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadsim/model_transport_generated.go",
		"backends/native/execution/simulation_time_wire_generated.go",
		"backends/native/execution/simulation_time_wire_generated_test.go",
		"backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated.go",
		"backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_export_generated_test.go",
		"backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated_test.go",
		"backends/native/build/internal/livecap/protocol_generated.go",
		"backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go",
		"world/compressed_wire_generated.go",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("generated endpoint %q: %v", relative, err)
		}
	}
	for _, relative := range []string{"choice/internal/wire/wire_generated.go", "internal/iowire/wire_generated.go", "backends/native/toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go", "backends/native/execution/simulation_time_wire_generated.go", "backends/native/execution/simulation_time_wire_generated_test.go", "backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated.go", "backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_export_generated_test.go", "backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated_test.go", "backends/native/build/internal/livecap/protocol_generated.go", "world/compressed_wire_generated.go"} {
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
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "backends/native/transcript", "schema", "iowire.json"))
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
