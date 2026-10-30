"""Verify immutable retention CAS bytes only; never read originals or extract an archive."""
import argparse
import hashlib
import json
import stat
import tarfile
from pathlib import Path, PurePosixPath

CAS = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal/.recovery/fn25415-native-proof')


def digest(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def safe_name(name):
    path = PurePosixPath(name)
    assert name and not path.is_absolute() and all(part not in ('', '.', '..') for part in name.split('/')), 'unsafe archive path'
    assert '\\' not in name and '\x00' not in name, 'unsafe archive path'
    return name


def regular(path):
    path = Path(path)
    assert path.is_absolute(), 'CAS path must be absolute'
    for part in [path, *path.parents]:
        assert not part.is_symlink(), 'symlinked CAS path'
    assert stat.S_ISREG(path.lstat().st_mode), 'CAS object is not regular'


def check_object(record):
    path = Path(record['path'])
    regular(path)
    assert path.name == record['sha256'], 'object is not content addressed'
    assert path.stat().st_size == record['size'], 'object size mismatch'
    assert digest(path) == record['sha256'], 'object hash mismatch'
    return path


def verify_payload(path, expected):
    names = [safe_name(row['path']) for row in expected]
    assert len(names) == len(set(names)), 'duplicate expected member'
    with tarfile.open(path, 'r:gz') as archive:
        actual = archive.getmembers()
        assert len(actual) == len(expected), 'missing or extra archive member'
        assert len({member.name for member in actual}) == len(actual), 'duplicate archive member'
        for member, row in zip(actual, expected):
            safe_name(member.name)
            assert member.isfile() and not member.issym() and not member.islnk(), 'non-regular archive member'
            assert member.name == row['path'], 'archive member name/order mismatch'
            assert member.size == row['size'], 'member size mismatch'
            assert member.mode == int(row['mode'], 8), 'member mode mismatch'
            value = hashlib.sha256()
            with archive.extractfile(member) as stream:
                for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                    value.update(chunk)
            assert value.hexdigest() == row['sha256'], 'member hash mismatch: ' + member.name
    return len(actual)


def verify(manifest_path, expected_manifest_sha256):
    manifest_path = Path(manifest_path)
    assert manifest_path.parent == CAS / 'manifests', 'unexpected manifest CAS'
    regular(manifest_path)
    assert digest(manifest_path) == expected_manifest_sha256, 'retention manifest hash mismatch'
    assert manifest_path.name == expected_manifest_sha256 + '.json', 'manifest is not content addressed'
    manifest = json.loads(manifest_path.read_text())
    assert manifest['schema'] == 'fn25415-retention-derived-v1'
    assert manifest['source_head'] == '9babff6c815beb7cff92d7001f647ddca19663bf'
    assert manifest['cleanup_performed'] is False
    assert Path(manifest['preparation_object']['path']).parent == CAS / 'objects', 'unexpected preparation CAS'
    assert Path(manifest['payload_object']['path']).parent == CAS / 'objects', 'unexpected payload CAS'
    assert Path(manifest['base_preparation_object']['path']).parent == CAS / 'objects', 'unexpected base preparation CAS'
    assert manifest['base_preparation_object']['sha256'] == 'cf8af5a758144fd6ece56e0849404d01387b38f5cbf9834b44693022b0ea3228', 'unexpected original preparation'
    base_path = check_object(manifest['base_preparation_object'])
    base = json.loads(base_path.read_text())
    preparation_path = check_object(manifest['preparation_object'])
    preparation = json.loads(preparation_path.read_text())
    assert preparation['proof_source_head'] == manifest['source_head']
    rows = preparation['required_retention_members']
    assert len(rows) == preparation['counts']['selected_retention_members'] == 4801
    assert preparation['schema'] == 'fn25415-proof-retention-derived-v1'
    assert base['schema'] == 'fn25415-proof-retention-preparation-v1'
    assert base['proof_source_head'] == manifest['source_head']
    for key in ['proof_root', 'private_target', 'protected_shared_target', 'mutable_result_alias_mismatches', 'exact_recorded_driver_recovery']:
        assert preparation[key] == base[key], 'original proof authority changed: ' + key
    assert len(base['required_retention_members']) == 4799
    assert preparation['base_preparation']['sha256'] == manifest['base_preparation_object']['sha256']
    assert preparation['base_preparation']['size'] == manifest['base_preparation_object']['size']
    assert rows[:4799] == base['required_retention_members'], 'original retention selection changed'
    additions = preparation['parent_bootstrap_additions']
    assert len(additions) == 2 and rows[4799:] == additions
    parent_root = Path(preparation['protected_shared_target']) / 'xtask-bootstrap/debug'
    assert {row['source'] for row in additions} == {str(parent_root / name) for name in ['xtask', 'umpire-guest-driver']}
    assert all(row['suggested_member'] == 'parent-bootstrap/' + Path(row['source']).name and
               row['roles'] == ['exact recorded parent bootstrap ELF component; original 9bab proof identity'] for row in additions)
    base_sources = {row['source']: row for row in base['required_retention_members']}
    support = preparation['parent_bootstrap_support']
    assert len(support) == 1 and base_sources[support[0]['source']]['sha256'] == support[0]['sha256']
    assert base_sources[support[0]['source']]['suggested_member'] == support[0]['member']
    expected = [dict(path=row['suggested_member'], source=row['source'], size=row['size'],
                     mode=row['mode'], sha256=row['sha256'], roles=row['roles'])
                for row in sorted(rows, key=lambda row: row['suggested_member'])]
    assert manifest['archive_members'] == expected, 'selection differs from sealed preparation'
    payload = check_object(manifest['payload_object'])
    assert payload.parent == preparation_path.parent, 'CAS objects have different stores'
    assert manifest_path.parent.parent == payload.parent.parent, 'manifest outside object CAS'
    count = verify_payload(payload, expected)
    parent_record = preparation['parent_bootstrap_record']
    assert base_sources[parent_record['path']]['sha256'] == parent_record['sha256']
    parent_member = base_sources[parent_record['path']]['suggested_member']
    with tarfile.open(payload, 'r:gz') as archive:
        with archive.extractfile(parent_member) as stream:
            recorded_parent_binaries = json.load(stream)
    assert {row['source']: row['sha256'] for row in additions} == recorded_parent_binaries, 'parent component differs from original proof record'
    recovery = preparation['exact_recorded_driver_recovery']
    aliases = preparation['mutable_result_alias_mismatches']
    assert len(aliases) == 1
    assert recovery['expected_sha256'] == 'cf6181b1f02565adfbad814323f9421c19884304978ead7e0f448e42f6fdd998'
    alias = aliases[0]
    assert alias['current_sha256'] == 'b2cae3f9f138bf009630ec366a44305c7ad955c41dc78c8c315ec0444de3900b'
    selected = {row['source']: row for row in expected}
    assert selected[alias['source']]['sha256'] == alias['current_sha256']
    copies = [selected[path] for path in recovery['verified_matching_sources']]
    assert len(copies) == 2 and all(row['sha256'] == recovery['expected_sha256'] == alias['expected_sha256'] for row in copies)
    wanted = {alias['source']: dict(recorded_sha256=alias['expected_sha256'],
        current_alias_sha256=alias['current_sha256'], current_alias_member=selected[alias['source']]['path'],
        retained_gate_finish_members=[row['path'] for row in copies],
        restoration='Use retained gate-finish bytes only on separately authorized recovery; never rewrite original receipts or this current alias during collection.')}
    assert manifest['gate_finish_restore_map'] == wanted, 'gate-finish restore mapping mismatch'
    assert any('pinned Rust loader/compiler support' in role for row in expected for role in row['roles']), 'pinned libstd role missing'
    return dict(verified=True, archive_members=count, original_source_reads=False,
                extracted_files=False, cleanup_performed=False, manifest_sha256=expected_manifest_sha256)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--manifest', required=True, type=Path)
    parser.add_argument('--expected-manifest-sha256', required=True)
    parser.add_argument('--receipt', required=True, type=Path)
    options = parser.parse_args()
    result = verify(options.manifest, options.expected_manifest_sha256)
    with options.receipt.open('x') as stream:
        json.dump(result, stream, indent=2)
        stream.write('\n')
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()
