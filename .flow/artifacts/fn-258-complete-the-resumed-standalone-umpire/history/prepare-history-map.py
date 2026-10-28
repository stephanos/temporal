"""Rebind original reviewed history rules to the frozen fn-258 source and manifest."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

SOURCE = 'ac60daaf47d2513b41eeff612f0e2758d33e186d'
p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--source-repo', type=Path, required=True)
p.add_argument('--original-history-map', type=Path, required=True)
p.add_argument('--manifest', type=Path, required=True)
p.add_argument('--output', type=Path, required=True)
a = p.parse_args()
assert not a.output.exists(), 'output must be new'
h = json.loads(a.original_history_map.read_bytes())
raw = a.manifest.read_bytes()
m = json.loads(raw)
assert m['source_commit'] == SOURCE and len(m['records']) == 1189
h.update(source_commit=SOURCE, source_framework_tree=m['framework_tree'],
         source_archive_tree=m['archive_tree'], framework_file_count=m['framework_files'],
         selected_file_count_including_license=len(m['records']),
         source_manifest='.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/source-manifest.json',
         source_manifest_sha256=hashlib.sha256(raw).hexdigest())
def git(*args):
    return subprocess.check_output(['git','--no-replace-objects','-C',str(a.source_repo),*args]).decode().splitlines()
sets = {'H': set(git('rev-list',SOURCE))}
for name, boundary in h['boundaries'].items():
    sets['D_'+name] = {boundary['commit']} | set(git('rev-list','--ancestry-path',boundary['commit']+'..'+SOURCE))
sets['R'] = sets['D_canonical_archive_move'] | sets['D_sibling_archive_move']
for name, evidence in h['ancestry_sets'].items():
    members = sets[name]
    evidence.update(count=len(members),sha256_sorted_lf_terminated=hashlib.sha256(('\n'.join(sorted(members))+'\n').encode()).hexdigest())
    if 'members' in evidence:
        evidence['members'] = sorted(members)
h['adapted_from'] = 'fn-257/task1/history-map.json; unchanged historical path/provenance qualification rules; recomputed fixed-source ancestry'
a.output.write_text(json.dumps(h,indent=2)+'\n')
