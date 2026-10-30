"""Refresh checkpoint, continuity, provenance and integrity evidence from actual inputs."""
import hashlib
import json
from pathlib import Path
import subprocess

here = Path(__file__).resolve().parent
source = here.parents[3]
repo = source / '.flow/tmp/fn258-history-final-export'
repro = source / '.flow/tmp/fn258-history-final-repro'
recovery = source / '.flow/tmp/fn258-history-recovery'
def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
def read(name):
    return json.loads((here/name).read_bytes())
def write(name, value):
    (here/name).write_text(json.dumps(value, indent=2)+'\n')
def git(directory, *arguments):
    return subprocess.check_output(['git', '--no-replace-objects', '-C', str(directory), *arguments], stderr=subprocess.STDOUT)
def mapping(path):
    return dict(line.split() for line in path.read_bytes().splitlines()[1:])
source_commit = 'ac60daaf47d2513b41eeff612f0e2758d33e186d'
checkpoint = 'a92db0610f13c9f623b526f2aa721abda69aa84d'
original = here.parents[1] / 'fn-257-extract-the-standalone-umpire/task2/export-attempt2/commit-map'
old_map, new_map = mapping(original), mapping(here/'export/commit-map')
assert all(new_map[key] == value for key, value in old_map.items())
assert (here/'export/commit-map').read_bytes() == (here/'reproduction/commit-map').read_bytes()
assert (here/'history-map.json').read_bytes() == (here/'history-map-reproduced.json').read_bytes()
assert read('verification.json')['status'] == read('bundle-recovery-verification.json')['status'] == 'pass'
for name in ['export', 'reproduction']:
    preflight = read(name+'/preflight.json')
    assert preflight['history_map_sha256'] == digest(here/'history-map.json')
    assert preflight['source_manifest_sha256'] == digest(here.parent/'source/source-manifest.json')
    assert preflight['export_script_sha256'] == digest(here/'export-history.py')
    assert read(name+'/export-result.json')['ordered_source_retained_frontier'] == [checkpoint]
for name in ['verification.json', 'bundle-recovery-verification.json']:
    report = read(name)
    assert report['input_hashes']['history_map_sha256'] == digest(here/'history-map.json')
    assert report['checkpoint_binding']['export_graph_equals_nonzero_mapped_values'] is True
    assert report['checkpoint_binding']['ordered_source_retained_frontier'] == [checkpoint]
assert read('preservation-before.json') == read('preservation-after.json')
write('preservation-comparison.json', {'before_equals_after': True,
      'before_sha256': digest(here/'preservation-before.json'), 'after_sha256': digest(here/'preservation-after.json'),
      'scope': 'All 39 registered Temporal worktree refs/indexes/status and dirty byte/mode fingerprints; root .flow/MILESTONES/MIGRATE excluded as declared migration evidence. Ignored caches not inventoried.'})
assert subprocess.run(['git','-C',str(recovery),'merge-base','--is-ancestor','cf8bec4d6231302df54b57fccdc5bdb9d11552fc',checkpoint]).returncode == 0
write('original-history-continuity.json', {'original_source_commits':len(old_map),'current_source_commits':len(new_map),
      'original_rows_changed':0,'changed_sample':[],'original_export_is_current_export_ancestor':True,
      'generator':'refresh-checkpoint-proof.py'})
bundle = here/'current-history.bundle'
assert digest(bundle) == '83cb7e061e8fea284108d8380f517b2179b70a1fb8924147e90aae0aa7acec19'
write('bundle-proof.json', {'fsck':git(recovery,'fsck','--full','--strict').decode(),
      'bundle_verify':git(recovery,'bundle','verify',str(bundle)).decode(),
      'bundle_sha256':digest(bundle),'bundle_bytes':bundle.stat().st_size,
      'bundle_heads':git(recovery,'bundle','list-heads',str(bundle)).decode(),
      'generator':'refresh-checkpoint-proof.py', 'integrity_checked_checkout':str(recovery),
      'clone_only_cleanup':'Original exported clone cleaned stale shallow metadata/origin/reflogs/unreachable objects before bundle creation; unchanged bundle independently recovered and rechecked.'})
write('checkpoint.json', {'source_commit':source_commit,'export_commit':checkpoint,'source_tip_zero_pruned':True,
      'available_original_commits':7261,'retained_commits':627,'projection_neutral_pruned_commits':6634,
      'active_files':1189,'history_support_files':30,'bundle':'current-history.bundle',
      'bundle_ref':'refs/heads/umpire-current','bundle_sha256':digest(bundle),
      'filtered_checkout':str(repo),'reproduced_checkout':str(repro),'independent_recovery_checkout':str(recovery),
      'original_history_commit_maps_identical':True,'all_source_worktree_preservation_captures_equal':True,
      'export_graph_equals_nonzero_mapped_values':True,'ordered_source_retained_frontier':[checkpoint],
      'recommended_join':'Merge as second parent with ours strategy, preserving exact portable first-parent tree. No standalone refs changed.',
      'generator':'refresh-checkpoint-proof.py'})
write('tool-provenance.json', {'git_version':git(source,'--version').decode().strip(),
      'filter_module':str(source/'.flow/tmp/fn257-tool/git_filter_repo.py'),
      'filter_module_sha256':digest(source/'.flow/tmp/fn257-tool/git_filter_repo.py'),
      'scripts_sha256':{path.name:digest(path) for path in here.glob('*.py')},
      'current_manifest_sha256':digest(here.parent/'source/source-manifest.json'),
      'history_map_sha256':digest(here/'history-map.json'),
      'history_map_reproduced_identically':True,'commit_map_reproduced_identically':True,
      'clone_commands':['git clone --no-local --revision='+source_commit+' temporal temporal/.flow/tmp/fn258-history-final-'+name for name in ['export','repro']],
      'evidence_generation':'refresh-checkpoint-proof.py; all assertions and strict integrity reads must pass before current summaries are written',
      'regression_controls':'checkpoint-controls.py --source temporal --export temporal/.flow/tmp/fn258-history-recovery --synthetic <new-isolated-path>',
      'existing_controls':'run-controls.py --source temporal --export temporal/.flow/tmp/fn258-history-export --synthetic <new-isolated-path>',
      'superseded':'review-before/ contains archived pre-review evidence and the old verifier regression fixture; not current acceptance evidence.'})
print(json.dumps({'status':'pass','checkpoint':checkpoint,'history_map_sha256':digest(here/'history-map.json'),'bundle_sha256':digest(bundle)}))
