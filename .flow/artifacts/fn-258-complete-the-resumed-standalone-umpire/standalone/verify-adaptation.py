#!/usr/bin/env python3
"""Compare a pinned standalone tree with the frozen source and reviewed path allowlist."""
import argparse
import copy
import hashlib
import json
import pathlib
import re
import subprocess

HERE = pathlib.Path(__file__).resolve().parent
TEMPORAL = next(p for p in HERE.parents if (p / '.flow/specs').is_dir())
UMPIRE = TEMPORAL.parent / 'umpire'
SOURCE = HERE.parent / 'source/source-manifest.json'


def git(repo, *args):
    return subprocess.check_output(['git', '-C', str(repo), *args])


def read(path):
    return json.loads(path.read_bytes())


def digest(data):
    return hashlib.sha256(data).hexdigest()


def tree(repo, revision):
    result = {}
    for row in git(repo, 'ls-tree', '-rz', revision).split(b'\0'):
        if row:
            metadata, path = row.split(b'\t', 1)
            mode, kind, blob = metadata.decode().split()
            result[path.decode()] = {'mode': mode, 'blob': blob}
    return result


def protected(path):
    parts = path.split('/')
    return (path == 'LICENSE' or pathlib.PurePosixPath(path).name in {'Cargo.lock', 'go.sum', 'go.work.sum', 'mise.lock'}
            # Freeze the whole fixture crate, including executable source and build.rs, for this migration.
            or any(p in {'fixtures', 'snapshots', 'expected', 'goldens'} for p in parts)
            or '.snap' in pathlib.PurePosixPath(path).suffixes
            or pathlib.PurePosixPath(path).name in {'metadata.json', 'snapshot.json', 'manifest.json'})


def compare(source, actual, allow, additions, removed):
    missing = sorted(source.keys() - actual.keys())
    extra = sorted(actual.keys() - source.keys())
    changed = sorted(p for p in source.keys() & actual.keys() if (source[p]['mode'], source[p]['blob']) != (actual[p]['mode'], actual[p]['blob']))
    errors = []
    if missing:
        errors.append({'kind': 'missing-source', 'paths': missing})
    unexpected = sorted(set(changed) - allow.keys())
    if unexpected:
        errors.append({'kind': 'unexpected-modified', 'paths': unexpected})
    protected_changes = [p for p in changed + missing if protected(p)]
    if protected_changes:
        errors.append({'kind': 'protected-difference', 'paths': protected_changes})
    modes = [p for p in source.keys() & actual.keys() if source[p]['mode'] != actual[p]['mode']]
    if modes:
        errors.append({'kind': 'mode-difference', 'paths': sorted(modes)})
    wrong_pins = [p for p in allow if p not in actual or actual[p] != allow[p]['approved_after']]
    if wrong_pins:
        errors.append({'kind': 'unreviewed-allowlisted-content', 'paths': wrong_pins})
    missing_support = sorted(additions.keys() - actual.keys())
    if missing_support:
        errors.append({'kind': 'missing-approved-support', 'paths': missing_support})
    unexpected_extra = sorted(set(extra) - additions.keys())
    if unexpected_extra:
        errors.append({'kind': 'unexpected-added', 'paths': unexpected_extra})
    wrong_extra = [p for p in extra if p in additions and actual[p] != additions[p]['approved_after']]
    if wrong_extra:
        errors.append({'kind': 'unreviewed-added-content', 'paths': wrong_extra})
    resurrected = sorted(set(removed) & actual.keys())
    if resurrected:
        errors.append({'kind': 'deleted-source-resurrected', 'paths': resurrected})
    return {'result': 'pass' if not errors else 'fail', 'source_files': len(source), 'unchanged_files': len(source)-len(changed)-len(missing),
            'modified_paths': changed, 'missing_source_paths': missing, 'unexpected_modified_paths': unexpected,
            'protected_files': sum(protected(p) for p in source), 'protected_differences': protected_changes,
            'mode_differences': sorted(modes), 'standalone_only_paths': extra, 'errors': errors}



