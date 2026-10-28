"""Explicit supplemental custody and source-free verification; contains no cache cleanup operation."""
import argparse
import hashlib
import importlib.util
import json
import os
import shlex
import stat
import tarfile
import uuid
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
CAS = ROOT / '.recovery/fn25415-native-proof'
HELPERS = ROOT / '.flow/tmp/fn25415-validation/retention-tools'
TARGET = Path('/tmp/umpire-fn25415-final-9babff6c815b-47070/proof-bKkF3n/target')
PRIMARY_SHA = '68208a5aa0e5df79783d97b740a07da3f0808dce4935aa076450630a5183b610'


def digest(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def helpers():
    seal = json.loads((HERE / 'tooling-manifest.json').read_text())
    loaded = []
    for name in ['collect.py', 'verify.py']:
        path = HELPERS / name
        assert digest(path) == seal['retention_helpers'][name], 'retention helper changed'
        spec = importlib.util.spec_from_file_location('supplement_' + name[:-3], path)
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        loaded.append(module)
    return loaded


def validate_map(census, collector):
    assert census['schema'] == 'fn25415-private-cache-classification-v1'
    assert census['target'] == str(TARGET)
    assert census['primary_manifest_sha256'] == PRIMARY_SHA
    assert census['current_files'] == len(census['all_files']) == 20183
    assert census['original_unselected_files'] == 16640
    assert not census['catalog_added'] and not census['catalog_missing'] and not census['special_nodes']
    assert not census['ownership']['matches'], 'census has live private target handles'
    assert census['cleanup_authorized'] is census['cleanup_performed'] is False
    rows = census['all_files']
    assert len({row['path'] for row in rows}) == len(rows)
    members = census['supplemental_members']
    assert len({row['path'] for row in members}) == len(members)
    by_member = {row['path']: row for row in members}
    by_path = {row['path']: row for row in rows}
    primary_path = Path(census['primary_manifest'])
    assert primary_path == CAS / 'manifests' / (PRIMARY_SHA + '.json')
    assert digest(primary_path) == PRIMARY_SHA
    primary = json.loads(primary_path.read_text())
    primary_by_member = {row['path']: row for row in primary['archive_members']}
    for row in rows:
        collector.member_name(row['path'])
        assert row['source'] == str(TARGET / row['path'])
        identity = row['sha256'], row['size'], row['mode']
        disposition = row['disposition']
        if disposition.startswith('primary-retained-'):
            retained = primary_by_member[row['primary_member']]
            assert identity == (retained['sha256'], retained['size'], retained['mode'])
        elif disposition == 'supplemental-custody-required-preserve-until-verified':
            retained = by_member[row['supplemental_member']]
            assert identity == (retained['sha256'], retained['size'], retained['mode'])
        else:
            assert disposition == 'proposed-disposable-cargo-compiled-object'
            evidence = row['provenance']
            magic = bytes.fromhex(row['magic_prefix_hex'])
            assert magic.startswith((b'\x7fELF', b'!<arch>\n', b'rust\x00')), 'compiled disposition lacks binary format'
            assert evidence['dep_info'] and evidence['cargo_fingerprint']
            assert len(evidence['unit_hash']) == 16
            for path in evidence['dep_info'] + evidence['cargo_fingerprint']:
                assert by_path[path]['disposition'] != 'proposed-disposable-cargo-compiled-object', 'producer evidence not in custody'
    mapped_sources = {(value['source'], value.get('supplemental_member')) for value in rows}
    for row in members:
        collector.member_name(row['path'])
        assert (row['source'], row['path']) in mapped_sources
    return primary


def current_snapshot(census, collector):
    paths = []
    for parent, directories, files in os.walk(TARGET, followlinks=False):
        for name in directories:
            assert stat.S_ISDIR((Path(parent) / name).lstat().st_mode), 'special directory appeared'
        for name in files:
            path = Path(parent) / name
            assert stat.S_ISREG(path.lstat().st_mode), 'special file appeared'
            paths.append(str(path.relative_to(TARGET)))
    assert sorted(paths) == sorted(row['path'] for row in census['all_files']), 'private cache membership changed'
    for row in census['all_files']:
        info = Path(row['source']).stat()
        assert (info.st_dev, info.st_ino) == (row['device'], row['inode']), 'private cache inode changed'
        with collector.checked_stream(row)[0]:
            pass


def collect(classification, expected_sha, expected_tools):
    assert digest(HERE / 'tooling-manifest.json') == expected_tools, 'tooling seal changed'
    seal = json.loads((HERE / 'tooling-manifest.json').read_text())
    for name, checksum in seal['files'].items():
        assert digest(HERE / name) == checksum, 'prepared input changed'
    raw = classification.read_bytes()
    assert hashlib.sha256(raw).hexdigest() == expected_sha, 'classification changed'
    census = json.loads(raw)
    collector, verifier = helpers()
    primary = validate_map(census, collector)
    spec = importlib.util.spec_from_file_location('custody_classify', HERE / 'classify.py')
    inspector = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(inspector)
    observed = inspector.ownership(TARGET)
    assert not observed['matches'], 'private cache became active'
    current_snapshot(census, collector)
    for directory in ['staging', 'objects', 'manifests']:
        collector.directory(CAS / directory)
    nonce = 'supplemental-' + uuid.uuid4().hex
    staged_census = CAS / 'staging' / (nonce + '.classification.json')
    with staged_census.open('xb') as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())
    retained_census, census_sha = collector.publish(staged_census, CAS / 'objects')
    staged = CAS / 'staging' / (nonce + '.tar.gz')
    collector.write_archive(staged, census['supplemental_members'])
    verifier.verify_payload(staged, census['supplemental_members'])
    current_snapshot(census, collector)
    assert classification.read_bytes() == raw, 'classification changed during collection'
    retained, payload_sha = collector.publish(staged, CAS / 'objects')
    manifest = dict(schema='fn25415-supplemental-custody-v1', primary_manifest=census['primary_manifest'], primary_manifest_sha256=PRIMARY_SHA,
        classification_object=dict(path=str(retained_census), sha256=census_sha, size=len(raw)),
        payload_object=dict(path=str(retained), sha256=payload_sha, size=retained.stat().st_size),
        supplemental_members=census['supplemental_members'], ownership_before=observed,
        cleanup_performed=False, cleanup_authorized=False, failed_staging_preserved=True,
        protected_shared_target=census['protected_shared_target'])
    staged_manifest = CAS / 'staging' / (nonce + '.manifest.json')
    with staged_manifest.open('x') as stream:
        json.dump(manifest, stream, indent=2)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())
    retained_manifest, manifest_sha = collector.publish(staged_manifest, CAS / 'manifests', '.json')
    return dict(collected=True, manifest=str(retained_manifest), manifest_sha256=manifest_sha,
                independent_verification_pending=True, cleanup_performed=False)



