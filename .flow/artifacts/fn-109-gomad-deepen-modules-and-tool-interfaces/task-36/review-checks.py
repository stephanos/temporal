import datetime
import hashlib
import json
import os
from pathlib import Path
import shlex
import subprocess
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PROOF = Path(__file__).resolve().parent
ADMISSION = json.loads((PROOF / 'root-admission.json').read_text())
ENV = json.loads((PROOF / 'final-package.receipt.json').read_text())['environment']


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def freeze():
    assert len(ADMISSION['protected_files']) == 1044
    assert all(sha(ROOT / p) == h for p, h in ADMISSION['protected_files'].items())
    tools = dict(json.loads((PROOF / 'final-package.receipt.json').read_text())['source_before']['tools'])
    tools[ENV['set']['PATH'].split(':')[0] + '/gofmt'] = sha(ENV['set']['PATH'].split(':')[0] + '/gofmt')
    assert all(sha(p) == h for p, h in tools.items())
    return {'sources': {p: sha(ROOT / p) for p in ADMISSION['source_paths']}, 'tools': tools, 'protected_count': 1044,
            'admission_sha256': sha(PROOF / 'root-admission.json'), 'head': subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip(),
            'index_sha256': sha(ROOT / '.git/index') if (ROOT / '.git/index').exists() else None}


evidence = json.loads((PROOF / 'evidence.json').read_text())
verified = []
for entry in evidence['receipts']:
    receipt_path = ROOT / entry['path']
    assert sha(receipt_path) == entry['sha256']
    receipt = json.loads(receipt_path.read_text())
    assert receipt['argv'] == shlex.split(receipt['command'])
    assert receipt['environment'] == ENV
    assert receipt['source_before'] == receipt['source_after'] and receipt['stability']
    assert receipt['source_before']['admission_sha256'] == sha(PROOF / 'root-admission.json')
    assert receipt['source_before']['protected_count'] == 1044
    assert receipt['source_before']['protected_unchanged']
    assert all(sha(p) == h for p, h in receipt['source_before']['tools'].items())
    assert sha(ROOT / receipt['log']) == receipt['log_sha256']
    start, end = map(datetime.datetime.fromisoformat, (receipt['start'], receipt['end']))
    assert start.utcoffset().total_seconds() == end.utcoffset().total_seconds() == 0
    assert start <= end and abs((end - start).total_seconds() - receipt['elapsed_seconds']) < 0.1
    verified.append({'path': entry['path'], 'sha256': entry['sha256'], 'start': receipt['start'], 'end': receipt['end'], 'exit_code': receipt['exit_code'], 'log_sha256': receipt['log_sha256']})

controls = json.loads((PROOF / 'controls-focused.receipt.json').read_text())
capture = json.loads((PROOF / 'capture-focused.receipt.json').read_text())
final = json.loads((PROOF / 'final-package.receipt.json').read_text())
production, tests = ADMISSION['source_paths']
assert capture['source_before']['sources'][production] == controls['source_before']['sources'][production] == ADMISSION['source_before_sha256'][production]
assert controls['source_before']['sources'][tests] == final['source_before']['sources'][tests] == sha(ROOT / tests)
assert datetime.datetime.fromisoformat(capture['end']) < datetime.datetime.fromisoformat(controls['start']) < datetime.datetime.fromisoformat(controls['end']) < datetime.datetime.fromisoformat(final['start'])
assert controls['exit_code'] == capture['exit_code'] == final['exit_code'] == 0
assert sha(PROOF / 'audit-environment.log') == 'ecc5d80c8f231082f3eec8d2705ed3d3ac9c9b6c7b813c310e6cc39b332d6b35'
assert (PROOF / 'audit-environment.log').read_bytes().splitlines()[9] == b''

audit_path = PROOF / 'source_audit.py'
audit = audit_path.read_text()
write = "(PROOF / 'source-preservation.json').write_text(json.dumps(report, indent=2) + '\\n')"
assert audit.count(write) == 1
audit = audit.replace(write, '', 1)
scope = {'__file__': str(audit_path)}
exec(compile(audit, str(audit_path), 'exec'), scope)
assert scope['report'] == json.loads((PROOF / 'source-preservation.json').read_text())

report = {'verification': 'PASS', 'source_audit_sha256': sha(audit_path), 'source_audit_execution': 'in-memory exact body with its sole source-preservation write removed', 'worker_receipts': verified,
          'source_preservation_reconstructed': scope['report'], 'literal_capture_precedes_controls_precedes_final': True,
          'review_script_sha256': sha(__file__), 'before': freeze(), 'gates': []}
environment = dict(os.environ)
for key in ENV['unset']:
    environment.pop(key, None)
environment.update(ENV['set'])
for name in ('package', 'focused', 'boundaries', 'lint', 'errortype', 'gofmt'):
    worker = json.loads((PROOF / ('final-' + name + '.receipt.json')).read_text())
    before = freeze()
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    tick = time.monotonic()
    log = PROOF / ('review-' + name + '.log')
    assert not log.exists()
    with log.open('wb') as output:
        result = subprocess.run(worker['argv'], cwd=worker['cwd'], env=environment, stdout=output, stderr=subprocess.STDOUT, timeout=600)
    after = freeze()
    receipt = {'argv': worker['argv'], 'command': worker['command'], 'cwd': worker['cwd'], 'environment': ENV,
               'start': started, 'end': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'elapsed_seconds': round(time.monotonic() - tick, 3),
               'exit_code': result.returncode, 'log': str(log.relative_to(ROOT)), 'log_sha256': sha(log), 'source_before': before, 'source_after': after, 'stability': before == after}
    (PROOF / ('review-' + name + '.receipt.json')).write_text(json.dumps(receipt, indent=2) + '\n')
    print(name, result.returncode, receipt['elapsed_seconds'], flush=True)
    assert result.returncode == 0 and before == after
    if name == 'gofmt':
        assert log.stat().st_size == 0
    report['gates'].append(receipt)
result = subprocess.run(['git', '-C', str(ROOT), 'diff', '--check', ADMISSION['base_commit'], '--', production, tests], text=True, capture_output=True)
assert result.returncode == 0 and result.stdout == result.stderr == ''
report['product_diff_check'] = {'argv': result.args, 'exit_code': 0, 'stdout': '', 'stderr': ''}
report['after'] = freeze()
assert report['after'] == report['before']
(PROOF / 'review-checks.json').write_text(json.dumps(report, indent=2) + '\n')
