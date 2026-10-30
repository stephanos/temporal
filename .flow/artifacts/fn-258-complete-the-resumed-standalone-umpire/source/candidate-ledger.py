#!/usr/bin/env python3
"""Capture read-only candidate ownership; verify frozen manifest and recovery refs."""
import argparse
import collections
import datetime
import hashlib
import json
import pathlib
import subprocess
from unittest import mock

HERE = pathlib.Path(__file__).resolve().parent
TEMPORAL = next(p for p in HERE.parents if (p / '.flow/specs').is_dir())
STANDALONE = TEMPORAL.parent / 'umpire'
SOURCE = 'ac60daaf47d2513b41eeff612f0e2758d33e186d'
OLD = TEMPORAL / '.flow/artifacts/fn-257-extract-the-standalone-umpire'


def git(repo, *args, check=True):
    return subprocess.run(['git', '-C', str(repo), *args], check=check,
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE)


def read(path):
    return json.loads(path.read_bytes())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(name, data):
    (HERE / name).write_text(json.dumps(data, indent=2, ensure_ascii=True) + '\n')


def refs(repo):
    return dict(line.split(' ', 1) for line in git(
        repo, 'for-each-ref', '--format=%(refname) %(objectname)').stdout.decode().splitlines())


def ancestor(repo, older, newer):
    return git(repo, 'merge-base', '--is-ancestor', older, newer, check=False).returncode == 0


def change_mapping(repository, path, temporal_mappings):
    if repository == 'umpire':
        return {'ownership': 'framework', 'repository': 'umpire', 'path': path,
                'mapping_status': 'standalone repository root; independent of Temporal path namespace'}
    return temporal_mappings.get(path)


def mapping_errors(entry, temporal_mappings):
    errors = []
    for change in entry['committed_changes'] + entry['dirty_changes']:
        mapping = change_mapping(entry['repository'], change['path'], temporal_mappings)
        if mapping is None:
            errors.append('candidate path lacks ownership mapping: ' + entry['repository'] + ':' + change['path'])
        elif change['ownership'] != mapping['ownership']:
            errors.append('candidate ownership differs from mapping: ' + entry['repository'] + ':' + change['path'])
    return errors


def original_recovery(repo, original_head, current_head):
    if original_head is None:
        return {'original_head_object_present': None,
                'original_head_reachable_from_current_ref': None,
                'current_refs_containing_original_head': [], 'original_recovery_gaps': []}
    present = git(repo, 'cat-file', '-e', original_head + '^{commit}', check=False).returncode == 0
    gaps, containing = [], []
    if present:
        result = git(repo, 'for-each-ref', '--contains', original_head, '--format=%(refname)', check=False)
        if result.returncode:
            gaps.append('Original head containing refs unknown: ' + result.stderr.decode(errors='replace').strip())
        else:
            containing = result.stdout.decode().splitlines()
    else:
        gaps.append('Original head object missing: ' + original_head + '; recovery reachability is unknown.')
    return {'original_head_object_present': present,
            'original_head_reachable_from_current_ref': ancestor(repo, original_head, current_head) if present and current_head else None,
            'current_refs_containing_original_head': containing, 'original_recovery_gaps': gaps}


def candidate_delta(repo, source, head):
    result = git(repo, 'merge-base', source, head, check=False)
    if result.returncode:
        diagnostic = result.stderr.decode(errors='replace').strip() or 'no common ancestor'
        return None, [], 'merge-base unavailable: ' + diagnostic
    base = result.stdout.decode().strip()
    result = git(repo, 'diff', '--no-renames', '--name-status', '-z', base, head, check=False)
    if result.returncode:
        return base, [], 'candidate diff unavailable: ' + result.stderr.decode(errors='replace').strip()
    parts = result.stdout.split(b'\0')
    changes = [{'status': parts[i].decode(), 'path': parts[i + 1].decode(errors='surrogateescape')}
               for i in range(0, len(parts) - 1, 2)]
    return base, changes, None


