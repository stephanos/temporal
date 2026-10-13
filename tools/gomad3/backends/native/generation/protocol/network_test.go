package protocol

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNetworkSemanticCodecGenerated(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	if err := generateNetworkProtocols(root, true); err != nil {
		t.Fatal(err)
	}
}

func TestNetworkWireConsumers(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	directory := t.TempDir()
	write := func(name string, contents []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", []byte("module networkwiretest\n\ngo 1.27.1\n"))
	for index, path := range []string{
		"../gomad3sim/runtime_network_wire.go",
		"backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go",
	} {
		source, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		parsed.Name.Name = "networkwiretest"
		var output bytes.Buffer
		if err := format.Node(&output, token.NewFileSet(), parsed); err != nil {
			t.Fatal(err)
		}
		write([]string{"host.go", "overlay.go"}[index], output.Bytes())
	}
	hostTypes := map[string][]string{
		"Spec":     {"Nodes", "Links", "Limits", "Replay"},
		"NodeSpec": {"ID", "Address"}, "LinkSpec": nil, "Limits": nil,
		"ReplayPlan": {"Network"},
		"NodeID":     nil, "NetworkTransitionKind": nil, "NetworkOutcome": nil,
		"NetworkEndpoint": nil, "NetworkTransition": nil, "NetworkNodeSnapshot": nil,
		"NetworkLinkSnapshot": nil, "NetworkListenerSnapshot": nil,
		"NetworkConnectionSnapshot": nil, "NetworkDeliverySnapshot": nil,
		"NetworkSnapshot": nil, "NetworkRecord": nil, "runtimeNetworkError": nil,
	}
	write("host_types.go", wireConsumerDeclarations(t, root, []string{
		"../gomad3sim/spec.go", "../gomad3sim/types.go", "../gomad3sim/runtime_network.go", "../gomad3sim/record.go",
	}, hostTypes, map[string]bool{"networkEndpointKey": true}))
	overlayTypes := map[string][]string{
		"simulationNetworkConfig": nil, "simulationLimits": nil, "simulationNodeConfig": nil,
		"simulationLink": nil, "simulationEndpoint": nil, "simulationTransition": nil,
		"simulationNodeSnapshot": nil, "simulationListenerSnapshot": nil,
		"simulationConnectionSnapshot": nil, "simulationDeliverySnapshot": nil,
		"simulationSnapshot": nil, "simulationRecord": nil, "simulationBridgeError": nil,
	}
	write("overlay_types.go", wireConsumerDeclarations(t, root, []string{
		"backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go",
	}, overlayTypes, map[string]bool{"Error": true, "simulationEndpointKey": true}))
	write("wire_test.go", []byte(networkWireConsumerTests))
	goCommand := os.Getenv("GOMAD3_STOCK_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	command := exec.Command(goCommand, "test", "-count=1", "-tags", "test_dep", "-v", ".")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("test actual network consumers: %v\n%s", err, output)
	}
	t.Logf("%s", output)
}

const networkWireConsumerTests = `
package networkwiretest

import (
 "bytes"
 "encoding/hex"
 "encoding/json"
 "errors"
 "math"
 "reflect"
 "strings"
 "testing"
)

func fixtureRecord() NetworkRecord {
 a := NetworkEndpoint{Node: "a", Incarnation: 1, Address: "A", Port: 2}
 b := NetworkEndpoint{Node: "b", Incarnation: 3, Address: "B", Port: 4}
 return NetworkRecord{
  Transitions: []NetworkTransition{{
   Ordinal: 5, Kind: "write", Source: a, Destination: b,
   Connection: 6, Delivery: 7, Bytes: 8, Count: 9, DelayNanos: 10,
   Outcome: "partial", PayloadSHA256: "p",
  }},
  Snapshot: NetworkSnapshot{
   Nodes: []NetworkNodeSnapshot{{Node: "a", Address: "A", LastIncarnation: 11, NextListenerPort: 12, NextClientPort: 13}},
   Links: []NetworkLinkSnapshot{{From: "a", To: "b", Enabled: true, DelayNanos: 14}},
   Listeners: []NetworkListenerSnapshot{{Endpoint: a, Closed: true}},
   Connections: []NetworkConnectionSnapshot{{Identity: 15, Client: a, Server: b, Closed: true, Reset: true}},
   Deliveries: []NetworkDeliverySnapshot{{Identity: 16, Connection: 15, Source: a, Destination: b, Bytes: 17, DelayNanos: 18}},
   NextConnection: 19, NextDelivery: 20, TransitionSHA256: "t", Identity: "s",
  },
 }
}

func toOverlayRecord(t *testing.T, record NetworkRecord) simulationRecord {
 t.Helper()
 encoded, err := json.Marshal(record)
 if err != nil { t.Fatal(err) }
 var result simulationRecord
 if err := json.Unmarshal(encoded, &result); err != nil { t.Fatal(err) }
 return result
}

func TestConfigCrossConsumer(t *testing.T) {
 spec := Spec{
  Nodes: []NodeSpec{{ID: "a", Address: "A"}},
  Links: []LinkSpec{{From: "a", To: "b", Enabled: true, DelayNanos: 14}},
  Limits: Limits{NetworkListeners: 23, NetworkConnections: 24, NetworkDeliveries: 25, NetworkBytes: 26, NetworkTransitions: 27},
 }
 spec.Limits.NetworkConnectionBytes, spec.Limits.NetworkStallNanos = 28, 29
 encoded, err := encodeRuntimeNetworkConfig(spec)
 if err != nil { t.Fatal(err) }
 if hex.EncodeToString(encoded) != "474f4d41444e43320100000000000000010000000000000061010000000000000041010000000000000001000000000000006101000000000000006201000000000000000e000000000000001700000000000000180000000000000019000000000000001a000000000000001c000000000000001d000000000000001b000000000000000000000000000000" { t.Fatalf("config vector: %x", encoded) }
 for _, replay := range []bool{false, true} {
  if replay { spec.Replay = &ReplayPlan{Network: fixtureRecord()} }
  encoded, err := encodeRuntimeNetworkConfig(spec)
  if err != nil { t.Fatal(err) }
  config, err := decodeSimulationConfig(encoded)
  if err != nil { t.Fatal(err) }
  if !reflect.DeepEqual(config.Nodes, []simulationNodeConfig{{Node: "a", Address: "A"}}) ||
   !reflect.DeepEqual(config.Links, []simulationLink{{From: "a", To: "b", Enabled: true, DelayNanos: 14}}) ||
   config.Limits != (simulationLimits{Listeners: 23, Connections: 24, Deliveries: 25, Bytes: 26, ConnectionBytes: 28, StallNanos: 29, Transitions: 27}) {
   t.Fatalf("config = %+v", config)
  }
  if replay {
   want := toOverlayRecord(t, fixtureRecord())
   if config.Replay == nil || !reflect.DeepEqual(*config.Replay, want) { t.Fatalf("replay config = %+v", config.Replay) }
  } else if config.Replay != nil { t.Fatal("unexpected replay") }
 }
}

func TestConfigLimitsAndOldVersionRejected(t *testing.T) {
 base := Spec{Limits: Limits{NetworkConnectionBytes: 1, NetworkStallNanos: 1}}
 encoded, err := encodeRuntimeNetworkConfig(base)
 if err != nil { t.Fatal(err) }
 old := bytes.Clone(encoded)
 old[7] = '1'
 if _, err := decodeSimulationConfig(old); err == nil { t.Fatal("old config header admitted") }
 omitted := append(bytes.Clone(encoded[:56]), encoded[72:]...)
 if _, err := decodeSimulationConfig(omitted); err == nil { t.Fatal("omitted limits admitted") }
 for _, field := range []struct{ offset int; invalid []uint64 }{
  {56, []uint64{0, (1<<30)+1, math.MaxUint64}},
  {64, []uint64{0, math.MaxInt64, uint64(math.MaxInt64)+1, math.MaxUint64}},
 } {
  for _, value := range field.invalid {
   malformed := bytes.Clone(encoded)
   copy(malformed[field.offset:field.offset+8], appendSimulationUint64(nil, value))
   if _, err := decodeSimulationConfig(malformed); err == nil { t.Fatalf("field %d accepted %d", field.offset, value) }
  }
 }
 for _, pair := range [][2]uint64{{1,1},{1<<30,math.MaxInt64-1}} {
  spec := base
  spec.Limits.NetworkConnectionBytes, spec.Limits.NetworkStallNanos = pair[0], pair[1]
  encoded, err := encodeRuntimeNetworkConfig(spec)
  if err != nil { t.Fatal(err) }
  config, err := decodeSimulationConfig(encoded)
  if err != nil || config.Limits.ConnectionBytes != pair[0] || config.Limits.StallNanos != pair[1] { t.Fatalf("boundaries = %+v %v", config, err) }
 }
}

func TestHeldTimeoutAndLegacySemanticsMatch(t *testing.T) {
 source := NetworkEndpoint{Node:"a",Incarnation:1,Address:"A",Port:2}
 destination := NetworkEndpoint{Node:"b",Incarnation:3,Address:"B",Port:4}
 valid := []NetworkTransition{
  {Kind:"write",Source:source,Destination:destination,Connection:6,Bytes:8,Outcome:"held"},
  {Kind:"write",Source:source,Destination:destination,Connection:6,Bytes:8,Outcome:"timeout"},
  {Kind:"reset",Source:source,Destination:destination,Connection:6,Outcome:"timeout"},
  {Kind:"reset",Source:source,Destination:destination,Connection:6,Outcome:"reset"},
  {Kind:"dial",Source:source,Destination:NetworkEndpoint{Node:"b",Address:"B",Port:4},Outcome:"timeout"},
 }
 for _, kind := range []NetworkTransitionKind{"partition","heal","delay","disconnect","reconnect","directional_delay"} {
  valid=append(valid,NetworkTransition{Kind:kind,Source:source,Destination:destination,Outcome:"ok"})
 }
 for _, kind := range []NetworkTransitionKind{"listen","dial","accept","write","deliver","close","listener_close","stop","crash"} {
  for _, outcome := range []NetworkOutcome{"ok","closed","refused","deadline","partition_drop","stale_drop","reset","capacity","unsupported"} {
   valid=append(valid,NetworkTransition{Kind:kind,Source:source,Connection:6,Delivery:7,Outcome:outcome})
  }
 }
 check := func(transition NetworkTransition, want bool) {
  t.Helper()
  encoded, err := json.Marshal(transition)
  if err != nil {t.Fatal(err)}
  var overlay simulationTransition
  if err:=json.Unmarshal(encoded,&overlay);err!=nil {t.Fatal(err)}
  if host, native:=validNetworkTransition(transition),validSimulationTransition(overlay);host!=want||native!=want {t.Fatalf("transition %+v validation host=%v overlay=%v want=%v",transition,host,native,want)}
  if host,native:=networkTransitionLane(transition),simulationTransitionLane(overlay);host!=native {t.Fatalf("transition %+v lane host=%q overlay=%q",transition,host,native)}
 }
 for _, transition:=range valid {check(transition,true)}
 reset:=valid[2]
 for _, mutate:=range []func(*NetworkTransition){
  func(v *NetworkTransition){v.Connection=0},
  func(v *NetworkTransition){v.Destination.Incarnation=0},
  func(v *NetworkTransition){v.Source.Port=65536},
  func(v *NetworkTransition){v.Destination.Port=0},
  func(v *NetworkTransition){v.Destination.Node=v.Source.Node},
  func(v *NetworkTransition){v.Delivery=1},
  func(v *NetworkTransition){v.PayloadSHA256="sha256:"+strings.Repeat("a",64)},
  func(v *NetworkTransition){v.Kind="timeout_reset"},
  func(v *NetworkTransition){v.Outcome="unknown"},
  func(v *NetworkTransition){v.Outcome="held"},
 } {invalid:=reset;mutate(&invalid);check(invalid,false)}
 held:=valid[0];held.Bytes=0;check(held,false)
 dial:=valid[4];dial.Connection=1;check(dial,false)
}

func TestFinishAndIdentityCrossConsumer(t *testing.T) {
 record := fixtureRecord()
 overlay := toOverlayRecord(t, record)
 encoded, err := encodeSimulationFinishResponse(overlay, nil)
 if err != nil { t.Fatal(err) }
 if hex.EncodeToString(encoded) != "474f4d41444e52310100000000000000050000000000000005000000000000007772697465010000000000000061010000000000000001000000000000004102000000000000000100000000000000620300000000000000010000000000000042040000000000000006000000000000000700000000000000080000000000000009000000000000000a0000000000000007000000000000007061727469616c01000000000000007001000000000000000100000000000000610100000000000000410b000000000000000c000000000000000d00000000000000010000000000000001000000000000006101000000000000006201000000000000000e00000000000000010000000000000001000000000000006101000000000000000100000000000000410200000000000000010000000000000001000000000000000f00000000000000010000000000000061010000000000000001000000000000004102000000000000000100000000000000620300000000000000010000000000000042040000000000000001000000000000000100000000000000010000000000000010000000000000000f00000000000000010000000000000061010000000000000001000000000000004102000000000000000100000000000000620300000000000000010000000000000042040000000000000011000000000000001200000000000000130000000000000014000000000000000100000000000000740100000000000000730000000000000000" { t.Fatalf("finish vector: %x", encoded) }
 decoded, bridge, err := decodeRuntimeNetworkFinish(encoded)
 if err != nil || bridge != nil || !reflect.DeepEqual(decoded, record) {
  t.Fatalf("finish = %+v bridge=%+v error=%v", decoded, bridge, err)
 }
 for _, record := range []NetworkRecord{{}, record} {
  overlay := toOverlayRecord(t, record)
  hostTransitions, err := encodeRuntimeNetworkTransitions(record.Transitions)
  if err != nil { t.Fatal(err) }
  overlayTransitions, err := encodeSimulationTransitions(overlay.Transitions)
  if err != nil || !bytes.Equal(hostTransitions, overlayTransitions) { t.Fatalf("transition identity: %v", err) }
  hostSnapshot, err := encodeRuntimeNetworkSnapshotIdentity(record.Snapshot)
  if err != nil { t.Fatal(err) }
  overlaySnapshot, err := encodeSimulationSnapshotIdentity(overlay.Snapshot)
  if err != nil || !bytes.Equal(hostSnapshot, overlaySnapshot) { t.Fatalf("snapshot identity: %v", err) }
 }
}

func TestStructuredErrorCrossConsumer(t *testing.T) {
 record := fixtureRecord()
 overlay := toOverlayRecord(t, record)
 expected, actual := overlay.Transitions[0], overlay.Transitions[0]
 actual.Count = 22
 for _, pointers := range []bool{false, true} {
  bridge := &simulationBridgeError{Kind: "replay", Message: "partial", Ordinal: 21, ExpectedSHA256: "e", ActualSHA256: "f"}
  want := runtimeNetworkError{Kind: "replay", Message: "partial", Ordinal: 21, ExpectedSHA256: "e", ActualSHA256: "f"}
  if pointers {
   bridge.Expected, bridge.Actual = &expected, &actual
   hostExpected, hostActual := record.Transitions[0], record.Transitions[0]
   hostActual.Count = 22
   want.Expected, want.Actual = &hostExpected, &hostActual
  }
  encoded := encodeSimulationBridgeError(bridge)
  decoded, err := decodeRuntimeNetworkErrorWire(encoded)
  if err != nil || !reflect.DeepEqual(decoded, want) { t.Fatalf("structured error = %+v, %v", decoded, err) }
  finish, err := encodeSimulationFinishResponse(overlay, bridge)
  if err != nil { t.Fatal(err) }
  decodedRecord, decodedBridge, err := decodeRuntimeNetworkFinish(finish)
  if err != nil || !reflect.DeepEqual(decodedRecord, record) || !reflect.DeepEqual(decodedBridge, &want) {
   t.Fatalf("finish with error = %+v %+v %v", decodedRecord, decodedBridge, err)
  }
 }
 fallback := encodeSimulationBridgeError(errors.New(strings.Repeat("x", maximumSimulationWireStringBytes+1)))
 decoded, err := decodeRuntimeNetworkErrorWire(fallback)
 if err != nil || decoded.Kind != "runtime" || decoded.Message != "encode simulation network error" { t.Fatalf("error fallback = %+v %v", decoded, err) }
}

func TestTruncatedRecordStopsAtFailure(t *testing.T) {
 host := &runtimeNetworkWireDecoder{data: appendRuntimeNetworkUint64(nil, 64)}
 hostRecord := decodeRuntimeNetworkRecord(host)
 overlay := &simulationWireDecoder{data: appendSimulationUint64(nil, 64)}
 overlayRecord := decodeSimulationRecord(overlay)
 if host.err == nil || overlay.err == nil { t.Fatal("truncated record was accepted") }
 if len(hostRecord.Transitions) > 1 || len(overlayRecord.Transitions) > 1 {
  t.Fatalf("decoder kept constructing transitions after failure: host=%d overlay=%d", len(hostRecord.Transitions), len(overlayRecord.Transitions))
 }
}

func TestMalformedCrossConsumer(t *testing.T) {
 config, err := encodeRuntimeNetworkConfig(Spec{Limits: Limits{NetworkConnectionBytes: 1, NetworkStallNanos: 1}})
 if err != nil { t.Fatal(err) }
 finish, err := encodeSimulationFinishResponse(simulationRecord{}, nil)
 if err != nil { t.Fatal(err) }
 bridge := encodeSimulationBridgeError(errors.New("partial"))
 for _, frame := range []struct{name string; data []byte; decode func([]byte) error}{
  {"config", config, func(data []byte) error {_, err := decodeSimulationConfig(data); return err}},
  {"finish", finish, func(data []byte) error {_, _, err := decodeRuntimeNetworkFinish(data); return err}},
  {"error", bridge, func(data []byte) error {_, err := decodeRuntimeNetworkErrorWire(data); return err}},
 } {
  t.Run(frame.name, func(t *testing.T) {
   for offset := range len(frame.data) {
    if err := frame.decode(frame.data[:offset]); err == nil { t.Fatalf("accepted truncation at %d", offset) }
   }
   badMagic := bytes.Clone(frame.data)
   badMagic[0] = 'X'
   for _, invalid := range [][]byte{badMagic, append(bytes.Clone(frame.data), 0)} {
    if err := frame.decode(invalid); err == nil { t.Fatal("accepted invalid header or trailing data") }
   }
  })
 }
 invalidBool := bytes.Clone(finish)
 invalidBool[len(invalidBool)-8] = 2
 if _, _, err := decodeRuntimeNetworkFinish(invalidBool); err == nil { t.Fatal("accepted invalid boolean") }
 invalidBool = bytes.Clone(config)
 invalidBool[len(invalidBool)-8] = 2
 if _, err := decodeSimulationConfig(invalidBool); err == nil { t.Fatal("accepted invalid config boolean") }
 for _, maximum := range []uint64{maximumSimulationNodes, maximumSimulationLinks, maximumSimulationListeners, maximumSimulationConnections, maximumSimulationDeliveries, maximumSimulationTransitions} {
  data := appendSimulationUint64([]byte(simulationConfigMagic), maximum+1)
  overlay := newSimulationWireDecoder(data, simulationConfigMagic)
  host := newRuntimeNetworkWireDecoder(data, runtimeNetworkConfigMagic)
  if overlay.count(maximum) != 0 || host.count(maximum) != 0 { t.Fatalf("rejected count above %d may still drive an allocation loop", maximum) }
  if overlay.finish() == nil || host.finish() == nil { t.Fatalf("accepted count above %d", maximum) }
 }
 for _, size := range []uint64{maximumSimulationWireStringBytes+1, ^uint64(0)} {
  data := appendSimulationUint64([]byte(simulationConfigMagic), size)
  overlay := newSimulationWireDecoder(data, simulationConfigMagic)
  host := newRuntimeNetworkWireDecoder(data, runtimeNetworkConfigMagic)
  overlay.string()
  host.string()
  if overlay.finish() == nil || host.finish() == nil { t.Fatalf("accepted string length %d", size) }
 }
}

`

func wireConsumerDeclarations(t *testing.T, root string, paths []string, selected map[string][]string, methods map[string]bool) []byte {
	t.Helper()
	output := []byte("package networkwiretest\n")
	if methods["networkEndpointKey"] || methods["simulationEndpointKey"] {
		output = append(output, []byte("import \"fmt\"\n")...)
	}
	found := make(map[string]bool)
	for _, path := range paths {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, filepath.Join(root, path), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				if declaration.Tok != token.TYPE {
					continue
				}
				for _, spec := range declaration.Specs {
					definition := spec.(*ast.TypeSpec)
					fields, ok := selected[definition.Name.Name]
					if !ok {
						continue
					}
					found[definition.Name.Name] = true
					if len(fields) != 0 {
						body := definition.Type.(*ast.StructType)
						retained := body.Fields.List[:0]
						for _, field := range body.Fields.List {
							for _, name := range fields {
								if field.Names[0].Name == name {
									retained = append(retained, field)
								}
							}
						}
						body.Fields.List = retained
					}
					var encoded bytes.Buffer
					if err := format.Node(&encoded, fset, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{definition}}); err != nil {
						t.Fatal(err)
					}
					output = append(output, encoded.Bytes()...)
					output = append(output, '\n')
				}
			case *ast.FuncDecl:
				if !methods[declaration.Name.Name] {
					continue
				}
				if declaration.Recv != nil {
					receiver := declaration.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
					if _, ok := selected[receiver]; !ok {
						continue
					}
				}
				var encoded bytes.Buffer
				if err := format.Node(&encoded, fset, declaration); err != nil {
					t.Fatal(err)
				}
				output = append(output, encoded.Bytes()...)
				output = append(output, '\n')
			}
		}
	}
	for name := range selected {
		if !found[name] {
			t.Fatalf("wire consumer declaration missing: %s", name)
		}
	}
	return output
}
