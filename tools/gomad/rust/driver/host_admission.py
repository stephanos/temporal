"""Original native-scratch and 12 GiB admission, before candidate dispatch."""
import os
from pathlib import Path
import re
import subprocess
import time

from driver import REPO, Refusal, read, require, write

MIN_FREE_BYTES = 12 * 1024 ** 3
# The probe is a frozen regular file; some hosts ship /usr/bin/stat as a coreutils multicall symlink.
STAT = next((path for path in (Path('/usr/bin/stat'), Path('/usr/bin/gnustat')) if path.is_file() and not path.is_symlink()),
            Path('/usr/bin/stat'))


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


def filesystem_type(path):
    # Ask the kernel about the path actually served, as the original shell gate
    # did. Mount-table sorting cannot identify overmounts or hidden descendants.
    result = subprocess.run([str(STAT), '-f', '-c', '%T', str(path)],
                            check=True, capture_output=True, text=True,
                            env={'LC_ALL': 'C'}, timeout=5)
    kind = result.stdout.strip()
    require(re.fullmatch(r'[a-zA-Z0-9._-]+', kind) is not None, 'unrecognized stat filesystem result')
    return kind


def host_fact(path):
    kind = filesystem_type(path)
    space = os.statvfs(path)
    return dict(path=str(path), filesystem=kind, probe=[str(STAT), '-f', '-c', '%T', str(path)],
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
    except (Refusal, OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        record['reason'] = str(error)
        raise Refusal(str(error)) from error
    finally:
        write(receipt, record)
    return record
