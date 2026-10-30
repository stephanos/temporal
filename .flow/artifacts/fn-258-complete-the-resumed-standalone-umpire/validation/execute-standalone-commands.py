#!/usr/bin/env python3
"""Run the remaining named standalone commands and retain actual results."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time

HERE = Path(__file__).resolve().parent
TEMPORAL = next(p for p in HERE.parents if (p / '.flow/specs').is_dir())
SOURCE = Path('/home/agent/.cache/codex-build/tmp/fn258-standalone-0547321/checkout')
HEAD = '05473210b48c3db862f22051eb7154ce98258f98'
TARGET = Path('/home/agent/.cache/codex-build/fn258-standalone-0547321')
RELEASE = TEMPORAL / '.flow/tmp/fn258-standalone-release-0547321'
OUTPUT = HERE / 'standalone-named-0547321'


def git(*args):
    return subprocess.check_output(['git', '-C', str(SOURCE), *args], text=True).strip()


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n')


assert git('rev-parse', 'HEAD') == HEAD
assert not git('status', '--porcelain=v1', '--untracked-files=all')
OUTPUT.mkdir(exist_ok=True)
environment = os.environ.copy()
for key in ('BASH_ENV', 'UMPIRE_BACKENDS', 'UMPIRE_APALACHE', 'UMPIRE_TOOL_ROOT',
            'UMPIRE_STANDALONE_CHILD', 'UMPIRE_STANDALONE_OUTPUT', 'UMPIRE_GATE_RECEIPTS',
            'UMPIRE_GATE_CONTROLLER_EXECUTABLE', 'RUSTC_WRAPPER', 'RUSTC_WORKSPACE_WRAPPER',
            'NEXTEST_PROFILE', 'RUSTFLAGS', 'CARGO_ENCODED_RUSTFLAGS'):
    environment.pop(key, None)
settings = {'CARGO_BUILD_JOBS': '1', 'CARGO_INCREMENTAL': '0', 'NEXTEST_TEST_THREADS': '4',
            'RUST_TEST_THREADS': '1', 'GOMAXPROCS': '1', 'GOFLAGS': '-p=1', 'CGO_ENABLED': '0',
            'MISE_AUTO_INSTALL': 'false', 'MISE_EXEC_AUTO_INSTALL': 'false',
            'MISE_NOT_FOUND_AUTO_INSTALL': 'false', 'MISE_TRUSTED_CONFIG_PATHS': str(SOURCE.parent)}
environment.update(settings)
for name, command, target, reservation in [
    ('make-check', ['make', 'check'], TARGET, 2),
    ('make-time', ['make', 'time'], TARGET, 2),
    ('release', ['mise', 'exec', '--', 'cargo', 'test', '--workspace', '--release', '--locked'], RELEASE, 8),
]:
    receipt = OUTPUT / (name + '.json')
    if receipt.exists():
        raise SystemExit(f'Refusing to overwrite existing command receipt: {receipt}')
    assert git('rev-parse', 'HEAD') == HEAD
    assert not git('status', '--porcelain=v1', '--untracked-files=all')
    ancestor = target
    while not ancestor.exists():
        ancestor = ancestor.parent
    free = shutil.disk_usage(ancestor).free
    if free - reservation * 2**30 < 30 * 2**30:
        raise SystemExit(f'Insufficient admitted headroom for {name}: {free} bytes free')
    target_existed = target.exists()
    target.mkdir(parents=True, exist_ok=True)
    env = dict(environment, CARGO_TARGET_DIR=str(target))
    log = OUTPUT / (name + '.log')
    row = {'source': HEAD, 'source_tree': git('rev-parse', 'HEAD^{tree}'), 'cwd': str(SOURCE),
           'argv': command, 'environment': dict(settings, CARGO_TARGET_DIR=str(target)),
           'backend_mode': 'default (UMPIRE_BACKENDS and UMPIRE_APALACHE unset)',
           'free_before_bytes': free, 'reserved_additional_gib_estimate': reservation,
           'target_existed_before': target_existed, 'log': log.name}
    print(f'START {name}: {command}', flush=True)
    started = time.monotonic()
    with log.open('wb') as stream:
        process = subprocess.run(command, cwd=SOURCE, env=env, stdout=stream, stderr=subprocess.STDOUT)
    row.update(exit_code=process.returncode, seconds=time.monotonic() - started,
               log_sha256=hashlib.sha256(log.read_bytes()).hexdigest(),
               free_after_bytes=shutil.disk_usage(ancestor).free,
               source_status_after=git('status', '--porcelain=v1', '--untracked-files=all'))
    save(receipt, row)
    print(f'END {name}: exit={process.returncode} seconds={row["seconds"]:.3f}', flush=True)
    if process.returncode or row['source_status_after']:
        raise SystemExit(process.returncode or 1)
