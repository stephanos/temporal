#!/usr/bin/env python3
"""Reconstruct the retained task-14 boundary without changing the checkout."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shlex
import subprocess
import tarfile
import tempfile
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
BASE = '58b718565044ab3bc3385d3323ee908a6d54328e'
MODULE = 'tools/gomad3'
OVERLAY = MODULE + '/toolchain/runtime/overlay/'
GO = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go')
MANIFEST_SHA = '7f6dd7279dd3465cf8ac9fa98fa4a033597203c9872b792204cd53932fcd226d'
PREIMAGES = {
    OVERLAY + 'src/internal/gomadio/' + name:
    Path('/tmp/gomad-task17.WFwzYI/baseline-overlay/src/internal/gomadio') / name
    for name in ('network.go', 'process_network.go', 'process_commands_export_test.go')
}
PREIMAGES.update({
    OVERLAY + 'src/internal/gomadfs/' + name:
    Path('/tmp/gomad-task17.WFwzYI/baseline-overlay/src/internal/gomadfs') / name
    for name in ('fs.go', 'process_volume.go')
})
PREIMAGES[MODULE + '/toolchain/version/version.json'] = Path('/tmp/gomad-task17.WFwzYI/baseline-version/version.json')
MIRRORS = {
    MODULE + '/choice/internal/wire/wire_generated.go',
    MODULE + '/target/internal/livecap/protocol_generated.go',
    OVERLAY + 'src/cmd/internal/gomadcap/protocol_generated.go',
    OVERLAY + 'src/internal/gomadchoicewire/wire_generated.go',
}


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)


def require(condition, detail):
    if not condition:
        raise RuntimeError(detail)


def safe_path(name):
    path = PurePosixPath(name)
    require(not path.is_absolute() and '..' not in path.parts, 'unsafe archive path: ' + name)
    require(name in ('tools', MODULE) or name.startswith(MODULE + '/'), 'outside module archive: ' + name)
    return path


def inventory(scratch):
    rows = []
    for path in sorted((scratch / MODULE).rglob('*')):
        relative = path.relative_to(scratch)
        if '.toolchain' in relative.parts or '.bin' in relative.parts:
            continue
        require(not path.is_symlink(), 'unexpected scratch symlink: ' + str(relative))
        if path.is_file():
            rows.append({'path': str(relative), 'sha256': sha(path.read_bytes())})
    return rows


def inventory_identity(rows):
    return sha((''.join(row['sha256'] + '  ' + row['path'] + '\n' for row in rows)).encode())


def run_checks(scratch, rows):
    require(GO.is_file(), 'pinned stock Go unavailable: ' + str(GO))
    env = os.environ.copy()
    for key in ('GOROOT', 'GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOEXPERIMENT'):
        env.pop(key, None)
    env.update(GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2', GOENV='off', GOFLAGS='')
    env['PATH'] = str(GO.parent) + os.pathsep + env.get('PATH', '')
    env['TMPDIR'] = str(scratch / 'tmp')
    Path(env['TMPDIR']).mkdir()
    go = str(GO)
    commands = [
        ('host', [go, 'env', 'GOVERSION', 'GOOS', 'GOARCH', 'GOROOT']),
        ('architecture-ownership', [go, 'test', '-count=1', '-tags', 'test_dep', '-run',
          '^(TestPackageArchitecture|TestProcessCommandsOwnModelWireTranslation)$', '.']),
        ('protocol-version', [go, 'test', '-count=1', '-tags', 'test_dep',
          './internal/gomadtool/generation/protocol', './toolchain/version']),
        ('version-generate-check', [go, 'run', './cmd/gomadtool', 'version-generate', '-check']),
        ('protocol-generate-check', [go, 'run', './cmd/gomadtool', 'protocol-generate', '-check']),
        ('validate-toolchain', ['make', 'validate-toolchain']),
    ]
    results = []
    for name, argv in commands:
        require(inventory(scratch) == rows, 'source changed before ' + name)
        started = time.monotonic()
        result = subprocess.run(argv, cwd=scratch / MODULE, env=env, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, timeout=300)
        elapsed = time.monotonic() - started
        log = OUT / ('checkpoint-' + name + '.log')
        require(not log.exists(), 'refusing to overwrite existing checkpoint log: ' + str(log))
        log.write_bytes(result.stdout)
        require(inventory(scratch) == rows, 'source changed during ' + name)
        row = {'argv': argv, 'command': shlex.join(argv), 'cwd': str(scratch / MODULE),
               'environment': {key: env[key] for key in ('GOWORK', 'GOTOOLCHAIN', 'GOMAXPROCS', 'GOENV', 'GOFLAGS', 'PATH', 'TMPDIR')},
               'exit_code': result.returncode, 'elapsed_seconds': round(elapsed, 6),
               'log': log.name, 'log_sha256': sha(result.stdout),
               'source_inventory_sha256': inventory_identity(rows), 'source_unchanged_after': True}
        if name == 'host':
            row['output'] = result.stdout.decode().strip().splitlines()
        results.append(row)
        print(json.dumps({'check': name, 'exit_code': result.returncode, 'elapsed_seconds': row['elapsed_seconds']}), flush=True)
    return results


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--checks', action='store_true')
    args = parser.parse_args()
    report_path = OUT / 'checkpoint-reconstruction.json'
    require(not report_path.exists(), 'refusing to overwrite existing checkpoint report')
    require(git('rev-parse', 'HEAD').decode().strip() == BASE, 'HEAD differs from committed task-13 predecessor')
    manifest = (OUT / 'final-source.sha256').read_bytes()
    require(sha(manifest) == MANIFEST_SHA, 'retained task-14 manifest SHA-256 mismatch')
    expected = {}
    for line in manifest.decode().splitlines():
        identity, path = line.split('  ', 1)
        require(re.fullmatch('[0-9a-f]{64}', identity) is not None, 'invalid retained hash')
        safe_path(path)
        require(path not in expected, 'duplicate retained path: ' + path)
        expected[path] = identity
    require(len(expected) == 17 and PREIMAGES.keys() <= expected.keys() and MIRRORS <= expected.keys(), 'unexpected retained manifest scope')
    source_data, origins = {}, []
    for path, identity in expected.items():
        source = PREIMAGES.get(path, ROOT / path)
        require(source.is_file() and not source.is_symlink(), 'input is not a regular file: ' + str(source))
        data = source.read_bytes()
        require(sha(data) == identity, 'INPUT SHA-256 MISMATCH: ' + str(source) + ' expected ' + identity + ' observed ' + sha(data))
        category = 'historical_preimage' if path in PREIMAGES else 'shared_task13_mirror' if path in MIRRORS else 'exclusive_current_task14'
        source_data[path] = data
        origins.append({'path': path, 'input': str(source), 'sha256': identity, 'origin': category})
    archive = git('archive', '--format=tar', BASE, MODULE)
    scratch = Path(tempfile.mkdtemp(prefix='fn109-task14-checkpoint.', dir='/tmp'))
    base_data = {}
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        seen = set()
        for member in tar.getmembers():
            safe_path(member.name)
            require(member.name not in seen, 'duplicate archive member: ' + member.name)
            seen.add(member.name)
            require(member.isdir() or member.isfile(), 'nonregular archive member: ' + member.name)
            destination = scratch / member.name
            if member.isdir():
                destination.mkdir(parents=True, exist_ok=True)
                continue
            data = tar.extractfile(member).read()
            base_data[member.name] = data
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)
            destination.chmod(member.mode & 0o777)
    for path, data in source_data.items():
        destination = scratch / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(data)
    rows = inventory(scratch)
    actual = {row['path']: row['sha256'] for row in rows}
    changed = []
    for path in sorted(set(base_data) | set(actual)):
        old = sha(base_data[path]) if path in base_data else None
        if old != actual.get(path):
            changed.append({'path': path, 'predecessor_sha256': old, 'sha256': actual.get(path),
                            'status': 'modified' if old is not None else 'new'})
    require(len(changed) == 13 and {row['path'] for row in changed} == set(expected) - MIRRORS,
            'checkpoint delta is not the exact 13 task-14 source paths')
    for path, identity in expected.items():
        require(actual[path] == identity, 'reconstructed SHA-256 mismatch: ' + path)
    require(all(sha(base_data[path]) == expected[path] for path in MIRRORS), 'task-13 mirrors unexpectedly changed')
    architecture = MODULE + '/architecture_test.go'
    require(actual[architecture] == sha(base_data[architecture]), 'later architecture source imported')
    support = {}
    for name in ('handover.md', 'source-audit.md', 'evidence.json', 'checkpoint-preparation-state.json', 'baseline-vectors.json'):
        data = (OUT / name).read_bytes()
        support[name] = sha(data)
    require(support['baseline-vectors.json'] == 'adf815e906f66690a448d329f266099973f5c2df1c27ee47cedc2682ba258545', 'baseline vector identity mismatch')
    report = {
        'task': 'fn-109-gomad-deepen-modules-and-tool-interfaces.14',
        'purpose': 'exact-source checkpoint preparation for conductor-only staging; not SHIP or acceptance',
        'base_revision': BASE, 'scratch': str(scratch), 'module_root': str(scratch / MODULE),
        'archive_sha256': sha(archive), 'retained_manifest_sha256': MANIFEST_SHA,
        'helper_sha256': sha(Path(__file__).read_bytes()), 'retained_hashes_matched': 17,
        'exclusive_current_inputs': 7, 'historical_preimage_inputs': 6, 'unchanged_task13_mirrors': 4,
        'origins': origins, 'changed_sources': changed, 'changed_source_count': len(changed),
        'source_inventory_sha256': inventory_identity(rows), 'source_files': len(rows), 'sources': rows,
        'architecture_sha256': actual[architecture], 'architecture_matches_committed_predecessor': True,
        'original_evidence_sha256': support,
        'original_review': 'Two fresh actual-source audits in source-audit.md found no concrete defect. Original 42-case external-GOROOT stand-in results remain developmental and were not rerun.',
        'model_routing': {'assigned_model': 'gpt-6.1-sol', 'assigned_effort': 'high',
                          'judge_result': 'no_key, already judged; not rejudged', 'actual_execution_metadata': 'unknown'},
        'stock_go': str(GO), 'host': {'os': platform.system().lower(), 'arch': platform.machine()},
        'checks': run_checks(scratch, rows) if args.checks else [],
        'full_make_validate': {'run': False, 'reason': 'Module-only archive has no root tests package or tools/gomad3integration qualification inputs required by validate-qualification. Available validate-toolchain checks run separately.'},
        'native_acceptance': False,
        'native_gates_open': ['darwin/arm64: rebuild, pinned overlay tests, real process/simulation and full host gates',
                              'linux/amd64: rebuild, pinned overlay tests, real process/simulation and full host gates'],
        'git_mutations': False, 'shared_source_writes': False, 'flow_state_writes': False,
        'commits': [], 'prs': [],
    }
    require(inventory(scratch) == rows, 'final source inventory changed')
    require(git('rev-parse', 'HEAD').decode().strip() == BASE, 'HEAD changed during reconstruction')
    report['all_requested_checkpoint_checks_passed'] = bool(args.checks) and all(row['exit_code'] == 0 for row in report['checks'])
    report_path.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps({key: report[key] for key in ('scratch', 'retained_hashes_matched', 'changed_source_count', 'source_files', 'all_requested_checkpoint_checks_passed')}), flush=True)
    if args.checks and not report['all_requested_checkpoint_checks_passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
