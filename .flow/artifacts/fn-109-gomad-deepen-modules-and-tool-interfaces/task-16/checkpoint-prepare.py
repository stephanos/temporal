#!/usr/bin/env python3
"""Prepare task16's retained boundary on the committed task15 module."""
import difflib
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
ARTIFACTS = Path(__file__).resolve().parent
HEAD = '230ffb0d8fafb7c2cd8b4341d56ccf4413c83e96'
GO = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go')
MODULE = 'tools/gomad3'
EXECUTION = MODULE + '/runner/internal/execution/'
EVIDENCE_SHA = 'a3ee543c0a329a9850e43160b9ad1e31f169bc52b4926d4832b8ea3c431072ba'
COMPARATOR_SHA = '2933200710392db987029a46486fd24b441250612c7fec2a9740cc66b0376631'
DESIGN = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md'
FOCUSED = 'SimulationTime|SimulationModel|SimulationCoordinator|ServeSimulation'
ARCHITECTURE = '^Test(PackageArchitecture|PublicPackagesDoNotExportTypeAliases|RunnerExecutionInjectionIsPrivate|RunnerRequestsCompileInExternalModule|CurrentVocabularyHasNoLegacyCampaignBoundary|MakeTargetsMatchTheirOwnership|ExactModuleEdges|CapabilityEvaluationHasNoHostEffect|DomainModulesDoNotExportWireFraming)$'
REVERSALS = [
    (b'_ = coordinator.time.progress.apply(modelRequestDispatched{coordinator: coordinator.coordinator})',
     b'coordinator.time.deliverExternal(coordinator.coordinator)'),
    (b'coordinator.time.progress.apply(modelAbandonedResponseDiscarded{coordinator: coordinator.coordinator, arrivals: frame.Arrivals})',
     b'coordinator.time.acknowledgeExternal(coordinator.coordinator, frame.Arrivals)'),
]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT)


def inventory(scratch):
    result = {}
    for path in sorted((scratch / MODULE).rglob('*')):
        require(not path.is_symlink(), 'unexpected module symlink: ' + str(path))
        if path.is_file():
            result[str(path.relative_to(scratch))] = sha(path.read_bytes())
    return result


def manifest(data):
    return ''.join(identity + '  ' + path + '\n' for path, identity in sorted(data.items())).encode()


def new_output(name, data):
    path = ARTIFACTS / name
    require(not path.exists(), 'refusing to overwrite checkpoint output: ' + str(path))
    path.write_bytes(data)
    return {'path': str(path), 'sha256': sha(data)}


