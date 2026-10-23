#!/usr/bin/env python3
"""Observe unchanged Go-owner assertions in an explicitly derived test checkout.

Actual provider root19 requires a genuine reference-transport typed helper build
receipt. Historical derive-only source proofs remain preparatory and unchanged.
"""
import argparse
import base64
import hashlib
import importlib.util
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import subprocess
import sys

sys.dont_write_bytecode = True

ROOTS = {
    19: 'TestProviderClusterStartsOnlyRequestedIncarnations',
    32: 'TestBrokerCrashRecoveryFreshGuest',
    33: 'TestClusterOutputBoundaryParity',
    34: 'TestClusterBrokerDurableRestartParity',
    35: 'TestClusterShutdownLiveNodesEvidenceOrder',
    36: 'TestBrokerIndependentGuestStreamsReplay',
    37: 'TestHelperModelPreservesGoOwnerEvidence',
    38: 'TestNamespaceOwnerFilesystemParity',
    39: 'TestBrokerCrashRetainsInterruptedRuntimePrefix',
}
PROVIDER_HELPER = r'''package simulation

import (
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync/atomic"
    "testing"

    "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
    gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

var providerObservationOrdinal atomic.Uint64

func providerObserveWrite(t *testing.T, row map[string]any) {
    ordinal := providerObservationOrdinal.Add(1)
    t.Cleanup(func() {
        if t.Failed() { return }
        row["test"], row["ordinal"] = t.Name(), ordinal
        raw, err := json.Marshal(row)
        if err != nil { t.Error(err); return }
        path := filepath.Join(os.Getenv("GOMAD_BROKER_OBSERVATION_ROOT"), fmt.Sprintf("provider-%06d.json", ordinal))
        if err := os.WriteFile(path, raw, 0600); err != nil { t.Error(err) }
    })
}

func providerObserveOpen(t *testing.T, provider *Provider, ctx context.Context, spec gomadsim.Spec) (gomadsim.WASMExecution, error) {
    opened, err := provider.Open(ctx, spec)
    message := ""
    if err != nil { message = err.Error() }
    role := "original-one-helper-provider"
    if provider.goBrokerReference { role = "original-private-go-broker-reference" }
    row := map[string]any{"kind":"open", "role":role, "spec":spec, "error":message,
        "admitted":opened != nil, "go_broker_reference":provider.goBrokerReference,
        "go_owner_implementation_sha256":wasi.ImplementationSHA256(),
        "cluster_model_implementation_sha256":gomadsim.ImplementationSHA256()}
    if execution, ok := opened.(*execution); ok {
        row["helper_started_at_open"] = execution.cluster != nil
        row["lazy_cluster_request_at_open"] = execution.clusterRequest != nil
        row["go_broker_present_at_open"] = execution.broker != nil
    }
    boots := make(map[string]wasi.Request)
    for id, request := range provider.Boots {
        module, readErr := os.ReadFile(request.ModulePath)
        if readErr != nil { t.Fatal(readErr) }
        path := filepath.Join(os.Getenv("GOMAD_BROKER_OBSERVATION_ROOT"), fmt.Sprintf("%x.wasm", sha256.Sum256(module)))
        if writeErr := os.WriteFile(path, module, 0600); writeErr != nil { t.Fatal(writeErr) }
        request.ModulePath, request.Broker, request.StdoutMirror, request.CompileCache = path, nil, nil, ""
        boots[string(id)] = request
    }
    row["boots"] = boots
    providerObserveWrite(t, row)
    return opened, err
}

func providerObserveFinish(t *testing.T, opened gomadsim.WASMExecution) (gomadsim.WASMEvidence, error) {
    evidence, err := opened.Finish()
    message := ""
    if err != nil { message = err.Error() }
    row := map[string]any{"kind":"finish", "role":"original-one-helper-provider", "evidence":evidence, "error":message}
    if execution, ok := opened.(*execution); ok && execution.cluster != nil {
        row["frames"] = execution.cluster.FrameCounts()
    }
    providerObserveWrite(t, row)
    return evidence, err
}
'''
HELPER = r'''package wasi

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "sync/atomic"
    "testing"
)

var brokerObservationOrdinal atomic.Uint64
var brokerObservationWorld sync.Map
var brokerObservationNetworks sync.Map

type brokerObservedNetwork struct { wire, document []byte }

func brokerObserveNetwork(state *clusterState) error {
    if (len(state.network) == 0) != (len(state.result.NetworkTranscript) == 0) {
        return fmt.Errorf("unpaired collected network evidence")
    }
    captured := brokerObservedNetwork{append([]byte(nil), state.network...), append([]byte(nil), state.result.NetworkTranscript...)}
    if _, loaded := brokerObservationNetworks.LoadOrStore(state, captured); loaded {
        return fmt.Errorf("duplicate pre-clear network observation")
    }
    return nil
}

type brokerObservedCollection struct {
    Status string `json:"status"`
    SinkRetained bool `json:"sink_retained"`
    StateError string `json:"state_error"`
    DoneClosed bool `json:"done_closed"`
    Classified bool `json:"classified"`
    Reaped bool `json:"reaped"`
    GroupGone bool `json:"group_gone"`
    WireBytes int `json:"wire_bytes"`
    DocumentBytes int `json:"document_bytes"`
}

func brokerObservedNetworkFor(state *clusterState, groupGone bool) ([]byte, brokerObservedCollection, error) {
    status := brokerObservedCollection{SinkRetained:state.sink != nil,
        Classified:!state.executionClassifiedAt.IsZero(), Reaped:state.result.Reaped,
        GroupGone:groupGone, WireBytes:len(state.network), DocumentBytes:len(state.result.NetworkTranscript)}
    if state.err != nil { status.StateError = state.err.Error() }
    select { case <-state.done: status.DoneClosed = true; default: }
    value, ok := brokerObservationNetworks.LoadAndDelete(state)
    if !ok {
        if status.SinkRetained && status.StateError != "" && status.DoneClosed && status.Classified &&
            status.Reaped && status.GroupGone && state.result.Termination == "" &&
            status.WireBytes == 0 && status.DocumentBytes == 0 {
            status.Status = "host_cleanup_without_result"
            return nil, status, nil
        }
        return nil, status, fmt.Errorf("missing pre-clear network observation")
    }
    captured, ok := value.(brokerObservedNetwork)
    if !ok || status.SinkRetained || !bytes.Equal(captured.document, state.result.NetworkTranscript) {
        return nil, status, fmt.Errorf("changed collected network document")
    }
    status.Status = "collected"
    status.WireBytes = len(captured.wire)
    return captured.wire, status, nil
}

func brokerObserveWorld(broker *Broker) ([]byte, error) {
    world, err := broker.EncodeWorldEvidence()
    if err == nil { brokerObservationWorld.Store(broker, append([]byte(nil), world...)) }
    return world, err
}

func brokerObserveRun(t *testing.T, ctx context.Context, request Request) (Result, error) {
    retained := request
    retained.Broker, retained.StdoutMirror, retained.CompileCache = nil, nil, ""
    requestJSON, snapshotErr := json.Marshal(retained)
    remote := request.remoteModel
    node := BrokerNode{}
    if request.Broker != nil { node = request.Broker.Node }
    var priorImage []byte
    if request.StorageImage != nil { priorImage = append([]byte(nil), request.StorageImage.data...) }
    result, err := Run(ctx, request)
    ordinal := brokerObservationOrdinal.Add(1)
    t.Cleanup(func() {
        if t.Failed() || err != nil { return }
        if snapshotErr != nil { t.Error(snapshotErr); return }
        root := os.Getenv("GOMAD_BROKER_OBSERVATION_ROOT")
        if root == "" { t.Fatal("observation root is required"); return }
        module, readErr := os.ReadFile(request.ModulePath)
        if readErr != nil { t.Error(readErr); return }
        digest := fmt.Sprintf("%x", sha256.Sum256(module))
        modulePath := filepath.Join(root, digest+".wasm")
        if writeErr := os.WriteFile(modulePath, module, 0600); writeErr != nil { t.Error(writeErr); return }
        var snapshot map[string]json.RawMessage
        if decodeErr := json.Unmarshal(requestJSON, &snapshot); decodeErr != nil { t.Error(decodeErr); return }
        snapshot["ModulePath"], _ = json.Marshal(modulePath)
        var image []byte
        imageIdentity := ""
        if result.StorageImage != nil { image, imageIdentity = result.StorageImage.data, result.StorageImage.Identity() }
        raw, marshalErr := json.Marshal(map[string]any{"test":t.Name(), "ordinal":ordinal, "go_owner_implementation_sha256":ImplementationSHA256(),
            "remote_model":remote, "node":node, "request":snapshot, "result":result,
            "storage_image":image, "storage_image_identity":imageIdentity, "request_storage_image":priorImage})
        if marshalErr != nil { t.Error(marshalErr); return }
        if writeErr := os.WriteFile(filepath.Join(root, fmt.Sprintf("observation-%06d.json",ordinal)), raw, 0600); writeErr != nil { t.Error(writeErr) }
    })
    return result, err
}

func brokerObserveNew(t *testing.T, configuration BrokerConfig) (*Broker, error) {
    broker, err := NewBroker(configuration)
    if err != nil { return broker, err }
    ordinal := brokerObservationOrdinal.Add(1)
    t.Cleanup(func() {
        if t.Failed() { return }
        var world []byte
        if captured, ok := brokerObservationWorld.LoadAndDelete(broker); ok {
            world = captured.([]byte)
        } else {
            var encodeErr error
            world, encodeErr = broker.EncodeWorldEvidence()
            if encodeErr != nil { t.Error(encodeErr); return }
        }
        var network any
        if configuration.Network != nil {
            record, recordErr := broker.NetworkRecord()
            if recordErr != nil { t.Error(recordErr); return }
            network = record
        }
        raw, marshalErr := json.Marshal(map[string]any{"test":t.Name(), "ordinal":ordinal, "go_owner_implementation_sha256":ImplementationSHA256(),
            "configuration":configuration, "world":world, "network":network})
        if marshalErr != nil { t.Error(marshalErr); return }
        path := filepath.Join(os.Getenv("GOMAD_BROKER_OBSERVATION_ROOT"), fmt.Sprintf("broker-%06d.json",ordinal))
        if writeErr := os.WriteFile(path, raw, 0600); writeErr != nil { t.Error(writeErr) }
    })
    return broker, nil
}


type brokerObservedCluster struct {
    *ClusterSession
    steps []map[string]any
    reverseInsertion bool
}

func brokerObserveOpen(t *testing.T, ctx context.Context, configuration ClusterRequest) (*brokerObservedCluster, error) {
    session, err := OpenCluster(ctx, configuration)
    if err != nil { return nil, err }
    wrapped := &brokerObservedCluster{ClusterSession:session}
    encoded, err := json.Marshal(configuration)
    if err != nil { t.Fatal(err) }
    ordinal := brokerObservationOrdinal.Add(1)
    t.Cleanup(func() {
        if t.Failed() { return }
        root := os.Getenv("GOMAD_BROKER_OBSERVATION_ROOT")
        var input any
        if err := json.Unmarshal(encoded, &input); err != nil { t.Error(err); return }
        var retainModules func(any) error
        retainModules = func(value any) error {
            switch value := value.(type) {
            case map[string]any:
                for key, child := range value {
                    if key == "ModulePath" {
                        path, ok := child.(string)
                        if !ok || path == "" { return fmt.Errorf("missing observed cluster module") }
                        raw, err := os.ReadFile(path)
                        if err != nil { return err }
                        copied := filepath.Join(root, fmt.Sprintf("%x.wasm", sha256.Sum256(raw)))
                        if err := os.WriteFile(copied, raw, 0600); err != nil { return err }
                        value[key] = copied
                    } else if err := retainModules(child); err != nil { return err }
                }
            case []any:
                for _, child := range value { if err := retainModules(child); err != nil { return err } }
            }
            return nil
        }
        if err := retainModules(input); err != nil { t.Error(err); return }
        var states []map[string]any
        session.mu.Lock()
        for handle, state := range session.states {
            streams := map[string][]byte{}
            if state.sink != nil {
                evidence := state.sink.helperEvidence
                streams = map[string][]byte{"choice":evidence.choice,"diagnostic":evidence.diagnostic,"transcript":evidence.transcript,
                    "stdout":evidence.stdout,"stderr":evidence.stderr,"storage":evidence.storage,"storage_state":evidence.storageState,"storage_image":evidence.storageImage}
            }
            network, collection, observeErr := brokerObservedNetworkFor(state, session.result.GroupGone)
            if observeErr != nil { session.mu.Unlock(); t.Error(observeErr); return }
            states = append(states, map[string]any{"handle":handle, "result":state.result, "network_wire":network, "collection":collection, "streams":streams})
        }
        result := session.result
        session.mu.Unlock()
        raw, err := json.Marshal(map[string]any{"test":t.Name(), "ordinal":ordinal, "go_owner_implementation_sha256":ImplementationSHA256(),
            "configuration":input, "steps":wrapped.steps, "states":states, "result":result, "network_wire":session.networkWire, "reverse_insertion":wrapped.reverseInsertion})
        if err != nil { t.Error(err); return }
        if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("cluster-%06d.json",ordinal)), raw, 0600); err != nil { t.Error(err) }
    })
    return wrapped, nil
}

func brokerObserveOpenWithOrder(t *testing.T, reverse bool, ctx context.Context, configuration ClusterRequest) (*brokerObservedCluster, error) {
    session, err := brokerObserveOpen(t, ctx, configuration)
    if err == nil { session.reverseInsertion = reverse }
    return session, err
}

func (session *brokerObservedCluster) Start(ctx context.Context, node string, incarnation uint64, boot string, runtime *RuntimeControl) error {
    err := session.ClusterSession.Start(ctx, node, incarnation, boot, runtime)
    if err == nil {
        action := "start"
        if incarnation > 1 { action = "restart" }
        var runtimeSnapshot any
        raw, encodeErr := json.Marshal(runtime)
        if encodeErr != nil { return encodeErr }
        if decodeErr := json.Unmarshal(raw, &runtimeSnapshot); decodeErr != nil { return decodeErr }
        session.steps = append(session.steps, map[string]any{"action":action, "node":node,"incarnation":incarnation,"boot":boot,"runtime":runtimeSnapshot})
    }
    return err
}

func (session *brokerObservedCluster) Wait(ctx context.Context, node string, incarnation uint64) (Result, error) {
    result, err := session.ClusterSession.Wait(ctx, node, incarnation)
    if err == nil { session.steps = append(session.steps, map[string]any{"action":"wait", "node":node,"incarnation":incarnation}) }
    return result, err
}

func (session *brokerObservedCluster) Link(ctx context.Context, action, from, to string, delay uint64, symmetric bool) error {
    err := session.ClusterSession.Link(ctx, action, from, to, delay, symmetric)
    if err == nil { session.steps = append(session.steps, map[string]any{"action":"link", "link_action":action,"from":from,"to":to,"delay":delay,"symmetric":symmetric}) }
    return err
}

'''


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def instrument(raw):
    source = raw.decode()
    functions = list(re.finditer(r'^func (\w+)\(', source, re.M))
    wanted = {name for root, name in ROOTS.items() if root != 19} | {'brokerCrashRecoveryParity'}
    found = set()
    for index in range(len(functions) - 1, -1, -1):
        match = functions[index]
        if match[1] not in wanted:
            continue
        found.add(match[1])
        end = functions[index + 1].start() if index + 1 < len(functions) else len(source)
        body = source[match.start():end]
        body = re.sub(r'(?<![\w.])Run\(', 'brokerObserveRun(t, ', body)
        body = re.sub(r'(?<![\w.])NewBroker\(', 'brokerObserveNew(t, ', body)
        body = re.sub(r'\b(\w+)\.EncodeWorldEvidence\(\)', r'brokerObserveWorld(\1)', body)
        body = re.sub(r'(?<![\w.])OpenCluster\(', 'brokerObserveOpen(t, ', body)
        if match[1] == ROOTS[35]:
            body = body.replace('brokerObserveOpen(t, ', 'brokerObserveOpenWithOrder(t, iteration%2 != 0, ')
        source = source[:match.start()] + body + source[end:]
    if found != wanted:
        raise ValueError(f'original test roots missing: {wanted - found}')
    return source.encode()


