"""Reproduce every declared ancestry definition and representative log from Git."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

SOURCE = 'ac60daaf47d2513b41eeff612f0e2758d33e186d'
p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--source-repo', type=Path, required=True)
p.add_argument('--history-map', type=Path, required=True)
p.add_argument('--output', type=Path, required=True)
a = p.parse_args()
raw = a.history_map.read_bytes()
history = json.loads(raw)
assert history['source_commit'] == SOURCE
manifest = json.loads((a.history_map.parent.parent/'source/source-manifest.json').read_bytes())
assert history['current_source_map'] == [{key: record[key] for key in ('source', 'destination', 'mode', 'blob')} for record in manifest['records']]
assert any(str(len(manifest['records']))+' including LICENSE' in text for text in history['filter_algorithm'])
assert history['original_input_map_provenance']['source_commit'] == 'c88fdc550694020eb1c6a2e09577813d906c9ffe'
def git(*args):
    return subprocess.check_output(['git', '--no-replace-objects', '-C', str(a.source_repo), *args])
sets = {'H': set(git('rev-list', SOURCE).decode().splitlines())}
assert history['ancestry_sets']['H']['definition_command'] == 'git rev-list '+SOURCE
for name, boundary in history['boundaries'].items():
    oid = boundary['commit']
    key = 'D_'+name
    assert history['ancestry_sets'][key]['definition_command'] == '{'+oid+'} union git rev-list --ancestry-path '+oid+'..'+SOURCE
    sets[key] = {oid} | set(git('rev-list', '--ancestry-path', oid+'..'+SOURCE).decode().splitlines())
assert history['ancestry_sets']['R']['definition_command'] == 'D_canonical_archive_move union D_sibling_archive_move'
sets['R'] = sets['D_canonical_archive_move'] | sets['D_sibling_archive_move']
for name, evidence in history['ancestry_sets'].items():
    encoded = ('\n'.join(sorted(sets[name]))+'\n').encode()
    assert evidence['count'] == len(sets[name])
    assert evidence['sha256_sorted_lf_terminated'] == hashlib.sha256(encoded).hexdigest()
    if 'members' in evidence:
        assert evidence['members'] == sorted(sets[name])
for representative in history['representative_history']:
    command = ['git', 'log', '--follow', '--format=%H%x09%aI%x09%an%x09%s', SOURCE, '--', representative['path']]
    assert representative['command'] == command
    payload = git(*command[1:])
    assert representative['history_records'] == payload.decode().splitlines()
    assert representative['history_commit_count'] == len(payload.splitlines())
    assert representative['history_sha256_lf_terminated'] == hashlib.sha256(payload).hexdigest()
    assert representative['present_at_source'] is True
    assert representative['blob'] == git('rev-parse', SOURCE+':'+representative['path']).decode().strip()
a.output.write_text(json.dumps({'status':'pass','source_commit':SOURCE,'ancestry_definitions_reproduced':len(sets),
                               'representative_histories_reproduced':len(history['representative_history']),
                               'history_map_sha256':hashlib.sha256(raw).hexdigest(),
                               'script_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest()},indent=2)+'\n')
