"""Retire exact idle task-owned compiler trees after preserving their proof files."""
from pathlib import Path
import collections
import gzip
import hashlib
import json
import os
import shutil
import stat
import subprocess

root = Path(__file__).resolve().parents[6]
out = Path(__file__).resolve().parent
main = root / '.flow/tmp/fn2546-resumed/target'
selection = root / '.flow/tmp/fn2546-selection-review/target'
sources = {
    'main-debug': main / 'debug',
    'trybuild': main / 'tests/trybuild',
    'renamed': main / 'tests/renamed-realization',
    'selection': selection,
}
backup = root.parent / '.recovery/fn25417-proof-outputs-20261027'

def sha(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()

def live_references():
    found = []
    for process in Path('/proc').iterdir():
        if not process.name.isdigit() or int(process.name) == os.getpid():
            continue
        try:
            if process.stat().st_uid != os.getuid():
                continue
            cwd = Path(os.readlink(process / 'cwd'))
            values = [cwd, Path(os.readlink(process / 'exe'))]
            for item in (process / 'environ').read_bytes().split(b'\0'):
                key, sep, value = item.partition(b'=')
                if sep and key in (b'CARGO_TARGET_DIR', b'GOCACHE', b'TMPDIR'):
                    path = Path(os.fsdecode(value))
                    values.append(path if path.is_absolute() else cwd / path)
            if any(value == base or base in value.parents for value in values for base in (main, selection)):
                found.append(int(process.name))
        except (OSError, PermissionError):
            pass
    return found

def rebuildable(path, info):
    if info.st_mode & 0o111:
        return False
    with path.open('rb') as stream:
        magic = stream.read(8)
    if path.suffix in ('.rlib', '.a'):
        return magic in (b'!<arch>\n', b'!<thin>\n')
    if path.suffix == '.rmeta':
        return magic.startswith(b'rust')
    if path.suffix == '.o':
        return magic.startswith((b'\x7fELF', b'\0asm', b'BC\xc0\xde'))
    return False

assert not live_references(), 'a process still references an owned cache'
assert not backup.exists(), 'preserve an existing recovery directory'
for base in sources.values():
    assert base.is_dir() and not base.is_symlink()
    assert root in base.resolve().parents
    tracked = subprocess.check_output(['git', 'ls-files', '-z', '--', str(base)], cwd=root)
    assert not tracked, 'a retirement tree contains tracked files'

protected = list((main / 'xtask-bootstrap/debug').glob('xtask'))
protected += list((main / 'xtask-bootstrap/debug').glob('umpire-guest-driver'))
protected += [path for path in (main / 'quick-cache').rglob('*') if path.is_file()]
protected_hashes = {str(path): sha(path) for path in protected}
assert len(protected_hashes) >= 356, 'bootstrap tools or retained TLC cache missing'
restore_tools = [main / 'debug' / name for name in ('xtask', 'temporal-xtask', 'umpire-guest-driver')]
restore_tools = [path for path in restore_tools if path.is_file()]
tool_hashes = {str(path): sha(path) for path in restore_tools}
before_free = shutil.disk_usage(root).free
# This is recovery, not build admission: hardlinks need little additional space.
cleanup_free_floor = 1024**3
assert before_free >= cleanup_free_floor
records = []
aliases = collections.Counter()
for label, base in sources.items():
    for path in sorted(base.rglob('*')):
        info = path.lstat()
        if stat.S_ISREG(info.st_mode):
            aliases[(info.st_dev, info.st_ino)] += 1
            records.append(dict(label=label, relative=str(path.relative_to(base)), size=info.st_size,
                                mode=stat.S_IMODE(info.st_mode), device=info.st_dev, inode=info.st_ino,
                                links=info.st_nlink, mtime_ns=info.st_mtime_ns,
                                discard=rebuildable(path, info), sha256=sha(path)))
        elif stat.S_ISLNK(info.st_mode):
            records.append(dict(label=label, relative=str(path.relative_to(base)), link=os.readlink(path)))
        else:
            assert stat.S_ISDIR(info.st_mode), f'non-file cache entry: {path}'
print('Inventory complete:', len(records), 'entries', flush=True)
backup.mkdir(mode=0o700, parents=True)
copied_bytes = 0
for item in records:
    if item.get('discard'):
        continue
    source = sources[item['label']] / item['relative']
    destination = backup / item['label'] / item['relative']
    destination.parent.mkdir(parents=True, exist_ok=True)
    if 'link' in item:
        destination.symlink_to(item['link'])
    elif item['links'] == aliases[(item['device'], item['inode'])]:
        os.link(source, destination)
        item['preservation'] = 'hardlink; every original alias in retired trees will be unlinked'
    else:
        assert copied_bytes + item['size'] <= 512 * 1024**2, 'external-alias copy budget exceeded'
        shutil.copy2(source, destination)
        copied_bytes += item['size']
        item['preservation'] = 'independent copy; unknown external inode aliases exist'
    if 'sha256' in item:
        assert sha(destination) == item['sha256'], str(destination)
    assert shutil.disk_usage(root).free >= cleanup_free_floor, 'preservation crossed recovery guard'

manifest = dict(source_root=str(root), sources={key: str(value) for key, value in sources.items()},
                records=records, protected_hashes=protected_hashes, tool_hashes=tool_hashes,
                free_before=before_free, external_alias_copy_bytes=copied_bytes)
manifest_bytes = (json.dumps(manifest, indent=2) + '\n').encode()
with gzip.open(backup / 'manifest.json.gz', 'wb') as stream:
    stream.write(manifest_bytes)
assert gzip.open(backup / 'manifest.json.gz', 'rb').read() == manifest_bytes
assert not live_references(), 'cache became active during preservation'
for item in records:
    path = sources[item['label']] / item['relative']
    if 'link' in item:
        assert path.is_symlink() and os.readlink(path) == item['link']
    else:
        info = path.stat()
        assert (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns) == (
            item['device'], item['inode'], item['size'], item['mtime_ns']), str(path)
assert all(sha(Path(path)) == digest for path, digest in protected_hashes.items())
print('All preserved bytes verified; retiring only the four exact idle trees.', flush=True)
for base in sources.values():
    shutil.rmtree(base)
    assert not base.exists()
for path in restore_tools:
    path.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(backup / 'main-debug' / path.name, path)
    assert sha(path) == tool_hashes[str(path)]
assert all(sha(Path(path)) == digest for path, digest in protected_hashes.items())
assert all((backup / item['label'] / item['relative']).exists() or
           (backup / item['label'] / item['relative']).is_symlink()
           for item in records if not item.get('discard'))
summary = dict(
    scope='Four exact idle task-owned compiler trees; no source or shared/global cache removed',
    recovery_directory=str(backup), manifest_gzip_sha256=sha(backup / 'manifest.json.gz'),
    manifest_uncompressed_sha256=hashlib.sha256(manifest_bytes).hexdigest(),
    source_entries=len(records), preserved_entries=sum(not item.get('discard') for item in records),
    preserved_executables=sum(bool(item.get('mode', 0) & 0o111) for item in records),
    discarded_rebuildable_bytes=sum(item.get('size', 0) for item in records if item.get('discard')),
    external_alias_copy_bytes=copied_bytes, restored_native_tools=tool_hashes,
    protected_bootstrap_and_TLC_files=len(protected_hashes), free_before=before_free,
    free_after=shutil.disk_usage(root).free,
    cargo_clean_refusal='Missing CACHEDIR.TAG; original caches were not retagged to bypass the guard',
    recovery='Restore retained files using manifest source paths; regenerate omitted compiler objects with pinned Cargo.',
)
(out / 'summary.json').write_text(json.dumps(summary, indent=2) + '\n')
print(json.dumps(summary, indent=2), flush=True)
