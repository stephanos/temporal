import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path("/Users/stephan/Workspace/skunkworks/gomad/temporal")
OUT = Path(__file__).resolve().parent
SPEC = 'fn-109-gomad-deepen-modules-and-tool-interfaces'
FLOW = '/home/agent/.codex/scripts/flowctl'
BASE = 'f931c9879e3017b346562667b9f6fbcc4db458ec'
DOC = 'e95d9fa1b61951388818715677e5c42b0c03ee0f'
PARENT_OUTCOME = "Task 32's independently reviewed error-provenance candidate preserves named-slice\nsingle-error Unwrap traversal, destination concrete conversion callbacks and fmt\nwriter-error results. Six fresh allocation alias families retain shared existing\nstorage; conversion-local array/struct copies isolate value cells while nested\nreferences retain their callback path. All 39 earlier fixture bodies remain an\nexact prefix of the final 67 stock-host fixtures and 134 supported-metadata\nobservations. Whole architecture, focused, five actual boundaries, broader\npurity/edges/host-vet, gomadtool consumer, errortype, source-scoped static and generator checks\npass on developmental linux/arm64. Actual unfiltered architecture lint retains\nfour byte-identical inherited findings, with none introduced or resolved.\nThe full staged diff-check retains one historical archived handover EOF blank;\nsource and newly written root-document checks are separately clean.\nFresh corrective review and root read-only audit permit SOURCE_PROGRESS_COMMIT_ONLY,\nwith no actionable introduced defects. Historical failures and inconclusive\nprobes remain disclosed; this is no universal copy, length/capacity or formatting\ncompleteness claim. Task 21 consumes the evidence. Original R8/R18/R19,\ntask 19/fn-105 D4, predecessors, matched first-baseline identities and complete/\nfull/formal/both-native/affected-consumer qualification remain open. Root commits\nreviewed progress before another source writer.\n"
SELF_REPORT = str((OUT / 'root-source-checkpoint-verification.json').relative_to(ROOT))

def sha(data):
    return hashlib.sha256(data).hexdigest()

def digest(path):
    return sha(Path(path).read_bytes())

def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)

def flow(*args):
    return json.loads(subprocess.check_output([FLOW, *args, '--json'], cwd=ROOT))

parser = argparse.ArgumentParser()
parser.add_argument('phase', choices=['pre', 'post'])
args = parser.parse_args()
bindings = json.loads((OUT / 'root-source-checks.json').read_text())
worker = json.loads((OUT / 'allocation-repair-evidence.json').read_text())
review = json.loads((OUT / 'allocation-source-review-checks.json').read_text())
admission = json.loads((OUT / 'source-admission.json').read_text())
assert git('branch', '--show-current').decode().strip() == 'gomad'
assert git('rev-parse', DOC + '^').decode().strip() == BASE
assert git('diff', '--name-only', BASE, DOC).decode().splitlines() == ['AGENTS.md']
assert digest(ROOT / 'AGENTS.md') == bindings['document_checkpoint']['sha256']
assert (ROOT / 'AGENTS.md').read_bytes() == git('show', DOC + ':AGENTS.md')
for path, expected in bindings['immutable_artifacts'].items():
    assert digest(OUT / path) == expected, path
for manifest in ('historical_files', 'proof_files'):
    for path, expected in worker[manifest].items():
        assert digest(OUT / path) == expected, path
for receipt in review['new_review_receipts']:
    assert digest(OUT / receipt['path']) == receipt['sha256']
    assert digest(OUT / receipt['log']) == receipt['log_sha256']
    current = json.loads((OUT / receipt['path']).read_text())
    assert current['source_before'] == current['source_after'] == review['source']
    assert current['protected_before'] == current['protected_after'] == review['protected']
assert review['verdict'] == 'SOURCE_PROGRESS_COMMIT_ONLY'
assert review['actionable_findings'] == []
assert [review[k] for k in ('architecture_top_level_tests', 'focused_top_level_tests', 'consumer_top_level_tests', 'causal_fixtures', 'metadata_observations', 'required_boundary_tests', 'broader_boundary_tests')] == [26,16,30,67,134,5,3]
assert review['lint'] == dict(inherited=4, introduced=0, resolved=0)
assert not review['qualified_native'] and not review['formal_review']
for path, expected in review['source'].items():
    assert digest(ROOT / path) == expected, path
assert bindings['source'] == {p:review['source'][p] for p in admission['protected_exclusions']}
paths = git('ls-files', 'tools/gomad3', 'tools/gomad3sim', 'tools/gomad3integration', 'go.mod', 'go.sum', '.github/.golangci.yml').decode().splitlines()
protected = {p:digest(ROOT / p) for p in paths if p not in admission['protected_exclusions']}
protection = dict(files=len(protected), aggregate_sha256=sha(json.dumps(protected, sort_keys=True).encode()))
assert protection == bindings['protected'] == review['protected']
for path, expected in admission['protected_original_documents'].items():
    if path == 'AGENTS.md':
        assert sha(git('show', BASE + ':' + path)) == expected
    else:
        assert digest(ROOT / path) == expected, path