NETWORK_CAPTURE = 'successful-original-collect/pre-clear-owned-copy'


def instrument_cluster(raw):
    collected = b'\t\tif err := state.collect(frame, s.config.Broker, inspecting); err != nil {\n\t\t\treturn err\n\t\t}\n'
    clearing = b'\t\tstate.sink = nil\n\t\tstate.network = nil\n'
    boundary = collected + clearing
    if raw.count(boundary) != 1:
        raise ValueError('original cluster collection boundary changed')
    return raw.replace(boundary, collected +
        b'\t\tif err := brokerObserveNetwork(state); err != nil { return err }\n' + clearing)


def validate_versions(index, derivation):
    version = index.get('schema')
    if version == 'gomad.live-broker-observation-index/v1':
        if derivation.get('schema') != 'gomad.live-broker-observation-derivation/v1' or 'network_capture' in index or 'network_capture' in derivation:
            raise ValueError('mixed original observation versions')
        return
    if version != 'gomad.live-broker-observation-index/v2' or derivation.get('schema') != 'gomad.live-broker-observation-derivation/v2':
        raise ValueError('unknown observation version')
    capture = index.get('network_capture')
    if not isinstance(capture, dict) or capture != derivation.get('network_capture') or set(capture) != {'boundary', 'derived_go_implementation_sha256'} or capture.get('boundary') != NETWORK_CAPTURE or not re.fullmatch(r'sha256:[0-9a-f]{64}', capture.get('derived_go_implementation_sha256', '')):
        raise ValueError('invalid network capture identity')
    source = derivation.get('cluster_source')
    if not isinstance(source, dict) or source.get('path') != 'tools/gomad/internal/backends/wasm/wasi/cluster.go' or any(not re.fullmatch(r'[0-9a-f]{64}', source.get(field, '')) for field in ('original_sha256', 'observed_sha256')) or source['original_sha256'] == source['observed_sha256']:
        raise ValueError('invalid derived cluster source binding')


