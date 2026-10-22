#!/usr/bin/env python3
"""Receipt admission checks; live owner equivalence runs in live-broker-owner-check.py."""
import importlib.util
import json
from pathlib import Path
import sys
import os
import shutil
import subprocess
import tempfile
from datetime import datetime, timezone
import unittest

sys.dont_write_bytecode = True
path = Path(__file__).with_name('live-broker-observe.py')
spec = importlib.util.spec_from_file_location('observe', path)
observe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(observe)

REFERENCE_SOURCE = {'integration': '5e2bfeff4dbd08e5ee9ae0ca6eda649802ab7cd64bca3e5b3c7c1a3f0002ab83', 'provider': 'bce7c6fe9e573eeea49be57d0882a8ad1da60bd23dbc7499d32c6253de7126c4'}


def reference_source(name):
    raw = (Path(__file__).with_name('reference') / ('live-broker-' + name + '.go.txt')).read_bytes()
    if observe.sha(raw) != REFERENCE_SOURCE[name]:
        raise ValueError('retained original source changed: ' + name)
    return raw


POINTER_PROBE = r'''
type choiceControl struct { Mode uint8; Trace []byte; Identity map[string]string }
type runtimeControl struct { Choice *choiceControl }
type configuration struct { Args []string; Env map[string]string }
type BrokerNode struct { ID string; Incarnation uint64 }
type brokerSession struct { Node BrokerNode }
type image struct { data []byte }
func (*image) Identity() string { return "probe-image" }
type Request struct {
    ModulePath string
    Broker *brokerSession
    StdoutMirror any
    CompileCache string
    StorageImage *image
    Runtime *runtimeControl
    Config configuration
    StorageCrashSelection map[string][]uint64
    remoteModel bool
}
type Result struct { StorageImage *image }
var runCalls int
func Run(_ context.Context, request Request) (Result, error) {
    runCalls++
    request.Runtime.Choice.Identity["inside-run"] = "mutated"
    return Result{}, nil
}

func TestCallTimeRequestSurvivesPointerMutation(t *testing.T) {
    root := t.TempDir()
    t.Setenv("GOMAD_BROKER_OBSERVATION_ROOT", root)
    module := filepath.Join(root, "input.wasm")
    if err := os.WriteFile(module, []byte("probe-module"), 0600); err != nil { t.Fatal(err) }
    request := Request{ModulePath:module, Runtime:&runtimeControl{Choice:&choiceControl{
        Mode:3, Trace:[]byte{1,2,3}, Identity:map[string]string{"source":"original"}}},
        Config:configuration{Args:[]string{"original"},Env:map[string]string{"mode":"original"}},
        StorageCrashSelection:map[string][]uint64{"write":{7}}, StorageImage:&image{data:[]byte{4,5}},
        Broker:&brokerSession{Node:BrokerNode{ID:"a",Incarnation:1}},remoteModel:true}
    t.Run("original-assertions",func(t *testing.T) {
        if _,err := brokerObserveRun(t, context.Background(),request); err != nil { t.Fatal(err) }
        if runCalls != 1 { t.Fatalf("original Run calls=%d",runCalls) }
        if _,err := os.Stat(filepath.Join(root,"observation-000001.json")); !os.IsNotExist(err) {
            t.Fatal("observation published before original assertions")
        }
        request.Runtime.Choice.Mode=1
        request.Runtime.Choice.Trace[0]=9
        request.Runtime.Choice.Identity["source"]="mutated"
        request.Config.Args[0]="mutated"
        request.Config.Env["mode"]="mutated"
        request.StorageCrashSelection["write"][0]=9
        request.StorageImage.data[0]=9
        request.Broker.Node.ID="mutated"
    })
    raw,err := os.ReadFile(filepath.Join(root,"observation-000001.json"))
    if err != nil { t.Fatal(err) }
    var row struct { Request Request `json:"request"`; Node BrokerNode `json:"node"`; Prior []byte `json:"request_storage_image"`; Remote bool `json:"remote_model"` }
    if err := json.Unmarshal(raw,&row); err != nil { t.Fatal(err) }
    got := row.Request
    if got.Runtime.Choice.Mode != 3 || string(got.Runtime.Choice.Trace) != string([]byte{1,2,3}) ||
        got.Runtime.Choice.Identity["source"] != "original" || got.Runtime.Choice.Identity["inside-run"] != "" ||
        got.Config.Args[0] != "original" || got.Config.Env["mode"] != "original" ||
        got.StorageCrashSelection["write"][0] != 7 || string(row.Prior) != string([]byte{4,5}) || row.Node.ID != "a" || !row.Remote {
        t.Fatalf("call-time request snapshot changed: %s",raw)
    }
}
'''

