"""Prepare or explicitly collect task15's exact selected proof bytes; never delete or restore originals."""
import argparse
import gzip
import hashlib
import json
import os
import re
import stat
import tarfile
import uuid
from pathlib import Path, PurePosixPath

ROOT = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
DEFAULT_INPUT = Path(__file__).resolve().parent / 'derived-preparation.json'
CAS = ROOT / '.recovery/fn25415-native-proof'
SOURCE_HEAD = '9babff6c815beb7cff92d7001f647ddca19663bf'
RECORDED_DRIVER = 'cf6181b1f02565adfbad814323f9421c19884304978ead7e0f448e42f6fdd998'
CURRENT_ALIAS = 'b2cae3f9f138bf009630ec366a44305c7ad955c41dc78c8c315ec0444de3900b'


def checksum(data):
    return hashlib.sha256(data).hexdigest()


def digest(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def member_name(name):
    assert isinstance(name, str) and name and not PurePosixPath(name).is_absolute(), 'unsafe member name'
    assert all(part not in ('', '.', '..') for part in name.split('/')), 'unsafe member name'
    assert '\\' not in name and '\x00' not in name, 'unsafe member name'
    return name


def safe_source(path):
    path = Path(path)
    assert path.is_absolute() and '..' not in path.parts, 'source must be absolute without parent traversal'
    for ancestor in [path, *path.parents]:
        assert not ancestor.is_symlink(), 'source symlink refused'
    assert stat.S_ISREG(path.lstat().st_mode), 'source is not a regular file'
    return path


def rows_from(preparation):
    selected = preparation['required_retention_members']
    rows = []
    for row in selected:
        assert re.fullmatch('[0-9a-f]{64}', row['sha256']), 'invalid input hash'
        assert isinstance(row['size'], int) and row['size'] >= 0
        mode = int(row['mode'], 8)
        assert 0 <= mode <= 0o7777
        rows.append(dict(path=member_name(row['suggested_member']), source=row['source'],
                         size=row['size'], mode=oct(mode), sha256=row['sha256'], roles=row['roles']))
    assert len({row['path'] for row in rows}) == len(rows), 'duplicate member'
    assert len({row['source'] for row in rows}) == len(rows), 'duplicate source'
    return sorted(rows, key=lambda row: row['path'])


def validate_production(preparation, rows):
    assert preparation['proof_source_head'] == SOURCE_HEAD
    assert preparation['schema'] == 'fn25415-proof-retention-derived-v1'
    assert len(rows) == preparation['counts']['selected_retention_members'] == 4801
    base_record = preparation['base_preparation']
    assert base_record['sha256'] == 'cf8af5a758144fd6ece56e0849404d01387b38f5cbf9834b44693022b0ea3228', 'unexpected original preparation'
    base_payload = Path(base_record['path']).read_bytes()
    assert checksum(base_payload) == base_record['sha256'] and len(base_payload) == base_record['size'], 'base preparation changed'
    base = json.loads(base_payload)
    assert base['schema'] == 'fn25415-proof-retention-preparation-v1'
    assert base['proof_source_head'] == SOURCE_HEAD
    for key in ['proof_root', 'private_target', 'protected_shared_target', 'mutable_result_alias_mismatches', 'exact_recorded_driver_recovery']:
        assert preparation[key] == base[key], 'original proof authority changed: ' + key
    assert len(base['required_retention_members']) == base_record['member_count'] == 4799
    assert preparation['required_retention_members'][:4799] == base['required_retention_members'], 'original selected members changed'
    additions = preparation['parent_bootstrap_additions']
    assert len(additions) == 2 and preparation['required_retention_members'][4799:] == additions
    parent_root = Path(preparation['protected_shared_target']) / 'xtask-bootstrap/debug'
    assert {row['source'] for row in additions} == {str(parent_root / name) for name in ['xtask', 'umpire-guest-driver']}
    record = preparation['parent_bootstrap_record']
    record_bytes = Path(record['path']).read_bytes()
    assert checksum(record_bytes) == record['sha256'], 'parent binary record changed'
    assert {row['source']: row['sha256'] for row in additions} == json.loads(record_bytes)
    base_sources = {row['source']: row for row in base['required_retention_members']}
    assert base_sources[record['path']]['sha256'] == record['sha256']
    assert all(row['suggested_member'] == 'parent-bootstrap/' + Path(row['source']).name and
               row['roles'] == ['exact recorded parent bootstrap ELF component; original 9bab proof identity'] for row in additions)
    support = preparation['parent_bootstrap_support']
    assert len(support) == 1
    assert base_sources[support[0]['source']]['suggested_member'] == support[0]['member']
    assert base_sources[support[0]['source']]['sha256'] == support[0]['sha256']
    assert preparation['exact_recorded_driver_recovery']['expected_sha256'] == RECORDED_DRIVER
    alias = preparation['mutable_result_alias_mismatches']
    assert len(alias) == 1 and alias[0]['expected_sha256'] == RECORDED_DRIVER and alias[0]['current_sha256'] == CURRENT_ALIAS
    selected = {row['source']: row for row in rows}
    assert selected[alias[0]['source']]['sha256'] == CURRENT_ALIAS
    recovery = preparation['exact_recorded_driver_recovery']['verified_matching_sources']
    assert len(recovery) == 2 and all(selected[path]['sha256'] == RECORDED_DRIVER for path in recovery)
    assert any('pinned Rust loader/compiler support' in role for row in rows for role in row['roles'])
    assert str(CAS).startswith(str(ROOT) + '/.recovery/')
    assert not CAS.is_relative_to(Path(preparation['proof_root']))
    assert not CAS.is_relative_to(Path(preparation['protected_shared_target']))
    return base_payload


def fingerprint(info):
    return (info.st_dev, info.st_ino, info.st_mode, info.st_size, info.st_mtime_ns, info.st_ctime_ns, info.st_nlink)


def checked_stream(row):
    path = safe_source(row['source'])
    descriptor = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    stream = os.fdopen(descriptor, 'rb')
    info = os.fstat(descriptor)
    try:
        assert stat.S_ISREG(info.st_mode) and info.st_size == row['size'], 'modified input size'
        assert stat.S_IMODE(info.st_mode) == int(row['mode'], 8), 'modified input mode'
        value = hashlib.sha256()
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(chunk)
        assert value.hexdigest() == row['sha256'], 'modified input hash'
        assert fingerprint(os.fstat(descriptor)) == fingerprint(info), 'input changed during admission'
        stream.seek(0)
        return stream, info
    except BaseException:
        stream.close()
        raise


class CheckedReader:
    def __init__(self, stream):
        self.stream = stream
        self.hash = hashlib.sha256()
        self.size = 0

    def read(self, size=-1):
        data = self.stream.read(size)
        self.hash.update(data)
        self.size += len(data)
        return data


def write_archive(destination, rows):
    with Path(destination).open('xb') as raw:
        with gzip.GzipFile(filename='', fileobj=raw, mode='wb', compresslevel=3, mtime=0) as compressed:
            with tarfile.open(fileobj=compressed, mode='w|', format=tarfile.PAX_FORMAT) as archive:
                for row in rows:
                    with checked_stream(row)[0] as stream:
                        initial = os.fstat(stream.fileno())
                        reader = CheckedReader(stream)
                        info = tarfile.TarInfo(row['path'])
                        info.size, info.mode = row['size'], int(row['mode'], 8)
                        info.uid = info.gid = info.mtime = 0
                        archive.addfile(info, reader)
                        assert reader.size == row['size'] and reader.hash.hexdigest() == row['sha256'], 'input changed during archive'
                        assert fingerprint(os.fstat(stream.fileno())) == fingerprint(initial), 'input changed during archive'
                raw.flush()
        raw.flush()
        os.fsync(raw.fileno())


def directory(path):
    path = Path(path)
    for ancestor in reversed([path, *path.parents]):
        assert not ancestor.is_symlink(), 'symlinked CAS directory'
        if not ancestor.exists():
            ancestor.mkdir(mode=0o700)
        assert ancestor.is_dir(), 'CAS parent is not a directory'


def publish(staged, folder, suffix=''):
    checksum = digest(staged)
    destination = folder / (checksum + suffix)
    try:
        os.link(staged, destination)
    except FileExistsError:
        safe_source(destination)
        assert digest(destination) == checksum and destination.stat().st_size == staged.stat().st_size, 'existing CAS differs'
    return destination, checksum


def collect(input_path, expected_sha256):
    payload = Path(input_path).read_bytes()
    assert checksum(payload) == expected_sha256, 'preparation input hash mismatch'
    preparation = json.loads(payload)
    rows = rows_from(preparation)
    base_payload = validate_production(preparation, rows)
    # Admit every original before making any retention archive. Recheck while streaming.
    for row in rows:
        with checked_stream(row)[0]:
            pass
    assert Path(input_path).read_bytes() == payload, 'preparation changed during admission'
    directory(CAS / 'staging')
    directory(CAS / 'objects')
    directory(CAS / 'manifests')
    nonce = uuid.uuid4().hex
    staged_base = CAS / 'staging' / (nonce + '.base-preparation.json')
    with staged_base.open('xb') as stream:
        stream.write(base_payload)
        stream.flush()
        os.fsync(stream.fileno())
    retained_base, retained_base_sha = publish(staged_base, CAS / 'objects')
    staged_input = CAS / 'staging' / (nonce + '.input.json')
    with staged_input.open('xb') as stream:
        stream.write(payload)
        stream.flush()
        os.fsync(stream.fileno())
    retained_input, retained_input_sha = publish(staged_input, CAS / 'objects')
    staged_archive = CAS / 'staging' / (nonce + '.payload.tar.gz')
    write_archive(staged_archive, rows)
    assert Path(input_path).read_bytes() == payload, 'preparation changed during collection'
    from verify import verify_payload
    verify_payload(staged_archive, rows)
    for row in rows:
        with checked_stream(row)[0]:
            pass
    retained_archive, retained_archive_sha = publish(staged_archive, CAS / 'objects')
    selected = {row['source']: row for row in rows}
    alias = preparation['mutable_result_alias_mismatches'][0]
    copies = [selected[path] for path in preparation['exact_recorded_driver_recovery']['verified_matching_sources']]
    restore = {alias['source']: dict(recorded_sha256=alias['expected_sha256'], current_alias_sha256=alias['current_sha256'],
        current_alias_member=selected[alias['source']]['path'], retained_gate_finish_members=[row['path'] for row in copies],
        restoration='Use retained gate-finish bytes only on separately authorized recovery; never rewrite original receipts or this current alias during collection.')}
    manifest = dict(schema='fn25415-retention-derived-v1', source_head=SOURCE_HEAD, selected_member_count=len(rows),
        base_preparation_object=dict(path=str(retained_base), sha256=retained_base_sha, size=len(base_payload)),
        preparation_object=dict(path=str(retained_input), sha256=retained_input_sha, size=len(payload)),
        payload_object=dict(path=str(retained_archive), sha256=retained_archive_sha, size=retained_archive.stat().st_size),
        archive_members=rows, gate_finish_restore_map=restore, cleanup_performed=False,
        staging_preserved=True, protected_shared_target=preparation['protected_shared_target'],
        scope='Selected proof evidence only; unknown entries preserved in place; no cleanup authority.')
    staged_manifest = CAS / 'staging' / (nonce + '.manifest.json')
    with staged_manifest.open('x') as stream:
        json.dump(manifest, stream, indent=2)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())
    manifest_path, manifest_sha = publish(staged_manifest, CAS / 'manifests', '.json')
    return dict(collected=True, independent_verification_pending=True, manifest=str(manifest_path),
                manifest_sha256=manifest_sha, selected_members=len(rows), cleanup_performed=False,
                all_staging_retained=True)



