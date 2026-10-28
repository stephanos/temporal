"""Prove unmapped identical-tree tips passed the old verifier and fail strict binding."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--source', type=Path, required=True)
parser.add_argument('--export', type=Path, required=True)
parser.add_argument('--synthetic', type=Path, required=True)
args = parser.parse_args()
here = Path(__file__).resolve().parent
source, export, synthetic = (path.resolve() for path in (args.source, args.export, args.synthetic))
assert source != synthetic and export != synthetic and not synthetic.exists()
subprocess.run(['git', 'clone', '--bare', '--no-local', str(export), str(synthetic)], check=True, capture_output=True)
def git(*arguments, data=None):
    return subprocess.check_output(['git', '--no-replace-objects', '-C', str(synthetic), *arguments], input=data)
checkpoint = git('rev-parse', 'refs/heads/umpire-current').decode().strip()
tree = git('rev-parse', checkpoint+'^{tree}').decode().strip()
def commit(parents, message):
    raw = ('tree '+tree+'\n'+''.join('parent '+parent+'\n' for parent in parents)+
           'author Synthetic Foreign <foreign@example.invalid> 1793000000 +0000\n'+
           'committer Synthetic Foreign <foreign@example.invalid> 1793000000 +0000\n\n'+message+'\n').encode()
    return git('hash-object', '-t', 'commit', '-w', '--stdin', data=raw).decode().strip()
foreign = commit([], 'Foreign root outside filtered source map')
controls = [('unmapped-tip', commit([checkpoint], 'Unmapped empty-diff tip')),
            ('foreign-parent-tip', commit([checkpoint, foreign], 'Unmapped foreign second-parent tip'))]
results = []
for label, tip in controls:
    git('update-ref', 'refs/heads/'+label, tip)
    assert git('rev-parse', tip+'^{tree}').decode().strip() == tree
    outcomes = {}
    for version, verifier, history_map, expected in [
        ('before', here/'review-before/verify-export.py', here/'review-before/history-map.json', 0),
        ('after', here/'verify-export.py', here/'history-map.json', 1),
    ]:
        output = here/(label+'-'+version+'.json')
        command = [sys.executable, str(verifier), '--source-repo', str(source), '--export-repo', str(synthetic),
                   '--checkpoint', tip, '--commit-map', str(here/'export/commit-map'),
                   '--manifest', str(here.parent/'source/source-manifest.json'),
                   '--history-map', str(history_map), '--output', str(output)]
        result = subprocess.run(command, capture_output=True)
        report = json.loads(output.read_bytes())
        assert result.returncode == expected, (label, version, result.returncode, report['failures'])
        if version == 'after':
            categories = {sample['category'] for sample in report['failures']['samples']}
            assert categories == {'export-graph-bounded', 'source-checkpoint-frontier'}, categories
        outcomes[version] = {'arguments': command, 'exit': result.returncode, 'status': report['status'], 'failures': report['failures']}
    results.append({'control': label, 'original_checkpoint': checkpoint, 'synthetic_checkpoint': tip,
                    'same_active_tree': True, 'outcomes': outcomes})
(here/'checkpoint-negative-controls.json').write_text(json.dumps({
    'old_verifier_sha256': hashlib.sha256((here/'review-before/verify-export.py').read_bytes()).hexdigest(),
    'new_verifier_sha256': hashlib.sha256((here/'verify-export.py').read_bytes()).hexdigest(),
    'scope': 'Only an isolated bare synthetic clone receives new commits/refs. No source or real export graph mutation.',
    'controls': results,
}, indent=2)+'\n')
