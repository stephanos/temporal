"""Read-only audit of worker and independent review receipts. Writes nothing."""
import datetime
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent


def sha(data):
    return hashlib.sha256(data).hexdigest()


def read_json(name):
    return json.loads((OUT / name).read_text())


started = datetime.datetime.now(datetime.timezone.utc).isoformat()
clock = time.monotonic()
bound = {
    'root-source-checks.json': '8609c23562facd83c292839e0a2030363be23257cb154d25167503d47ee75418',
    'run-gate.py': 'abcda3996f80446219b3a9f381de402af2d2e3b605022b56a7b4d59ad0bd54dd',
    'audit.py': 'a335be20c15ee54e53d51eeab783204443fbf974dc86d666bfe2367d2fbaacf6',
    'source-admission.json': '7681244eb5a84b517b0e72b261b59f15a73c139a985237dc97ae65fad10d2679',
    'evidence.json': '7faeb5c115c5d14863944b3f2f857e2e595d4ec623ebf70d04ecb07477550f19',
    'handover.md': '1560670e3052fa4624f58458899c6a183c8edebb461a202daf5bbd74beee3aa5',
}
assert all(sha((OUT / name).read_bytes()) == digest for name, digest in bound.items())
root_check = read_json('root-source-checks.json')['checks'][0]
assert root_check['cwd'] == str(ROOT)
assert sha(root_check['command'].encode()) == '982644872b109d1da3abfebb44fb3ceef2765e2d30b18ed8165e4f23ede8deb7'
worker = subprocess.run(root_check['command'], shell=True, cwd=ROOT, text=True,
                        capture_output=True, timeout=60)
assert worker.returncode == 0 and not worker.stderr, (worker.returncode, worker.stderr)
worker_result = json.loads(worker.stdout)
assert worker_result['status'] == 'verified_source_bindings'
assert all(sha((OUT / name).read_bytes()) == digest for name, digest in bound.items())
evidence = read_json('evidence.json')
source = {str(path.relative_to(ROOT)): sha(path.read_bytes())
          for path in sorted((ROOT / 'tools/gomad3/artifact').glob('*.go'))}
assert len(source) == 23 and source == evidence['source_final']
names = ('package', 'focused', 'retained-private-public', 'boundary', 'errortype', 'lint', 'static')
gates = []
for name in names:
    receipt_name = 'review-' + name + '.json'
    receipt = read_json(receipt_name)
    original = read_json('final-' + name + '.json')
    log = (OUT / receipt['log']).read_bytes()
    assert receipt['command'] == original['command']
    assert receipt['cwd'] == str(ROOT / 'tools/gomad3')
    assert receipt['environment'] == original['environment']
    assert receipt['tools'] == evidence['pinned_tools']
    assert receipt['config_sha256'] == evidence['config_sha256']
    assert all(sha(Path(path).read_bytes()) == digest for path, digest in receipt['tools'].items())
    assert sha((ROOT / '.github/.golangci.yml').read_bytes()) == receipt['config_sha256']
    assert receipt['source_before'] == source == receipt['source_after']
    assert receipt['stable'] and not receipt['timed_out'] and receipt['timeout_seconds'] == 600
    assert receipt['exit'] == (1 if name == 'lint' else 0)
    assert receipt['log'] == 'review-' + name + '.log'
    assert sha(log) == receipt['log_sha256']
    begin = datetime.datetime.fromisoformat(receipt['started'])
    end = datetime.datetime.fromisoformat(receipt['ended'])
    assert begin <= end and receipt['elapsed_seconds'] >= 0
    assert abs((end - begin).total_seconds() - receipt['elapsed_seconds']) < 1
    run_entries = None
    if receipt['command'][1] == 'test':
        run_entries = len(re.findall(rb'^=== RUN   ', log, re.MULTILINE))
        assert run_entries == evidence['test_run_entries']['final-' + name]
        assert not re.search(rb'^(--- FAIL:|--- SKIP:|FAIL\b)', log, re.MULTILINE)
        assert b'\nPASS\n' in log
    if name in ('static', 'errortype'):
        assert not log
    if name == 'lint':
        findings = [dict(path=path, line=int(line), column=int(column), message=message, linter=linter)
                    for path, line, column, message, linter in re.findall(
                        r'^(.*\.go):(\d+):(\d+): (.*) \((\w+)\)$', log.decode(), re.MULTILINE)]
        assert findings == evidence['lint']['after'] and len(findings) == 4
        assert b'4 issues:' in log
    if name == 'boundary':
        assert all(('--- PASS: ' + test + ' (').encode() in log
                   for test in evidence['boundaries']['executed'])
        assert b'TestRecordAndArtifactHaveSeparateOwners' not in log
    gates.append(dict(receipt=receipt_name, receipt_sha256=sha((OUT / receipt_name).read_bytes()),
                      log=receipt['log'], log_sha256=receipt['log_sha256'], exit=receipt['exit'],
                      elapsed_seconds=receipt['elapsed_seconds'], run_entries=run_entries))
assert all(sha((OUT / name).read_bytes()) == digest for name, digest in bound.items())
print(json.dumps(dict(
    status='SOURCE_PROGRESS_COMMIT_ONLY', base_commit=evidence['base_commit'], branch='gomad',
    script_sha256=sha(Path(__file__).read_bytes()), bound_inputs=bound,
    worker_read_only_command_ref='root-source-checks.json checks[0].command',
    worker_read_only_command_sha256=sha(root_check['command'].encode()),
    worker_audit_exit=worker.returncode, worker_receipts_verified=len(worker_result['receipts']),
    worker_source_recovery=worker_result['preservation']['recovered_baseline'],
    characterization_pool_recipe='restore original opened.Close defer; remove originalEntry capture; weaken only added pool condition to len(entries)!=1',
    characterization_pool_sha256=read_json('characterization-baseline.json')['source_before']['tools/gomad3/artifact/target_pool_test.go'],
    source_inventory_ref='evidence.json source_final', source_files_verified=23,
    protected_files=worker_result['protected_files'],
    protected_aggregate_sha256=worker_result['protected_aggregate_sha256'], fresh_gates=gates,
    reused_generator_receipt='final-validation.json',
    reused_generator_log_sha256=read_json('final-validation.json')['log_sha256'],
    lint_before=10, lint_after=4, resolved=6, introduced=0,
    formal_ship=False, flow_status='in_progress', owned_delegates=0, live_command_handles=0,
    started=started, ended=datetime.datetime.now(datetime.timezone.utc).isoformat(),
    elapsed_seconds=time.monotonic() - clock)))
