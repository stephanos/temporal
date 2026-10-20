package protocol

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChoiceImplementationIdentityBindsEveryParentInput(t *testing.T) {
	inputs := choiceImplementationInputs{
		Schema:           []byte("schema"),
		CodecTemplate:    []byte("codec"),
		RuntimeTemplate:  []byte("runtime-codec"),
		TestTemplate:     []byte("codec-test"),
		RuntimeSemantics: []byte("runtime-semantics"),
		HostTrace:        []byte("host-trace"),
		HostTape:         []byte("host-tape"),
	}
	want := choiceImplementationIdentity(inputs)
	if want == ([32]byte{}) {
		t.Fatal("choice implementation identity is empty")
	}
	fields := []*[]byte{&inputs.Schema, &inputs.CodecTemplate, &inputs.RuntimeTemplate, &inputs.TestTemplate, &inputs.RuntimeSemantics, &inputs.HostTrace, &inputs.HostTape}
	if len(fields) != len(choiceImplementationSources) {
		t.Fatalf("identity binds %d inputs, sources list %d", len(fields), len(choiceImplementationSources))
	}
	for index, field := range fields {
		original := *field
		*field = append(bytes.Clone(original), '!')
		if got := choiceImplementationIdentity(inputs); got == want {
			t.Fatalf("identity did not bind input %d (%s)", index, choiceImplementationSources[index])
		}
		*field = original
	}
	// Moving bytes between adjacent inputs must change the identity.
	shifted := inputs
	shifted.Schema, shifted.CodecTemplate = []byte("schemac"), []byte("odec")
	if choiceImplementationIdentity(shifted) == want {
		t.Fatal("identity does not length-prefix its inputs")
	}
}

func TestReadSimulationModelSchemaPinsBoundedOperationVocabulary(t *testing.T) {
	definition, err := readSimulationModelSchema(filepath.Join("..", "..", "..", "..", "internal", "model", "schema", "modelwire.json"))
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
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "internal", "model", "schema", "modelwire.json"))
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
	root := filepath.Join("..", "..", "..", "..")
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
	generated, err := generate(filepath.Join(root, "internal", "model", "world", "schema", "snapshotwire.go.tmpl"), worldSnapshotTemplateData{Package: "world", Schema: definition}, SharedHeader)
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
	root := filepath.Join("..", "..", "..", "..")
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
	generated, err := render(filepath.Join(root, "internal", "model", "world", "schema", "snapshotwire.rs.tmpl"), worldSnapshotTemplateData{Package: "world", Schema: definition}, SharedHeader)
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

func TestReadSchemaRejectsTrailingData(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "internal/model/schema/iowire.json"))
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
