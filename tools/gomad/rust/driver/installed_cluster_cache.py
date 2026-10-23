"""Authenticate genuine Go cache modules against their archives and candidate go.sum."""
import base64
import hashlib
from pathlib import Path
import shutil
import zipfile

from driver import require, sha
import live

MODULE = 'golang.org/x/sys'
VERSION = 'v0.48.0'
METADATA = ('.info', '.mod', '.zip', '.ziphash', '.lock')


def paths(root, module=MODULE, version=VERSION):
    directory = root / (module + '@' + version)
    metadata = root / 'cache/download' / module / '@v'
    return directory, metadata


def verify(root, go_sum, module=MODULE, version=VERSION):
    directory, metadata = paths(root, module, version)
    require(root.is_absolute() and root.resolve() == root, 'module cache input must be canonical')
    expected_files = {metadata / (version + suffix) for suffix in METADATA}
    require(all(path.is_file() and not path.is_symlink() for path in expected_files),
            'original Go dependency cache metadata is missing or aliased')
    prefix = module + '@' + version + '/'
    source = live.inventory(directory)
    archive_records = []
    present = set()
    with zipfile.ZipFile(metadata / (version + '.zip')) as archive:
        names = archive.namelist()
        require(len(names) == len(set(names)) and len(names) <= 10_000
                and sum(info.file_size for info in archive.infolist()) <= 64 << 20
                and all(info.file_size <= 16 << 20 for info in archive.infolist()),
                'duplicate or oversized Go module archive member')
        for name in sorted(names):
            require(name.startswith(prefix), 'Go module archive has foreign prefix')
            relative = name[len(prefix):]
            require(relative and all(part not in ('', '.', '..') for part in relative.split('/'))
                    and '\\' not in relative and '\0' not in relative and '\n' not in relative, 'unsafe Go module archive member')
            raw = archive.read(name)
            require(relative in source and source[relative] == dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest()),
                    'extracted Go dependency differs from genuine module archive')
            archive_records.append(hashlib.sha256(raw).hexdigest() + '  ' + name + '\n')
            present.add(relative)
        require(present == set(source), 'extracted Go dependency has extra or missing files')
        if prefix + 'go.mod' in names:
            require(archive.read(prefix + 'go.mod') == (metadata / (version + '.mod')).read_bytes(),
                    'Go dependency module metadata differs from genuine archive')
    digest = 'h1:' + base64.b64encode(hashlib.sha256(''.join(archive_records).encode()).digest()).decode()
    sums = go_sum.read_text().splitlines()
    require((metadata / (version + '.ziphash')).read_text().strip() == digest
            and module + ' ' + version + ' ' + digest in sums,
            'Go dependency archive differs from candidate go.sum')
    mod = (metadata / (version + '.mod')).read_bytes()
    mod_record = hashlib.sha256(mod).hexdigest() + '  go.mod\n'
    mod_digest = 'h1:' + base64.b64encode(hashlib.sha256(mod_record.encode()).digest()).decode()
    require(module + ' ' + version + '/go.mod ' + mod_digest in sums,
            'Go dependency module metadata differs from candidate go.sum')
    result = {str(directory.relative_to(root) / name): value for name, value in source.items()}
    result.update({str(path.relative_to(root)): dict(bytes=path.stat().st_size, sha256=sha(path)) for path in expected_files})
    return result


def snapshot(source, destination, go_sum):
    return snapshot_modules(source, destination, go_sum, ((MODULE, VERSION),))


def snapshot_modules(source, destination, go_sum, modules):
    modules = tuple(modules)
    require(modules and len(modules) == len(set(modules)), 'missing/duplicate dependency cache module')
    expected = {}
    for module, version in modules:
        expected.update(verify(source, go_sum, module, version))
    require(not destination.exists(), 'private dependency cache already exists')
    for module, version in modules:
        source_directory, source_metadata = paths(source, module, version)
        directory, metadata = paths(destination, module, version)
        metadata.mkdir(mode=0o700, parents=True)
        shutil.copytree(source_directory, directory, symlinks=False)
        for suffix in METADATA:
            shutil.copyfile(source_metadata / (version + suffix), metadata / (version + suffix))
        require(verify(destination, go_sum, module, version).items() <= expected.items(),
                'private dependency module copy differs from genuine source')
    require(live.inventory(destination) == expected, 'private dependency cache copy differs from genuine source')
    return expected
