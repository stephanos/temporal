#!/usr/bin/env python3
"""Verify the retained task-15 checkpoint on an isolated committed module."""
import difflib
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import tarfile
import tempfile

ROOT = Path('/Users/stephan/Workspace/skunkworks/gomad/temporal')
ARTIFACTS = Path(__file__).resolve().parent
HEAD = '5350185a3601921c0a5f9ba07e1f05bdad7df81f'
GO = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go')
PREFIX = 'tools/gomad3/'
TEST = PREFIX + 'runner/internal/execution/simulation_progress_test.go'
FIXTURE = PREFIX + 'runner/internal/execution/simulation_progress_fixture_test.go'
DESIGN = '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md'
EXPECTED = {
    TEST: '4d91f01eb5e378e5aa4824c2af655862d9fe0fe57772bd74f2dfa648418c0880',
    FIXTURE: '35ed448549b3aa5d6ce959d86a631b37979056642144e31274e18bcbfb0e8e5c',
    DESIGN: '1f2fc94d417ad3ffe2d64a2b255787d3ad74e13701bc85a7294522a0629a60c9',
}
CURRENT_FIXTURE = '12cbd8911c03b5ed4e0717397fec2742508daab3e28b8e91a6ca376671db8add'
RETAINED = {
    TEST: ROOT / '.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-16/simulation_progress_test.before.txt',
    FIXTURE: Path('/tmp/fn109-task15-preimage.B9MU5kXH') / FIXTURE,
}
REVERSALS = [
    (b'_ = coordinator.time.progress.apply(modelRequestDispatched{coordinator: coordinator.coordinator})',
     b'coordinator.time.deliverExternal(coordinator.coordinator)'),
    (b'coordinator.time.progress.apply(modelAbandonedResponseDiscarded{coordinator: coordinator.coordinator, arrivals: frame.Arrivals})',
     b'coordinator.time.acknowledgeExternal(coordinator.coordinator, frame.Arrivals)'),
]
ARCHITECTURE = '^Test(PackageArchitecture|PublicPackagesDoNotExportTypeAliases|RunnerExecutionInjectionIsPrivate|RunnerRequestsCompileInExternalModule|CurrentVocabularyHasNoLegacyCampaignBoundary|MakeTargetsMatchTheirOwnership|ExactModuleEdges|CapabilityEvaluationHasNoHostEffect|DomainModulesDoNotExportWireFraming)$'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*arguments):
    return subprocess.check_output(['git', '-C', str(ROOT), *arguments])


def inventory(directory):
    found = {}
    for path in sorted(directory.rglob('*')):
        if path.is_symlink():
            raise ValueError('unexpected scratch symlink: ' + str(path))
        if path.is_file():
            found[path.relative_to(directory).as_posix()] = sha(path.read_bytes())
    return found


