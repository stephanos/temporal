#!/usr/bin/env python3
"""Admit current-source cycle roles without the superseded formal build-receipt ceremony.

Cargo observations bind each role to its actual compiler, feature set and implementation source.
Matching builds may be reused across commits; build_head remains their original provenance while
returned custody describes the authenticated current execution inputs. A loaded configuration must
opt in with dry_run=true; the resulting source gates alone claim no qualification pass.
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
# Preparation scripts without a loaded driver configuration opt in explicitly.
_CONFIG = None
_ENABLED = os.environ.get('GOMAD_DRIVER_DRY_RUN') == '1'


def enable(config):
    """Admit dry-run records only for a configuration that declares itself a dry run."""
    global _ENABLED, _CONFIG
    _CONFIG = config if config.get('dry_run') is True else None
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
    role = record['role']
    require(record['features'] == main_cli_build.ROLES[role]
            and record['helper_features'] == main_cli_build.helper_features(role), 'dry-run compiled role features differ')
    cargo = Path(path).parent / 'cargo.stdout'
    frames = [json.loads(line) for line in cargo.read_text().splitlines() if line.startswith('{')]
    finished = [row for row in frames if row.get('reason') == 'build-finished']
    require(len(finished) == 1 and finished[0].get('success') is True and frames[-1] == finished[0],
            'dry-run Cargo build did not complete')
    helper = [row for row in frames if row.get('reason') == 'build-script-executed'
              and dict(row.get('env', [])).get('GOMAD_LINKED_IMPLEMENTATION_SHA256') == record['helper_sha256']]
    require(len(helper) == 1, 'dry-run linked build identity missing or ambiguous')
    output = Path(helper[0]['out_dir'])
    compiler = output / 'identity-compiler.txt'
    bindings = output / 'identity-bindings.tsv'
    observed = dict(line.split('\t', 1) for line in bindings.read_text().splitlines())
    import native_roles
    require(observed == native_roles.feature_bindings(record['helper_features'], observed.get('TARGET'))
            and observed.get('TARGET') and compiler.read_text().startswith('rustc 1.99.1 '),
            'dry-run compiler/feature context differs')
    context = dict(compiler_utf8=compiler.read_text(), bindings={key: observed[key] for key in ('PROFILE', 'TARGET')})
    selected = [row for row in frames if row.get('reason') == 'compiler-artifact' and row.get('executable')
                and Path(row['manifest_path']).parent.name == ('wasmhost' if role in main_cli_build.HELPERS else 'gomad')
                and row['target']['name'] == main_cli_build.EXAMPLES.get(role,
                    'gomad_wasmhost' if role in main_cli_build.HELPERS and role in main_cli_build.LIBTESTS else
                    'gomad-wasmhost' if role in main_cli_build.HELPERS else 'gomad')]
    require(len(selected) == 1 and sorted(selected[0]['features']) == sorted(record['features'])
            and selected[0]['profile']['opt_level'] == '3' and not selected[0]['profile']['debug_assertions']
            and selected[0]['profile']['test'] == (role in main_cli_build.LIBTESTS), 'dry-run Cargo artifact role differs')
    candidate = frozen if frozen is not None else _CONFIG or {}
    repo = Path(candidate.get('repo', REPO))
    files, source, head, links = main_cli_build.source_state(repo, '/usr/bin/git')
    identity, runtime, _ = main_cli_build.identities(dict(repo=str(repo), files=files, main_role=role), context)
    require(record['host_sha256'] == identity['host'] and record['helper_sha256'] == identity['helper']
            and record['runtime_implementation_sha256'] == runtime, 'dry-run compiled implementation differs from current source/features')
    if role not in main_cli_build.HELPERS:
        hosts = [dict(row.get('env', [])) for row in frames if row.get('reason') == 'build-script-executed'
                 and 'GOMAD_HOST_IMPLEMENTATION_SHA256' in dict(row.get('env', []))]
        require(len(hosts) == 1 and hosts[0]['GOMAD_HOST_IMPLEMENTATION_SHA256'] == identity['host']
                and hosts[0]['GOMAD_RUNTIME_IMPLEMENTATION_SHA256'] == runtime, 'dry-run host build identity differs')
    inputs = dict(files)
    inputs.update({str(name): sha(name) for name in (Path(record['binary']), Path(path), cargo, compiler, bindings)})
    receipt = dict(record, exit=0, profile='release', inputs=inputs, root=str(Path(path).parent), source_sha256=source,
                   head=head, symlinks=links, tools_plan=None, gnu_plan=None, context=context)
    if candidate:
        require((candidate['source_sha256'], candidate['head'], candidate['symlinks']) == (source, head, links),
                'dry-run source custody differs from current candidate')
        if not record['helper_features']:
            require(candidate['identity']['helper'] == record['helper_sha256'], 'dry-run helper differs from the candidate helper')
        if role in ('release', 'runtime-harness', 'fn222-support', 'interception-harness', 'ordinary-host-test'):
            require(candidate['identity']['host'] == record['host_sha256'], 'dry-run host differs from the candidate host')
        if frozen is not None:
            require(all(frozen['files'].get(name) == value for name, value in inputs.items()), 'dry-run build custody is incompletely sealed')
            if role == 'release':
                require(frozen['binary'] == record['binary'] and frozen['binary_sha256'] == record['binary_sha256'],
                        'dry-run release differs from the admitted candidate')
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
                  argv=argv, scope='current-source cycle preparation; no qualification pass')
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
