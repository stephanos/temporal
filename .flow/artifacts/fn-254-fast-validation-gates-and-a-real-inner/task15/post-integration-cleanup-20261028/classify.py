"""Read-only private-cache census. Only the preparation directory receives new evidence."""
import collections
import datetime
import hashlib
import json
import os
import re
import shlex
import stat
from pathlib import Path

ROOT = Path('/Users/stephan/Workspace/skunkworks/umpire/temporal')
HERE = Path(__file__).resolve().parent
PREPARATION = ROOT / '.flow/tmp/fn25415-validation/proof-retention-preparation.json'
PRIMARY = ROOT / '.recovery/fn25415-native-proof/manifests/68208a5aa0e5df79783d97b740a07da3f0808dce4935aa076450630a5183b610.json'
TAG = b'Signature: 8a477f597d28d172789f06886806bc55'


def digest(path):
    value = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def write(path, value):
    Path(path).write_text(json.dumps(value, indent=2) + '\n')


def within(path, root):
    return path == root or root in path.parents


def ownership(target):
    prefix = os.fsencode(target)
    inspector = set()
    pid = os.getpid()
    while pid and pid not in inspector:
        inspector.add(pid)
        try:
            pid = int((Path('/proc') / str(pid) / 'stat').read_text().rsplit(')', 1)[1].split()[1])
        except (OSError, IndexError, ValueError):
            break
    matches, inaccessible = [], []
    for proc in Path('/proc').iterdir():
        if not proc.name.isdigit() or int(proc.name) in inspector:
            continue
        found, denied = [], []
        for key in ['cwd', 'exe']:
            try:
                value = os.fsencode(os.readlink(proc / key))
                if value == prefix or value.startswith(prefix + b'/'):
                    found.append(key)
            except PermissionError:
                denied.append(key)
            except (FileNotFoundError, ProcessLookupError):
                pass
        for key in ['cmdline', 'environ', 'maps']:
            try:
                if prefix in (proc / key).read_bytes():
                    found.append(key)
            except PermissionError:
                denied.append(key)
            except (FileNotFoundError, ProcessLookupError):
                pass
        try:
            for descriptor in (proc / 'fd').iterdir():
                try:
                    value = os.fsencode(os.readlink(descriptor))
                    if value == prefix or value.startswith(prefix + b'/'):
                        found.append('fd:' + descriptor.name)
                except (FileNotFoundError, ProcessLookupError):
                    pass
                except PermissionError:
                    denied.append('fd:' + descriptor.name)
        except PermissionError:
            denied.append('fd-list')
        except (FileNotFoundError, ProcessLookupError):
            pass
        if found:
            matches.append(dict(pid=int(proc.name), matches=found))
        if denied:
            try:
                identity = dict(pid=int(proc.name), uid=proc.stat().st_uid,
                                comm=(proc / 'comm').read_text().strip())
            except OSError:
                identity = dict(pid=int(proc.name))
            identity['unreadable'] = denied
            inaccessible.append(identity)
    return dict(matches=matches, unreadable=inaccessible, inspector_ancestry_excluded=sorted(inspector),
                scope='private target only; active shared parent is protected, not asserted idle')


def producer_indices(target, paths):
    dependencies, fingerprints = {}, collections.defaultdict(list)
    for path in paths:
        if path.suffix == '.d':
            for line in path.read_text(errors='replace').splitlines():
                if ': ' not in line or line.startswith('#'):
                    continue
                left, right = line.split(': ', 1)
                if not right.strip():
                    continue
                try:
                    outputs = shlex.split(left)
                except ValueError:
                    continue
                for name in outputs:
                    output = Path(name)
                    if output.is_absolute() and within(output, target):
                        dependencies.setdefault(str(output), []).append(str(path.relative_to(target)))
        if path.suffix == '.json' and path.parent.parent.name == '.fingerprint':
            unit = re.search(r'-([0-9a-f]{16})$', path.parent.name)
            if not unit:
                continue
            try:
                record = json.loads(path.read_text())
            except (ValueError, OSError):
                continue
            if all(type(record.get(key)) is int for key in ['rustc', 'target', 'profile']) and isinstance(record.get('deps'), list) and isinstance(record.get('local'), list):
                profile = path.parent.parent.parent
                fingerprints[(str(profile), unit[1])].append(str(path.relative_to(target)))
    return dependencies, fingerprints