def validate_collection(state, group_gone):
    collection = state.get('collection')
    fields = {'status', 'sink_retained', 'state_error', 'done_closed', 'classified', 'reaped',
              'group_gone', 'wire_bytes', 'document_bytes'}
    if not isinstance(collection, dict) or set(collection) != fields:
        raise ValueError('missing typed collection status')
    if any(type(collection[name]) is not bool for name in ('sink_retained', 'done_closed', 'classified', 'reaped', 'group_gone')) or not isinstance(collection['state_error'], str):
        raise ValueError('invalid collection lifecycle types')
    if any(collection[name] is not True for name in ('done_closed', 'classified', 'reaped', 'group_gone')) or group_gone is not True or state['result'].get('Reaped') is not True:
        raise ValueError('unclosed original collection lifecycle')
    if 'network_wire' not in state:
        raise ValueError('missing observed network wire field')
    wire = base64.b64decode(state['network_wire'] or '', validate=True)
    document = base64.b64decode(state['result'].get('NetworkTranscript') or '', validate=True)
    if any(type(collection[name]) is not int for name in ('wire_bytes', 'document_bytes')) or collection['wire_bytes'] != len(wire) or collection['document_bytes'] != len(document):
        raise ValueError('collection byte accounting mismatch')
    if collection['status'] == 'collected':
        if collection['sink_retained'] or collection['state_error'] or not state['result'].get('Termination') or bool(wire) != bool(document):
            raise ValueError('invalid collected network evidence')
    elif collection['status'] == 'host_cleanup_without_result':
        if not collection['sink_retained'] or not collection['state_error'] or state['result'].get('Termination') != '' or wire or document:
            raise ValueError('ambiguous uncollected host cleanup')
    else:
        raise ValueError('unknown collection status')
    return collection['status']


