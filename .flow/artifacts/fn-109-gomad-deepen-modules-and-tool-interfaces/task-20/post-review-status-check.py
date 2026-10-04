"""Verify this task's reviewed guides and subsequent status-only receipt update."""
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys

root = Path.cwd()
artifact = Path('.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces')
task = artifact / 'task-20'
evidence = json.loads((task / 'evidence.json').read_text())
for path, digest in evidence['document_sha256'].items():
    if path != 'MILESTONES.md':
        assert hashlib.sha256(Path(path).read_bytes()).hexdigest() == digest, path
milestone_hash = '24e4613b5c6fb32a74775b431c54d8b8247d3e697cb4e80f3a7d4d5f1f775bea'
assert hashlib.sha256(Path('MILESTONES.md').read_bytes()).hexdigest() == milestone_hash
checker = Path('.flow/artifacts/fn-111-gomad-consolidate-vocabulary-and-update/verify-guides.py')
sys.argv = ['post-review-document-check', str(root), '/home/agent/.codex/scripts/flowctl', str(root / task)]
ns = {'__file__': str(checker)}
exec(compile(checker.read_text().split('# --- actual command help ---')[0], str(checker), 'exec'), ns)
assert not ns['errors'], ns['errors']
paths = [artifact / 'documentation-evidence.md', task / 'handover.md', task / 'source-checkpoint.md', task / 'acceptance-open.md']
paths += [task / ('formal-review-' + axis + '.md') for axis in ('correctness', 'contracts', 'integration')]
paths += [Path('.flow/tasks/' + name + '.md') for name in ('fn-109-gomad-deepen-modules-and-tool-interfaces.20', 'fn-105-gomad-follow-ups-deferred-scope.5')]
links = 0
for path in paths:
    prose, _, defects = ns['scan_fences'](str(path))
    assert not defects, (str(path), defects)
    for destination in re.findall(r'\[[^]]+\]\(([^)\s]+)\)', ns['squash']('\n'.join(prose))):
        if re.match(r'^[a-zA-Z][a-zA-Z0-9+.-]*:', destination):
            continue
        target, _, fragment = destination.partition('#')
        resolved = ((root / path).parent / ns['urllib'].parse.unquote(target)).resolve() if target else root / path
        assert resolved.exists(), (str(path), destination)
        if fragment and resolved.suffix == '.md':
            assert ns['urllib'].parse.unquote(fragment) in ns['anchors'](str(resolved.relative_to(root))), (str(path), destination)
        links += 1
origin = Path('.flow/review-fanout/22e75a53e12345df9db888ccc682aa1f')
receipt = json.loads((task / 'formal-review-receipt.json').read_text())
assert receipt['verdict'] == 'SHIP' and receipt['findings']['items'] == [] and receipt['unaddressed'] == []
assert receipt['findings']['headSha'] == '7cf8855c5e12280b4ff132e96e43fca9ac6b58c7'
assert (task / 'formal-review-receipt.json').read_bytes() == Path('/tmp/impl-review-receipt-657da2bc4466-fn-109-gomad-deepen-modules-and-tool-interfaces.20.json').read_bytes()
assert (task / 'formal-review-fanout-meta.json').read_bytes() == (origin / 'meta.json').read_bytes()
for axis in ('correctness', 'contracts', 'integration'):
    metadata_path = task / ('formal-review-' + axis + '.json')
    assert metadata_path.read_bytes() == (origin / (axis + '.json')).read_bytes()
    metadata = json.loads(metadata_path.read_text())
    assert metadata['verdict'] == 'SHIP' and not metadata['failed'] and metadata['exit_code'] == 0
    assert metadata['model'] == 'gpt-6.1-sol' and metadata['effort'] == 'high'
    original = (origin / (axis + '.review.md')).read_bytes()
    copy = (task / ('formal-review-' + axis + '.md')).read_bytes()
    assert copy == original or (not original.endswith(b'\n') and copy == original + b'\n')
for path in task.glob('formal-review-*.json'):
    json.loads(path.read_text())
subprocess.run(['git', 'diff', '--check'], check=True)
print(json.dumps({'guide_links': len(ns['navigation']), 'other_links': links, 'fences': 'balanced', 'reviewed_guide_and_evidence_hashes': 'unchanged', 'milestone_status_sha256': milestone_hash, 'formal_receipt_and_metadata': 'byte-identical', 'review_text': 'unchanged except added final newline where absent', 'draws': '3 SHIP, no findings, gpt-6.1-sol high'}, sort_keys=True))
