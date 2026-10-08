import base64
import datetime
import hashlib
import json
import os
from pathlib import Path
import shlex
import subprocess
import sys

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
HERE = Path(__file__).resolve().parent
PARENT = HERE.parent
BASE = '1fcef5ea4bfbf3ec5f7279a8a6c67070d5159a9d'
ENV = dict(os.environ)
for key in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
    ENV.pop(key, None)


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def reference(path):
    return {'path': str(path.relative_to(ROOT)), 'sha256': digest(path)}


old_path = PARENT / 'evidence.json'
historical = HERE / 'parent-evidence.json'
saved = subprocess.check_output(['git', 'show', BASE + ':' + str(old_path.relative_to(ROOT))], cwd=ROOT)
if historical.exists():
    assert historical.read_bytes() == saved
else:
    historical.write_bytes(saved)
original = json.loads(saved)
observations = []
for name, argv in [('documentation-source-correction-audit', [sys.executable, str(HERE / 'audit.py')]),
                   ('whitespace', ['git', 'diff', '--check'])]:
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    result = subprocess.run(argv, cwd=ROOT, env=ENV, capture_output=True, timeout=600)
    item = {'name': name, 'argv': argv, 'cwd': str(ROOT), 'started': started,
            'ended': datetime.datetime.now(datetime.timezone.utc).isoformat(),
            'exit': result.returncode, 'environment': {'PATH': ENV['PATH']},
            'unset_environment': ['GOMADSEED', 'GOMAD3_CHILD_SEED'],
            'stdout_base64': base64.b64encode(result.stdout).decode(),
            'stderr_base64': base64.b64encode(result.stderr).decode()}
    observations.append(item)
    (HERE / 'checks.json').write_text(json.dumps({'observations': observations,
        'python_sha256': digest(Path(sys.executable).resolve())}, indent=2) + '\n')
    print(name, 'exit', result.returncode, flush=True)
    if result.returncode:
        raise SystemExit(result.returncode)
audit = json.loads((HERE / 'final-docs.json').read_text())
assert not audit['errors']
for path, expected in audit['inputs'].items():
    assert digest(ROOT / path) == expected, path
for group in original['command_receipts']:
    ref = group['receipt']
    assert digest(ROOT / ref['path']) == ref['sha256'], ref['path']
    json.loads((ROOT / ref['path']).read_text())
changed = {path: digest(ROOT / path) for path in original['changed_product_hashes']}
report = {
    'task_id': original['task_id'], 'status': 'in_progress', 'base_commit': BASE,
    'review_id': 'f32f5cc32c5747aab7c57fb40ac3ae94', 'review_round': 1,
    'finding': 'introduced P3/R12; source clause count differs from polled non-nil case count',
    'check_tier': 'doc-only; all Go/runtime inputs unchanged',
    'checks': reference(HERE / 'checks.json'), 'source_audit': reference(HERE / 'final-docs.json'),
    'tools': [reference(HERE / name) for name in ('audit.py', 'verify.py')],
    'handover': reference(HERE / 'handover.md'),
    'milestone_recommendations': reference(HERE / 'milestone-recommendations.md'),
    'changed_product_hashes': changed,
    'historical_checkpoint_evidence': reference(historical),
    'original_source_report': reference(PARENT / 'final-docs.json'),
    'reuse_limit': 'Unchanged precise Go/runtime/generator/manifest inputs only; original command observations and historical reports keep their checkpoint meaning.',
    'native_credit': False, 'owned_commands_terminal': True, 'go_cache_lane': 'released'
}
(HERE / 'evidence.json').write_text(json.dumps(report, indent=2) + '\n')
current = dict(original)
current['changed_product_hashes'] = changed
current['final_source_audit'] = reference(HERE / 'final-docs.json')
current['handover'] = reference(PARENT / 'handover.md')
current['milestone_recommendations'] = reference(PARENT / 'milestone-recommendations.md')
current['review_fix_1'] = reference(HERE / 'evidence.json')
current['tests'] = original['tests'] + [shlex.join(item['argv']) for item in observations]
current['historical_pre_fix_snapshot'] = {
    'commit': BASE, 'evidence': reference(historical),
    'source_audit': reference(PARENT / 'final-docs.json'),
    'command_scope': 'Existing command_receipts keep their original pre-fix observations; no command was overwritten or rerun for Go/native credit.'}
current['recorded_at'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
old_path.write_text(json.dumps(current, indent=2) + '\n')
print(json.dumps({'source_inputs': len(audit['inputs']), 'errors': audit['errors'],
                  'correction_proof': audit['correction_proof'],
                  'evidence': reference(HERE / 'evidence.json'),
                  'current_parent_evidence': reference(old_path)}))
