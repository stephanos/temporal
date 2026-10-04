#!/usr/bin/env python3
"""Prepare task18 on committed task17 with bounded validation AST inputs."""
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import shlex
import subprocess
import tarfile
import tempfile
import time

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
OUT = Path(__file__).resolve().parent
HEAD = '9c0438b314b5a8368b1067a02abb865c24077f70'
GO = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go')
MANIFEST_SHA = '1d42629bd964f696412940bf75619878c965b07f54cfe693f7eb31e91d5861be'
EVIDENCE_SHA = '89ede09e01d2fc8143f894e838b02547c2bf34816b8fbede1b9d852be82a7131'
MODULE = 'tools/gomad3'
SIM = 'tools/gomad3sim'
JSON_INPUTS = ['tools/gomad3integration/qualification/tests.generator.json',
               'tools/gomad3integration/qualification/tests.json']
ARCHITECTURE = 'PackageArchitecture|PublicPackagesDoNotExportTypeAliases|RunnerExecutionInjectionIsPrivate|RunnerRequestsCompileInExternalModule|CurrentVocabularyHasNoLegacyCampaignBoundary|MakeTargetsMatchTheirOwnership|ExactModuleEdges|CapabilityEvaluationHasNoHostEffect|DomainModulesDoNotExportWireFraming'
FILTER = '^Test(' + ARCHITECTURE + '|FilesystemHandlesOwnOneImplementation|NetworkHandlesOwnOneImplementation|SimulationGateSelectsProcessNetworkHandles|ProcessCommandsOwnModelWireTranslation)$'
PREDECESSOR = {
    MODULE + '/Makefile': '28318e32d3360cfcd644298944e864e704670ac80f55c550d32461a55afc8497',
    MODULE + '/simulation_gate_selection_test.go': '57febc14101facc1b4db77ba3f8dba55669e51036e23790eb1372c27dec38bbb',
    MODULE + '/runner/internal/execution/simulation_root_integration_test.go': 'bc4c86b6210232026f0eb94edbee209fa9817a62b66d58a2da7b627e9bbbf833',
    MODULE + '/toolchain/version/version.json': '75d2429a49e42df374e2fc2b77710d1694322c5198a4b54f6c9bfb95ab88c8de',
}


def sha(data):
    return hashlib.sha256(data).hexdigest()


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)


def source_inventory(scratch):
    result = {}
    for path in sorted(scratch.rglob('*')):
        relative = path.relative_to(scratch)
        if '.toolchain' in relative.parts or relative.parts[0] == 'tmp':
            continue
        require(not path.is_symlink(), 'unexpected source symlink: ' + str(relative))
        if path.is_file():
            result[str(relative)] = sha(path.read_bytes())
    return result


def manifest(data):
    return ''.join(identity + '  ' + path + '\n' for path, identity in sorted(data.items())).encode()


def new_output(name, data):
    path = OUT / name
    require(not path.exists(), 'refusing to overwrite checkpoint output: ' + str(path))
    path.write_bytes(data)
    return {'path': str(path), 'sha256': sha(data)}


