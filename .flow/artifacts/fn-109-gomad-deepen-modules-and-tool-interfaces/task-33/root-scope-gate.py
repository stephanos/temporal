"""Read-only task-33 preservation and exact checkpoint-scope audit."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
SPEC = 'fn-109-gomad-deepen-modules-and-tool-interfaces'
FLOW = '/home/agent/.codex/scripts/flowctl'
ORIGINAL_TASK_PREFIX_SHA256 = 'd0f46e3876a4e3d93ce858123e3933b862abaa45f2fd7ee67aafdf96aba1121f'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)


def flow(*args):
    return json.loads(subprocess.check_output([FLOW, *args, '--json'], cwd=ROOT))


admission = json.loads((OUT / 'root-admission.json').read_text())
base = admission['base_commit']
assert git('branch', '--show-current').decode().strip() == admission['branch']
paths = git('ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration',
            'go.mod', 'go.sum', '.github/.golangci.yml').decode().splitlines()
protected = {p: sha((ROOT / p).read_bytes()) for p in paths
             if p not in admission['source_exclusions']}
protection = dict(files=len(protected),
                  aggregate_sha256=sha(json.dumps(protected, sort_keys=True).encode()))
assert protection == admission['protected'], protection
for path, expected in admission['original_documents'].items():
    assert sha((ROOT / path).read_bytes()) == expected, path
assert sha((ROOT / '.flow/tmp/artifact-remaining-diagnostics-owner-plan.md').read_bytes()) == admission['source_plan_sha256']
task_document = (ROOT / ('.flow/tasks/' + SPEC + '.33.md')).read_text()
assert sha(task_document.split('## Done summary')[0].encode()) == ORIGINAL_TASK_PREFIX_SHA256
parent_path = '.flow/specs/' + SPEC + '.md'
parent_before = git('show', base + ':' + parent_path).decode()
parent_now = (ROOT / parent_path).read_text()
assert parent_now.split("Task 21's retained lint failure")[0] == parent_before.split("Task 21's retained lint failure")[0]
assert parent_now.split('## Finding coverage')[1] == parent_before.split('## Finding coverage')[1]
metadata = ['MILESTONES.md', '.flow/specs/' + SPEC + '.md',
            '.flow/specs/' + SPEC + '.json', '.flow/tasks/' + SPEC + '.21.json',
            '.flow/tasks/' + SPEC + '.33.md', '.flow/tasks/' + SPEC + '.33.json']
proofs = [str(p.relative_to(ROOT)) for p in sorted(OUT.rglob('*')) if p.is_file()]
assert all('__pycache__' not in p and not p.endswith(('.pyc', '.tar')) for p in proofs)
allowed = sorted(set(metadata + admission['source_exclusions'] + proofs))
changed = git('diff', '--name-only', base).decode().splitlines()
assert set(changed) <= set(allowed), sorted(set(changed) - set(allowed))
index = git('diff', '--cached', '--name-only').decode().splitlines()
assert set(index) <= set(allowed), index
task = flow('show', SPEC + '.33')
spec = flow('show', SPEC)
assert task['status'] in ('in_progress', 'blocked')
assert task['assignee'] == 'stephanos@users.noreply.github.com'
assert spec['status'] == 'open' and len(spec['tasks']) == 33
assert sum(t['status'] == 'done' for t in spec['tasks']) == 2
consumer = next(t for t in spec['tasks'] if t['id'] == SPEC + '.21')
assert consumer['status'] == 'blocked' and SPEC + '.33' in consumer['depends_on']
assert flow('validate', '--spec', SPEC)['valid']
checks_path = OUT / 'root-source-checks.json'
if checks_path.exists():
    checks = json.loads(checks_path.read_text())
    assert sha((ROOT / admission['source_exclusions'][0]).read_bytes()) == checks['source_sha256']
    assert task['status'] == checks['flow_status'] == 'blocked'
    assert checks['assessment'] == 'SOURCE_PROGRESS_COMMIT_ONLY' and not checks['actionable_findings']
    assert not checks['formal_review'] and not checks['qualified_native']
    assert checks['original_task_prefix_sha256'] == ORIGINAL_TASK_PREFIX_SHA256
    for name, expected in checks['immutable_artifacts'].items():
        assert sha((OUT / name).read_bytes()) == expected, name
    assert parent_now == parent_before.replace('## Finding coverage', checks['parent_insert'] + '## Finding coverage')
    milestones = (ROOT / 'MILESTONES.md').read_text()
    assert milestones.count(checks['milestone_insert']) == 1
    restored = milestones.replace(checks['milestone_insert'], '')
    assert restored.count(checks['milestone_rows']['after']) == 1
    restored = restored.replace(checks['milestone_rows']['after'], checks['milestone_rows']['before'])
    assert restored == git('show', base + ':MILESTONES.md').decode()
    assert task_document.count('stage: impl-review - ') == task_document.count('stage: plan-sync - ') == 1
    assert 'TBD' not in task_document.split('## Done summary')[1]
if len(sys.argv) > 1 and sys.argv[1] == 'staged':
    selected = sorted(set(changed + proofs + ['.flow/tasks/' + SPEC + '.33.md',
                                           '.flow/tasks/' + SPEC + '.33.json']))
    assert sorted(index) == selected, (index, selected)
    for path in index:
        assert git('show', ':' + path) == (ROOT / path).read_bytes(), path
    subprocess.check_call(['git', 'diff', '--cached', '--check'], cwd=ROOT)
print(json.dumps(dict(base=base, head=git('rev-parse', 'HEAD').decode().strip(),
                      source_sha256=sha((ROOT / admission['source_exclusions'][0]).read_bytes()),
                      protected=protection, changed=changed, staged=index,
                      proof_files=len(proofs), task_status=task['status'],
                      spec_status=spec['status'], completed_tasks=2, total_tasks=33,
                      formal_review=False, qualified_native=False, writes=0), indent=2))