def candidate_errors(entry, mappings):
    errors = mapping_errors(entry, mappings)
    if entry['repository'] != 'temporal' or not entry['observed_head']:
        return errors
    base, changes, delta_error = candidate_delta(TEMPORAL, SOURCE, entry['observed_head'])
    if delta_error:
        errors.append('candidate delta cannot be verified: ' + entry['ref'] + ': ' + delta_error)
        return errors
    observed = [(e['status'], e['path']) for e in changes]
    recorded = [(e['status'], e['path']) for e in entry['committed_changes']]
    if base != entry['merge_base_with_frozen_source'] or observed != recorded:
        errors.append('candidate delta differs: ' + entry['ref'])
    return errors


def synthetic_controls():
    controls = []
    path = '.flow/shared-path.json'
    mappings = {path: {'ownership': 'tracker-evidence', 'repository': 'temporal', 'path': path}}
    entry = {'repository': 'umpire', 'observed_head': 'synthetic-head', 'committed_changes': [],
             'dirty_changes': [{'status': '??', 'path': path, 'ownership': 'framework'}]}
    controls.append({'name': 'dirty-standalone-path-has-independent-framework-owner',
                     'passed': not candidate_errors(entry, mappings)
                     and change_mapping('umpire', path, mappings)['repository'] == 'umpire'
                     and mappings[path]['ownership'] == 'tracker-evidence'})
    bad = dict(entry, dirty_changes=[dict(entry['dirty_changes'][0], ownership='tracker-evidence')])
    controls.append({'name': 'wrong-standalone-owner-is-refused', 'passed': bool(candidate_errors(bad, mappings))})
    calls = []

    def missing_object(repo, *args, check=True):
        calls.append(args)
        if args != ('cat-file', '-e', 'missing-object^{commit}') or check:
            raise AssertionError('Unavailable object must not reach a checked contains/ancestry query')
        return subprocess.CompletedProcess(args, 128, b'', b'fatal: not a valid object name')

    with mock.patch.dict(globals(), {'git': missing_object}):
        recovery = original_recovery(TEMPORAL, 'missing-object', 'current-head')
    controls.append({'name': 'missing-original-object-retains-unknown-recovery-gap',
                     'passed': recovery['original_head_object_present'] is False
                     and recovery['original_head_reachable_from_current_ref'] is None
                     and recovery['current_refs_containing_original_head'] == []
                     and bool(recovery['original_recovery_gaps']) and len(calls) == 1})

    def orphan(repo, *args, check=True):
        if args != ('merge-base', SOURCE, 'orphan-head') or check:
            raise AssertionError('Orphan branch must report missing merge-base without running diff')
        return subprocess.CompletedProcess(args, 1, b'', b'')

    orphan_entry = {'repository': 'temporal', 'observed_head': 'orphan-head', 'ref': 'refs/heads/synthetic-orphan',
                    'committed_changes': [], 'dirty_changes': [], 'merge_base_with_frozen_source': None}
    with mock.patch.dict(globals(), {'git': orphan}):
        base, changes, gap = candidate_delta(TEMPORAL, SOURCE, 'orphan-head')
        errors = candidate_errors(orphan_entry, {})
    controls.append({'name': 'orphan-capture-gap-and-verifier-error-are-explicit',
                     'passed': base is None and changes == [] and gap == 'merge-base unavailable: no common ancestor'
                     and errors == ['candidate delta cannot be verified: refs/heads/synthetic-orphan: ' + gap]})
    result = {'result': 'pass' if all(c['passed'] for c in controls) else 'fail', 'controls': controls,
              'method': 'In-memory entries and synthetic Git responses exercise the same capture/verification helpers; no repository/ref/worktree changes.'}
    save('candidate-ledger-controls.json', result)
    return result