def validate_row(row, index):
    if index['schema'] != 'gomad.live-broker-observation-index/v2':
        return
    if row.get('go_owner_implementation_sha256') != index['network_capture']['derived_go_implementation_sha256']:
        raise ValueError('observed derived Go identity mismatch')
    for state in row.get('states', []):
        validate_collection(state, row['result'].get('GroupGone'))


def instrument_provider(raw):
    source = raw.decode()
    pattern = r'(^func ' + ROOTS[19] + r'\(.*?)(?=^func |\Z)'
    match = re.search(pattern, source, re.M | re.S)
    if not match:
        raise ValueError('original provider root19 missing')
    body, count = re.subn(r'\b(provider|reference)\.Open\(', r'providerObserveOpen(t, \1, ', match[1])
    if count != 3:
        raise ValueError('original provider admission path count changed')
    body = body.replace('opened.Finish()', 'providerObserveFinish(t, opened)')
    return (source[:match.start()] + body + source[match.end():]).encode()


def validate_events(raw, roots):
    events = [json.loads(line) for line in raw.splitlines()]
    statuses = {event['Test']: event['Action'] for event in events
                if 'Test' in event and event['Action'] in ('pass', 'fail', 'skip')}
    for root in roots:
        name = ROOTS[root]
        if statuses.get(name) != 'pass':
            raise ValueError(f'{name}: exact root did not pass')
        if any(status != 'pass' for test, status in statuses.items() if test.startswith(name + '/')):
            raise ValueError(f'{name}: failed or skipped subtest')
    leaves = {
        32: [f'read-step-{step}' for step in (0, 1000)],
        33: [f'{fixture}/{action}' for fixture in ('boundary-stderr', 'boundary-split') for action in ('crash', 'stop')],
        34: [f'{fixture}/seed-{seed}' for fixture in ('crash', 'stop', 'open-streams', 'storage-crash') for seed in (7, 19, 42)],
        36: [f'seed-{seed}/{fixture}' for seed in (7, 19, 42) for fixture in ('partition-heal-delay', 'stall', 'live-link')],
        37: [f'{clock}/digest-{digest}' for clock in ('read-step', 'frozen') for digest in ('false', 'true')],
    }
    for root in roots:
        for suffix in leaves.get(root, []):
            name = ROOTS[root] + '/' + suffix
            if statuses.get(name) != 'pass':
                raise ValueError(f'{name}: required cell did not pass')
    return statuses


