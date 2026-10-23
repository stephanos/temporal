"""The overlay helper's one external dependency, authenticated against its real Go archive."""
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


def paths(root):
    directory = root / (MODULE + '@' + VERSION)
    metadata = root / 'cache/download' / MODULE / '@v'
    return directory, metadata


def verify(root, go_sum):
    directory, metadata = paths(root)
    require(root.is_absolute() and root.resolve() == root, 'module cache input must be canonical')
    expected_files = {metadata / (VERSION + suffix) for suffix in METADATA}
    require(all(path.is_file() and not path.is_symlink() for path in expected_files),
            'original Go dependency cache metadata is missing or aliased')
    prefix = MODULE + '@' + VERSION + '/'
    source = live.inventory(directory)
    archive_records = []
    present = set()
    with zipfile.ZipFile(metadata / (VERSION + '.zip')) as archive:
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
        require(archive.read(prefix + 'go.mod') == (metadata / (VERSION + '.mod')).read_bytes(),
                'Go dependency module metadata differs from genuine archive')
    digest = 'h1:' + base64.b64encode(hashlib.sha256(''.join(archive_records).encode()).digest()).decode()
    require((metadata / (VERSION + '.ziphash')).read_text().strip() == digest
            and MODULE + ' ' + VERSION + ' ' + digest in go_sum.read_text().splitlines(),
            'Go dependency archive differs from candidate go.sum')
    result = {str(directory.relative_to(root) / name): value for name, value in source.items()}
    result.update({str(path.relative_to(root)): dict(bytes=path.stat().st_size, sha256=sha(path)) for path in expected_files})
    return result


def snapshot(source, destination, go_sum):
    expected = verify(source, go_sum)
    require(not destination.exists(), 'private dependency cache already exists')
    source_directory, source_metadata = paths(source)
    directory, metadata = paths(destination)
    metadata.mkdir(mode=0o700, parents=True)
    shutil.copytree(source_directory, directory, symlinks=False)
    for suffix in METADATA:
        shutil.copyfile(source_metadata / (VERSION + suffix), metadata / (VERSION + suffix))
    require(live.inventory(destination) == expected and verify(destination, go_sum) == expected,
            'private dependency cache copy differs from genuine source')
    return expected
