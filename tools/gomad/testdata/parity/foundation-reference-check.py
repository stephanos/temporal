#!/usr/bin/env python3
"""Authenticate fresh per-lineage evidence before comparing guest termination classes."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import subprocess

p = argparse.ArgumentParser()
p.add_argument('--baseline', type=Path, required=True)
p.add_argument('--go-results', type=Path, required=True)
p.add_argument('--rust-results', type=Path, required=True)
args = p.parse_args()
corpus = Path(__file__).resolve().parent

def sha(raw):
    return hashlib.sha256(raw).hexdigest()

for expected, relative in [line.split() for line in (args.baseline / 'inputs.sha256').read_text().splitlines()]:
    assert sha((args.baseline / relative).read_bytes()) == expected, relative
subprocess.run(['python3', str(corpus.parents[1] / 'scripts/behavioral-parity'), '--corpus', str(corpus), '--foundation-root', str(args.rust_results), '--foundation-result', str(args.rust_results / 'foundation-result.json')], check=True)
index_raw = (corpus / 'foundation-execution.json').read_bytes()
index = json.loads(index_raw)
go = json.loads((args.go_results / 'reference.json').read_bytes())
assert go['schema'] == 'gomad.foundation-go-reference/v1'
assert go['corpus_sha256'] == sha(index_raw)
report = json.loads((args.rust_results / 'foundation-result.json').read_bytes())
expected = [(case, seed) for case in index['cases'] for seed in index['seeds']]
assert len(expected) == len(go['records']) == len(report['records']) == 12
for (case, seed), reference, item in zip(expected, go['records'], report['records']):
    assert reference['name'] == case['name'] and reference['seed'] == seed
    raw_name = f"{case['name']}-seed{seed}.json"
    assert reference['raw_result'] == raw_name
    raw = (args.go_results / raw_name).read_bytes()
    assert sha(raw) == reference['raw_sha256'], raw_name
    decoded = json.loads(raw)
    assert reference['helper_sha256'] == 'b25439140f0c850a82f99471c9fd48ac864569a75e407a76850873f6ece055a1'
    assert reference['source_revision'] == '1d7b37f00a74a85399993f1edcad6fe742e778e3'
    assert decoded['Reaped'] and not decoded['Cancelled']
    assert decoded['Termination'] == reference['raw_termination']
    assert decoded['ExitCode'] == reference['exit_code']
    assert decoded['WatchdogTimeout'] == reference['watchdog']
    termination = 'watchdog_timeout' if decoded['WatchdogTimeout'] else decoded['Termination']
    assert termination == reference['termination']
    module_sha = sha((corpus / case['module']).read_bytes())
    assert reference['module_sha256'] == module_sha
    if decoded['Started'] is not None:
        assert decoded['Started']['module_sha256'] == module_sha
    rust = json.loads((args.rust_results / item['record_file']).read_bytes())
    assert rust['seed'] == seed and rust['identity']['target'] == module_sha
    assert rust['limits']['fuel'] == case['fuel'] and rust['limits']['memory_bytes'] == case['memory_bytes']
    assert rust['limits']['wall_millis'] == case['wall_millis']
    assert rust['outcome']['termination'] == termination, case['name'] + '.termination'
    assert rust['outcome']['exit_code'] == decoded['ExitCode'], case['name'] + '.exit_code'
    artifact = args.rust_results / item['artifact']
    for go_name, rust_name in [('Stdout', 'output/stdout'), ('Stderr', 'output/stderr'), ('Transcript', 'evidence/transcript')]:
        payload = base64.b64decode(decoded[go_name] or '', validate=True)
        actual = (artifact / rust_name).read_bytes() if rust_name in rust['streams'] else b''
        assert payload == actual, case['name'] + '.' + go_name
        if go_name == 'Transcript' and payload:
            assert sha(payload) == bytes(decoded['TranscriptSHA256']).hex()
            assert rust['streams'][rust_name]['records'] == decoded['Calls']
print('authenticated Go/Rust termination parity: 12/12 fresh executions')