WORLD_PROBE = r'''
type BrokerConfig struct { Network any }
type Broker struct { calls int }
func NewBroker(BrokerConfig) (*Broker,error) { return &Broker{},nil }
func (broker *Broker) EncodeWorldEvidence() ([]byte,error) {
    broker.calls++
    if broker.calls != 1 { return nil,fmt.Errorf("broker World finalized") }
    return []byte{1,2,3},nil
}
func (*Broker) NetworkRecord() (any,error) { return nil,nil }

func TestOriginalWorldEncodingIsNotRepeatedByCleanup(t *testing.T) {
    for _, originalEncoded := range []bool{true,false} {
        t.Run(fmt.Sprint(originalEncoded),func(t *testing.T) {
            root := t.TempDir()
            t.Setenv("GOMAD_BROKER_OBSERVATION_ROOT",root)
            var broker *Broker
            t.Run("original-assertions",func(t *testing.T) {
                var err error
                broker,err=brokerObserveNew(t,BrokerConfig{})
                if err != nil { t.Fatal(err) }
                if originalEncoded {
                    ORIGINAL_ENCODING_CALL
                    if err != nil || string(wantWorld) != string([]byte{1,2,3}) { t.Fatalf("original encoding: %v",err) }
                    if broker.calls != 1 { t.Fatal("original call count changed") }
                    wantWorld[0]=9
                }
                matches,err := filepath.Glob(filepath.Join(root,"broker-*.json"))
                if err != nil || len(matches) != 0 { t.Fatal("observation published before assertions") }
            })
            if broker.calls != 1 { t.Fatalf("encoder calls=%d, want exactly1",broker.calls) }
            matches,err := filepath.Glob(filepath.Join(root,"broker-*.json"))
            if err != nil || len(matches) != 1 { t.Fatalf("postassertion observations=%v %v",matches,err) }
            raw,err := os.ReadFile(matches[0])
            if err != nil { t.Fatal(err) }
            var row struct { World []byte `json:"world"` }
            if err := json.Unmarshal(raw,&row); err != nil { t.Fatal(err) }
            if string(row.World) != string([]byte{1,2,3}) { t.Fatalf("raw World changed: %x",row.World) }
        })
    }
}
'''


class RequestSnapshot(unittest.TestCase):
    def test_actual_go_observer_freezes_nested_call_time_request(self):
        function = observe.HELPER.split('func brokerObserveRun(', 1)[1].split('\nfunc brokerObserveNew(', 1)[0]
        source = '''package wasi
import ("context"; "crypto/sha256"; "encoding/json"; "fmt"; "os"; "path/filepath"; "sync/atomic"; "testing")
var brokerObservationOrdinal atomic.Uint64
func brokerObserveRun(''' + function + POINTER_PROBE
        self.run_probe(source, 'TestCallTimeRequestSurvivesPointerMutation')

    def test_actual_go_observer_reuses_original_single_use_world_encoding(self):
        original = reference_source('integration')
        observed = observe.instrument(original).decode()
        call = next(line.strip() for line in observed.splitlines() if 'wantWorld, err :=' in line)
        helper = observe.HELPER.split('func brokerObserveNew(', 1)[1].split('\ntype brokerObservedCluster', 1)[0]
        capture = ''
        if 'func brokerObserveWorld(' in observe.HELPER:
            capture = '\nvar brokerObservationWorld sync.Map\nfunc brokerObserveWorld(' + observe.HELPER.split('func brokerObserveWorld(',1)[1].split('\nfunc ',1)[0]
        source = '''package wasi
import ("encoding/json"; "fmt"; "os"; "path/filepath"; "sync/atomic"; "testing"; SYNC_IMPORT)
var brokerObservationOrdinal atomic.Uint64
'''.replace('SYNC_IMPORT','"sync"' if capture else '') + capture + '\nfunc brokerObserveNew(' + helper + WORLD_PROBE.replace('ORIGINAL_ENCODING_CALL',call)
        self.run_probe(source, 'TestOriginalWorldEncodingIsNotRepeatedByCleanup')

    def run_probe(self, source, name):
        compiler = os.environ.get('GOMAD_OBSERVATION_PROBE_GO') or shutil.which('go')
        self.assertIsNotNone(compiler, 'Go compiler required for actual observer probe')
        if os.environ.get('GOMAD_OBSERVATION_PROBE_ROOT'):
            evidence = Path(os.environ['GOMAD_OBSERVATION_PROBE_ROOT']) / name
        else:
            temporary = tempfile.TemporaryDirectory(prefix='gomad-observer-probe-')
            self.addCleanup(temporary.cleanup)
            evidence = Path(temporary.name) / 'probe'
        evidence.mkdir(parents=True, exist_ok=False)
        (evidence / 'go.mod').write_text('module gomad-observer-pointer-probe\n\ngo 1.18\n')
        (evidence / 'snapshot_test.go').write_text(source)
        command = [compiler, 'test', '-count=1', '-v', '-run', '^' + name + '$', '.']
        started = datetime.now(timezone.utc).isoformat()
        with subprocess.Popen(command, cwd=evidence, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                              env={**os.environ, 'GOWORK':'off', 'GOTOOLCHAIN':'local', 'GOENV':'off'}) as process:
            try:
                stdout, stderr = process.communicate(timeout=60)
            except subprocess.TimeoutExpired:
                process.kill()
                process.communicate()
                raise
            (evidence / 'stdout.log').write_bytes(stdout)
            (evidence / 'stderr.log').write_bytes(stderr)
            (evidence / 'command.json').write_text(json.dumps(dict(argv=command, exit=process.returncode,
                pid=process.pid, started_at=started, ended_at=datetime.now(timezone.utc).isoformat(),
                source_sha256=observe.sha(source.encode()), helper_sha256=observe.sha(observe.HELPER.encode()))) + '\n')
            self.assertEqual(process.returncode, 0, stdout.decode() + stderr.decode())