task_path = '.flow/tasks/' + SPEC + '.32.md'
task = (ROOT / task_path).read_text()
assert sha(task.split('## Done summary')[0].encode()) == bindings['original_task_prefix_sha256']
assert task.count('stage: impl-review - ') == task.count('stage: plan-sync - ') == 1
assert 'TBD' not in task.split('## Done summary')[1]
parent_path = '.flow/specs/' + SPEC + '.md'
parent_baseline = git('show', BASE + ':' + parent_path).decode()
assert parent_baseline.count('## Finding coverage') == 1
assert (ROOT / parent_path).read_text() == parent_baseline.replace('## Finding coverage', PARENT_OUTCOME + '\n## Finding coverage')
milestones = (ROOT / 'MILESTONES.md').read_text()
start = milestones.index("Task 32's independently reviewed error-provenance repair")
end = milestones.index('The remaining nested findings retain bounded source owners', start)
restored = milestones[:start] + milestones[end:]
old_milestones = git('show', DOC + ':MILESTONES.md').decode()
old_line = next(line for line in old_milestones.splitlines() if line.startswith('| Deep modules'))
new_line = old_line.replace('tasks 23–31', 'tasks 23–32').replace('while the historical', 'and architecture provenance passes 67 fixtures with four inherited lint findings, while the historical')
assert restored.count(new_line) == 1
assert restored.replace(new_line, old_line) == old_milestones
task_state = flow('show', SPEC + '.32')
spec_state = flow('show', SPEC)
assert task_state['status'] == 'blocked' and task_state['status_source'] == 'flow-state'
assert task_state['claimed_at'] == '2026-10-04T17:27:39.986836Z'
assert task_state['assignee'] == 'stephanos@users.noreply.github.com'
assert spec_state['status'] == 'open'
assert len(spec_state['tasks']) == 32
assert sum(t['status'] == 'done' for t in spec_state['tasks']) == 2
consumer = next(t for t in spec_state['tasks'] if t['id'] == SPEC + '.21')
assert consumer['depends_on'] == [SPEC+'.20'] + [SPEC+'.'+str(n) for n in range(23,33)]
assert flow('validate', '--spec', SPEC)['valid']
owned = [
    'MILESTONES.md',
    '.flow/specs/' + SPEC + '.md',
    '.flow/specs/' + SPEC + '.json',
    '.flow/tasks/' + SPEC + '.21.json',
    '.flow/tasks/' + SPEC + '.32.md',
    '.flow/tasks/' + SPEC + '.32.json',
    *admission['protected_exclusions'],
]
proofs = [str(p.relative_to(ROOT)) for p in sorted(OUT.rglob('*')) if p.is_file()]
assert all('__pycache__' not in p and not p.endswith(('.pyc', '.tar')) for p in proofs)
owned = sorted(set(owned + proofs))
assert SELF_REPORT in owned
index = git('diff', '--cached', '--name-only').decode().splitlines()
if args.phase == 'pre':
    assert git('rev-parse', 'HEAD').decode().strip() == DOC
    assert index == owned, (len(index), len(owned), set(index)^set(owned))
    for path in owned:
        assert git('show', ':' + path) == (ROOT / path).read_bytes(), path
else:
    assert not index
    source_commit = bindings['source_progress_commit']
    assert source_commit and git('merge-base', '--is-ancestor', source_commit, 'HEAD') == b''
    assert sorted(git('diff', '--name-only', DOC, source_commit).decode().splitlines()) == owned
    assert not git('diff', '--name-only', 'HEAD', '--', *owned)
    for path in owned:
        assert git('show', 'HEAD:' + path) == (ROOT / path).read_bytes(), path
assert set(git('diff', '--name-only', 'HEAD').decode().splitlines()) <= set(owned)
subprocess.run(['git', 'diff', '--check'], cwd=ROOT, check=True)
full_args = ['git', 'diff', '--cached', '--check'] if args.phase == 'pre' else ['git', 'diff', '--check', DOC, bindings['source_progress_commit']]
full_diff = subprocess.run(full_args, cwd=ROOT, capture_output=True, text=True)
expected_whitespace = str((OUT / 'allocation-repair-handover.md').relative_to(ROOT)) + ':20: new blank line at EOF.\n'
assert full_diff.returncode == 2 and full_diff.stdout == expected_whitespace and full_diff.stderr == ''
new_document_paths = [p for p in owned if not p.startswith(str(OUT.relative_to(ROOT)) + '/')] + [str((OUT / p).relative_to(ROOT)) for p in ('acceptance-open.md', 'root-source-checks.json', 'root-source-checkpoint-gate.py', 'root-source-checkpoint-verification.json')]
scope_args = ['git', 'diff', '--cached', '--check'] if args.phase == 'pre' else ['git', 'diff', '--check', DOC, bindings['source_progress_commit']]
subprocess.run([*scope_args, '--', *new_document_paths], cwd=ROOT, check=True)
content = {p:digest(ROOT / p) for p in owned if p != SELF_REPORT}
print(json.dumps(dict(
    phase=args.phase, task=SPEC+'.32', status=task_state['status'],
    spec_status=spec_state['status'], done_tasks=2, total_tasks=32,
    head=git('rev-parse','HEAD').decode().strip(),
    source_progress_commit=bindings['source_progress_commit'],
    selected_paths=len(owned), selected_content_files=len(content),
    selected_content_aggregate_sha256=sha(json.dumps(content, sort_keys=True).encode()),
    proof_files=len(proofs), immutable_writer_files=len(worker['historical_files'])+len(worker['proof_files']),
    new_review_receipts=len(review['new_review_receipts']),
    protected=protection, causal_fixtures=67, metadata_observations=134,
    inherited_lint=4, introduced_lint=0, original_task_prefix_unchanged=True,
    original_parent_criteria_unchanged=True, original_milestone_scope_unchanged=True,
    document_only_checkpoint=DOC, gate_script_sha256=digest(__file__),
    root_source_checks_sha256=digest(OUT/'root-source-checks.json'),
    formal_ship=False, native_qualified=False, writes=0,
    full_checkpoint_diff_check=dict(command=full_args, exit=full_diff.returncode, raw_output=full_diff.stdout, classification='one frozen historical archived handover EOF blank; not Git-BASE-inherited or a passing full-index quality gate'),
    source_and_new_root_document_diff_check_exit=0,
), indent=2))
