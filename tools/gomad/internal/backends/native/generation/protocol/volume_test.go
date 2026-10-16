package protocol

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVolumeGeneratedConsumersAreCurrent(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	if err := generateVolumeProtocols(root, true); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(root, "internal", "model", "cluster", "internal", "wire", "volume.go"),
		filepath.Join(root, "internal", "backends", "native", "toolchain", "runtime", "overlay", "src", "internal", "gomadfs", "simulation_volume_wire.go"),
	} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(contents) < 18 || string(contents[:18]) != "// Code generated " {
			t.Fatalf("volume semantic consumer is not generated: %s", path)
		}
	}
}

func TestVolumeSemanticCrossConsumerVectors(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	dir := t.TempDir()
	for _, consumer := range []struct{ path, name string }{
		{filepath.Join(root, "internal", "model", "cluster", "internal", "wire", "volume.go"), "host.go"},
		{filepath.Join(root, "internal", "backends", "native", "toolchain", "runtime", "overlay", "src", "internal", "gomadfs", "simulation_volume_wire.go"), "runtime.go"},
	} {
		fileSet := token.NewFileSet()
		sourceBytes, err := os.ReadFile(consumer.path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fileSet, consumer.path, clusterWireFixtureSource(t, sourceBytes), 0)
		if err != nil {
			t.Fatal(err)
		}
		file.Name.Name = "volume"
		var declarations []ast.Decl
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "encodeRuntimeVolumeConfig" {
				continue
			}
			declarations = append(declarations, declaration)
		}
		file.Decls = declarations
		var source bytes.Buffer
		if err := format.Node(&source, fileSet, file); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, consumer.name), source.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	types := volumeWireDomainTypes(t, root)
	for name, source := range map[string][]byte{
		"go.mod":       []byte("module volume\n\ngo 1.27.1\n"),
		"types.go":     types,
		"wire_test.go": []byte(volumeWireCrossConsumerTests),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	goCommand := os.Getenv("GOMAD_STOCK_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	command := exec.Command(goCommand, "test", "-tags", "test_dep", "-count=1", ".")
	command.Dir = dir
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("volume cross-consumer vectors: %v\n%s", err, output)
	}
}

func TestVolumeSchemaRejectsIncompleteAndAmbiguousLayouts(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "internal", "model", "schema", "volumewire.json")
	definition, err := readVolumeWireSchema(path)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		change func(*volumeWireSchema)
	}{
		{"version", func(value *volumeWireSchema) { value.Version++ }},
		{"magic", func(value *volumeWireSchema) { value.Magic["config"] = value.Magic["finish"] }},
		{"limit", func(value *volumeWireSchema) { delete(value.Limits, "operations") }},
		{"layout", func(value *volumeWireSchema) { value.Layouts = value.Layouts[1:] }},
		{"duplicate", func(value *volumeWireSchema) { value.Layouts = append(value.Layouts, value.Layouts[0]) }},
		{"empty", func(value *volumeWireSchema) { value.Layouts[0].Fields = nil }},
		{"field", func(value *volumeWireSchema) {
			value.Layouts[0].Fields = append(value.Layouts[0].Fields, value.Layouts[0].Fields[0])
		}},
		{"kind", func(value *volumeWireSchema) { value.Layouts[0].Fields[0].Kind = "unknown" }},
		{"type", func(value *volumeWireSchema) { value.Layouts[0].Fields[3].Type = "unknown" }},
		{"field-limit", func(value *volumeWireSchema) { value.Layouts[0].Fields[3].Limit = "unknown" }},
	}
	original, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			var changed volumeWireSchema
			if err := json.Unmarshal(original, &changed); err != nil {
				t.Fatal(err)
			}
			mutation.change(&changed)
			contents, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "volumewire.json")
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readVolumeWireSchema(path); err == nil {
				t.Fatal("accepted malformed semantic schema")
			}
		})
	}
}