class ExactSelection(unittest.TestCase):
    def test_provider_observer_preserves_exact_original_admission_paths(self):
        self.assertTrue(hasattr(observe, 'instrument_provider'), 'missing provider observer')
        source = reference_source('provider')
        observed = observe.instrument_provider(source)
        self.assertEqual(observed.count(b'providerObserveOpen(t, '), 3)
        self.assertIn(b'providerObserveOpen(t, reference, t.Context(), hard)', observed)
        self.assertEqual(observed.count(b't.Fatal('), source.count(b't.Fatal('))
        self.assertEqual(observed.count(b't.Fatalf('), source.count(b't.Fatalf('))
        self.assertEqual(observed.count(b'.Start('), source.count(b'.Start('))
        self.assertEqual(observed.count(b'.Wait('), source.count(b'.Wait('))

    def test_root_pass_does_not_admit_missing_or_skipped_cell(self):
        root = observe.ROOTS[37]
        cells = [f'{root}/{clock}/digest-{digest}' for clock in ('read-step', 'frozen') for digest in ('false', 'true')]
        passed = [dict(Test=name, Action='pass') for name in [root, *cells]]
        for replacement in (None, 'skip', 'fail'):
            with self.subTest(replacement=replacement):
                events = passed[:-1]
                if replacement:
                    events = [*events, dict(Test=cells[-1], Action=replacement)]
                with self.assertRaisesRegex(ValueError, 'required cell did not pass|failed or skipped subtest'):
                    observe.validate_events(b'\n'.join(json.dumps(event).encode() for event in events), [37])
        self.assertEqual(len(observe.validate_events(b'\n'.join(json.dumps(event).encode() for event in passed), [37])), 5)

    def test_observer_preserves_subtest_calls_and_assertions(self):
        source = reference_source('integration')
        observed = observe.instrument(source)
        self.assertNotIn(b't.brokerObserveRun', observed)
        self.assertEqual(observed.count(b't.Fatal('), source.count(b't.Fatal('))
        self.assertEqual(observed.count(b't.Fatalf('), source.count(b't.Fatalf('))
        self.assertEqual(observed.count(b't.Run('), source.count(b't.Run('))
        self.assertGreater(observed.count(b'brokerObserveRun(t, '), 0)


class DiagnosticEncoding(unittest.TestCase):
    def test_expected_header_mutations_cannot_rewrite_actual_framing(self):
        path = Path(__file__).with_name('live-broker-owner-check.py')
        spec = importlib.util.spec_from_file_location('owner', path)
        owner = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(owner)
        self.assertTrue(hasattr(owner, 'require_diagnostic'), 'missing independent diagnostic header comparison')
        header = bytes.fromhex('474f4d4144444701' '00000001' '01000000'
                               '0000000000000100' '00000000000000a0' '0000000000000001' + '00' * 24)
        records = bytes(96)
        owner.require_diagnostic(records, header + records, 256)
        for offset in (0, 11, 12, 13, 23, 31, 39, 63):
            with self.subTest(offset=offset):
                changed = bytearray(header + records)
                changed[offset] ^= 1
                with self.assertRaises(AssertionError):
                    owner.require_diagnostic(records, bytes(changed), 256)


if __name__ == '__main__':
    unittest.main()
