import base64
import datetime
import hashlib
import json
from pathlib import Path
import shlex
import subprocess

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
HERE = Path(__file__).resolve().parent


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def reference(path):
    return {'path': str(path.relative_to(ROOT)), 'sha256': digest(path)}


audit = json.loads((HERE / 'final-docs.json').read_text())
assert not audit['errors']
for path, expected in audit['inputs'].items():
    assert digest(ROOT / path) == expected, path
baseline = json.loads((HERE / 'baseline-docs.json').read_text())
assert audit['qualification_manifests'] == baseline['qualification_manifests']
receipts, tests = [], []
for name in ('baseline-checks.json', 'final-checks.json',
             'portable-completion-checks.json', 'freeze-checks.json', 'conductor-checks.json'):
    path = HERE / name
    saved = json.loads(path.read_text())
    observations = []
    for item in saved['observations']:
        output = base64.b64decode(item['stdout_base64'], validate=True)
        base64.b64decode(item['stderr_base64'], validate=True)
        events = []
        for line in output.decode().splitlines():
            try:
                row = json.loads(line)
            except ValueError:
                continue
            if isinstance(row, dict) and 'Test' in row:
                events.append(row)
        top = [row for row in events if '/' not in row['Test']]
        counts = {action: sum(row.get('Action') == action for row in top)
                  for action in ('run', 'pass', 'fail', 'skip')}
        observations.append({'name': item['name'], 'exit': item['exit'],
                             'executed_top_level_tests': counts,
                             'nested_passes': sum(row.get('Action') == 'pass' and '/' in row['Test'] for row in events)})
        tests.append(shlex.join(item['argv']))
    receipts.append({'receipt': reference(path), 'observations': observations})
    if name in ('baseline-checks.json', 'portable-completion-checks.json', 'freeze-checks.json', 'conductor-checks.json'):
        assert all(item['exit'] == 0 for item in saved['observations']), name
changed = [f'tools/gomad3/{name}.md' for name in ('README', 'CLI', 'SPEC', 'ARCHITECTURE', 'TUTORIAL')]
changed.append('.plans/GOMAD_CMP.md')
head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
report = {
    'task_id': 'fn-114-gomad-correct-search-path-defects-and.14',
    'status': 'in_progress',
    'base_commit': (HERE / 'base_commit').read_text().strip(),
    'observed_head': head, 'commits': [], 'prs': [], 'tests': tests,
    'recorded_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
    'tier': 'explicit implementer gpt-6.1-sol at high retained by conductor after judge no_key',
    'execution_metadata': 'Host execution model unavailable; no actual_model inferred.',
    'check_tier': 'doc-only; no executable or generator inputs changed',
    'review': 'deferred to conductor; no writer review verdict',
    'command_receipts': receipts,
    'final_source_audit': reference(HERE / 'final-docs.json'),
    'baseline_source_audit': reference(HERE / 'baseline-docs.json'),
    'changed_product_hashes': {path: digest(ROOT / path) for path in changed},
    'evidence_tools': [reference(HERE / name) for name in ('run-checks.py', 'audit-docs.py', 'build-evidence.py')],
    'handover': reference(HERE / 'handover.md'),
    'milestone_recommendations': reference(HERE / 'milestone-recommendations.md'),
    'source_preservation': {
        'receipt': audit['prior_receipt'], 'unchanged_scoped_inputs': audit['unchanged_preservation_inputs'],
        'retained_raw': audit['verified_retained_raw'],
        'reuse_limit': audit['reuse_limit'],
        'qualification_manifests_unchanged': len(audit['qualification_manifests'])},
    'native_observation': {
        'state': 'inconclusive before native fixture execution',
        'receipt': reference(HERE / 'final-checks.json'),
        'test': 'TestPinnedCoverageBuildSettingsRejectInstrumentation',
        'cause': 'Absent tools/gomad3/.toolchain/bin/go; selected native-dependent fixture did not run.',
        'native_credit': False,
        'owners': ['fn-149-gomad-deferred-darwin-qualification', 'fn-128-gomad-deferred-linux-qualification-and'],
        'transfer': reference(ROOT / '.flow/artifacts/native-scope-transfer-2026-10-07.md')},
    'audit_calibrations': [
        'Initial helper omitted patch-materialize alias; corrected helper, no product/parser edit.',
        'Expanded helper first named nonexistent corpus/guide.go; corrected to corpus/corpus.go.',
        'Raw-chain helper first used manifest name instead of retained_name and failed before completing. Corrected exact utf8-json-string container/decoded-byte handling; final audit passes. The freeze receipt was replaced by the final frozen observation; calibration receives no green credit.'
    ],
    'limits': [
        'No native runtime/cause/count/build-key/full-host/core/smoke/representative/replay/soak qualification.',
        'No current whole-tree/module blanket reuse of fn-114.13 historical snapshot.',
        'No new whole-module/full-project lint pass; fn-109 aggregate lint remains open.',
        'No worker Git, Flow/lifecycle, review/completion, PR/push/CI or native actions.',
        'Conductor must bind its later commits and lifecycle status before final task receipt.'
    ],
    'owned_commands_terminal': True, 'go_cache_lane': 'released'
}
(HERE / 'evidence.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps({'evidence': reference(HERE / 'evidence.json'),
                  'audit_inputs': len(audit['inputs']), 'source_errors': audit['errors'],
                  'changed_product_hashes': report['changed_product_hashes']}))