def verify_disposable_evidence(census, primary_payload, supplemental_payload):
    by_path = {row['path']: row for row in census['all_files']}
    needed = set()
    for row in census['all_files']:
        if row['disposition'] == 'proposed-disposable-cargo-compiled-object':
            needed.update(row['provenance']['dep_info'])
            needed.update(row['provenance']['cargo_fingerprint'])
    wanted = {'primary': {}, 'supplemental': {}}
    for name in needed:
        row = by_path[name]
        if row['disposition'].startswith('primary-retained-'):
            wanted['primary'].setdefault(row['primary_member'], []).append(name)
        else:
            wanted['supplemental'].setdefault(row['supplemental_member'], []).append(name)
    evidence = {}
    for role, path in [('primary', primary_payload), ('supplemental', supplemental_payload)]:
        if not wanted[role]:
            continue
        with tarfile.open(path, 'r|gz') as archive:
            for member in archive:
                if member.name in wanted[role]:
                    assert member.size <= 16 * 1024 * 1024, 'producer metadata exceeds bounded parser input'
                    with archive.extractfile(member) as stream:
                        value = stream.read().decode(errors='replace')
                    for name in wanted[role][member.name]:
                        evidence[name] = value
    assert set(evidence) == needed, 'retained producer evidence missing'
    for row in census['all_files']:
        if row['disposition'] != 'proposed-disposable-cargo-compiled-object':
            continue
        proof = row['provenance']
        for name in proof['dep_info']:
            explicit = False
            for line in evidence[name].splitlines():
                if ': ' not in line or line.startswith('#'):
                    continue
                left, right = line.split(': ', 1)
                try:
                    explicit = explicit or (bool(right.strip()) and row['source'] in shlex.split(left))
                except ValueError:
                    pass
            assert explicit, 'disposed artifact absent from retained dep-info output rules'
        for name in proof['cargo_fingerprint']:
            path = Path(name)
            record = json.loads(evidence[name])
            assert path.parent.parent.name == '.fingerprint' and path.parent.name.endswith('-' + proof['unit_hash'])
            assert all(type(record.get(key)) is int for key in ['rustc', 'target', 'profile'])
            assert isinstance(record.get('deps'), list) and isinstance(record.get('local'), list)
    return dict(disposable_artifacts=sum(row['disposition'] == 'proposed-disposable-cargo-compiled-object' for row in census['all_files']),
                producer_files_verified=len(evidence), source_reads=False)