def main():
    require(git('rev-parse', 'HEAD').decode().strip() == HEAD, 'HEAD differs from exact committed task17')
    manifest_data = (OUT / 'final-source.sha256').read_bytes()
    require(sha(manifest_data) == MANIFEST_SHA, 'retained task18 manifest hash mismatch')
    evidence_data = (OUT / 'evidence.json').read_bytes()
    require(sha(evidence_data) == EVIDENCE_SHA, 'original task18 evidence hash mismatch')
    evidence = json.loads(evidence_data)
    expected = {}
    source_data = {}
    for line in manifest_data.decode().splitlines():
        identity, path = line.split('  ', 1)
        require(re.fullmatch('[0-9a-f]{64}', identity) is not None and path not in expected, 'invalid source identity')
        require(path.startswith(MODULE + '/') or path.startswith(SIM + '/'), 'task18 source outside approved modules')
        source = ROOT / path
        require(source.is_file() and not source.is_symlink(), 'source is not regular: ' + path)
        data = source.read_bytes()
        require(sha(data) == identity, 'INPUT HASH MISMATCH: ' + path)
        expected[path], source_data[path] = identity, data
    require(len(expected) == 14, 'unexpected task18 source scope')
    tree = git('ls-tree', '-r', '--name-only', HEAD, '--', 'tests').decode().splitlines()
    tests = [path for path in tree if re.fullmatch(r'tests/[^/]+_test\.go', path)]
    require(len(tests) == 113 and len(set(tests)) == 113, 'root AST inputs differ from exact 113 committed top-level test files')
    selected = [MODULE, SIM, 'go.mod', 'go.sum', *JSON_INPUTS, *tests]
    archive_data = git('archive', '--format=tar', HEAD, *selected)
    scratch = Path(tempfile.mkdtemp(prefix='fn109-task18-checkpoint.', dir='/tmp'))
    baseline = {}
    with tarfile.open(fileobj=io.BytesIO(archive_data)) as archive:
        seen = set()
        for member in archive.getmembers():
            name = PurePosixPath(member.name)
            require(not name.is_absolute() and '..' not in name.parts, 'unsafe archive member')
            require(member.name not in seen and (member.isdir() or member.isfile()), 'duplicate or nonregular archive member')
            seen.add(member.name)
            destination = scratch / member.name
            if member.isdir():
                destination.mkdir(parents=True, exist_ok=True)
                continue
            allowed = member.name in ('go.mod', 'go.sum', *JSON_INPUTS, *tests) or member.name.startswith(MODULE + '/') or member.name.startswith(SIM + '/')
            require(allowed, 'unrequested archived input: ' + member.name)
            data = archive.extractfile(member).read()
            require(len(data) == member.size, 'truncated archive input')
            baseline[member.name] = sha(data)
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)
            destination.chmod(member.mode & 0o777)
    require(source_inventory(scratch) == baseline, 'extracted archive inventory mismatch')
    require({p for p in baseline if p.startswith('tests/')} == set(tests), 'root test archive exceeded AST-only scope')
    for path, identity in PREDECESSOR.items():
        require(baseline[path] == identity, 'historical task17 source predecessor mismatch: ' + path)
    for path, data in source_data.items():
        destination = scratch / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(data)
    candidate = source_inventory(scratch)
    changed = sorted(path for path in set(baseline) | set(candidate) if baseline.get(path) != candidate.get(path))
    require(len(changed) == 14 and set(changed) == set(expected), 'candidate delta exceeds exactly fourteen task18 paths')
    require(all(candidate[path] == identity for path, identity in expected.items()), 'final retained source mismatch')
    require(not (scratch / MODULE / 'internal/gomadtool/architecture').exists(), 'later task19 checker imported')
    require(candidate[MODULE + '/architecture_test.go'] == baseline[MODULE + '/architecture_test.go'], 'inherited architecture changed')
    descriptor = json.loads(source_data[MODULE + '/toolchain/version/version.json'])
    overlay_prefix = MODULE + '/toolchain/runtime/overlay/'
    overlay_paths = sorted(path[len(overlay_prefix):] for path in candidate if path.startswith(overlay_prefix))
    require(len(descriptor['overlay_allowlist']) == 79 and descriptor['overlay_allowlist'] == overlay_paths, 'descriptor differs from exact 79 overlay files')
    manifests = {
        'before': new_output('checkpoint-source-before.log', manifest(baseline)),
        'after': new_output('checkpoint-source-after.log', manifest(candidate)),
        'module_before': new_output('checkpoint-module-before.log', manifest({p: v for p, v in baseline.items() if p.startswith(MODULE + '/')})),
        'module_after': new_output('checkpoint-module-after.log', manifest({p: v for p, v in candidate.items() if p.startswith(MODULE + '/')})),
    }
    env = os.environ.copy()
    unset = ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOROOT', 'GOEXPERIMENT']
    for key in unset:
        env.pop(key, None)
    env.update(GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2', GOENV='off', GOFLAGS='', GOMAD3_STOCK_GO=str(GO), PATH=str(GO.parent) + os.pathsep + env.get('PATH', ''), TMPDIR=str(scratch / 'tmp'))
    Path(env['TMPDIR']).mkdir()
    observations = []

    def run(label, argv, listed=None):
        require(source_inventory(scratch) == candidate, 'source changed before ' + label)
        started = time.monotonic()
        result = subprocess.run(argv, cwd=scratch / MODULE, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=300)
        elapsed = time.monotonic() - started
        output = result.stdout.decode(errors='replace')
        row = {'label': label, 'argv': argv, 'command': shlex.join(argv), 'cwd': str(scratch / MODULE), 'environment': {key: env[key] for key in ('GOWORK', 'GOTOOLCHAIN', 'GOMAXPROCS', 'GOENV', 'GOFLAGS', 'GOMAD3_STOCK_GO', 'PATH', 'TMPDIR')}, 'unset': unset, 'exit_code': result.returncode, 'elapsed_seconds': round(elapsed, 6), 'log': new_output('checkpoint-' + label + '.log', result.stdout), 'source_manifest_sha256': manifests['after']['sha256']}
        if '-list' in argv:
            row['selected_tests'] = re.findall(r'^Test\w+$', output, re.MULTILINE)
        if listed is not None:
            row['passed_top_level_tests'] = re.findall(r'^--- PASS: (Test\w+) \(', output, re.MULTILINE)
            require(set(row['passed_top_level_tests']) == set(listed), 'executed tests differ from listing')
        observations.append(row)
        print(json.dumps({'label': label, 'exit_code': result.returncode, 'elapsed_seconds': row['elapsed_seconds'], 'scratch': str(scratch)}), flush=True)
        require(result.returncode == 0, 'checkpoint command failed: ' + label + '\n' + output)
        require(source_inventory(scratch) == candidate, 'source changed during ' + label)
        return row

    go = str(GO)
    run('host', [go, 'env', 'GOVERSION', 'GOOS', 'GOARCH', 'GOROOT'])
    listed = run('ownership-selection-architecture-list', [go, 'test', '-count=1', '-tags', 'test_dep', '.', '-list', FILTER])['selected_tests']
    require(len(listed) == 13, 'ownership/selection/architecture count differs from expected thirteen')
    run('ownership-selection-architecture', [go, 'test', '-count=1', '-timeout=120s', '-tags', 'test_dep', '.', '-run', FILTER, '-v'], listed)
    run('generation-version', [go, 'test', '-count=1', '-timeout=120s', '-tags', 'test_dep', './internal/gomadtool/generation/...', './toolchain/version', '-v'])
    run('version-check', [go, 'run', './cmd/gomadtool', 'version-generate', '-check'])
    run('protocol-check', [go, 'run', './cmd/gomadtool', 'protocol-generate', '-check'])
    run('validate', ['make', 'validate'])
    run('scoped-vet', [go, 'vet', '-tags', 'test_dep', '.', './internal/gomadtool/generation/...', './toolchain/version'])
    formatted = subprocess.check_output([str(GO.parent / 'gofmt'), '-l', *(str(scratch / path) for path in expected if path.endswith('.go'))])
    new_output('checkpoint-gofmt.log', formatted)
    require(formatted == b'', 'retained source formatting changed')
    require(source_inventory(scratch) == candidate, 'final source inventory changed')
    require(git('rev-parse', 'HEAD').decode().strip() == HEAD, 'HEAD changed')
    require(all((ROOT / path).read_bytes() == data for path, data in source_data.items()), 'shared retained source changed')
    retained_logs = {}
    for row in evidence['command_logs']:
        if row['log'] in ('ownership-red.log', 'ownership-green.log', 'gate-selection-red.log', 'gate-selection-green.log', 'developmental-race.log', 'developmental-repeat.log', 'behavior-old.log', 'behavior-new.log', 'root-developmental-link.log', 'runner-integration-compile.log', 'preservation.log'):
            retained_logs[row['log']] = {'sha256': sha((OUT / row['log']).read_bytes()), 'historical_exit_code': row['exit'], 'scope': row['scope']}
    root_fixture = SIM + '/filesystem_handles_toolchain_test.go'
    report = {
        'task': evidence['task_id'], 'scope': 'bounded source checkpoint for conductor-only staging',
        'base_head': HEAD, 'archive_sha256': sha(archive_data), 'archive_paths': selected,
        'scratch': str(scratch), 'module_root': str(scratch / MODULE), 'helper_sha256': sha(Path(__file__).read_bytes()),
        'retained_manifest_sha256': MANIFEST_SHA, 'retained_source_hashes_matched': 14, 'changed_count': 14,
        'changed_source_paths': changed, 'source_origins': {path: {'input': str(ROOT / path), 'sha256': identity, 'origin': 'full-hash-verified current task18 bytes'} for path, identity in expected.items()},
        'delta': [{'path': path, 'before_sha256': baseline.get(path), 'after_sha256': candidate[path]} for path in changed],
        'source_files_before': len(baseline), 'source_files_after': len(candidate), 'complete_source_manifests': manifests,
        'root_tests_ast_inputs': {'count': 113, 'paths': tests, 'compiled_or_executed': False},
        'committed_task17_predecessor_sha256': PREDECESSOR, 'overlay_inventory_count': 79,
        'inherited_architecture_sha256': candidate[MODULE + '/architecture_test.go'], 'task19_checker_imported': False,
        'full_source_unchanged_after_each_check': True, 'ignored_generated_cache_prefix': MODULE + '/.toolchain/',
        'observations': observations, 'original_source_audit_sha256': sha((OUT / 'source-audit.md').read_bytes()),
        'original_evidence_sha256': EVIDENCE_SHA, 'original_handover_sha256': sha((OUT / 'handover.md').read_bytes()),
        'reused_original_logs': retained_logs,
        'root_fixture': {'path': root_fixture, 'sha256': expected[root_fixture], 'prior_link_only_log': 'root-developmental-link.log', 'prior_link_only_log_sha256': retained_logs['root-developmental-link.log']['sha256'], 'new_compilation_or_execution': False},
        'reuse_reason': 'The exact original candidate and source-audit identities match. Original behavior RED/GREEN, race/repeat, developmental shim and link-only evidence remains developmental; no new source or concurrency change warrants repeating it.',
        'model_routing': {'tier_judgments': 1, 'judge_reason': 'no_key', 'assigned_model': 'gpt-6.1-sol', 'assigned_effort': 'high', 'actual_model_metadata': 'unknown', 'original_writer_and_reviewer_same_family': True, 'judge_log_sha256': sha((OUT / 'checkpoint-judge.log').read_bytes()), 'judged_state_sha256': sha((OUT / 'checkpoint-preparation-state.json').read_bytes()), 'agents_or_bridges_spawned': 0},
        'native_acceptance': False, 'native_platforms_open': ['darwin/arm64', 'linux/amd64'], 'native_gaps': evidence['native_gaps'],
        'formal_ship_verdict': False, 'task_completion': False, 'shared_source_writes': False, 'git_index_writes': False, 'git_history_writes': False, 'flow_state_writes': False, 'commits': [], 'prs': [],
    }
    output = new_output('checkpoint-verification.json', (json.dumps(report, indent=2) + '\n').encode())
    print(json.dumps({'scratch': str(scratch), 'retained_source_hashes_matched': 14, 'changed_count': 14, 'overlay_inventory_count': 79, 'report': output}), flush=True)


if __name__ == '__main__':
    main()