def index_observations(root):
    files = {path.name: dict(bytes=path.stat().st_size, sha256=sha(path.read_bytes()))
             for path in sorted((root / 'observations').iterdir()) if path.is_file()}
    derivation_path = root / 'derivation.json'
    derivation = json.loads(derivation_path.read_bytes())
    version = 2 if derivation['schema'] == 'gomad.live-broker-observation-derivation/v2' else 1
    capture = None
    if version == 2:
        rows = [json.loads((root / 'observations' / name).read_bytes()) for name in files if name.endswith('.json')]
        identities = {row.get('go_owner_implementation_sha256') for row in rows}
        if len(identities) != 1 or not re.fullmatch(r'sha256:[0-9a-f]{64}', next(iter(identities), '') or ''):
            raise ValueError('missing or inconsistent derived Go identity')
        capture = dict(boundary=NETWORK_CAPTURE, derived_go_implementation_sha256=identities.pop())
        derivation['network_capture'] = capture
        derivation_path.write_text(json.dumps(derivation, indent=2) + '\n')
    index = dict(schema=f'gomad.live-broker-observation-index/v{version}',
        derivation_sha256=sha(derivation_path.read_bytes()),
        go_test_sha256=sha((root / 'go-test.jsonl').read_bytes()), files=files)
    if capture is not None:
        index['network_capture'] = capture
    validate_versions(index, derivation)
    if version == 2:
        for row in rows:
            validate_row(row, index)
    (root / 'observation-index.json').write_text(json.dumps(index, indent=2) + '\n')


