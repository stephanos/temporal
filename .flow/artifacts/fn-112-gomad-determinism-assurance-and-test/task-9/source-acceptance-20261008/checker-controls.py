import contextlib
import copy
import importlib.util
import io
import json
from pathlib import Path

root = Path(__file__).resolve().parents[5]
original = root / '.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-9'
spec = importlib.util.spec_from_file_location('original_mapping_checker', original / 'mapping-check.py')
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)
pairs = [str(original / ('behaviors-' + phase + '.tsv')) + ':' + package
         for package in ('deterministicio', 'root', 'cli', 'runner')
         for phase in ('before', 'after')]
baseline = io.StringIO()
with contextlib.redirect_stdout(baseline):
    status = checker.main(pairs)
assert status == 0, baseline.getvalue()
print(json.dumps({'canonical_original_exit': status, 'stdout': baseline.getvalue()}))

old = 'TestValidateConfigRequiresBoundedSingleSeedChoiceExploration/multiple_seeds'
new = checker.GOLDEN + 'choice_multiple_seeds'
base_before = {old: 'pass'}
base_after = {new: 'pass'}
errors = checker.golden_errors()
load = checker.load
golden = checker.golden_errors
controls = [
    ('unmapped_behavior', {'TestUnmappedPreservationControl': 'pass'}, {}, errors, 'UNMAPPED'),
    ('missing_replacement', base_before, {}, errors, 'MISSING'),
    ('worse_status', base_before, {new: 'fail'}, errors, 'STATUS WORSE'),
    ('wrong_golden_error', base_before, base_after, {**errors, new: 'a different failure'}, 'GOLDEN ERROR MISMATCH'),
    ('missing_golden_row', base_before, base_after, {k: v for k, v in errors.items() if k != new}, 'GOLDEN ROW MISSING'),
]
for label, before, after, golden_errors, expected in controls:
    checker.load = lambda path: copy.deepcopy(before if path == 'before:runner' else after)
    checker.golden_errors = lambda: golden_errors
    stream = io.StringIO()
    with contextlib.redirect_stdout(stream):
        status = checker.main(['before:runner', 'after:runner'])
    assert status == 1 and expected in stream.getvalue(), (label, status, stream.getvalue())
    print(json.dumps({'control': label, 'exit': status, 'rejected': True, 'stdout': stream.getvalue()}))
checker.load = load
checker.golden_errors = golden
