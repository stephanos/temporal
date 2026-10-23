"""Original native-scratch and 12 GiB admission, before candidate dispatch."""
import os
from pathlib import Path
import re
import time

from driver import REPO, Refusal, read, require, write

MIN_FREE_BYTES = 12 * 1024 ** 3


def bindings(config):
    scratch = Path(config['scratch'])
    evidence = Path(config['root'])
    require(evidence.is_absolute() and evidence.resolve() == evidence,
            'evidence root must be an absolute canonical path')
    require(scratch.is_absolute() and scratch.resolve() == scratch,
            'scratch must be an absolute canonical path')
    require(scratch.parent.is_dir(), 'scratch parent must already exist')
    require(not scratch.is_relative_to(REPO) and not REPO.is_relative_to(scratch),
            'scratch overlaps candidate source')
    require(not scratch.is_relative_to(evidence) and not evidence.is_relative_to(scratch),
            'scratch overlaps evidence root')
    paths = config['temporary_paths']
    require(isinstance(paths, dict) and set(paths) == {'TMPDIR', 'GOTMPDIR'},
            'explicit TMPDIR and GOTMPDIR scratch bindings required')
    for key, name in paths.items():
        path = Path(name)
        require(path.is_absolute() and path.resolve() == path and path.parent == scratch,
                key + ' must be a direct canonical scratch child')
        require(key not in config.get('environment', {}) or config['environment'][key] == name,
                'conflicting temporary environment binding: ' + key)
    require(len(set(paths.values())) == 2, 'temporary paths must be distinct')
    return scratch, paths


def unescape_mount(value):
    return re.sub(r'\\([0-7]{3})', lambda match: chr(int(match[1], 8)), value)


def filesystem(path, mounts):
    resolved = Path(path).resolve(strict=True)
    candidates = []
    for line in mounts.splitlines():
        fields = line.split()
        separator = fields.index('-')
        mount = Path(unescape_mount(fields[4]))
        if resolved.is_relative_to(mount):
            candidates.append((len(mount.parts), fields[separator + 1], str(mount)))
    require(candidates, 'cannot identify native filesystem: ' + str(path))
    _, kind, mount = max(candidates)
    return kind, mount


def host_fact(path):
    kind, mount = filesystem(path, Path('/proc/self/mountinfo').read_text())
    space = os.statvfs(path)
    return dict(path=str(path), filesystem=kind, mount=mount,
                available_bytes=space.f_bavail * space.f_frsize,
                fragment_bytes=space.f_frsize, available_fragments=space.f_bavail)


def admit(config, receipt, *, initial, fact=host_fact):
    """Retain actual facts on refusal too; never lower the original threshold."""
    record = dict(schema='gomad.driver-host-admission/v1', status='REFUSED',
                  config_sha256=config['_config_sha256'], sent_epoch=config['sent_epoch'],
                  minimum_free_bytes=MIN_FREE_BYTES, observed_ns=time.time_ns(), volumes=[], reason='')
    try:
        scratch, paths = bindings(config)
        if initial:
            require(not scratch.exists(), 'scratch must be fresh')
        else:
            prior = read(Path(config['root']) / 'host-admission.json')
            require(prior['status'] == 'PASS' and prior['config_sha256'] == config['_config_sha256']
                    and prior['scratch'] == str(scratch), 'missing authenticated initial host admission')
            require(scratch.is_dir() and all(Path(name).is_dir() for name in paths.values()),
                    'admitted temporary paths disappeared')
        record.update(scratch=str(scratch), temporary_paths=paths)
        record['volumes'] = [fact(Path(config['root'])), fact(scratch.parent)]
        evidence, scratch_volume = record['volumes']
        kind = scratch_volume['filesystem']
        require('fuse' not in kind and kind != 'virtiofs', 'scratch filesystem refused: ' + kind)
        for volume in (evidence, scratch_volume):
            require(volume['available_bytes'] >= MIN_FREE_BYTES,
                    'less than 12 GiB free on ' + volume['path'])
        if initial:
            scratch.mkdir()
            for name in paths.values():
                Path(name).mkdir()
        record['status'] = 'PASS'
    except (Refusal, OSError, ValueError, KeyError, TypeError) as error:
        record['reason'] = str(error)
        raise Refusal(str(error)) from error
    finally:
        write(receipt, record)
    return record
