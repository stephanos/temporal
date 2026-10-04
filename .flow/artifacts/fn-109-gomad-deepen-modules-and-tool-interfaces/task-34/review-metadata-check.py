import datetime
import hashlib
import json
from pathlib import Path
import subprocess
import sys

root = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
proof = Path(__file__).resolve().parent
task_id = 'fn-109-gomad-deepen-modules-and-tool-interfaces.34'
spec_id = task_id.rsplit('.', 1)[0]
flow = '/home/agent/.codex/scripts/flowctl'
paths = ['.flow/tasks/' + task_id + '.md', '.flow/tasks/' + task_id + '.json', '.flow/specs/' + spec_id + '.md', 'MILESTONES.md', str((proof / 'acceptance-open.md').relative_to(root))]
start = datetime.datetime.now(datetime.timezone.utc).isoformat()
hashes = {p: hashlib.sha256((root / p).read_bytes()).hexdigest() for p in paths}
task = json.loads(subprocess.check_output([flow, 'show', task_id, '--json'], cwd=root))
consumer = json.loads(subprocess.check_output([flow, 'show', spec_id + '.21', '--json'], cwd=root))
parent = json.loads(subprocess.check_output([flow, 'show', spec_id, '--json'], cwd=root))
assert task['status'] == 'blocked' and task['status_source'] == 'flow-state'
assert consumer['status'] == 'blocked' and task_id in consumer['depends_on']
assert parent['status'] == 'open' and parent['completion_review_status'] == 'unknown'
assert len(parent['tasks']) == 34 and sum(t['status'] == 'done' for t in parent['tasks']) == 2
fallback = json.loads((root / paths[1]).read_text())
assert fallback['status'] == 'todo'
text = (root / paths[0]).read_text()
prefix = text.split('## Done summary', 1)[0]
description = text.split('## Description\n', 1)[1].split('## Acceptance\n', 1)[0].strip()
acceptance = text.split('## Acceptance\n', 1)[1].split('## Done summary', 1)[0].strip()
assert description == (root / '.flow/tmp/task34-description.md').read_text().strip()
assert acceptance == (root / '.flow/tmp/task34-acceptance.md').read_text().strip()
assert 'SOURCE_PROGRESS_COMMIT_ONLY' in text and 'qualification remain required and open' in text
assert 'stage: impl-review - skipped(policy: conductor-deferred;' in text
assert 'stage: plan-sync - skipped(config: disabled;' in text
assert 'Genuine Close and simultaneous primary/cleanup failure execution remain unproved.' in text
assert '+Task 34' not in (root / 'MILESTONES.md').read_text()
for p in [paths[2], paths[3], paths[4]]:
    body = (root / p).read_text()
    assert '53' in body and 'linux/arm64' in body
    assert 'primary/cleanup' in body
    assert 'first-baseline' in body or 'first-task' in body
    assert ('darwin/arm64' in body and 'linux/amd64' in body) or 'both-native' in body
    assert 'formal' in body and 'required' in body
reaudit = subprocess.run([sys.executable, str(proof / 'review-reaudit.py')], cwd=root, capture_output=True, text=True, check=True)
assert hashes == {p: hashlib.sha256((root / p).read_bytes()).hexdigest() for p in paths}
report = {'result': 'PASS', 'source_verdict': 'SOURCE_PROGRESS_COMMIT_ONLY', 'start': start, 'end': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'exit_code': 0, 'tests_repeated': 0, 'source_reaudit': json.loads(reaudit.stdout), 'document_sha256': hashes, 'original_task_prefix_sha256': hashlib.sha256(prefix.encode()).hexdigest(), 'original_description_acceptance_preserved': True, 'task_status': task['status'], 'task_status_source': task['status_source'], 'tracked_task_fallback_status': fallback['status'], 'fallback_interpretation': 'Flow runtime flow-state is authoritative; tracked todo snapshot does not supersede blocked runtime status.', 'consumer_status': consumer['status'], 'consumer_depends_on_task34': True, 'parent_status': parent['status'], 'parent_completion_review_status': parent['completion_review_status'], 'parent_done': 2, 'parent_total': 34, 'corrected_root_metadata_preflight': 'Literal +Task 34 paragraph marker was corrected by root; no source change.', 'all_handles_terminal': True, 'metadata_script_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest()}
destination = proof / 'review-metadata-check.json'
assert not destination.exists()
with destination.open('x') as output:
    output.write(json.dumps(report, indent=2) + '\n')
print(json.dumps({'result': report['result'], 'task_status': task['status'], 'fallback': fallback['status'], 'documents': len(hashes), 'prefix': report['original_task_prefix_sha256'], 'tests_repeated': 0}))