def main():
    require(git('rev-parse', 'HEAD').decode().strip() == HEAD, 'HEAD differs from committed task15')
    evidence_data = (ARTIFACTS / 'evidence.json').read_bytes()
    require(sha(evidence_data) == EVIDENCE_SHA, 'task16 evidence hash mismatch')
    evidence = json.loads(evidence_data)
    preserved = evidence['characterization_preservation']
    expected = {EXECUTION + name: identity for name, identity in evidence['final_source_sha256'].items()}
    require(len(expected) == 10, 'unexpected task16 source scope')
    source_data = {}
    for path, identity in expected.items():
        source = ROOT / path
        require(source.is_file() and not source.is_symlink(), 'source is not a regular file: ' + path)
        source_data[path] = source.read_bytes()
        require(sha(source_data[path]) == identity, 'INPUT HASH MISMATCH: ' + path)
    design_data = (ROOT / DESIGN).read_bytes()
    require(sha(design_data) == preserved['unchanged_design_sha256'], 'selected design hash mismatch')
    require(git('show', HEAD + ':' + DESIGN) == design_data, 'design differs from task15 commit')
    comparator = (ARTIFACTS / 'compare-test-bodies.go').read_bytes()
    require(sha(comparator) == COMPARATOR_SHA, 'retained AST comparator hash mismatch')
    archive_data = git('archive', '--format=tar', HEAD, MODULE)
    scratch = Path(tempfile.mkdtemp(prefix='fn109-task16-checkpoint.', dir='/tmp'))
    base_data = {}
    with tarfile.open(fileobj=io.BytesIO(archive_data)) as archive:
        seen = set()
        for member in archive.getmembers():
            name = PurePosixPath(member.name)
            require(not name.is_absolute() and '..' not in name.parts, 'unsafe archive member')
            require(member.name in ('tools', MODULE) or member.name.startswith(MODULE + '/'), 'archive member outside module')
            require(member.name not in seen and (member.isdir() or member.isfile()), 'duplicate or nonregular archive member')
            seen.add(member.name)
            destination = scratch / member.name
            if member.isdir():
                destination.mkdir(parents=True, exist_ok=True)
                continue
            data = archive.extractfile(member).read()
            require(len(data) == member.size, 'truncated archived source')
            base_data[member.name] = data
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)
            destination.chmod(member.mode & 0o777)
    before = {path: sha(data) for path, data in base_data.items()}
    require(inventory(scratch) == before, 'archive bytes differ after extraction')
    for name, identity in evidence['source_baseline_sha256'].items():
        require(before[EXECUTION + name] == identity, 'production predecessor mismatch: ' + name)
    test = EXECUTION + 'simulation_progress_test.go'
    fixture = EXECUTION + 'simulation_progress_fixture_test.go'
    require(before[test] == preserved['before_test_sha256'], 'task15 characterization predecessor mismatch')
    require(before[fixture] == preserved['before_fixture_sha256'], 'task15 fixture predecessor mismatch')
    require((ARTIFACTS / 'simulation_progress_test.before.txt').read_bytes() == base_data[test], 'retained characterization preimage differs from committed task15')
    reversed_fixture = source_data[fixture]
    for original, replacement in REVERSALS:
        require(reversed_fixture.count(original) == 1, 'fixture callback is not unique')
        reversed_fixture = reversed_fixture.replace(original, replacement)
    require(reversed_fixture == base_data[fixture], 'fixture changes exceed exactly two callback substitutions')
    for path, data in source_data.items():
        destination = scratch / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(data)
    after = inventory(scratch)
    unchanged_model = EXECUTION + 'simulation_model.go'
    changed = sorted(path for path in set(before) | set(after) if before.get(path) != after.get(path))
    require(len(changed) == 9 and set(changed) == set(expected) - {unchanged_model}, 'delta exceeds exact nine task16 source paths')
    require(all(after[path] == identity for path, identity in expected.items()), 'final source mismatch')
    require(before[unchanged_model] == after[unchanged_model], 'model transport changed')
    require(before[MODULE + '/architecture_test.go'] == after[MODULE + '/architecture_test.go'], 'later architecture imported')
    module_manifests = {
        'before': new_output('checkpoint-module-before.log', manifest(before)),
        'after': new_output('checkpoint-module-after.log', manifest(after)),
    }
    fixture_diff = ''.join(difflib.unified_diff(base_data[fixture].decode().splitlines(True), source_data[fixture].decode().splitlines(True), fromfile='committed-task15-fixture', tofile='retained-task16-fixture'))
    new_output('checkpoint-fixture-diff.log', fixture_diff.encode())
    support = scratch / 'checkpoint-support'
    support.mkdir()
    (support / 'compare-test-bodies.go').write_bytes(comparator)
    (support / 'task15-characterization.go.txt').write_bytes(base_data[test])
    env = os.environ.copy()
    unset = ['GOMADSEED', 'GOMAD3_CHILD_SEED', 'GOROOT', 'GOEXPERIMENT']
    for key in unset:
        env.pop(key, None)
    env.update(GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2', GOENV='off', GOFLAGS='', GOMAD3_STOCK_GO=str(GO), PATH=str(GO.parent) + os.pathsep + env.get('PATH', ''), TMPDIR=str(scratch / 'tmp'))
    Path(env['TMPDIR']).mkdir()
    observations = []

    def run(label, args, expected_names=None):
        require(inventory(scratch) == after, 'source changed before ' + label)
        argv = [str(GO), *args]
        started = time.monotonic()
        result = subprocess.run(argv, cwd=scratch / MODULE, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=180)
        elapsed = time.monotonic() - started
        log = new_output('checkpoint-' + label + '.log', result.stdout)
        output = result.stdout.decode(errors='replace')
        row = {'label': label, 'argv': argv, 'command': shlex.join(argv), 'cwd': str(scratch / MODULE), 'environment': {key: env[key] for key in ('GOWORK', 'GOTOOLCHAIN', 'GOMAXPROCS', 'GOENV', 'GOFLAGS', 'GOMAD3_STOCK_GO', 'PATH', 'TMPDIR')}, 'unset': unset, 'exit_code': result.returncode, 'elapsed_seconds': round(elapsed, 6), 'log': log, 'source_manifest_sha256': module_manifests['after']['sha256']}
        names = re.findall(r'^Test\w+$', output, re.MULTILINE)
        if '-list' in args:
            row['selected_tests'] = names
            require(names, 'selector matched no tests')
        if expected_names is not None:
            actual_names = re.findall(r'^--- PASS: (Test\w+) \(', output, re.MULTILINE)
            require(set(actual_names) == set(expected_names), 'executed tests differ from listing')
            row['passed_top_level_tests'] = actual_names
        observations.append(row)
        print(json.dumps({'label': label, 'exit_code': result.returncode, 'elapsed_seconds': row['elapsed_seconds']}), flush=True)
        require(result.returncode == 0, 'checkpoint command failed: ' + label + '\n' + output)
        require(inventory(scratch) == after, 'source changed during ' + label)
        return output, names

    run('host', ['env', 'GOVERSION', 'GOOS', 'GOARCH', 'GOROOT'])
    body_output, _ = run('body-preservation', ['run', str(support / 'compare-test-bodies.go'), str(support / 'task15-characterization.go.txt'), str(scratch / test)])
    bodies = json.loads(body_output)
    require(len(bodies) == 11, 'characterization count changed')
    unchanged_bodies = [name for name, row in bodies.items() if row['body_byte_identical']]
    strengthened = [name for name, row in bodies.items() if not row['body_byte_identical']]
    require(len(unchanged_bodies) == 9 and set(strengthened) == set(preserved['allowed_changes']), 'unexpected test-body change')
    require(bodies == json.loads((ARTIFACTS / preserved['comparison']).read_bytes()), 'fresh body comparison differs from original retained comparison')
    _, selected = run('focused-list', ['test', '-count=1', '-tags', 'test_dep', './runner/internal/execution', '-list', FOCUSED])
    require(set(bodies) <= set(selected), 'characterization coverage missing from selector')
    lifecycle = re.findall(r'^func (Test\w+)\(', source_data[EXECUTION + 'simulation_progress_lifecycle_test.go'].decode(), re.MULTILINE)
    require(len(lifecycle) == 5 and set(lifecycle) <= set(selected), 'new lifecycle coverage missing')
    run('focused', ['test', '-count=1', '-timeout=60s', '-tags', 'test_dep', './runner/internal/execution', '-run', FOCUSED, '-v'], selected)
    _, architectural = run('architecture-list', ['test', '-count=1', '-tags', 'test_dep', '.', '-list', ARCHITECTURE])
    require(len(architectural) == 9 and 'TestPackageArchitecture' in architectural, 'inherited architecture selector changed')
    run('architecture', ['test', '-count=1', '-timeout=120s', '-tags', 'test_dep', '.', '-run', ARCHITECTURE, '-v'], architectural)
    require(inventory(scratch) == after, 'final complete module inventory changed')
    require(git('rev-parse', 'HEAD').decode().strip() == HEAD, 'HEAD changed')
    require((ROOT / DESIGN).read_bytes() == design_data, 'shared design changed')
    require(all((ROOT / path).read_bytes() == data for path, data in source_data.items()), 'shared task16 source changed')
    retained_names = ['evidence.json', 'source-audit.md', 'handover.md', 'conductor-verification.md', 'regressions-red.log', 'regressions-green.log', 'final-race.log', 'final-repeat.log', 'final-vet.log', 'compare-test-bodies.go', 'test-body-comparison.json', 'checkpoint-preparation-state.json', 'checkpoint-judge.log']
    report = {
        'task': evidence['task'], 'scope': 'bounded checkpoint preparation for conductor-only staging',
        'base_head': HEAD, 'archive_sha256': sha(archive_data), 'scratch': str(scratch), 'module_root': str(scratch / MODULE),
        'helper_sha256': sha(Path(__file__).read_bytes()), 'retained_source_hashes_matched': 10,
        'source_origins': {path: {'input': str(ROOT / path), 'sha256': identity, 'changed_from_predecessor': path in changed} for path, identity in expected.items()},
        'changed_module_paths': changed, 'changed_count': 9, 'unchanged_model_transport_sha256': after[unchanged_model],
        'module_files_before': len(before), 'module_files_after': len(after), 'complete_module_manifests': module_manifests,
        'delta': [{'path': path, 'before_sha256': before.get(path), 'after_sha256': after[path]} for path in changed],
        'design': {'path': DESIGN, 'sha256': sha(design_data), 'matches_committed_task15': True, 'unchanged': True},
        'characterization_preservation': {'before_test_sha256': before[test], 'after_test_sha256': after[test], 'unchanged_valid_bodies': unchanged_bodies, 'strengthened_negative_bodies': strengthened, 'fresh_comparison_matches_original': True, 'before_fixture_sha256': before[fixture], 'after_fixture_sha256': after[fixture], 'only_two_fixture_callback_substitutions': True},
        'observations': observations, 'full_module_unchanged_after_each_check': True,
        'original_evidence': {name: sha((ARTIFACTS / name).read_bytes()) for name in retained_names},
        'original_source_audit': 'Two independent same-family actual-source audits found no defects in this exact retained candidate. Audits are source evidence, not formal implementation-review receipts.',
        'model_routing': {'tier_judgments': 1, 'argument_validation_failures': 1, 'judge_reason': 'no_key', 'assigned_model': 'gpt-6.1-sol', 'assigned_effort': 'high', 'actual_model_metadata': 'unknown', 'original_writer_and_reviewers_same_family': True, 'agents_or_bridges_spawned': 0},
        'rerun_scope': 'Fresh focused source tests and inherited architecture only. Retained RED/GREEN, race, 100-repeat and vet evidence reused because inputs match and no new concurrency or source discrepancy arose. Known stock broad Simulation child-exit49 and unavailable native Quick commands were not retried.',
        'native_acceptance': False, 'native_platforms_open': ['darwin/arm64', 'linux/amd64'], 'native_gaps': evidence['native_gaps'], 'lint_gap': evidence['lint_gap'],
        'formal_ship_verdict': False, 'task_completion': False, 'shared_source_writes': False, 'git_index_writes': False, 'git_history_writes': False, 'flow_state_writes': False, 'commits': [], 'prs': [],
    }
    output = new_output('checkpoint-verification.json', (json.dumps(report, indent=2) + '\n').encode())
    print(json.dumps({'scratch': str(scratch), 'changed_count': 9, 'retained_source_hashes_matched': 10, 'focused_tests': len(selected), 'architecture_tests': len(architectural), 'report': output}), flush=True)


if __name__ == '__main__':
    main()
