#!/usr/bin/env python3
"""Verify that the provenance join retains the portable tree and both ancestries."""
import hashlib
import json
import pathlib
import subprocess

HERE = pathlib.Path(__file__).resolve().parent
TEMPORAL = next(p for p in HERE.parents if (p / '.flow/specs').is_dir())
UMPIRE = TEMPORAL.parent / 'umpire'
PORTABLE = '5ee8911480c8acb429625bef3f3183e572b9d18f'
EXPORT = 'a92db0610f13c9f623b526f2aa721abda69aa84d'
JOIN = 'c71c3090fa0f89e744631c6590f1d8b4b0ffd94f'
FINAL = 'ac79114a9ad43a9d9fc2a022d07406d217878760'
ORIGINAL = 'cf8bec4d6231302df54b57fccdc5bdb9d11552fc'
CURRENT = '05473210b48c3db862f22051eb7154ce98258f98'
CURRENT_TREE = '46829511b6d582b40f3fa5095f42bdf980c7d466'
RECEIVER_PATHS = ['dev/xtask/src/gate.rs', 'dev/xtask/tests/gate_wiring.rs']


def git(*args):
    return subprocess.check_output(['git', '-C', str(UMPIRE), *args]).decode().strip()


def ancestor(old, new):
    return subprocess.run(['git', '-C', str(UMPIRE), 'merge-base', '--is-ancestor', old, new], check=False).returncode == 0


def tree(revision):
    result = {}
    raw = subprocess.check_output(['git', '-C', str(UMPIRE), 'ls-tree', '-rz', revision])
    for row in raw.split(b'\0'):
        if row:
            metadata, path = row.split(b'\t', 1)
            mode, kind, blob = metadata.decode().split()
            result[path.decode()] = dict(mode=mode, blob=blob)
    return result


parents = git('rev-list', '--parents', '-n', '1', JOIN).split()[1:]
portable_tree, join_tree, final_tree = [git('rev-parse', commit + '^{tree}') for commit in [PORTABLE, JOIN, FINAL]]
portable_commits = git('rev-list', '--reverse', ORIGINAL + '..' + PORTABLE).splitlines()
ancestry = [{'commit': oid, 'reachable_from_join': ancestor(oid, JOIN), 'reachable_from_final': ancestor(oid, FINAL)} for oid in [ORIGINAL, *portable_commits, EXPORT]]
manifest_path = HERE.parent / 'source/source-manifest.json'
manifest = json.loads(manifest_path.read_text())
export_tree = tree(EXPORT)
selected = {r['destination']: dict(mode=r['mode'], blob=r['blob']) for r in manifest['records']}
missing = sorted(selected.keys() - export_tree.keys())
modified = sorted(p for p in selected.keys() & export_tree.keys() if selected[p] != export_tree[p])
final_paths = git('diff-tree', '--no-commit-id', '--name-only', '-r', JOIN, FINAL).splitlines()
relative = 'dev/xtask/src/gate/boundary.rs'
before = git('show', JOIN + ':' + relative)
after = git('show', FINAL + ':' + relative)
# Exact production text must survive; test helper spellings may change below cfg(test).
production_unchanged = before.split('#[cfg(test)]', 1)[0] == after.split('#[cfg(test)]', 1)[0]
checks = {'exact_ordered_parents': parents == [PORTABLE, EXPORT],
          'join_tree_equals_first_parent': join_tree == portable_tree == '9324a0e75c455bd060cc1faeb328769a556d0f2d',
          'all_portable_and_export_ancestors_reachable': all(a['reachable_from_join'] and a['reachable_from_final'] for a in ancestry),
          'export_selected_tree_equals_frozen_source': not missing and not modified,
          'final_parent_is_history_join': git('rev-parse', FINAL + '^') == JOIN,
          'final_changes_only_boundary_tests': final_paths == [relative] and production_unchanged}
policy_path = HERE/'adaptation-allowlist.json'
policy = json.loads(policy_path.read_bytes())
receiver_tree = tree(CURRENT)
receiver_paths = git('diff-tree', '--no-commit-id', '--name-only', '-r', FINAL, CURRENT).splitlines()
receiver_diff = subprocess.check_output(['git', '-C', str(UMPIRE), 'diff', '--binary', FINAL, CURRENT, '--', *RECEIVER_PATHS])
receiver_ancestry = [{'commit': oid, 'reachable_from_current_receiver': ancestor(oid, CURRENT)}
                     for oid in [ORIGINAL, *portable_commits, EXPORT, JOIN, FINAL]]
receiver_checks = {
    'current_receiver_matches_explicit_reviewed_pin': policy['standalone_head'] == CURRENT and git('rev-parse', CURRENT+'^{tree}') == CURRENT_TREE,
    'current_receiver_parent_is_historical_final': git('rev-list', '--parents', '-n', '1', CURRENT).split()[1:] == [FINAL],
    'exactly_two_explicitly_approved_receiver_paths': receiver_paths == RECEIVER_PATHS,
    'receiver_blobs_and_modes_match_reviewed_allowlist': all(receiver_tree[path] == policy['modified_paths'][path]['approved_after'] for path in RECEIVER_PATHS),
    'all_historical_and_filtered_ancestries_retained_by_current_receiver': all(item['reachable_from_current_receiver'] for item in receiver_ancestry),
}
checks.update(receiver_checks)
result = dict(result='pass' if all(checks.values()) else 'fail', checks=checks,
              join_commit=JOIN, ordered_parents=parents, first_parent_tree=portable_tree, join_tree=join_tree,
              final_commit=FINAL, final_tree=final_tree, final_changed_paths=final_paths,
              original_export=ORIGINAL, current_export=EXPORT, frozen_source_commit=manifest['source_commit'],
              manifest_sha256=hashlib.sha256(manifest_path.read_bytes()).hexdigest(),
              export_selected_records=len(selected), export_missing_selected_paths=missing, export_modified_selected_paths=modified,
              export_extra_provenance_paths=sorted(export_tree.keys()-selected.keys()), ancestry_checks=ancestry,
              current_receiver_head=CURRENT, current_receiver_tree=CURRENT_TREE,
              current_receiver_base=FINAL, current_receiver_ordered_parents=git('rev-list', '--parents', '-n', '1', CURRENT).split()[1:],
              current_receiver_changed_paths=receiver_paths,
              current_receiver_approved_records={path: receiver_tree[path] for path in RECEIVER_PATHS},
              current_receiver_diff_sha256=hashlib.sha256(receiver_diff).hexdigest(),
              current_receiver_ancestry_checks=receiver_ancestry,
              current_receiver_allowlist_sha256=hashlib.sha256(policy_path.read_bytes()).hexdigest(),
              historical_final_qualification='final_commit/final_tree and final_changes_only_boundary_tests refer exclusively to historical ac79114; current_receiver fields separately prove the approved two-path descendant.',
              scope='Exact Git ancestry/tree and production-text checks; full export parent-edge verification belongs to the separate history lane. No runtime or publication claim.')
(HERE/'history-join-verification.json').write_text(json.dumps(result,indent=2)+'\n')
print(json.dumps(result,indent=2))
if result['result'] != 'pass':
    raise SystemExit(1)
