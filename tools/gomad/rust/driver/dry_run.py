#!/usr/bin/env python3
"""Pre-freeze dry-run builds: role executables from the working tree, recorded by SHA, never a frozen receipt.

The dry run ("Find bugs early" rule 2) executes every final-cycle stage on the current head before the
freeze. It does not require the fixed-role build custody, head binding or receipts of `main_cli_build.py`;
those happen once at the freeze. A dry-run record is refused by every reader unless the loaded driver
configuration declares `dry_run: true`, and `configure.py` never freezes one into a qualifying configuration.
"""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

from driver import REPO, Refusal, read, require, sha, write
import main_cli_build

SCHEMA = 'gomad.dry-run-build/v1'
UNEXECUTED = 'UNEXECUTED: dry-run composition left this group unbound'
# Pre-freeze preparation scripts without a driver configuration opt in explicitly.
_ENABLED = os.environ.get('GOMAD_DRIVER_DRY_RUN') == '1'


def enable(config):
    """Admit dry-run records only for a configuration that declares itself a dry run."""
    global _ENABLED
    _ENABLED = config.get('dry_run') is True
    return _ENABLED


def enabled():
    return _ENABLED


def is_record(path):
    try:
        return read(path).get('schema') == SCHEMA
    except (OSError, ValueError, AttributeError):
        return False


def read_record(path, frozen=None, expected_role=None):
    """Return a receipt-shaped view of a dry-run build; custody fields bind to the dry-run configuration."""
    require(_ENABLED, 'dry-run build record outside a dry-run configuration')
    record = read(path)
    require(record.get('schema') == SCHEMA and record.get('role') in main_cli_build.ROLES
            and (expected_role is None or record['role'] == expected_role), 'foreign dry-run build role')
    require(Path(record['binary']).is_file() and sha(record['binary']) == record['binary_sha256'],
            'dry-run binary changed after its build')
    receipt = dict(record, exit=0, profile='release', inputs={}, root=str(Path(path).parent), source_sha256=SCHEMA,
                   head=SCHEMA, symlinks={}, tools_plan=None, gnu_plan=None)
    if frozen is not None:
        receipt.update(source_sha256=frozen['source_sha256'], head=frozen['head'], symlinks=frozen['symlinks'])
        if not record['helper_features']:
            require(frozen['identity']['helper'] == record['helper_sha256'], 'dry-run helper differs from the candidate helper')
        if record['role'] in ('release', 'runtime-harness', 'fn222-support', 'interception-harness', 'ordinary-host-test'):
            require(frozen['identity']['host'] == record['host_sha256'], 'dry-run host differs from the candidate host')
    return receipt


def cargo_command(cargo, role):
    test = role in main_cli_build.LIBTESTS
    helper = role in main_cli_build.HELPERS
    examples = main_cli_build.EXAMPLES
    target = (['--lib', '--no-run'] if test else ['--example', examples[role]] if role in examples
              else ['--bin', 'gomad-wasmhost' if helper else 'gomad'])
    features = ['conformance'] if role == 'conformance-reference' else main_cli_build.ROLES[role]
    return [cargo, 'test' if test else 'build', '--release', '--locked', '--no-default-features',
            '-p', 'gomad-wasmhost' if helper else 'gomad', *target, '--message-format=json',
            '--manifest-path', str(REPO / 'tools/gomad/rust/Cargo.toml')] + (['--features', ','.join(features)] if features else [])


def build(role, root, cargo, target):
    """Build one role into ROOT/ROLE with a shared incremental target; return the record path."""
    require(role in main_cli_build.ROLES, 'unknown role')
    output = root / role
    require(not output.exists(), 'dry-run role output must be fresh')
    output.mkdir(mode=0o700, parents=True)
    env = dict(os.environ, CARGO_TARGET_DIR=str(target), RUSTUP_TOOLCHAIN='1.99.1')
    argv = cargo_command(cargo, role)
    completed = subprocess.run(argv, cwd=REPO, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    (output / 'cargo.stdout').write_text(completed.stdout)
    (output / 'cargo.stderr').write_text(completed.stderr)
    require(completed.returncode == 0, role + ': dry-run cargo build failed; see ' + str(output / 'cargo.stderr'))
    frames = [json.loads(line) for line in completed.stdout.splitlines() if line.startswith('{')]
    helper = role in main_cli_build.HELPERS
    selected = 'gomad-wasmhost' if helper else 'gomad'
    name = main_cli_build.EXAMPLES.get(role, 'gomad_wasmhost' if helper and role in main_cli_build.LIBTESTS else
                                       'gomad-wasmhost' if helper else 'gomad')
    rows = [row for row in frames if row.get('reason') == 'compiler-artifact' and row.get('executable')
            and Path(row['manifest_path']).parent.name == ('wasmhost' if helper else 'gomad')
            and row['target']['name'] == name]
    require(len(rows) == 1, role + ': dry-run artifact missing or ambiguous')
    identities = {}
    for row in frames:
        if row.get('reason') == 'build-script-executed':
            identities.update(dict(row.get('env', [])))
    binary = output / main_cli_build.binary_name(role)
    shutil.copyfile(rows[0]['executable'], binary)
    binary.chmod(0o755)
    head = subprocess.check_output(['git', '-C', str(REPO), 'rev-parse', 'HEAD'], text=True).strip()
    dirty = bool(subprocess.check_output(['git', '-C', str(REPO), 'status', '--porcelain', '--untracked-files=no'], text=True).strip())
    record = dict(schema=SCHEMA, role=role, binary=str(binary), binary_sha256=sha(binary), build_head=head, build_dirty=dirty,
                  features=main_cli_build.ROLES[role], helper_features=main_cli_build.helper_features(role),
                  host_sha256=None if helper else identities.get('GOMAD_HOST_IMPLEMENTATION_SHA256'),
                  helper_sha256=identities.get('GOMAD_LINKED_IMPLEMENTATION_SHA256'),
                  runtime_implementation_sha256=None if helper else identities.get('GOMAD_RUNTIME_IMPLEMENTATION_SHA256'),
                  argv=argv, scope='pre-freeze dry run only; no qualification, no frozen custody')
    require(record['helper_sha256'] and (helper or record['host_sha256']), role + ': build identities not observed')
    path = output / 'dry-build.json'
    write(path, record)
    return path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--target', type=Path, required=True, help='shared incremental Cargo target')
    parser.add_argument('--cargo', default=shutil.which('cargo') or 'cargo')
    parser.add_argument('roles', nargs='+')
    args = parser.parse_args()
    try:
        for role in args.roles:
            print(json.dumps(dict(role=role, record=str(build(role, args.root.resolve(), args.cargo, args.target.resolve())))), flush=True)
        return 0
    except (Refusal, OSError, ValueError, subprocess.CalledProcessError) as error:
        print('REFUSED: ' + str(error), file=sys.stderr)
        return 2


if __name__ == '__main__':
    raise SystemExit(main())