def worktrees(repo):
    entries = []
    for block in git(repo, 'worktree', 'list', '--porcelain', '-z').stdout.split(b'\0\0'):
        fields = dict(piece.decode(errors='surrogateescape').split(' ', 1)
                      if b' ' in piece else (piece.decode(), True)
                      for piece in block.split(b'\0') if piece)
        if not fields:
            continue
        path = pathlib.Path(fields['worktree'])
        status = git(path, 'status', '--porcelain=v1', '-z', '--untracked-files=all', check=False)
        fields['status_hex'] = status.stdout.hex()
        fields['status_exit'] = status.returncode
        fields['dirty_paths'] = []
        parts = iter(status.stdout.split(b'\0'))
        for item in parts:
            if not item:
                continue
            code, source = item[:2].decode(), item[3:].decode(errors='surrogateescape')
            fields['dirty_paths'].append({'status': code, 'path': source})
            if 'R' in code or 'C' in code:
                fields['dirty_paths'].append({'status': 'rename-source', 'path': next(parts).decode(errors='surrogateescape')})
        fields['index_stage_sha256'] = hashlib.sha256(git(path, 'ls-files', '--stage', '-z', check=False).stdout).hexdigest()
        fields['tracked_worktree_diff_sha256'] = hashlib.sha256(git(path, 'diff', '--binary', 'HEAD', check=False).stdout).hexdigest()
        fields['backup_status'] = 'observation only; no dirty/untracked/unique ignored backup claimed'
        entries.append(fields)
    return entries


