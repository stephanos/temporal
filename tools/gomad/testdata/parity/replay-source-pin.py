#!/usr/bin/env python3
"""Authenticate an extracted Go reference source tree against its pinned git archive."""
import argparse
import hashlib
import io
import os
from pathlib import Path
import shutil
import sys
import tarfile
import tempfile

sys.dont_write_bytecode = True


def verify(archive, source, revision):
    """Refuse unless source holds exactly the archive's files, links and modes and the archive names revision."""
    source = Path(source)
    with tarfile.open(archive) as bundle:
        comment = bundle.pax_headers.get('comment')
        if comment != revision:
            raise ValueError(f'pinned source archive revision {comment!r}, expected {revision!r}')
        expected = set()
        for member in bundle.getmembers():
            name = member.name.rstrip('/')
            if not name or member.isdir():
                continue
            if Path(name).is_absolute() or '..' in Path(name).parts:
                raise ValueError('pinned source archive member escapes: ' + name)
            expected.add(name)
            path = source / name
            if member.issym():
                if not path.is_symlink() or os.readlink(path) != member.linkname:
                    raise ValueError('pinned source symlink differs: ' + name)
            elif member.isfile():
                if path.is_symlink() or not path.is_file():
                    raise ValueError('pinned source file missing: ' + name)
                if hashlib.sha256(bundle.extractfile(member).read()).digest() != hashlib.sha256(path.read_bytes()).digest():
                    raise ValueError('pinned source file differs: ' + name)
                if bool(member.mode & 0o111) != bool(path.stat().st_mode & 0o111):
                    raise ValueError('pinned source file mode differs: ' + name)
            else:
                raise ValueError('unsupported pinned source archive member: ' + name)
    for root, directories, files in os.walk(source):
        for name in files + [entry for entry in directories if (Path(root) / entry).is_symlink()]:
            relative = os.path.relpath(os.path.join(root, name), source)
            if relative not in expected:
                raise ValueError('pinned source has unarchived file: ' + relative)
    return len(expected)


def selftest():
    with tempfile.TemporaryDirectory(prefix='replay-source-pin-') as scratch:
        scratch = Path(scratch)
        archive = scratch / 'source.tar.gz'
        with tarfile.open(archive, 'w:gz', format=tarfile.PAX_FORMAT, pax_headers={'comment': 'r1'}) as bundle:
            for name, data, mode in (('tools/gomad/go.mod', b'module x\n', 0o644), ('tools/gomad/run.sh', b'#!/bin/sh\n', 0o755)):
                info = tarfile.TarInfo(name)
                info.size, info.mode = len(data), mode
                bundle.addfile(info, io.BytesIO(data))
            link = tarfile.TarInfo('tools/link')
            link.type, link.linkname = tarfile.SYMTYPE, 'gomad/go.mod'
            bundle.addfile(link)

        def fresh():
            tree = scratch / 'tree'
            shutil.rmtree(tree, ignore_errors=True)
            with tarfile.open(archive) as bundle:
                bundle.extractall(tree, filter='tar')
            return tree

        assert verify(archive, fresh(), 'r1') == 3
        cases = {
            'revision': lambda tree: 'r2',
            'modified': lambda tree: (tree / 'tools/gomad/go.mod').write_bytes(b'module y\n'),
            'missing': lambda tree: (tree / 'tools/gomad/go.mod').unlink(),
            'unarchived': lambda tree: (tree / 'tools/gomad/extra.go').write_bytes(b'package x\n'),
            'mode': lambda tree: (tree / 'tools/gomad/run.sh').chmod(0o644),
            'symlink': lambda tree: ((tree / 'tools/link').unlink(), (tree / 'tools/link').symlink_to('gomad/run.sh')),
        }
        for name, mutate in cases.items():
            tree = fresh()
            revision = mutate(tree) if name == 'revision' else (mutate(tree), 'r1')[1]
            try:
                verify(archive, tree, revision)
            except ValueError:
                continue
            raise AssertionError('modified pinned source accepted: ' + name)
    print('pinned source authentication: exact tree accepted; revision, modified, missing, unarchived, mode and symlink changes refused')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--selftest', action='store_true')
    parser.add_argument('--archive', type=Path)
    parser.add_argument('--source', type=Path)
    parser.add_argument('--revision')
    args = parser.parse_args()
    if args.selftest:
        selftest()
    else:
        print('pinned source authenticated:', verify(args.archive, args.source, args.revision), 'entries')
