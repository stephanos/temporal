import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
ARCH = ROOT / 'tools/gomad3/internal/gomadtool/architecture'

def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()

evidence = json.loads((OUT / 'evidence.json').read_text())
admission = json.loads((OUT / 'source-admission.json').read_text())
assert evidence['base_commit'] == admission['base_commit']
paths = subprocess.check_output(['git', 'ls-files', 'tools/gomad3', 'tools/gomad3sim',
    'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml'], cwd=ROOT, text=True).splitlines()
protected = {path: digest(ROOT / path) for path in paths if path not in admission['protected_exclusions']}
assert len(protected) == admission['protected_files']
assert hashlib.sha256(json.dumps(protected, sort_keys=True).encode()).hexdigest() == admission['protected_aggregate_sha256']
for path, expected in admission['protected_original_documents'].items():
    assert digest(ROOT / path) == expected, path
for reference in evidence['snapshots']:
    assert digest(OUT / reference['path']) == reference['sha256']
    stage = json.loads((OUT / reference['path']).read_text())
    for name in ('effects.go', 'standard.go', 'error_provenance_test.go'):
        saved = OUT / 'sources' / Path(reference['path']).stem / name
        key = str((ARCH / name).relative_to(ROOT))
        if key in stage['source']:
            assert digest(saved) == stage['source'][key], saved
        else:
            assert not saved.exists(), saved
baseline = json.loads((OUT / 'baseline.json').read_text())['source']
final = json.loads((OUT / 'writer-stage.json').read_text())['source']
stages = {name: json.loads((OUT / (name + '.json')).read_text())['source']
    for name in ('baseline', 'test-first', 'unwrap-stage', 'conversion-stage', 'writer-stage')}
for name in ('effects.go', 'standard.go'):
    path = str((ARCH / name).relative_to(ROOT))
    admitted = subprocess.check_output(['git', 'show', admission['base_commit'] + ':' + path], cwd=ROOT)
    assert hashlib.sha256(admitted).hexdigest() == baseline[path]
    assert admitted == (OUT / 'sources/baseline' / name).read_bytes()
actual = {str(path.relative_to(ROOT)): digest(path) for path in sorted(ARCH.glob('*.go'))}
assert len(baseline) == 14 and len(final) == 15 and actual == final
for path, expected in baseline.items():
    if path not in admission['protected_exclusions']:
        assert actual[path] == expected, path
logs = {}
for reference in evidence['command_receipts']:
    assert digest(OUT / reference['path']) == reference['sha256'], reference['path']
    receipt = json.loads((OUT / reference['path']).read_text())
    assert receipt['exit'] == reference['expected_exit'] and not receipt['timed_out']
    assert receipt['stable'] and receipt['source_before'] == receipt['source_after']
    assert receipt['protected_before'] == receipt['protected_after']
    assert receipt['protected_before']['files'] == 1042
    assert receipt['protected_before']['aggregate_sha256'] == admission['protected_aggregate_sha256']
    assert receipt['protected_after']['aggregate_sha256'] == admission['protected_aggregate_sha256']
    assert digest(OUT / receipt['log']) == receipt['log_sha256']
    assert digest(ROOT / '.github/.golangci.yml') == receipt['config_sha256']
    for tool, expected in receipt['tools'].items():
        assert digest(tool) == expected, tool
    assert receipt['environment']['GOWORK'] == 'off'
    assert receipt['environment']['GOTOOLCHAIN'] == 'local'
    assert receipt['environment']['GOPROXY'] == 'off'
    assert receipt['environment']['GOFLAGS'] == ''
    assert receipt['environment']['GOMADSEED'] is None and receipt['environment']['GOMAD3_CHILD_SEED'] is None
    name = Path(reference['path']).stem
    stage = 'writer-stage'
    if name.startswith('baseline-'):
        stage = 'baseline'
    elif name == 'causal-red':
        stage = 'test-first'
    elif name in ('unwrap-stage-test', 'conversion-stage-test', 'writer-stage-test'):
        stage = name.removesuffix('-test')
    assert receipt['source_before'] == receipt['source_after'] == stages[stage], name
    logs[Path(reference['path']).stem] = (OUT / receipt['log']).read_text()
