import hashlib
import json
import pathlib
import subprocess

ROOT = pathlib.Path('/Users/stephan/Workspace/skunkworks/gomad-fn109-next.nDsJ0uMh/task-62').resolve()
OUT = pathlib.Path(__file__).resolve().parent
BASE = '7b75ae312a6641c6b00afbb63f44ae9a9dd0c2b2'
RUNNER = 'tools/gomad3/runner/runner_test.go'
CONTROL = 'tools/gomad3/runner/progress_start_test.go'

if pathlib.Path.cwd().resolve() != ROOT:
    raise SystemExit('Wrong workspace')
records = {}
for name in ['baseline-healthy', 'baseline-sentinel', 'baseline-vet', 'baseline-format',
             'baseline-errortype', 'baseline-configured-lint', 'baseline-make-fast',
             'red-controls', 'final-controls', 'final-healthy', 'final-sentinel',
             'sensitivity-original-helper', 'final-vet', 'final-errortype',
             'final-format', 'final-configured-lint', 'final-make-fast']:
    record = json.loads((OUT / (name + '.json')).read_text())
    assert record['terminal'] and record['source_unchanged'], name
    for stream in ['stdout', 'stderr']:
        assert hashlib.sha256((OUT / (name + '.' + stream)).read_bytes()).hexdigest() == record[stream + '_sha256'], name
    assert hashlib.sha256((OUT / 'run_gate.py').read_bytes()).hexdigest() == record['gate_runner_sha256'], name
    records[name] = record

baseline = records['baseline-healthy']['source_files']
final = records['final-controls']['source_files']
assert final.keys() - baseline.keys() == {CONTROL}
assert not baseline.keys() - final.keys()
assert {path for path in baseline if baseline[path] != final[path]} == {RUNNER}
for probe, healthy, expected in [('baseline-sentinel', 'baseline-healthy', RUNNER),
                                 ('final-sentinel', 'final-controls', RUNNER),
                                 ('sensitivity-original-helper', 'final-controls', CONTROL)]:
    actual = records[probe]['source_files']
    reference = records[healthy]['source_files']
    assert actual.keys() == reference.keys(), probe
    assert {path for path in actual if actual[path] != reference[path]} == {expected}, probe
assert records['red-controls']['source_files'][CONTROL] == records['sensitivity-original-helper']['source_files'][CONTROL]
assert records['baseline-configured-lint']['stdout_sha256'] == records['final-configured-lint']['stdout_sha256']
assert all(record['tools'] == records['baseline-healthy']['tools'] for record in records.values())
assert all(record['source_files'] == final for name, record in records.items()
           if name in ['final-controls', 'final-healthy', 'final-vet', 'final-errortype',
                       'final-format', 'final-configured-lint', 'final-make-fast'])

base = subprocess.check_output(['git', 'show', BASE + ':' + RUNNER], cwd=ROOT)
current = (ROOT / RUNNER).read_bytes()
start = b'func TestRunReportsPeriodicProgressWhileTargetIsRunning'
end = b'func TestRunReportsPassiveSemanticCoverage'
old_prefix, old_tail = base.split(start, 1)
new_prefix, new_tail = current.split(start, 1)
old_body, old_suffix = old_tail.split(end, 1)
new_body, new_suffix = new_tail.split(end, 1)
assert old_prefix == new_prefix and old_suffix == new_suffix
old_heartbeat = old_body.split(b'\theartbeats := 0\n', 1)[1].split(b'\tclose(executor.release)\n', 1)[0]
new_heartbeat = new_body.split(b'\theartbeats := 0\n', 1)[1].split(b'\trelease()\n', 1)[0]
assert old_heartbeat == new_heartbeat
assert old_body.split(b'\tclose(executor.release)\n', 1)[1] == new_body.split(b'\trelease()\n', 1)[1]
auxiliary = {}
for path in ['go.mod', 'go.sum']:
    original = subprocess.check_output(['git', 'show', BASE + ':' + path], cwd=ROOT)
    actual = (ROOT / path).read_bytes()
    assert original == actual, path
    auxiliary[path] = hashlib.sha256(actual).hexdigest()
print(json.dumps({
    'receipts_checked': len(records), 'protected_bound_inputs': len(baseline) - 1,
    'changed': [RUNNER], 'added': [CONTROL],
    'outside_periodic_fixture_byte_identical': True,
    'heartbeat_block_byte_identical': True, 'completion_assertion_byte_identical': True,
    'probe_deltas_exact': True, 'red_and_mutant_control_source_identical': True,
    'inherited_runner_lint_stdout_identical': True,
    'tools_identical_at_all_retained_observations': True,
    'protected_overlay_files': sum(path.startswith('tools/gomad3/toolchain/runtime/overlay/') for path in baseline),
    'root_module_inputs_identical_to_immutable_base': auxiliary,
}, indent=2))