PRIOR_SOURCE = 'c88fdc550694020eb1c6a2e09577813d906c9ffe'
PRIOR_HEAD = 'f7b42ce25558344ea2d38f1e05d002259427a8f6'
PRIOR_INVENTORY_PATH = '.flow/artifacts/fn-257-extract-the-standalone-umpire/task3/adaptation-inventory.json'
PRIOR_INVENTORY_SHA256 = '48d9f6f485aaf0bc649be076fe433d4c0dacdd43cb962e868cd15663b951b5c2'


def load_prior_adaptations():
    inventory_path = TEMPORAL / PRIOR_INVENTORY_PATH
    raw = inventory_path.read_bytes()
    assert digest(raw) == PRIOR_INVENTORY_SHA256, 'Original approved adaptation inventory changed'
    inventory = json.loads(raw)
    assert inventory['source_checkpoint'] == PRIOR_SOURCE
    assert inventory['standalone_head'] == PRIOR_HEAD
    prior_manifest_path = TEMPORAL / '.flow/artifacts/fn-257-extract-the-standalone-umpire/task1/source-manifest.json'
    prior_manifest = read(prior_manifest_path)
    assert prior_manifest['source_commit'] == PRIOR_SOURCE
    original_tree = tree(TEMPORAL, PRIOR_SOURCE)
    prior_tree = tree(UMPIRE, PRIOR_HEAD)
    manifest_records = {record['destination']: record for record in prior_manifest['records']}
    verified = {}
    for record in inventory['modified_files']:
        path = record['path']
        before = manifest_records[path]
        assert original_tree[before['source']] == {'mode': before['mode'], 'blob': before['blob']}
        before_hash = digest(git(TEMPORAL, 'cat-file', 'blob', before['blob']))
        assert before_hash == before['sha256'] == record['before_sha256']
        after = prior_tree[path]
        after_hash = digest(git(UMPIRE, 'cat-file', 'blob', after['blob']))
        assert after_hash == record['after_sha256']
        assert before['mode'] == after['mode']
        verified[path] = {'before_sha256': before_hash, 'after_sha256': after_hash, **after}
    return {'inventory_path': PRIOR_INVENTORY_PATH, 'inventory_sha256': digest(raw),
            'source_manifest_sha256': digest(prior_manifest_path.read_bytes()),
            'source_commit': PRIOR_SOURCE, 'standalone_head': PRIOR_HEAD, 'records': verified}