def provider_helper_receipt(path, helper):
    path, helper = path.resolve(strict=True), helper.resolve(strict=True)
    binding = dict(path=str(path),sha256=sha(path.read_bytes()))
    checker_path = Path(__file__).with_name('live-cluster-check.py')
    spec = importlib.util.spec_from_file_location('provider_helper_admission',checker_path)
    checker = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(checker)
    build = checker.reference_helper(binding,sha(helper.read_bytes()))
    if build['binary'] != str(helper):
        raise ValueError('provider helper differs from its actual typed build publication')
    return binding


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('source', 'derived', 'work', 'compiler', 'helper'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--roots')
    parser.add_argument('--provider-roots', choices=('19',))
    parser.add_argument('--derive-only', action='store_true')
    parser.add_argument('--helper-build-receipt',type=Path)
    args = parser.parse_args()
    if args.roots is not None and args.provider_roots:
        parser.error('provider and broker observations require separate command receipts')
    provider = args.provider_roots is not None
    if args.helper_build_receipt is not None and not provider:
        parser.error('typed provider helper admission is scoped to root19')
    if provider and not args.derive_only and args.helper_build_receipt is None:
        parser.error('actual provider root19 requires --helper-build-receipt from reference-transport')
    helper_binding = (provider_helper_receipt(args.helper_build_receipt,args.helper)
        if args.helper_build_receipt is not None else None)
    roots = [19] if provider else [int(value) for value in (args.roots or '32,33,34,35,36,37,38,39').split(',')]
    if not roots or len(set(roots)) != len(roots) or (not provider and (set(roots) - (ROOTS.keys() - {19}))):
        parser.error('roots must be distinct root numbers 32 through 39')
    source, derived, work = args.source.resolve(), args.derived.resolve(), args.work.resolve()
    if derived.parent != Path('/home/agent/wt'):
        parser.error('derived checkout must be directly under /home/agent/wt')
    work.mkdir(mode=0o700, parents=True, exist_ok=False)
    base = subprocess.check_output(['git', '-C', str(source), 'rev-parse', 'HEAD'], text=True).strip()
    subprocess.run(['git', '-C', str(source), 'worktree', 'add', '--detach', str(derived), base], check=True)
    relative = Path('tools/gomad/internal/backends/wasm/cluster/provider_test.go' if provider else
                    'tools/gomad/internal/backends/wasm/wasi/integration_test.go')
    original = (source / relative).read_bytes()
    if (derived / relative).read_bytes() != original:
        raise ValueError('source test differs from its committed derivation base')
    observed = instrument_provider(original) if provider else instrument(original)
    (derived / relative).write_bytes(observed)
    helper = derived / relative.parent / 'live_broker_observation_test.go'
    helper.write_text(PROVIDER_HELPER if provider else HELPER)
    subprocess.run([str(args.compiler.resolve().with_name('gofmt')), '-w', str(helper)], check=True)
    changed = [str(relative)]
    cluster_source = None
    if not provider:
        cluster_path = Path('tools/gomad/internal/backends/wasm/wasi/cluster.go')
        original_cluster = (source / cluster_path).read_bytes()
        if (derived / cluster_path).read_bytes() != original_cluster:
            raise ValueError('cluster source differs from committed derivation base')
        observed_cluster = instrument_cluster(original_cluster)
        (derived / cluster_path).write_bytes(observed_cluster)
        changed.append(str(cluster_path))
        cluster_source = dict(path=str(cluster_path), original_sha256=sha(original_cluster), observed_sha256=sha(observed_cluster))
    diff = subprocess.check_output(['git', '-C', str(derived), 'diff', '--', *changed])
    (work / 'test-only.diff').write_bytes(diff)
    (work / 'test-only-helper.go').write_bytes(helper.read_bytes())
    identity = dict(schema='gomad.live-broker-observation-derivation/v1' if provider else 'gomad.live-broker-observation-derivation/v2',
                    scope='derived test-only observation; not frozen baseline or qualification',
                    base_commit=base, source=str(source), derived=str(derived),
                    compiler_sha256=sha(args.compiler.resolve().read_bytes()), oracle_helper_sha256=sha(args.helper.resolve().read_bytes()),
                    original_test_sha256=sha(original), observed_test_sha256=sha(observed),
                    helper_sha256=sha(helper.read_bytes()), diff_sha256=sha(diff), roots=roots,
                    original_source_path=str(relative),
                    oracle_role='original-private-go-broker-reference-and-original-one-helper-provider' if provider else 'original-go-owner')
    if helper_binding is not None:
        identity['helper_build_receipt'] = helper_binding
    if cluster_source is not None:
        identity['cluster_source'] = cluster_source
        identity['derived_go_identity_role'] = 'actual derived embedded source identity; not frozen baseline identity'
    (work / 'derivation.json').write_text(json.dumps(identity, indent=2) + '\n')
    if args.derive_only:
        return
    environment = {key: value for key, value in os.environ.items() if not key.startswith('GOMAD_')}
    output = work / 'observations'
    output.mkdir(mode=0o700)
    environment.update(GOMAD_STOCK_GO=str(args.compiler.resolve()), GOMAD_WASM_TEST_HELPER=str(args.helper.resolve()),
                       GOMAD_BROKER_OBSERVATION_ROOT=str(output), GOENV='off', GOTOOLCHAIN='local', GOWORK='off')
    command = [str(args.compiler.resolve()), 'test', '-json', '-tags', 'test_dep', '-parallel=1', '-count=1', '-timeout=30m',
               '-run', '^(' + '|'.join(ROOTS[root] for root in roots) + ')$',
               './internal/backends/wasm/cluster' if provider else './internal/backends/wasm/wasi']
    started = datetime.now(timezone.utc).isoformat()
    result = subprocess.run(command, cwd=derived / 'tools/gomad', env=environment, capture_output=True, timeout=1900)
    (work / 'go-test.jsonl').write_bytes(result.stdout)
    (work / 'go-test.stderr').write_bytes(result.stderr)
    (work / 'command.json').write_text(json.dumps(dict(argv=command, exit=result.returncode, started_at=started, ended_at=datetime.now(timezone.utc).isoformat(), qualification=False), indent=2) + '\n')
    if result.returncode:
        raise RuntimeError(f'original Go assertions failed: exit {result.returncode}; see {work}')
    statuses = validate_events(result.stdout, roots)
    (work / 'original-assertions.json').write_text(json.dumps(statuses, indent=2) + '\n')
    if helper_binding is not None and provider_helper_receipt(args.helper_build_receipt,args.helper) != helper_binding:
        raise ValueError('provider helper build custody changed during actual Go test')
    index_observations(work)
    print(json.dumps(dict(observations=str(output), derivation=str(work / 'derivation.json'))))


if __name__ == '__main__':
    main()
