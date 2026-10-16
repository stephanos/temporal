package protocol

import (
	"bytes"
	"encoding/json"
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
	definition, err := readSimulationModelSchema(filepath.Join("..", "..", "..", "..", "..", "internal", "model", "schema", "modelwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 1 || definition.Profile != "gomad.simulation-model/v1" || definition.TransportMagic != "GOMADPM\x03" || definition.Limits.FrameBytes != 128<<20 || definition.Limits.StringBytes != 4096 || definition.Limits.DataBytes != 64<<20 || definition.Limits.Entries != 100_000 || definition.Limits.NodeBytes != 256 || definition.Limits.ErrorBytes != 4096 {
		t.Fatalf("simulation model protocol = %+v", definition)
	}
	if definition.Models.Network != 1 || definition.Models.Volume != 2 || definition.NetworkOperations.Listen != 1 || definition.NetworkOperations.ConnSetWriteDeadline != 13 || definition.VolumeOperations.Resolve != 1 || definition.VolumeOperations.MappingClose != 28 {
		t.Fatalf("simulation model operation vocabulary = network %+v volume %+v", definition.NetworkOperations, definition.VolumeOperations)
	}
}

func TestReadSimulationModelSchemaRequiresDeclaredStallTimeout(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "internal", "model", "schema", "modelwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(contents, &document); err != nil {
		t.Fatal(err)
	}
	var codes map[string]uint16
	if err := json.Unmarshal(document["errors"], &codes); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "modelwire.json")
	write := func() {
		t.Helper()
		encoded, err := json.Marshal(codes)
		if err != nil {
			t.Fatal(err)
		}
		document["errors"] = encoded
		encoded, err = json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	codes["stall_timeout"] = 22
	write()
	if _, err := readSimulationModelSchema(path); err != nil {
		t.Fatalf("declared stall timeout rejected: %v", err)
	}
	for _, invalid := range []uint16{0, 21, 23} {
		codes["stall_timeout"] = invalid
		write()
		if _, err := readSimulationModelSchema(path); err == nil {
			t.Fatalf("invalid code accepted: %d", invalid)
		}
	}
	delete(codes, "stall_timeout")
	write()
	if _, err := readSimulationModelSchema(path); err == nil {
		t.Fatal("omitted stall timeout accepted")
	}
}