func volumeWireDomainTypes(t *testing.T, root string) []byte {
	t.Helper()
	paths := []string{
		filepath.Join(root, "internal", "model", "cluster", "internal", "history", "types.go"),
	}
	overlay := filepath.Join(root, "internal", "backends", "native", "toolchain", "runtime", "overlay", "src", "internal", "gomadfs")
	for _, name := range []string{"simulation_volume.go", "volume.go", "fs.go"} {
		paths = append(paths, filepath.Join(overlay, name))
	}
	paths = append(paths, filepath.Join(overlay, "..", "gomadwire", "wire_generated.go"))
	wanted := strings.Fields("NodeID VolumeID VolumeTransitionKind VolumeOutcome NodeHandle VolumeRecord VolumeTransition VolumeSnapshot VolumeStateSnapshot VolumeEntrySnapshot VolumeCrashFrontier VolumeCrashEntry VolumeCrashState VolumeCrashEnumeration VolumeCrashCapacity runtimeVolumeError simulationVolumeConfig simulationVolumeLimits simulationVolumeNodeConfig simulationVolumeHandle simulationVolumeTransition simulationVolumeStateSnapshot simulationVolumeSnapshot simulationVolumeRecord simulationVolumeBridgeError VolumeConfig CrashCapacity CrashExploration CrashEntry CrashState CrashEnumeration SnapshotEntry Kind")
	selected := make(map[string]bool)
	selected["MountKind"] = true
	for _, name := range wanted {
		selected[name] = true
	}
	fileSet := token.NewFileSet()
	var source bytes.Buffer
	source.WriteString("package volume\nimport \"strconv\"\n")
	for _, path := range paths {
		sourceBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fileSet, path, clusterWireFixtureSource(t, sourceBytes), 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			if group, ok := declaration.(*ast.GenDecl); ok && group.Tok == token.TYPE {
				for _, spec := range group.Specs {
					value := spec.(*ast.TypeSpec)
					if !selected[value.Name.Name] {
						continue
					}
					delete(selected, value.Name.Name)
					if value.Name.Name == "Kind" {
						value.Type = ast.NewIdent("MountKind")
					}
					if err := format.Node(&source, fileSet, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{value}}); err != nil {
						t.Fatal(err)
					}
					source.WriteByte('\n')
				}
			}
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "Error" && function.Recv != nil {
				receiver, ok := function.Recv.List[0].Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				name, ok := receiver.X.(*ast.Ident)
				if !ok || name.Name != "simulationVolumeBridgeError" {
					continue
				}
				if err := format.Node(&source, fileSet, function); err != nil {
					t.Fatal(err)
				}
				source.WriteByte('\n')
			}
		}
	}
	if len(selected) != 0 {
		t.Fatalf("volume domain types not found: %v", selected)
	}
	return source.Bytes()
}