def capture():
    old = read(OLD / 'task1/candidate-ledger.json')
    transfer = read(OLD / 'task6/provisional-transfer-plan.json')
    old_entries = {e['ref']: (i, e) for i, e in enumerate(old['entries'])}
    old_transfer = {e['candidate_id']: e for e in transfer['entries']}
    mappings = dict(old['path_mappings'])

    def ownership(path):
        if path in mappings:
            return mappings[path]
        if path.startswith('model/umpire/'):
            result = {'ownership': 'framework', 'repository': 'umpire', 'path': path.removeprefix('model/umpire/'), 'mapping_status': 'current-framework-prefix'}
        elif path.startswith(('model/archived/', 'model2/archived/')) or path.endswith('.scala'):
            result = {'ownership': 'retained-legacy-scala', 'repository': 'temporal', 'path': path, 'mapping_status': 'retain historical lineage; no replay'}
        elif path.startswith(('model2/framework/', 'model2/umpire/', 'model2/hosts/go/', 'model/hosts/go/')):
            result = {'ownership': 'historical-framework', 'repository': 'umpire', 'path': None, 'mapping_status': 'requires reviewed historical-to-current path map'}
        elif path.startswith('.flow/'):
            result = {'ownership': 'tracker-evidence', 'repository': 'temporal', 'path': path, 'mapping_status': 'retain original record; link receiving obligations separately'}
        elif path.startswith(('model/temporal/', 'tools/', 'common/', 'tests/', 'service/', 'api/', 'proto/', 'components/')):
            result = {'ownership': 'temporal', 'repository': 'temporal', 'path': path, 'mapping_status': 'retain consumer/system owner'}
        else:
            result = {'ownership': 'shared-or-uncertain', 'repository': 'temporal', 'path': path, 'mapping_status': 'retain original; inspect ownership/hunks before integration or retirement'}
        mappings[path] = result
        return result

    now = datetime.datetime.now(datetime.timezone.utc).isoformat()
    repositories = {'temporal': TEMPORAL, 'umpire': STANDALONE}
    snapshots = {name: {'refs': refs(repo), 'worktrees': worktrees(repo)} for name, repo in repositories.items()}
    entries = []
    for name, repo in repositories.items():
        current = snapshots[name]['refs']
        names = {ref for ref in current if ref.startswith('refs/heads/')}
        if name == 'temporal':
            names.update(old_entries)
        for ref in sorted(names):
            head = current.get(ref)
            original_index, original = old_entries.get(ref, (None, {})) if name == 'temporal' else (None, {})
            original_head = original.get('head')
            prior = old_transfer.get(ref, {}) if name == 'temporal' else {}
            related = [w for w in snapshots[name]['worktrees'] if w.get('branch') == ref]
            changes, base, delta_error = [], None, None
            if head and name == 'temporal':
                base, changes, delta_error = candidate_delta(repo, SOURCE, head)
                for change in changes:
                    change['ownership'] = ownership(change['path'])['ownership']
            recovery = original_recovery(repo, original_head, head)
            dirty = [dict(d, worktree=w['worktree']) for w in related for d in w['dirty_paths']]
            classes = collections.Counter(e['ownership'] for e in changes)
            for d in dirty:
                mapping = ownership(d['path']) if name == 'temporal' else change_mapping(name, d['path'], mappings)
                d['ownership'] = mapping['ownership']
            all_classes = set(classes) | {d['ownership'] for d in dirty}
            has_framework = bool(all_classes & {'framework', 'export-framework', 'historical-framework'}) or any('framework' in c for c in all_classes)
            has_other = bool(all_classes - {'framework', 'export-framework', 'historical-framework'})
            if name == 'umpire':
                category = 'framework'
            elif has_framework and has_other:
                category = 'mixed'
            elif has_framework:
                category = 'framework'
            elif all_classes and all(c == 'retained-legacy-scala' for c in all_classes):
                category = 'legacy'
            else:
                category = 'Temporal'
            tasks = prior.get('source_task_records', original.get('intended_tasks', []))
            ongoing = ref in {'refs/heads/work/fn2583-consumers', 'refs/heads/work/fn2573-standalone', 'refs/heads/work/fn2577-quick', 'refs/heads/work/fn2574-consumers'}
            entries.append({
                'repository': name, 'ref': ref, 'observed_head': head,
                'original_ledger_index': original_index, 'original_head': original_head,
                **recovery,
                'missing_ref': head is None, 'preservation': 'retain current ref and worktree unchanged; missing/uncertain candidates stay explicitly frozen here',
                'merge_base_with_frozen_source': base, 'committed_delta_error': delta_error,
                'head_integrated_in_frozen_source': ancestor(repo, head, SOURCE) if head and name == 'temporal' else None,
                'ownership': category, 'committed_delta_counts': dict(classes), 'committed_changes': changes,
                'worktrees': [w['worktree'] for w in related], 'dirty_changes': dirty,
                'original_task_links': tasks,
                'current_task_record_observations': [{'id': t.get('id'), 'record_exists': (TEMPORAL / t.get('task_record', '__missing__')).is_file(), 'status': read(TEMPORAL / t['task_record'].replace('.md', '.json')).get('status') if t.get('task_record') and (TEMPORAL / t['task_record'].replace('.md', '.json')).is_file() else 'unresolved'} for t in tasks if isinstance(t, dict)],
                'migration_preparation_tracking': 'fn-258-complete-the-resumed-standalone-umpire' if ongoing else None,
                'task_assignment_basis': original.get('task_assignment_basis', 'migration preparation ref; no historical task inferred from branch name'),
                'ongoing_preparation_only': ongoing,
                'provisional_future_repositories': sorted({'umpire' if has_framework else 'temporal'} | ({'temporal'} if has_other else set())) if name == 'temporal' else ['umpire'],
                'cross_repository_prerequisites': ['verified-current-standalone-and-consumer-pair', 'selective-candidate-path-and-hunk-disposition', 'receiving-ref-and-obligation-records', 'dirty-untracked-unique-ignored-backup-before-retirement'],
                'historical_evidence_paths': prior.get('retained_evidence_paths', original.get('changed_tracker_evidence_paths', [])),
                'destination_obligations': {'receiving_candidate_refs': [], 'receiving_tracker_records': [], 'required_action': 'Review actual candidate hunks, deduplicate inherited assembly changes, preserve original task/proof links and dependencies; establish destination recovery and tracker records before claiming transfer.'},
                'transfer_status': 'ongoing-preparation-only' if ongoing else 'not-transferred', 'r6_disposition_complete': False,
                'gaps': recovery['original_recovery_gaps'] + ([delta_error] if delta_error else []) + (['Observed ref is missing; retain original OID and historical evidence; resolve recovery before retirement.'] if head is None else []) + ([] if tasks else ['No corroborated historical task assignment; ownership remains provisional.']) + ['Dirty/untracked/unique ignored source or evidence has not been backed up by this inventory.'],
            })
    counts = {'original_candidates': len(old_entries), 'original_refs_found': sum(e['original_ledger_index'] is not None and not e['missing_ref'] for e in entries), 'repository_ref_entries': len(entries), 'entries_by_ownership': dict(collections.Counter(e['ownership'] for e in entries)), 'r6_completed_dispositions': 0, 'original_candidates_transferred': 0}
    ledger = {'schema_version': 1, 'status': 'renewed provisional inventory; no R6 transfer completion', 'observed_at': now, 'source_commit': SOURCE,
              'inputs': {str(p.relative_to(TEMPORAL)): sha(p) for p in [OLD/'task1/candidate-ledger.json', OLD/'task6/provisional-transfer-plan.json', HERE/'source-manifest.json', HERE/'candidate-observation.json']},
              'scope': 'All current local branch refs in both repositories plus every original candidate, all registered worktrees and every other observed ref namespace. Merge-base deltas show inherited assemblies, not exclusive contribution. Ignored files are not inventoried or protected by branch refs.',
              'retained_preparation_checkpoints': [{'repository': 'umpire', 'commit': '58a6989fe5ccd6a1ae734326d568f60dce2dab43', 'purpose': 'path-relative c88fdc5506..ac60daaf47 integrated framework transfer preparation', 'reachable_from_observed_standalone_head': ancestor(STANDALONE, '58a6989fe5ccd6a1ae734326d568f60dce2dab43', git(STANDALONE, 'rev-parse', 'HEAD').stdout.decode().strip()), 'r6_original_candidate_transfer': False}],
              'historical_prerequisites': old['prerequisites'], 'historical_prerequisite_note': 'Historical context only: old validation hold is superseded by resumed fn-258 execution; existing fn-254/fn-255/fn-246 obligations remain open.',
              'counts': counts, 'repository_snapshots': snapshots, 'entries': entries, 'path_mappings': mappings,
              'remaining_r6_gaps': ['No original candidate receiving recovery refs or destination tracker records created.', 'Mixed/shared/historical candidates require selective path/hunk mapping and deduplication.', 'Unresolved tasks stay unresolved; no numeric branch-name attribution.', 'Dirty/untracked/unique ignored source/evidence must be checkpointed and verified before any retirement.', 'Standalone 58a6989 and current work/fn2583-consumers are preparation only, not original-candidate transfer.', 'Current standalone/paired validation and local mutation proofs, source ownership cutover, destination-access/publication disposition and final closure remain separate gates.']}
    save('candidate-ledger.json', ledger)