assert logs['baseline-lint'] == logs['final-lint']
assert logs['baseline-errortype'] == logs['final-errortype'] == ''
assert logs['final-static'] == ''
red = logs['causal-red']
assert red.count('stock-host causal fixture:') == 29
assert len(re.findall(r'^    --- FAIL: TestErrorProvenance', red, re.M)) == 14
assert 'invalid stock-host fixture' not in red
assert 'invalid fixture edges' not in red
green = logs['final-causal']
assert green.count('stock-host causal fixture:') == 29
assert green.count('package edges=0 effects=') == 58
assert len(re.findall(r'^    --- PASS: TestErrorProvenance', green, re.M)) == 29
failed = {
    'TestErrorProvenanceUnwrap/single-named-slice', 'TestErrorProvenanceUnwrap/joined-named-slice',
    'TestErrorProvenanceUnwrap/wrapped-named-slice', 'TestErrorProvenanceUnwrap/as-named-slice',
    'TestErrorProvenanceUnwrap/nil-slice-node', 'TestErrorProvenanceConversion/typed-source',
    'TestErrorProvenanceConversion/supplier', 'TestErrorProvenanceConversion/interface-dynamic-type',
    'TestErrorProvenanceConversion/external-typed-source', 'TestErrorProvenanceWriter/fprint',
    'TestErrorProvenanceWriter/fprintf', 'TestErrorProvenanceWriter/fprintln',
    'TestErrorProvenanceWriter/wrapped-is', 'TestErrorProvenanceWriter/unknown-return',
}
assert set(re.findall(r'^    --- FAIL: (TestErrorProvenance\S+) ', red, re.M)) == failed
dirty = failed - {'TestErrorProvenanceWriter/unknown-return'} | {
    'TestErrorProvenanceUnwrap/direct-named-slice', 'TestErrorProvenanceUnwrap/true-multi',
    'TestErrorProvenanceConversion/function-payload', 'TestErrorProvenanceConversion/slice-alias',
    'TestErrorProvenanceConversion/pointer-alias',
}
cases = {}
for label, log in (('red', red), ('green', green)):
    cases[label] = dict(re.findall(r'^=== RUN   (TestErrorProvenance[^/\n]+/[^ \n]+)\n(.*?)(?=^=== RUN   |\Z)', log, re.M | re.S))
    assert len(cases[label]) == 29
    for name, body in cases[label].items():
        calls = 1 if name in dirty else 0
        count = 7 if name in {'TestErrorProvenanceWriter/fprint', 'TestErrorProvenanceWriter/fprintf',
            'TestErrorProvenanceWriter/fprintln', 'TestErrorProvenanceWriter/nil-return'} else 0
        assert re.findall(r'actual callbacks=(\d+) writer count=(\d+)', body) == [(str(calls), str(count))], name
        assert '--- PASS: TestBehavior' in body, name
        if label == 'red' and name in failed:
            for platform in ('linux amd64', 'darwin arm64'):
                assert re.search(r'returned callback escaped \{' + platform + r'\} \([^\n]*\): \[\]', body), name
        if label == 'green':
            for platform in ('linux amd64', 'darwin arm64'):
                effect = re.search(r'metadata \{' + platform + r'\} package edges=0 effects=(.*)', body).group(1)
                if calls == 1:
                    assert 'host-effect' in effect and 'record.Check' in effect and 'time.Now' in effect, name
                    if name.startswith('TestErrorProvenanceUnwrap/'):
                        callback = 'canonicaljson.Leaf.As' if name.endswith('/as-named-slice') else 'canonicaljson.Leaf.Is'
                    elif name.endswith('/external-typed-source'):
                        callback = 'dependency.Leaf.Error'
                    elif name.endswith(('/function-payload', '/slice-alias', '/pointer-alias')):
                        callback = 'canonicaljson.Dirty'
                    else:
                        callback = 'canonicaljson.Leaf.Is' if name.endswith('/wrapped-is') else 'canonicaljson.Leaf.Error'
                    assert callback in effect, name
                elif name == 'TestErrorProvenanceWriter/unknown-return':
                    assert 'unresolved-effect' in effect and 'unknown dynamic callback receiver' in effect and 'record.Check' in effect
                else:
                    assert effect == '[]', name
for name in evidence['required_boundaries']:
    combined = logs['final-boundaries'] + logs['final-public-consumer-boundaries']
    assert len(re.findall(r'^=== RUN   ' + re.escape(name) + r'$', combined, re.M)) == 1, name
    assert re.search(r'^--- PASS: ' + re.escape(name) + r' \(', combined, re.M), name
for name in ('TestEffectCallbackContextsAndUnwrapReturns', 'TestDependencyInitialization', 'TestThirdPartyInitialization',
    'TestStandardStartupIdentity', 'TestMemorySummarySourceIdentity', 'TestCallbackContainerMutations',
    'TestRangeAssignmentSlots', 'TestImplicitCallbackPrecedence', 'TestPureMemoryFormattingAndJSON'):
    assert re.search(r'^--- PASS: ' + name + r' \(', logs['final-package'], re.M), name
environment = json.loads(logs['environment'])
assert environment['GOOS'] == 'linux' and environment['GOARCH'] == 'arm64' and environment['GOVERSION'] == 'go1.27.1'
assert not (ROOT / 'tools/gomad3/.toolchain/bin/go').exists()
print(json.dumps(dict(protected_files=len(protected), architecture_sources=len(actual), causal_fixtures=29,
    metadata_observations=58, required_boundaries=5, causal_red_cases=14, inherited_lint=4,
    introduced_lint=0, resolved_lint=0, errortype=0, qualification='developmental linux/arm64 only',
    immutable_evidence='verified; audit writes no files'), sort_keys=True))