def verify(manifest_path, expected_sha):
    collector, verifier = helpers()
    assert manifest_path.parent == CAS / 'manifests' and manifest_path.name == expected_sha + '.json'
    verifier.regular(manifest_path)
    assert digest(manifest_path) == expected_sha, 'supplemental manifest mismatch'
    manifest = json.loads(manifest_path.read_text())
    assert manifest['schema'] == 'fn25415-supplemental-custody-v1'
    assert manifest['cleanup_performed'] is manifest['cleanup_authorized'] is False
    for key in ['classification_object', 'payload_object']:
        assert Path(manifest[key]['path']).parent == CAS / 'objects'
    census_path = verifier.check_object(manifest['classification_object'])
    payload_path = verifier.check_object(manifest['payload_object'])
    census = json.loads(census_path.read_text())
    assert manifest['primary_manifest'] == census['primary_manifest'] and manifest['primary_manifest_sha256'] == PRIMARY_SHA
    primary = validate_map(census, collector)
    assert manifest['supplemental_members'] == census['supplemental_members']
    members = verifier.verify_payload(payload_path, manifest['supplemental_members'])
    primary_verification = verifier.verify(Path(manifest['primary_manifest']), manifest['primary_manifest_sha256'])
    generated_provenance = verify_disposable_evidence(census, Path(primary['payload_object']['path']), payload_path)
    return dict(verified=True, supplemental_unique_members=members, classified_original_paths=len(census['all_files']),
                primary_verification=primary_verification, generated_provenance=generated_provenance, original_source_reads=False,
                archive_extracted=False, cleanup_performed=False, cleanup_authorized=False)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--classification', type=Path, default=HERE / 'classification.json')
    parser.add_argument('--expected-classification-sha256')
    parser.add_argument('--expected-tools-sha256')
    parser.add_argument('--execute', action='store_true')
    parser.add_argument('--verify-manifest', type=Path)
    parser.add_argument('--expected-manifest-sha256')
    parser.add_argument('--receipt', type=Path)
    options = parser.parse_args()
    if options.verify_manifest:
        result = verify(options.verify_manifest, options.expected_manifest_sha256)
    elif options.execute:
        result = collect(options.classification, options.expected_classification_sha256, options.expected_tools_sha256)
    else:
        result = dict(prepared_only=True, collection_executed=False, cleanup_performed=False)
    if options.receipt:
        with options.receipt.open('x') as stream:
            json.dump(result, stream, indent=2)
            stream.write('\n')
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()