func TestReadWorldSnapshotSchemaPinsBoundedTransitionVocabulary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshotwire.json")
	contents := []byte(`{
  "version": 1,
  "profile": "gomad.world-compressed-snapshot/v1",
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
	if definition.Version != 1 || definition.Profile != "gomad.world-compressed-snapshot/v1" || definition.Magic != "GMWSNAP1" || definition.Limits.Depth != 4 || definition.Limits.EnumBytes != 32 {
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
	root := filepath.Join("..", "..", "..", "..", "..")
	contents, err := os.ReadFile(filepath.Join(root, "internal", "model", "world", "schema", "snapshotwire.json"))
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
	generated, err := generate(filepath.Join(root, "internal", "model", "world", "schema", "snapshotwire.go.tmpl"), worldSnapshotTemplateData{Package: "world", Schema: definition})
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

func TestRustWorldSnapshotWriterFollowsSchemaFieldAndTagOrder(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	contents, err := os.ReadFile(filepath.Join(root, "internal", "model", "world", "schema", "snapshotwire.json"))
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
	definition.Transitions[0].Tag, definition.Transitions[1].Tag = 7, 9
	generated, err := generateUnformatted(filepath.Join(root, "internal", "model", "world", "schema", "snapshotwire.rs.tmpl"), worldSnapshotTemplateData{Package: "world", Schema: definition})
	if err != nil {
		t.Fatal(err)
	}
	adapter := bytes.Index(generated, []byte("self.text(&request.resource.adapter)?;"))
	kind := bytes.Index(generated, []byte("self.text(&request.kind)?;"))
	if adapter < 0 || kind < 0 || adapter >= kind {
		t.Fatalf("generated Rust request field order did not follow schema: adapter=%d kind=%d", adapter, kind)
	}
	ready := bytes.Index(generated, []byte("TransitionBody::Ready {"))
	register := bytes.Index(generated, []byte("TransitionBody::Register {"))
	if ready < 0 || register < 0 || ready >= register {
		t.Fatalf("generated Rust transition order did not follow schema: ready=%d register=%d", ready, register)
	}
	if !bytes.Contains(generated, []byte("self.uint8(7)?;")) || !bytes.Contains(generated, []byte("self.uint8(9)?;")) || !bytes.Contains(generated, []byte("[0x47, 0x4d, 0x57, 0x53, 0x4e, 0x41, 0x50, 0x31]")) {
		t.Fatal("generated Rust writer did not take tags or magic from the schema")
	}
}

func TestReadLiveCapabilitySchemaPinsClosedProtocol(t *testing.T) {
	definition, err := readLiveCapabilitySchema(filepath.Join("..", "..", "..", "..", "..", "internal", "backends", "native", "build", "internal", "livecap", "livecap.json"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Version != 2 || definition.Schema != "gomad.live-capability-manifest/v2" || definition.Symbol != "runtime.gomadCapabilities" || definition.GuardSymbol != "runtime.gomadCapabilityGuard" {
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
		Version: 1, Schema: "gomad.live-capability-manifest/v1",
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
		"internal/model/schema/processwire.json",
		"internal/model/schema/processwire.go.tmpl",
		"internal/model/schema/processwire_test.go.tmpl",
		"internal/model/schema/processwire_transport.go.tmpl",
		"internal/model/schema/processwire_transport_test.go.tmpl",
		"internal/model/schema/networkwire.json",
		"internal/model/schema/networkwire.go.tmpl",
		"internal/model/schema/volumewire.json",
		"internal/model/schema/volumewire.go.tmpl",
		"internal/backends/native/toolchain/runtime/go1.27.1.patch",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadcap.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadguard.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go",
		"internal/model/choice/tape.go",
		"internal/model/choice/trace.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go",
		"internal/model/runtimesource/gomad_semantics.go.txt",
		"internal/backends/native/transcript/boundary/manifest.json",
		"internal/backends/native/transcript/boundary_generated.go",
		"internal/model/choice/schema/choicewire.go.tmpl",
		"internal/model/choice/schema/choicewire.json",
		"internal/model/choice/schema/choicewire_runtime.go.tmpl",
		"internal/model/choice/schema/choicewire_test.go.tmpl",
		"internal/backends/native/transcript/schema/iowire.go.tmpl",
		"internal/backends/native/transcript/schema/iowire.json",
		"internal/backends/native/transcript/schema/iowire_runtime.go.tmpl",
		"internal/backends/native/transcript/schema/iowire_test.go.tmpl",
		"internal/model/schema/modelwire.go.tmpl",
		"internal/model/schema/modelwire.json",
		"internal/model/schema/netstream_status.go.tmpl",
		"internal/model/schema/modelwire_test.go.tmpl",
		"internal/model/schema/modeltransport.go.tmpl",
		"internal/model/schema/timewire.json",
		"internal/model/schema/timewire_host.go.tmpl",
		"internal/model/schema/timewire_host_test.go.tmpl",
		"internal/model/schema/timewire_runtime.go.tmpl",
		"internal/model/schema/timewire_runtime_export_test.go.tmpl",
		"internal/model/schema/timewire_runtime_test.go.tmpl",
		"internal/model/world/schema/snapshotwire.go.tmpl",
		"internal/model/world/schema/snapshotwire.rs.tmpl",
		"internal/model/world/schema/snapshotwire.json",
		"internal/backends/native/build/internal/livecap/livecap.go",
		"internal/backends/native/build/internal/livecap/livecap.go.tmpl",
		"internal/backends/native/build/internal/livecap/livecap.json",
		"internal/backends/native/build/internal/livecap/project.go",
	} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", relative))
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
		"internal/model/network/stream_status_generated.go",
		"internal/backends/wasm/toolchain/net_stream_status_generated.go.txt",
		"internal/model/cluster/internal/wire/process.go",
		"internal/model/cluster/internal/wire/process_generated_test.go",
		"internal/model/cluster/internal/wire/network.go",
		"internal/model/cluster/internal/wire/volume.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/simulation_volume_wire.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated_test.go",
		"internal/model/choice/internal/wire/wire_generated.go",
		"internal/model/choice/internal/wire/wire_generated_test.go",
		"internal/model/iowire/wire_generated.go",
		"internal/model/iowire/wire_generated_test.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated_test.go",
		"internal/model/runtimesource/gomad_choicewire_generated.go.txt",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad_iowire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated_test.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated_test.go",
		"internal/backends/native/execution/simulation_model_wire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadsim/model_transport_generated.go",
		"internal/backends/native/execution/simulation_time_wire_generated.go",
		"internal/backends/native/execution/simulation_time_wire_generated_test.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_export_generated_test.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated_test.go",
		"internal/backends/native/build/internal/livecap/protocol_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go",
		"internal/model/world/compressed_wire_generated.go",
		"internal/backends/wasm/wasmhost/src/world/wire_generated.rs",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("generated endpoint %q: %v", relative, err)
		}
	}
	for _, relative := range []string{
		"internal/model/network/stream_status_generated.go",
		"internal/backends/wasm/toolchain/net_stream_status_generated.go.txt",
		"internal/model/cluster/internal/wire/process.go",
		"internal/model/cluster/internal/wire/process_generated_test.go",
		"internal/model/cluster/internal/wire/network.go",
		"internal/model/cluster/internal/wire/volume.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/simulation_volume_wire.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated_test.go",
		"internal/model/choice/internal/wire/wire_generated.go",
		"internal/model/iowire/wire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go",
		"internal/backends/native/execution/simulation_time_wire_generated.go",
		"internal/backends/native/execution/simulation_time_wire_generated_test.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_export_generated_test.go",
		"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad_timewire_generated_test.go",
		"internal/backends/native/build/internal/livecap/protocol_generated.go",
		"internal/model/world/compressed_wire_generated.go",
		"internal/backends/wasm/wasmhost/src/world/wire_generated.rs",
	} {
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
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "internal/backends/native/transcript", "schema", "iowire.json"))
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
