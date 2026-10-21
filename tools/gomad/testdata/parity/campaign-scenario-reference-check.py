#!/usr/bin/env python3
"""Authenticate fresh public Go CLI captures and compare their scenario behavior with Rust."""
import argparse
import base64
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
from pathlib import Path
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent


def load(name, file):
    loader = SourceFileLoader(name, str(CORPUS / file))
    module = module_from_spec(spec_from_loader(name, loader))
    loader.exec_module(module)
    return module


go = load('scenario_go_reference', 'campaign-reference-check.py')
rust = load('scenario_rust_reference', 'campaign-check.py')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('baseline', 'go-work', 'rust-work', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    for line in (args.baseline / 'inputs.sha256').read_text().splitlines():
        expected, relative = line.split()
        go.equal(go.digest((args.baseline / relative).read_bytes()), expected, 'baseline.' + relative)
    cases = {}
    for name in ('durable', 'network', 'assertion', 'report-only', 'crash-assertion', 'crash-report-only'):
        expected_status = 1 if name in ('assertion', 'crash-assertion') else 0
        go.equal(int((args.go_work / (name + '.status')).read_text()), expected_status, name + '.Go.status')
        root = next((args.go_work / (name + '-artifacts')).rglob('store/log')).parent.parent
        original = go.authenticate_campaign(root)
        go.equal(len(original['records']), 1, name + '.Go.records')
        execution = original['records'][0]
        artifact = root / (execution.get('artifact') or execution['success_artifact'])
        manifest = go.authenticate_artifact(artifact)
        bundle = json.loads((artifact / 'backend/evidence.bin').read_bytes())
        go.equal(bundle['schema'], 'gomad.wasm-simulation-retained/v2', name + '.Go.cluster_schema')
        cluster = json.loads(base64.b64decode(bundle['record']))
        candidate_root = next((args.rust_work / (name + '-0/campaigns')).glob('campaign-*'))
        candidate = rust.authenticate_campaign(candidate_root)
        retained = candidate['executions'][0]
        candidate_artifact = candidate_root / retained['artifact']
        inputs = rust.canonical_file(candidate_artifact / 'evidence/cluster-replay.json')
        go.equal([dict(action=row['action'], handle=row['handle']) for row in bundle['lifecycle']],
                 inputs['configuration']['steps'], name + '.lifecycle')
        go.equal(go.semantic(original['records'])[0]['outcome'], retained['record']['outcome']['class'], name + '.outcome')
        go.equal(execution['termination'], retained['record']['outcome']['termination'], name + '.termination')
        events = [json.loads(line) for line in (args.go_work / (name + '.stdout')).read_bytes().splitlines()]
        final = next(event for event in events if event['type'] == 'result')
        go.equal(final['classification'], retained['record']['outcome']['class'], name + '.Go.public_classification')
        output_hashes = {}
        for observation in cluster['outputs']:
            raw = base64.b64decode(observation['bytes'])
            go.equal(go.digest(raw), observation['full_sha256'].removeprefix('sha256:'), name + '.Go.output_hash')
            handle = observation['handle']
            relative = f"evidence/cluster/{handle['node']}/{handle['incarnation']}/{observation['stream']}"
            go.equal((candidate_artifact / relative).read_bytes(), raw, name + '.' + relative)
            output_hashes[relative] = go.digest(raw)
        reports = {}
        for oracle in cluster['oracles'] or []:
            if not oracle.get('soft_assertion'):
                continue
            fields = {}
            for evidence in oracle['evidence']:
                raw = base64.b64decode(evidence['value'])
                encoded = b'gomad-oracle-evidence/v1\0' + rust.parity.canonical(evidence['value'])
                go.equal(go.digest(encoded), evidence['full_sha256'].removeprefix('sha256:'), name + '.Go.oracle_evidence')
                fields[evidence['label']] = raw
            reports[fields['instance'].decode()] = dict(messages=json.loads(fields['messages']) or [],
                report_only=oracle.get('report_only', False), inconclusive=oracle.get('inconclusive', False))
        if reports:
            observed = rust.canonical_file(candidate_artifact / 'evidence/soft-assertions.json')
            go.equal(reports, {f"{row['handle']['node']}/{row['handle']['incarnation']}": row['report']
                              for row in observed['incarnations']}, name + '.handled_soft_assertions')
        go.equal(int((args.go_work / (name + '-replay.status')).read_text()), expected_status, name + '.Go.replay_status')
        replay = (args.go_work / (name + '-replay.stdout')).read_bytes()
        go.equal(b'reproduced=true ' in replay, True, name + '.Go.replay_reproduced')
        cases[name] = dict(outcome=go.semantic(original['records'])[0], output_sha256=output_hashes,
            soft_assertions=reports, Go_record_hash=manifest['record_hash'],
            Rust_record_sha256=rust.parity.digest(rust.parity.canonical(retained['record'])),
            Go_capture_sha256={suffix: go.digest((args.go_work / (name + suffix)).read_bytes())
                               for suffix in ('.stdout', '.stderr', '.status', '-replay.stdout', '-replay.status')})
    result = dict(schema='gomad.scenario-public-cli-reference/v1', scope='scenario reference; no all-verb API waiver',
        producer='fresh original public Go CLI captures; no provider wrapper', cases=cases,
        Go_binary_sha256=go.digest((args.baseline / 'bin/gomad').read_bytes()))
    args.output.write_bytes(rust.parity.canonical(result) + b'\n')
    print('six fresh public Go scenario captures authenticated; lifecycle, outputs, handled assertions and replay match Rust')


if __name__ == '__main__':
    main()