def main():
    assert git('rev-parse', 'HEAD').decode().strip() == HEAD
    evidence = json.loads((ARTIFACTS / 'evidence.json').read_text())
    design_data = (ROOT / DESIGN).read_bytes()
    assert sha(design_data) == EXPECTED[DESIGN]
    overlays = {path: source.read_bytes() for path, source in RETAINED.items()}
    for path, data in overlays.items():
        assert sha(data) == EXPECTED[path], path
    current = (ROOT / FIXTURE).read_bytes()
    assert sha(current) == CURRENT_FIXTURE
    reversed_data = current
    for before, after in REVERSALS:
        assert reversed_data.count(before) == 1
        reversed_data = reversed_data.replace(before, after)
    assert reversed_data == overlays[FIXTURE]
    fixture_diff = ''.join(difflib.unified_diff(
        overlays[FIXTURE].decode().splitlines(True), current.decode().splitlines(True),
        fromfile='retained-task15-fixture', tofile='current-task16-fixture'))
    (ARTIFACTS / 'checkpoint-fixture-diff.log').write_text(fixture_diff)
    archive_data = git('archive', '--format=tar', HEAD, 'tools/gomad3')
    scratch = Path(tempfile.mkdtemp(prefix='fn109-task15-checkpoint.'))
    baseline = {}
    with tarfile.open(fileobj=io.BytesIO(archive_data), mode='r:') as archive:
        members = archive.getmembers()
        for member in members:
            name = PurePosixPath(member.name)
            assert not name.is_absolute() and '..' not in name.parts
            assert member.name == 'tools' or member.name == 'tools/gomad3' or member.name.startswith(PREFIX)
            assert member.isdir() or member.isreg(), (member.name, member.type)
        for member in members:
            destination = scratch / member.name
            if member.isdir():
                destination.mkdir(parents=True, exist_ok=True)
                continue
            assert member.name not in baseline
            stream = archive.extractfile(member)
            assert stream is not None
            data = stream.read()
            assert len(data) == member.size
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)
            destination.chmod(member.mode & 0o777)
            baseline[member.name] = sha(data)
    assert inventory(scratch) == baseline
    assert not (scratch / PREFIX / 'runner/internal/execution/simulation_progress.go').exists()
    assert not (scratch / PREFIX / 'internal/gomadtool/architecture').exists()
    production = {}
    for name, expected in evidence['production_final_sha256'].items():
        path = PREFIX + 'runner/internal/execution/' + name
        assert baseline[path] == expected, path
        production[path] = expected
    for path, data in overlays.items():
        assert path not in baseline
        (scratch / path).write_bytes(data)
    candidate = inventory(scratch)
    changed = sorted(path for path in baseline.keys() | candidate.keys()
                     if baseline.get(path) != candidate.get(path))
    assert changed == sorted(overlays), changed
    env = os.environ.copy()
    for name in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
        env.pop(name, None)
    env.update(GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2',
               PATH=str(GO.parent) + os.pathsep + env.get('PATH', ''),
               GOMAD3_STOCK_GO=str(GO))
    module = scratch / 'tools/gomad3'
    observations = []

    def run(label, arguments, expected_count=None):
        command = [str(GO), *arguments]
        completed = subprocess.run(command, cwd=module, env=env,
                                   stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                   timeout=600)
        output = completed.stdout.decode(errors='replace')
        log = ARTIFACTS / ('checkpoint-' + label + '.log')
        log.write_text(output)
        observation = dict(label=label, command=command, cwd=str(module),
                           environment={k: env[k] for k in ('GOWORK', 'GOTOOLCHAIN', 'GOMAXPROCS', 'PATH', 'GOMAD3_STOCK_GO')},
                           unset=['GOMADSEED', 'GOMAD3_CHILD_SEED'],
                           exit_code=completed.returncode, log=str(log), log_sha256=sha(completed.stdout))
        if expected_count is not None:
            names = re.findall(r'^Test\w+$', output, re.MULTILINE)
            assert len(names) == expected_count, names
            observation['selected_tests'] = names
        observations.append(observation)
        print(label + ': exit ' + str(completed.returncode), flush=True)
        assert completed.returncode == 0, output
        return output

    run('go-version', ['version'])
    run('characterization-list', ['test', '-count=1', '-tags', 'test_dep',
        './runner/internal/execution', '-list', '^TestSimulation(Time|Model)Progress'], 11)
    run('characterization', ['test', '-count=1', '-timeout=60s', '-tags', 'test_dep',
        './runner/internal/execution', '-run', '^TestSimulation(Time|Model)Progress', '-v'])
    run('focused-list', ['test', '-count=1', '-tags', 'test_dep', './runner/internal/execution',
        '-list', 'SimulationTime|SimulationModel|SimulationCoordinator|ServeSimulation'])
    run('focused', ['test', '-count=1', '-timeout=60s', '-tags', 'test_dep',
        './runner/internal/execution', '-run', 'SimulationTime|SimulationModel|SimulationCoordinator|ServeSimulation', '-v'])
    run('architecture-list', ['test', '-count=1', '-tags', 'test_dep', '.', '-list', ARCHITECTURE], 9)
    run('architecture', ['test', '-count=1', '-timeout=120s', '-tags', 'test_dep', '.', '-run', ARCHITECTURE, '-v'])
    formatted = subprocess.check_output([str(GO.parent / 'gofmt'), '-l', *(str(scratch / p) for p in overlays)])
    assert formatted == b'', formatted
    (ARTIFACTS / 'checkpoint-gofmt.log').write_bytes(formatted)
    assert inventory(scratch) == candidate
    assert sha((ROOT / DESIGN).read_bytes()) == EXPECTED[DESIGN]
    assert sha((ROOT / FIXTURE).read_bytes()) == CURRENT_FIXTURE
    assert git('rev-parse', 'HEAD').decode().strip() == HEAD
    manifest = dict(base_head=HEAD, archived_prefix='tools/gomad3',
                    archive_sha256=sha(archive_data), scratch=str(scratch),
                    archived_files=len(baseline), baseline_sha256=baseline,
                    candidate_sha256=candidate, changed_module_paths=changed,
                    retained_blobs={p: dict(source=str(RETAINED[p]), sha256=EXPECTED[p], scratch=str(scratch / p)) for p in overlays},
                    production_sha256=production,
                    design=dict(path=DESIGN, sha256=EXPECTED[DESIGN], mutation=False),
                    independent_fixture_recovery=dict(current_sha256=CURRENT_FIXTURE,
                        recovered_sha256=EXPECTED[FIXTURE], reversals=[dict(before=a.decode(), after=b.decode()) for a,b in REVERSALS]),
                    observations=observations, full_module_unchanged_after_checks=True,
                    model_routing='Previously judged assignment; no_key; explicit gpt-6.1-sol high; same family; actual backend model unknown; no rejudge.',
                    shared_source_writes=False, git_index_writes=False, flow_mutations=False,
                    native_acceptance=False, formal_ship_verdict=False,
                    unrun_native_quick_commands=evidence['native_gaps'],
                    race_rerun='No concrete need; exact test blobs retain prior bounded race/100-repeat/vet evidence.',
                    broad_simulation='Known pre-edit child-exit-49 developmental failures retained; not retried.')
    destination = ARTIFACTS / 'checkpoint-verification.json'
    destination.write_text(json.dumps(manifest, indent=2) + '\n')
    print(json.dumps(dict(report=str(destination), report_sha256=sha(destination.read_bytes()),
                          scratch=str(scratch), changed_module_paths=changed)))


if __name__ == '__main__':
    main()
