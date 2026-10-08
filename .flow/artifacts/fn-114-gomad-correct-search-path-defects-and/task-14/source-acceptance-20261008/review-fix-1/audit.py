import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
HERE = Path(__file__).resolve().parent
PARENT = HERE.parent
BASE = '1fcef5ea4bfbf3ec5f7279a8a6c67070d5159a9d'
original = json.loads((PARENT / 'final-docs.json').read_text())
docs = {f'tools/gomad3/{name}.md' for name in ('README', 'CLI', 'SPEC', 'ARCHITECTURE', 'TUTORIAL')}
docs.update(('.plans/GOMAD_CMP.md', '.plans/GOMAD_NEXT.md'))
for path, expected in original['inputs'].items():
    data = (ROOT / path).read_bytes()
    if path in docs:
        data = subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)
    assert hashlib.sha256(data).hexdigest() == expected, path
source_path = PARENT / 'audit-docs.py'
source = source_path.read_text()
old_destination = "output = HERE / 'final-docs.json'"
assert source.count(old_destination) == 1
routed = source.replace(old_destination, "output = HERE / 'review-fix-1/final-docs.json'")
namespace = {'__file__': str(source_path), '__name__': '__main__'}
try:
    exec(compile(routed, str(source_path), 'exec'), namespace)
except SystemExit as result:
    assert result.code == 0, result.code
report_path = HERE / 'final-docs.json'
report = json.loads(report_path.read_text())
fixture_path = 'tools/gomad3/internal/gomadtool/conformance/testdata/select_readiness/main.go'
fixture = (ROOT / fixture_path).read_text()
nil_shape = fixture.split('case "nil-channel":', 1)[1].split('case "timer-channel-due":', 1)[0]
assert nil_shape.count('\n\t\tcase <-') == 3
assert 'var disabled chan int' in nil_shape
assert all('case <-' + channel + ':' in nil_shape for channel in ('disabled', 'first', 'second'))
shape_source = (ROOT / 'tools/gomad3/choice/no_op_select.go').read_text()
assert shape_source.count('{PolledCases: 2, Readiness:') == 7
assert 'Ready: 1, NilChannel: true' in shape_source
predicate = (ROOT / 'tools/gomad3/runner/internal/exploration/choice/select_readiness.go').read_text()
assert 'PolledCases: trace.Decisions[end-1].Alternatives' in predicate
for path in docs - {'.plans/GOMAD_NEXT.md'}:
    text = (ROOT / path).read_text()
    assert 'two-case' not in text and 'multi-case' not in text, path
    normalized = ' '.join(text.split())
    assert 'polled non-nil' in normalized and 'three or more polled cases' in normalized, path
report['inputs'][fixture_path] = hashlib.sha256((ROOT / fixture_path).read_bytes()).hexdigest()
report['snapshot'] = 'Current review-fix-1 documents; original final-docs.json remains bound to checkpoint ' + BASE
report['correction_proof'] = {
    'fixture': fixture_path, 'source_clauses': 3, 'non_nil_polled_cases': 2,
    'shape_list': 'tools/gomad3/choice/no_op_select.go', 'listed_shapes': 7,
    'predicate': 'tools/gomad3/runner/internal/exploration/choice/select_readiness.go',
    'predicate_key': 'PolledCases: trace.Decisions[end-1].Alternatives',
    'output_routing_only': 'The original audit body runs unchanged except its final report destination.'}
report_path.write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps({'current_inputs': len(report['inputs']), 'source_clauses': 3,
                  'polled_non_nil_cases': 2, 'errors': report['errors']}))