def compare_prior_adaptations(policy, actual, prior):
    errors, checks = [], []
    if policy['prior_adaptation_inventory_sha256'] != prior['inventory_sha256']:
        errors.append({'kind': 'prior-inventory-hash-mismatch'})
    for path, entry in policy['modified_paths'].items():
        historical = prior['records'].get(path)
        fields = {'prior_adaptation_inventory_path', 'prior_adaptation_after_sha256',
                  'unchanged_since_prior_approved_adaptation'}
        if historical is None:
            if fields & entry.keys():
                errors.append({'kind': 'unknown-prior-adaptation', 'path': path})
            continue
        if not fields <= entry.keys():
            errors.append({'kind': 'missing-prior-adaptation-metadata', 'path': path})
            continue
        if entry['prior_adaptation_inventory_path'] != prior['inventory_path']:
            errors.append({'kind': 'prior-inventory-path-mismatch', 'path': path})
        if entry['prior_adaptation_after_sha256'] != historical['after_sha256']:
            errors.append({'kind': 'stale-prior-adaptation-hash', 'path': path})
        current = actual.get(path)
        current_hash = digest(git(UMPIRE, 'cat-file', 'blob', current['blob'])) if current else None
        unchanged = current is not None and current['mode'] == historical['mode'] and current_hash == historical['after_sha256']
        flag = entry['unchanged_since_prior_approved_adaptation']
        if not isinstance(flag, bool) or flag != unchanged:
            errors.append({'kind': 'prior-adaptation-flag-mismatch', 'path': path})
        checks.append({'path': path, 'prior_blob': historical['blob'],
                       'prior_mode': historical['mode'], 'prior_after_sha256': historical['after_sha256'],
                       'actual_after_sha256': current_hash, 'unchanged_verified': unchanged})
    return {'result': 'pass' if not errors else 'fail', 'errors': errors,
            'verified_prior_blob_count': len(prior['records']), 'verified_claim_count': len(checks),
            'verified_unchanged_count': sum(check['unchanged_verified'] for check in checks),
            'verified_changed_count': sum(not check['unchanged_verified'] for check in checks),
            'inventory_path': prior['inventory_path'], 'inventory_sha256': prior['inventory_sha256'],
            'source_manifest_sha256': prior['source_manifest_sha256'],
            'source_commit': prior['source_commit'], 'standalone_head': prior['standalone_head'],
            'checks': checks}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--head', help='Default: reviewed pinned head, never implicit live HEAD')
    args = parser.parse_args()
    policy = read(HERE/'adaptation-allowlist.json')
    head = args.head or policy['standalone_head']
    manifest = read(SOURCE)
    assert manifest['source_commit'] == policy['frozen_source_commit']
    assert digest(SOURCE.read_bytes()) == policy['source_manifest_sha256']
    source = {r['destination']: {'mode': r['mode'], 'blob': r['blob']} for r in manifest['records']}
    actual = tree(UMPIRE, head)
    result = compare(source, actual, policy['modified_paths'], policy['standalone_only_paths'], policy['deleted_since_original_checkpoint'])
    normalized_checks = []
    for path in policy['modified_paths']:
        if not ((path.endswith('.go') and not path.endswith('.pb.go')) or path.endswith('.proto')):
            continue
        before = git(TEMPORAL, 'cat-file', 'blob', source[path]['blob']).decode()
        after = git(UMPIRE, 'cat-file', 'blob', actual[path]['blob']).decode()
        if path.endswith('.proto'):
            normalize = lambda data: '\n'.join(line for line in data.splitlines() if not line.startswith('option go_package ='))
            kind = 'only-go-package-option-differs'
        else:
            def normalize(data):
                data = data.replace('umpire.local/model', 'github.com/temporalio/umpire/hosts/go').replace('umpire.local/kettle', 'github.com/temporalio/umpire/reference/kettle')
                def imports(match):
                    lines = sorted(line.strip() for line in match.group(1).splitlines() if line.strip())
                    return 'import (' + '\n'.join(lines) + ')'
                return re.sub(r'import \((.*?)\)', imports, data, flags=re.S)
            kind = 'only-canonical-import-identity-and-order-differs'
        passed = normalize(before) == normalize(after)
        normalized_checks.append(dict(path=path, check=kind, passed=passed))
        if not passed:
            result['errors'].append({'kind': 'non-identity-Go-or-proto-change', 'path': path})
            result['result'] = 'fail'
    result['normalized_source_identity_checks'] = normalized_checks
    prior = load_prior_adaptations()
    prior_checks = compare_prior_adaptations(policy, actual, prior)
    result['prior_approved_adaptation_checks'] = prior_checks
    if prior_checks['errors']:
        result['errors'].extend(prior_checks['errors'])
        result['result'] = 'fail'
    result.update({'frozen_source_commit': manifest['source_commit'], 'standalone_head': git(UMPIRE, 'rev-parse', head).decode().strip(),
                   'live_head': git(UMPIRE, 'rev-parse', 'HEAD').decode().strip(), 'standalone_tree': git(UMPIRE, 'rev-parse', head + '^{tree}').decode().strip(),
                   'manifest_sha256': digest(SOURCE.read_bytes()), 'allowlist_sha256': digest((HERE/'adaptation-allowlist.json').read_bytes()),
                   'deleted_source_paths_checked_absent': len(policy['deleted_since_original_checkpoint']),
                   'new_source_paths_checked_present': len(policy['added_since_original_checkpoint']),
                   'protected_records': [dict(path=p, **source[p]) for p in sorted(source) if protected(p)],
                   'scope': 'Git tree byte/mode identity, exact approved adaptations and source path evolution only; no runtime, regeneration, wire-semantic or guest execution claim.'})
    controls = []
    ordinary = next(p for p in sorted(source) if p not in policy['modified_paths'] and not protected(p))
    for name, path, action, expected in [
        ('missing-source', ordinary, 'delete', 'missing-source'),
        ('unexpected-modified', ordinary, 'modify', 'unexpected-modified'),
        ('protected-license', 'LICENSE', 'modify', 'protected-difference'),
        ('protected-fixture', next(p for p in sorted(source) if '/fixtures/' in p and protected(p)), 'modify', 'protected-difference'),
        ('protected-lock', 'Cargo.lock', 'modify', 'protected-difference'),
        ('protected-fixture-build-script', 'dev/fixtures/build.rs', 'modify', 'protected-difference'),
        ('protected-fixture-crate-source', 'dev/fixtures/interpreted.rs', 'modify', 'protected-difference'),
        ('allowlisted-content-change', next(iter(policy['modified_paths'])), 'modify', 'unreviewed-allowlisted-content'),
        ('missing-approved-support', 'Makefile', 'delete', 'missing-approved-support'),
        ('unexpected-added', '__unapproved_source__.rs', 'add', 'unexpected-added'),
        ('deleted-source-resurrection', policy['deleted_since_original_checkpoint'][0], 'add', 'deleted-source-resurrected'),
    ]:
        candidate = copy.deepcopy(actual)
        if action == 'delete':
            del candidate[path]
        else:
            candidate[path] = {'mode': '100644', 'blob': '0'*40}
        refusal = compare(source, candidate, policy['modified_paths'], policy['standalone_only_paths'], policy['deleted_since_original_checkpoint'])
        passed = refusal['result'] == 'fail' and expected in {e['kind'] for e in refusal['errors']}
        controls.append({'name': name, 'mutated_path': path, 'expected_refusal': expected, 'passed': passed})
    prior_path = next(path for path, entry in policy['modified_paths'].items()
                      if entry.get('unchanged_since_prior_approved_adaptation') is True)
    for name, field, value, expected in [
        ('stale-prior-hash', 'prior_adaptation_after_sha256', '0'*64, 'stale-prior-adaptation-hash'),
        ('mismatched-prior-flag', 'unchanged_since_prior_approved_adaptation', False, 'prior-adaptation-flag-mismatch'),
    ]:
        candidate = copy.deepcopy(policy)
        candidate['modified_paths'][prior_path][field] = value
        refusal = compare_prior_adaptations(candidate, actual, prior)
        controls.append({'name': name, 'mutated_path': prior_path, 'mutated_field': field,
                         'expected_refusal': expected,
                         'passed': refusal['result'] == 'fail' and expected in {error['kind'] for error in refusal['errors']}})
    changed_prior_path = next(path for path, entry in policy['modified_paths'].items()
                              if entry.get('unchanged_since_prior_approved_adaptation') is False)
    candidate = copy.deepcopy(policy)
    candidate['modified_paths'][changed_prior_path]['unchanged_since_prior_approved_adaptation'] = True
    refusal = compare_prior_adaptations(candidate, actual, prior)
    controls.append({'name': 'mismatched-prior-changed-flag', 'mutated_path': changed_prior_path,
                     'mutated_field': 'unchanged_since_prior_approved_adaptation',
                     'expected_refusal': 'prior-adaptation-flag-mismatch',
                     'passed': refusal['result'] == 'fail' and 'prior-adaptation-flag-mismatch' in {error['kind'] for error in refusal['errors']}})
    result['negative_controls'] = controls
    if not all(c['passed'] for c in controls):
        result['errors'].append({'kind': 'negative-control-failure'})
        result['result'] = 'fail'
    for path in policy['added_since_original_checkpoint']:
        if path not in source or path not in actual:
            result['errors'].append({'kind': 'new-source-missing', 'path': path})
            result['result'] = 'fail'
    (HERE/'adaptation-equivalence.json').write_text(json.dumps(result, indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k not in {'protected_records','modified_paths'}},indent=2))
    if result['result'] != 'pass':
        raise SystemExit(1)


if __name__ == '__main__':
    main()