def compiled_provenance(path, magic, target, dependencies, fingerprints):
    # File suffixes alone never confer a disposable disposition.
    kind = 'ELF' if magic.startswith(b'\x7fELF') else 'Rust ar archive' if magic.startswith(b'!<arch>\n') else 'Rust metadata' if magic.startswith(b'rust\x00') else None
    if kind is None or str(path) not in dependencies:
        return None
    unit = re.search(r'-([0-9a-f]{16})(?:\.[A-Za-z0-9_.-]+)?$', path.name)
    if not unit:
        return None
    profile = next((parent for parent in path.parents if (str(parent), unit[1]) in fingerprints and within(parent, target)), None)
    if profile is None:
        return None
    return dict(binary_format=kind, dep_info=dependencies[str(path)],
                cargo_fingerprint=fingerprints[(str(profile), unit[1])], unit_hash=unit[1],
                reason='Explicit Cargo dep-info output rule + matching rustc/profile/target fingerprint + actual binary format')


def main():
    preparation = json.loads(PREPARATION.read_text())
    primary_bytes = PRIMARY.read_bytes()
    assert hashlib.sha256(primary_bytes).hexdigest() == PRIMARY.stem
    primary = json.loads(primary_bytes)
    assert primary['selected_member_count'] == len(primary['archive_members']) == 4801
    target = Path(preparation['private_target'])
    assert target == Path('/tmp/umpire-fn25415-final-9babff6c815b-47070/proof-bKkF3n/target') and not target.is_symlink()
    original = {row['path']: row for row in preparation['target_catalog']}
    paths, directories, specials = [], [], []
    for parent, names, files in os.walk(target, followlinks=False):
        for name in names + files:
            path = Path(parent) / name
            info = path.lstat()
            if stat.S_ISREG(info.st_mode):
                paths.append(path)
            elif stat.S_ISDIR(info.st_mode):
                directories.append(path)
            else:
                specials.append(dict(path=str(path), mode=oct(info.st_mode), preserve_in_place=True))
    actual = {str(path.relative_to(target)) for path in paths}
    added, missing = sorted(actual - original.keys()), sorted(original.keys() - actual)
    dependencies, fingerprints = producer_indices(target, paths)
    selected = {row['source']: row for row in primary['archive_members']}
    content = {(row['sha256'], row['size'], row['mode']): row['path'] for row in primary['archive_members']}
    hashes, rows, supplemental = {}, [], {}
    allocated_inodes = {}
    for directory in [target, *directories]:
        info = directory.lstat()
        allocated_inodes[(info.st_dev, info.st_ino)] = info.st_blocks * 512
    for path in sorted(paths):
        before = path.stat()
        inode = before.st_dev, before.st_ino
        allocated_inodes[inode] = before.st_blocks * 512
        if inode not in hashes:
            hashes[inode] = digest(path)
        checksum = hashes[inode]
        after = path.stat()
        assert (before.st_size, before.st_mtime_ns, before.st_ctime_ns, before.st_mode) == (after.st_size, after.st_mtime_ns, after.st_ctime_ns, after.st_mode), 'file changed during census'
        with path.open('rb') as stream:
            magic = stream.read(12)
        relative = str(path.relative_to(target))
        row = dict(path=relative, source=str(path), size=before.st_size, mode=oct(stat.S_IMODE(before.st_mode)),
                   sha256=checksum, magic_prefix_hex=magic.hex(), allocated_bytes=before.st_blocks * 512, device=before.st_dev, inode=before.st_ino)
        key = checksum, row['size'], row['mode']
        if str(path) in selected:
            old = selected[str(path)]
            assert key == (old['sha256'], old['size'], old['mode']), 'primary-retained source changed'
            row.update(disposition='primary-retained-exact-path', primary_member=old['path'])
        elif key in content:
            row.update(disposition='primary-retained-identical-content', primary_member=content[key],
                       reason='Same SHA256, size and mode already independently retained; this additional original path is mapped explicitly')
        else:
            evidence = compiled_provenance(path, magic, target, dependencies, fingerprints)
            if evidence:
                row.update(disposition='proposed-disposable-cargo-compiled-object', provenance=evidence,
                           cleanup_authorized=False)
            else:
                name = 'supplemental/' + checksum + '-' + row['mode'][2:]
                row.update(disposition='supplemental-custody-required-preserve-until-verified', supplemental_member=name,
                           reason='Generated source/data/docs/records/cache proof or unknown file; no suffix-only disposal')
                supplemental.setdefault(key, dict(path=name, source=str(path), size=row['size'], mode=row['mode'],
                                                 sha256=checksum, roles=['supplemental generated source/data/proof or unknown custody']))
        if relative in original:
            prior = original[relative]
            row['original_catalog_metadata_matches'] = all(row[k] == prior[k] for k in ['size', 'mode', 'device', 'inode'])
        rows.append(row)
    retained_by_path = {row['path']: row for row in rows}
    for row in rows:
        if row['disposition'] == 'proposed-disposable-cargo-compiled-object':
            evidence = row['provenance']
            for name in evidence['dep_info'] + evidence['cargo_fingerprint']:
                assert retained_by_path[name]['disposition'] != 'proposed-disposable-cargo-compiled-object', 'producer proof would not be retained'
    tags = []
    for path in paths:
        if path.name == 'CACHEDIR.TAG' and path.read_bytes().startswith(TAG):
            tags.append(dict(target=str(path.parent), tag_sha256=digest(path),
                             command=['/home/agent/.rustup/toolchains/1.99.1-aarch64-unknown-linux-gnu/bin/cargo', 'clean', '--target-dir', str(path.parent)],
                             execute=False))
    count = collections.Counter(row['disposition'] for row in rows)
    unique_payloads = sorted(supplemental.values(), key=lambda row: row['path'])
    protected = [ROOT, ROOT/'model', Path(preparation['proof_root'])/'framework', Path(preparation['proof_root'])/'result.json',
                 ROOT/'.recovery', ROOT.parent/'.recovery', Path(preparation['protected_shared_target']),
                 ROOT/'.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task15']
    result = dict(schema='fn25415-private-cache-classification-v1', prepared_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
        target=str(target), proof_source_head=preparation['proof_source_head'], primary_manifest=str(PRIMARY), primary_manifest_sha256=PRIMARY.stem,
        preparation_sha256=digest(PREPARATION), original_catalog_files=len(original), original_unselected_files=16640,
        current_files=len(paths), current_directory_count=len(directories), catalog_added=added, catalog_missing=missing,
        special_nodes=specials, dispositions=dict(count), all_files=rows, supplemental_members=unique_payloads,
        supplemental_unique_members=len(unique_payloads), supplemental_logical_bytes=sum(row['size'] for row in unique_payloads),
        allocated_private_target_bytes=sum(allocated_inodes.values()), ownership=ownership(target),
        filesystems=[dict(path=str(p), realpath=str(p.resolve()), device=p.stat().st_dev,
                          free_bytes=os.statvfs(p).f_bavail * os.statvfs(p).f_frsize) for p in [ROOT, target, Path('/tmp')]],
        protected_paths={str(p):p.exists() for p in protected}, protected_shared_target=preparation['protected_shared_target'],
        proposed_tagged_cargo_clean=sorted(tags, key=lambda row: row['target']),
        guarded_root_remainder=dict(target=str(target), execute=False, method='No cleanup function prepared. Root may consider guarded exact-root remainder only after approved custody and fresh complete classification/ownership checks following tagged Cargo clean.'),
        cleanup_authorized=False, cleanup_performed=False, supplemental_collection_executed=False,
        blockers=['Supplemental custody and independent verification pending', 'Root must recheck ownership and exact current classification before any clean',
                  'Unknown/unclassified content preserved until supplemental custody; special nodes or unexplained catalog changes block cleanup',
                  'Unreadable foreign process metadata remains an explicit observation limitation; active shared parent is protected'])
    write(HERE/'classification.json', result)
    summary = {k:result[k] for k in ['current_files','original_unselected_files','dispositions','supplemental_unique_members','supplemental_logical_bytes','allocated_private_target_bytes','catalog_added','catalog_missing','special_nodes','ownership','filesystems']}
    write(HERE/'summary.json', summary)
    print(json.dumps(summary, indent=2), flush=True)


if __name__ == '__main__':
    main()
