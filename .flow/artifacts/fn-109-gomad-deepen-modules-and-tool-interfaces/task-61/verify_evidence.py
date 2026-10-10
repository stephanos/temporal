import collections
import hashlib
import json
import pathlib
import re
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-61')
OUT = pathlib.Path(__file__).resolve().parent
if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
expected = {
    'baseline-builds': 0, 'baseline-lint': 1, 'baseline-vet': 0, 'baseline-format': 0,
    'baseline-original-base': 2, 'final-builds': 0, 'context-controls': 0,
    'muted-observer': 1, 'waited-false': 1, 'bypassed-wait-sequential': 1,
    'final-lint': 0, 'final-vet': 0, 'final-errortype': 0, 'final-format': 0,
    'final-fast': 0, 'final-original-base': 2,
}
records = {}
for name, code in expected.items():
    receipt = json.loads((OUT / (name + '.json')).read_text())
    assert receipt['exit_code'] == code, name
    assert receipt['terminal'] and receipt['source_unchanged'] and not receipt['timed_out'], name
    assert receipt.get('auxiliary_before', {}) == receipt.get('auxiliary_after', {}), name
    for path, sha in receipt.get('auxiliary_before', {}).items():
        assert hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest() == sha, path
    for suffix in ['stdout', 'stderr']:
        assert receipt[suffix + '_sha256'] == hashlib.sha256((OUT / (name + '.' + suffix)).read_bytes()).hexdigest(), name
    manifest_bytes = (OUT / receipt['source_manifest']).read_bytes()
    assert hashlib.sha256(manifest_bytes).hexdigest() == receipt['source_manifest_sha256'], name
    manifest = json.loads(manifest_bytes)
    assert hashlib.sha256(json.dumps(manifest, sort_keys=True).encode()).hexdigest() == receipt['source_before_sha256'], name
    for path, sha in receipt['tools'].items():
        assert hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest() == sha, path
    runner = 'run_gate.py' if name.startswith('baseline-') else 'run_final_gate.py'
    assert hashlib.sha256((OUT / runner).read_bytes()).hexdigest() == receipt['gate_runner_sha256'], name
    records[name] = {key: receipt[key] for key in ['command', 'exit_code', 'elapsed_seconds', 'source_before_sha256']}

def diagnostics(name):
    lines = (OUT / (name + '.stdout')).read_text().splitlines()
    return collections.Counter(tuple(lines[index:index + 3]) for index, line in enumerate(lines)
                               if re.match(r'^tools/[^:]+:\d+:\d+: ', line))

old, new = diagnostics('baseline-original-base'), diagnostics('final-original-base')
sleep = diagnostics('baseline-lint')
assert sum(old.values()) == 60 and sum(new.values()) == 59, (sum(old.values()), sum(new.values()))
assert sum(sleep.values()) == 1
assert next(iter(sleep))[0].startswith('tools/gomad3/toolchain/build_test.go:97:2: use of `time.Sleep` forbidden')
assert new == old - sleep
assert not diagnostics('final-lint')
for name in ['baseline-format', 'final-format']:
    assert (OUT / (name + '.stdout')).read_bytes() == b'', name
counts = {}
for name in ['baseline-builds', 'final-builds']:
    text = (OUT / (name + '.stdout')).read_text()
    counts[name] = {'top_level_passes': len(re.findall(r'^--- PASS:', text, re.M)),
                    'subtest_passes': len(re.findall(r'^\s+--- PASS:', text, re.M))}
    assert counts[name]['top_level_passes'] > 0 and 'TestBuildSerializesConcurrentSameKey' in text
    assert not re.search(r'--- (FAIL|SKIP):', text)
assert counts['baseline-builds'] == counts['final-builds']
for name, reason in [('muted-observer', 'second builder did not observe lock contention before timeout'),
                     ('waited-false', 'neither concurrent result reported lock waiting'),
                     ('bypassed-wait-sequential', 'neither concurrent result reported lock waiting')]:
    text = (OUT / (name + '.stdout')).read_text()
    assert reason in text and 'controlled cleanup drained every launched builder' in text
    assert 'did not exit after cancellation' not in text and 'panic: test timed out' not in text
old_manifest = json.loads((OUT / json.loads((OUT / 'baseline-builds.json').read_text())['source_manifest']).read_text())
new_manifest = json.loads((OUT / json.loads((OUT / 'final-builds.json').read_text())['source_manifest']).read_text())
assert set(old_manifest) == set(new_manifest)
assert [path for path in old_manifest if old_manifest[path] != new_manifest[path]] == ['tools/gomad3/toolchain/build_test.go']
for path, sha in new_manifest.items():
    assert hashlib.sha256((ROOT / path).read_bytes()).hexdigest() == sha, path
assert subprocess.run(['python3', str(OUT / 'verify_source.py')], cwd=ROOT, stdout=subprocess.DEVNULL).returncode == 0
print(json.dumps({'receipts': records, 'matched_build_test_counts': counts,
                  'original_base_lint_before': 60, 'original_base_lint_after': 59,
                  'removed': list(sleep.elements()), 'introduced': [],
                  'residual_diagnostic_blocks_exact': True,
                  'original_base_integrated_errortype': 'unreached after lint failure',
                  'mutants_rejected_for_fixed_reasons_and_drained': True,
                  'source_manifest_only_change': 'tools/gomad3/toolchain/build_test.go',
                  'runner_raw_tool_source_auxiliary_hashes_valid': True}, indent=2))
