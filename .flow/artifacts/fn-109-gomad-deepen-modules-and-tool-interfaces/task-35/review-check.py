import contextlib
import datetime
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import sys
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
PROOF = Path(__file__).resolve().parent
admission = json.loads((PROOF / 'root-admission.json').read_text())
evidence = json.loads((PROOF / 'evidence.json').read_text())
template = json.loads((PROOF / 'final-package.receipt.json').read_text())


def sha(data):
    return hashlib.sha256(data).hexdigest()


def freeze():
    assert all(sha((ROOT / p).read_bytes()) == h for p, h in admission['protected_files'].items())
    assert all(sha(Path(p).read_bytes()) == h for p, h in template['source_before']['tools'].items())
    sources = {p: sha((ROOT / p).read_bytes()) for p in evidence['sources']}
    assert sources == evidence['sources']
    assert all(sha((ROOT / p).read_bytes()) == h for h, p in [line.split('  ', 1) for line in (PROOF / 'worker-freeze.sha256').read_text().splitlines()])
    return {'sources': sources, 'tools': template['source_before']['tools'], 'admission_sha256': sha((PROOF / 'root-admission.json').read_bytes()), 'protected_count': len(admission['protected_files']), 'protected_unchanged': True, 'worker_freeze_sha256': sha((PROOF / 'worker-freeze.sha256').read_bytes()), 'worker_freeze_verified': True}


def audit():
    assert subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT).decode().strip() == admission['base_commit']
    for p, h in admission['source_before_sha256'].items():
        assert sha(subprocess.check_output(['git', 'show', admission['base_commit'] + ':' + p], cwd=ROOT)) == h
    saved = json.loads((PROOF / 'source-check.json').read_text())
    saved_names = [Path(r['path']).name for r in saved['receipts']]
    assert set(saved_names) <= set(evidence['receipts'])
    source = (PROOF / 'source_audit.py').read_text()
    source = source.replace("(PROOF / 'lint-delta.json').write_text(json.dumps(delta, indent=2) + '\\n')", "assert json.loads((PROOF / 'lint-delta.json').read_text()) == delta")
    source = source.replace("for path in sorted(PROOF.glob('*.receipt.json')):", "for path in sorted(PROOF / name for name in saved_names):")
    source = source.replace("(PROOF / 'source-check.json').write_text(json.dumps(result, indent=2) + '\\n')", "assert json.loads((PROOF / 'source-check.json').read_text()) == result")
    assert 'write_text(' not in source
    stream = io.StringIO()
    with contextlib.redirect_stdout(stream):
        exec(compile(source, str(PROOF / 'source_audit.py'), 'exec'), {'__file__': str(PROOF / 'source_audit.py'), 'saved_names': saved_names})
    print(stream.getvalue(), end='')
    for name in evidence['receipts']:
        receipt = json.loads((PROOF / name).read_text())
        raw = (ROOT / receipt['log']).read_bytes()
        assert sha(raw) == receipt['log_sha256']
        assert receipt['stability'] and receipt['source_before'] == receipt['source_after']
        assert receipt['exit_code'] == (1 if name == 'baseline-lint.receipt.json' else 0)
        assert receipt['environment'] == template['environment']
        assert receipt['source_before']['tools'] == template['source_before']['tools']
        assert receipt['source_before']['admission_sha256'] == template['source_before']['admission_sha256']
        assert receipt['source_before']['protected_count'] == len(admission['protected_files'])
        assert receipt['source_before']['protected_unchanged']
        if name.startswith('baseline-'):
            assert receipt['source_before']['sources'] == admission['source_before_sha256']
        if name.startswith(('final-', 'audit-')):
            assert receipt['source_before']['sources'] == evidence['sources']
        if 'go test ' in receipt['command']:
            assert b'\nFAIL' not in raw and re.search(rb'^ok\s', raw, re.M)
    print('All 16 archived receipts/logs verified; read-only audit comparisons match archived JSON; BASE Git source hashes and worker freeze verified.')


name = sys.argv[1]
before = freeze()
environment = dict(os.environ)
for key in template['environment']['unset']:
    environment.pop(key, None)
environment.update(template['environment']['set'])
started = time.monotonic()
start = datetime.datetime.now(datetime.timezone.utc).isoformat()
if name == 'audit':
    command = [sys.executable, str(Path(__file__).resolve()), 'audit']
    stream = io.StringIO()
    with contextlib.redirect_stdout(stream):
        audit()
    raw = stream.getvalue()
    code = 0
else:
    archived = json.loads((PROOF / ('final-' + name + '.receipt.json')).read_text())
    command = shlex.split(archived['command'])
    outcome = subprocess.run(command, cwd=ROOT / 'tools/gomad3', env=environment, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=600)
    raw = outcome.stdout.decode()
    code = outcome.returncode
after = freeze()
receipt = {'name': name, 'argv': command, 'cwd': str(ROOT / 'tools/gomad3' if name != 'audit' else ROOT), 'environment': template['environment'], 'source_before': before, 'source_after': after, 'stability': before == after, 'start_utc': start, 'end_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'elapsed_seconds': round(time.monotonic() - started, 3), 'exit_code': code, 'log': str((PROOF / ('review-' + name + '.log')).relative_to(ROOT)), 'log_sha256': sha(raw.encode()), 'top_level_pass_count': len(re.findall(r'^--- PASS:', raw, re.M)) if name in ('package', 'focused', 'boundaries') else None}
print(json.dumps({'receipt': receipt, 'raw_log': raw}))