def verify():
    manifest = read(HERE/'source-manifest.json')
    ledger = read(HERE/'candidate-ledger.json')
    errors, drift = [], []
    assert manifest['source_commit'] == ledger['source_commit'] == SOURCE
    framework = git(TEMPORAL, 'ls-tree', '-rz', SOURCE, '--', 'model/umpire').stdout
    tree_records = {}
    for raw in framework.split(b'\0'):
        if raw:
            metadata, path = raw.split(b'\t', 1)
            mode, kind, blob = metadata.decode().split()
            tree_records[path.decode()] = (mode, blob)
    selected = {r['source']: r for r in manifest['records'] if r['source'].startswith('model/umpire/')}
    if set(selected) != set(tree_records):
        errors.append('manifest framework paths differ from frozen tracked tree')
    for record in manifest['records']:
        raw = git(TEMPORAL, 'cat-file', 'blob', record['blob']).stdout
        if len(raw) != record['size'] or hashlib.sha256(raw).hexdigest() != record['sha256']:
            errors.append('manifest blob mismatch: ' + record['source'])
        if record['source'] not in tree_records:
            metadata = git(TEMPORAL, 'ls-tree', SOURCE, '--', record['source']).stdout.split(b'\t', 1)[0].decode().split()
            if not metadata or (metadata[0], metadata[2]) != (record['mode'], record['blob']):
                errors.append('supplemental mode/blob mismatch: ' + record['source'])
        if record['source'] in tree_records and tree_records[record['source']] != (record['mode'], record['blob']):
            errors.append('manifest mode/blob mismatch: ' + record['source'])
    delta = read(HERE/'integrated-framework-delta.json')
    patch = git(TEMPORAL, 'diff', '--binary', '--relative=model/umpire', delta['source_baseline'], SOURCE, '--', 'model/umpire').stdout
    if hashlib.sha256(patch).hexdigest() != delta['path_relative_binary_patch_sha256']:
        errors.append('integrated framework source patch differs')
    delta_parts = git(TEMPORAL, *delta['command'][1:]).stdout.split(b'\0')
    expected_delta = []
    for i in range(0, len(delta_parts) - 1, 2):
        oldmode, newmode, oldblob, newblob, status = delta_parts[i].decode().lstrip(':').split()
        path = delta_parts[i + 1].decode()
        expected_delta.append(dict(source_path=path, destination_path=path.removeprefix('model/umpire/'), status=status, old_mode=oldmode, new_mode=newmode, old_blob=oldblob, new_blob=newblob))
    if expected_delta != delta['entries']:
        errors.append('integrated framework delta entries differ')
    old = read(OLD/'task1/candidate-ledger.json')
    original = {e['ref'] for e in old['entries']}
    seen = {e['ref'] for e in ledger['entries'] if e['repository'] == 'temporal' and e['original_ledger_index'] is not None}
    if original != seen:
        errors.append('original candidate coverage differs')
    for path, digest in ledger['inputs'].items():
        if sha(TEMPORAL/path) != digest:
            errors.append('input digest differs: ' + path)
    for entry in ledger['entries']:
        errors.extend(candidate_errors(entry, ledger['path_mappings']))
    for name, repo in [('temporal', TEMPORAL), ('umpire', STANDALONE)]:
        snapshot = ledger['repository_snapshots'][name]
        frozen_heads = {r for r in snapshot['refs'] if r.startswith('refs/heads/')}
        recorded = {e['ref'] for e in ledger['entries'] if e['repository'] == name and not e['missing_ref']}
        if frozen_heads != recorded:
            errors.append(name + ' snapshot local ref coverage differs')
        current = refs(repo)
        for ref, oid in snapshot['refs'].items():
            if current.get(ref) != oid:
                drift.append({'repository': name, 'ref': ref, 'observed': oid, 'current': current.get(ref)})
            if git(repo, 'cat-file', '-e', oid, check=False).returncode:
                errors.append(name + ' frozen recovery object missing: ' + ref)
        for ref in current.keys() - snapshot['refs'].keys():
            drift.append({'repository': name, 'new_ref_after_observation': ref})
    controls = synthetic_controls()
    if controls['result'] != 'pass':
        errors.append('synthetic ledger controls failed')
    result = {'synthetic_controls': controls, 'source_commit': SOURCE, 'ledger_sha256': sha(HERE/'candidate-ledger.json'), 'manifest_sha256': sha(HERE/'source-manifest.json'), 'verified_manifest_records': len(manifest['records']), 'verified_original_candidates': len(original), 'errors': errors, 'live_ref_drift_since_capture': drift, 'result': 'pass' if not errors else 'fail', 'interpretation': 'Completeness and frozen recovery-object verification only; no transfer, dirty backup or current-pair execution claim.'}
    save('candidate-ledger-verification.json', result)
    print(json.dumps(result, indent=2))
    if errors:
        raise SystemExit(1)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--capture', action='store_true', help='Replace only renewed ledger observation; never alters refs/worktrees')
    parser.add_argument('--controls-only', action='store_true', help='Run in-memory synthetic controls without reading live refs')
    args = parser.parse_args()
    if args.controls_only:
        controls = synthetic_controls()
        print(json.dumps(controls, indent=2))
        raise SystemExit(0 if controls['result'] == 'pass' else 1)
    if args.capture:
        capture()
    verify()