def check_tools(expected_sha256):
    here = Path(__file__).resolve().parent
    path = here / 'tools-manifest.json'
    assert checksum(path.read_bytes()) == expected_sha256, 'retention tooling manifest changed'
    manifest = json.loads(path.read_text())
    for name, expected in manifest['files'].items():
        assert member_name(name) == name and '/' not in name
        assert digest(here / name) == expected, 'retention tooling input changed: ' + name
    controls = json.loads((here / 'controls.json').read_text())
    assert controls['success'] and controls['tests_run'] == 4 and controls['errors'] == controls['failures'] == 0
    for name, expected in controls['inputs_sha256'].items():
        assert digest(here / name) == expected, 'safe controls are stale: ' + name


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--input', type=Path, default=DEFAULT_INPUT)
    parser.add_argument('--expected-input-sha256', required=True)
    parser.add_argument('--execute', action='store_true')
    parser.add_argument('--expected-tools-sha256')
    options = parser.parse_args()
    assert re.fullmatch('[0-9a-f]{64}', options.expected_input_sha256)
    if not options.execute:
        print(json.dumps(dict(prepared_only=True, collected=False, execute_required=True,
                              input=str(options.input), cas=str(CAS), cleanup_performed=False)))
        return
    assert options.expected_tools_sha256 and re.fullmatch('[0-9a-f]{64}', options.expected_tools_sha256), '--expected-tools-sha256 required for execution'
    check_tools(options.expected_tools_sha256)
    print(json.dumps(collect(options.input, options.expected_input_sha256), indent=2))


if __name__ == '__main__':
    main()