const volumeWireCrossConsumerTests = `package volume
import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestVolumeCrossConsumerConfigAndEmptyVectors(t *testing.T) {
	config := runtimeVolumeWireConfig{Seed:7}
	config.Limits.Operations, config.Limits.Transitions = 9, 11
	config.Nodes = []runtimeVolumeWireNode{{Node:"n", Volumes:[]runtimeVolumeWireMount{{ID:"v", Path:"/v", CapacityBytes:19}}}}
	writer := &runtimeVolumeWireEncoder{data:[]byte(runtimeVolumeConfigMagic)}
	encodeRuntimeVolumeConfigBody(writer, config)
	decoded, err := decodeSimulationVolumeConfig(writer.data)
	want := simulationVolumeConfig{Seed:7, Limits:simulationVolumeLimits{Operations:9, Transitions:11}, Nodes:[]simulationVolumeNodeConfig{{Node:"n", Volumes:[]VolumeConfig{{ID:"v", Path:"/v", CapacityBytes:19}}}}}
	if writer.err != nil || err != nil || !reflect.DeepEqual(decoded, want) { t.Fatalf("config = %+v, %v / %v", decoded, err, writer.err) }
	for _, vector := range []struct { name, hex string; got []byte }{
		{"config", "474f4d414456433100000000000000000000000000000000000000000000000000000000000000000000000000000000", nil},
		{"finish", "474f4d414456523100000000000000000000000000000000000000000000000000000000000000000000000000000000", nil},
		{"enumeration", "474f4d41445651310000000000000000000000000000000000000000000000000000000000000000", nil},
	} {
		var encoded []byte
		switch vector.name {
		case "config": w := &runtimeVolumeWireEncoder{data:[]byte(runtimeVolumeConfigMagic)}; encodeRuntimeVolumeConfigBody(w, runtimeVolumeWireConfig{}); encoded = w.data
		case "finish": encoded, err = encodeSimulationVolumeFinish(simulationVolumeRecord{}, nil)
		case "enumeration": encoded, err = encodeSimulationCrashEnumeration(CrashEnumeration{})
		}
		if err != nil || hex.EncodeToString(encoded) != vector.hex { t.Fatalf("%s vector = %x, %v", vector.name, encoded, err) }
	}
}

func TestVolumeCrossConsumerRecordAndTypedError(t *testing.T) {
	transition := simulationVolumeTransition{
		Ordinal:3, Kind:"write", Handle:simulationVolumeHandle{Node:"n", Incarnation:2}, Volume:"v", Operation:7,
		Dependencies:[]uint64{4,6}, SelectedOperations:[]uint64{}, Path:"/x", Destination:"/y",
		Inode:9, Offset:2, Bytes:3, PayloadSHA256:"payload", EffectSHA256:"effect", Outcome:"capacity",
	}
	snapshot := simulationVolumeSnapshot{
		Volumes:[]simulationVolumeStateSnapshot{{Volume:"v", Node:"n", Mount:"/v", CapacityBytes:1024,
			Persisted:[]SnapshotEntry{{Path:"/", Mode:493, Kind:"directory", ModTime:-7}},
			Volatile:[]SnapshotEntry{{Path:"/x", Mode:384, Kind:"file", ModTime:17, Size:3, DataSHA256:"data"}},
			PendingOperations:1, PendingSHA256:"pending", NextOperation:8, Identity:"volume"}},
		TransitionSHA256:"transitions", Identity:"snapshot",
	}
	record := simulationVolumeRecord{Transitions:[]simulationVolumeTransition{transition}, Snapshot:snapshot}
	bridge := &simulationVolumeBridgeError{Kind:"replay_divergence", Message:"diverged", Ordinal:3, ExpectedSHA256:"expected", ActualSHA256:"actual", Expected:&transition, Actual:&transition}
	encoded, err := encodeSimulationVolumeFinish(record, bridge)
	got, gotErr, decodeErr := decodeRuntimeVolumeFinish(encoded)
	if err != nil || decodeErr != nil || gotErr == nil || gotErr.Message != "diverged" || gotErr.Kind != "replay_divergence" || gotErr.Ordinal != 3 ||
		gotErr.ExpectedSHA256 != "expected" || gotErr.ActualSHA256 != "actual" || gotErr.Expected == nil || gotErr.Actual == nil {
		t.Fatalf("finish error = %+v, %v / %v", gotErr, err, decodeErr)
	}
	w := &runtimeVolumeWireEncoder{}
	encodeRuntimeVolumeRecord(w, got)
	d := newSimulationVolumeWireDecoder(w.data, "")
	roundTrip := decodeSimulationVolumeRecord(d)
	if err := d.finish(); err != nil || !reflect.DeepEqual(roundTrip, record) { t.Fatalf("record = %+v, %v", roundTrip, err) }
	if got.Transitions[0].Bytes != 3 || got.Transitions[0].Outcome != "capacity" || got.Snapshot.Volumes[0].Persisted[0].ModTime != -7 ||
		!reflect.DeepEqual(*gotErr.Expected, got.Transitions[0]) || !reflect.DeepEqual(*gotErr.Actual, got.Transitions[0]) { t.Fatalf("typed partial result = %+v / %+v", got, gotErr) }
	replayConfig := runtimeVolumeWireConfig{Replay:&got}
	w = &runtimeVolumeWireEncoder{data:[]byte(runtimeVolumeConfigMagic)}
	encodeRuntimeVolumeConfigBody(w, replayConfig)
	config, err := decodeSimulationVolumeConfig(w.data)
	if err != nil || config.Replay == nil || !reflect.DeepEqual(*config.Replay, record) { t.Fatalf("replay = %+v, %v", config, err) }
	errorWire := encodeSimulationVolumeBridgeError(bridge)
	errorValue, err := decodeRuntimeVolumeErrorWire(errorWire)
	if err != nil || !reflect.DeepEqual(errorValue, *gotErr) { t.Fatalf("standalone error = %+v, %v", errorValue, err) }
	fallback := encodeSimulationVolumeBridgeError(&simulationVolumeBridgeError{Message:strings.Repeat("x", maximumSimulationVolumeWireStringBytes+1)})
	errorValue, err = decodeRuntimeVolumeErrorWire(fallback)
	if err != nil || errorValue.Kind != "runtime" || errorValue.Message != "encode simulation volume error" { t.Fatalf("fallback = %+v, %v", errorValue, err) }
}

func TestVolumeCrossConsumerCrashEnumerationAndFrontier(t *testing.T) {
	frontier := VolumeCrashFrontier{Volume:"v", PendingSHA256:"pending", Cursor:[]byte{0,1}, Seen:[]string{"seen"}, Identity:"frontier"}
	encoded, err := encodeRuntimeVolumeFrontier(&frontier)
	exploration, decodeErr := decodeSimulationCrashExploration(encoded)
	wantExploration := &CrashExploration{Volume:"v", PendingSHA256:"pending", Cursor:[]byte{0,1}, Seen:[]string{"seen"}, Identity:"frontier"}
	if err != nil || decodeErr != nil || !reflect.DeepEqual(exploration, wantExploration) { t.Fatalf("frontier = %+v, %v / %v", exploration, err, decodeErr) }
	page := CrashEnumeration{States:[]CrashState{{Volume:"v", PendingSHA256:"pending", SelectedOperations:[]uint64{2,4}, Entries:[]CrashEntry{
		{Path:"/x", Mode:384, Kind:Kind(1), ModTime:-9, Data:[]byte{0,255,7}},
		{Path:"/d", Mode:493, Kind:Kind(2), ModTime:11}},
		Identity:"state"}}, ChoiceExploration:exploration, Complete:false, Capacity:"states"}
	encoded, err = encodeSimulationCrashEnumeration(page)
	got, decodeErr := decodeRuntimeVolumeEnumeration(encoded)
	want := VolumeCrashEnumeration{States:[]VolumeCrashState{{Volume:"v", PendingSHA256:"pending", SelectedOperations:[]uint64{2,4},
		Entries:[]VolumeCrashEntry{{Path:"/x", Mode:384, Kind:"file", ModTime:-9, Data:[]byte{0,255,7}}, {Path:"/d", Mode:493, Kind:"directory", ModTime:11}},
		Identity:"state"}}, Frontier:&frontier, Complete:false, Capacity:"states"}
	if err != nil || decodeErr != nil || !reflect.DeepEqual(got, want) { t.Fatalf("page = %+v, %v / %v", got, err, decodeErr) }
	w := &runtimeVolumeWireEncoder{data:[]byte(runtimeVolumeEnumerationMagic)}
	encodeRuntimeVolumeEnumerationBody(w, got)
	if w.err != nil || !bytes.Equal(w.data, encoded) { t.Fatalf("enumeration round trip = %x, %v", w.data, w.err) }
}

func TestVolumeCrossConsumerRejectsMalformedFrames(t *testing.T) {
	configWriter := &runtimeVolumeWireEncoder{data:[]byte(runtimeVolumeConfigMagic)}
	encodeRuntimeVolumeConfigBody(configWriter, runtimeVolumeWireConfig{})
	finish, _ := encodeSimulationVolumeFinish(simulationVolumeRecord{}, nil)
	enumeration, _ := encodeSimulationCrashEnumeration(CrashEnumeration{})
	frontier, _ := encodeRuntimeVolumeFrontier(&VolumeCrashFrontier{})
	errorWire := encodeSimulationVolumeBridgeError(&simulationVolumeBridgeError{Message:"bad"})
	for _, frame := range []struct {name string; data []byte; decode func([]byte) error}{
		{"config", configWriter.data, func(b []byte) error { _, err := decodeSimulationVolumeConfig(b); return err }},
		{"finish", finish, func(b []byte) error { _, _, err := decodeRuntimeVolumeFinish(b); return err }},
		{"enumeration", enumeration, func(b []byte) error { _, err := decodeRuntimeVolumeEnumeration(b); return err }},
		{"frontier", frontier, func(b []byte) error { _, err := decodeSimulationCrashExploration(b); return err }},
		{"error", errorWire, func(b []byte) error { _, err := decodeRuntimeVolumeErrorWire(b); return err }},
	} {
		for n:=1; n<len(frame.data); n++ { if err := frame.decode(frame.data[:n]); err == nil { t.Fatalf("%s accepted truncation %d", frame.name, n) } }
		if err := frame.decode(append(bytes.Clone(frame.data), 0)); err == nil { t.Fatalf("%s accepted trailing data", frame.name) }
		changed := bytes.Clone(frame.data); changed[0] ^= 1
		if err := frame.decode(changed); err == nil { t.Fatalf("%s accepted invalid magic", frame.name) }
	}
	for _, maximum := range []uint64{65, ^uint64(0)} {
		changed := bytes.Clone(configWriter.data); binary.LittleEndian.PutUint64(changed[32:40], maximum)
		if _, err := decodeSimulationVolumeConfig(changed); err == nil { t.Fatal("accepted oversized node count") }
	}
	changed := bytes.Clone(finish); binary.LittleEndian.PutUint64(changed[len(changed)-8:], 2)
	if _, _, err := decodeRuntimeVolumeFinish(changed); err == nil { t.Fatal("accepted invalid boolean") }
	changed = bytes.Clone(enumeration); binary.LittleEndian.PutUint64(changed[8:16], ^uint64(0))
	if _, err := decodeRuntimeVolumeEnumeration(changed); err == nil { t.Fatal("accepted oversized state count") }
	w := &runtimeVolumeWireEncoder{data:[]byte(runtimeVolumeErrorMagic)}; encodeRuntimeVolumeErrorBody(w, runtimeVolumeError{})
	if _, err := decodeRuntimeVolumeErrorWire(w.data); err == nil { t.Fatal("accepted empty standalone error") }
	if _, err := encodeRuntimeVolumeFrontier(&VolumeCrashFrontier{Cursor:make([]byte, 1<<20+1)}); err == nil { t.Fatal("accepted oversized cursor") }
}
`
