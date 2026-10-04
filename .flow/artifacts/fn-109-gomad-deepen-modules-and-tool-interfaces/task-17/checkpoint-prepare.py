#!/usr/bin/env python3
"""Prepare and check exact historical task-17 source in isolated scratch."""
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
OUT = Path(__file__).resolve().parent
HEAD = 'add55cf6116fbed23e8c4051e120521b2f3f033c'
GO = Path('/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go')
MANIFEST = OUT / 'final-source-post-correction.sha256'
MANIFEST_SHA = '1692362f697d85d42aeabf6b96e878e77245c03848280e0ac7b72cf6bd487095'
PREFIX = 'tools/gomad3/'
PREIMAGES = {
    PREFIX + 'runner/internal/execution/simulation_root_integration_test.go': Path('/tmp/fn109-task17-preimage.yZkyiGoM/tools/gomad3/runner/internal/execution/simulation_root_integration_test.go'),
    PREFIX + 'toolchain/version/version.json': Path('/tmp/gomad-task18.U1uS6Z/baseline-version/version.json'),
    PREFIX + 'Makefile': Path('/tmp/gomad-task18.U1uS6Z/Makefile'),
    PREFIX + 'simulation_gate_selection_test.go': Path('/tmp/gomad-task18.U1uS6Z/simulation_gate_selection_test.go'),
}
GENERATED = [PREFIX + p for p in (
    'choice/internal/wire/wire_generated.go', 'target/internal/livecap/protocol_generated.go',
    'toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go',
    'toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go')]
ARCHITECTURE = 'PackageArchitecture|PublicPackagesDoNotExportTypeAliases|RunnerExecutionInjectionIsPrivate|RunnerRequestsCompileInExternalModule|CurrentVocabularyHasNoLegacyCampaignBoundary|MakeTargetsMatchTheirOwnership|ExactModuleEdges|CapabilityEvaluationHasNoHostEffect|DomainModulesDoNotExportWireFraming'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def git(*arguments):
    return subprocess.check_output(['git', '-C', str(ROOT), *arguments])


def inventory(directory):
    found = {}
    for path in sorted(directory.rglob('*')):
        if '.toolchain' in path.relative_to(directory).parts:
            continue
        assert not path.is_symlink(), str(path)
        if path.is_file():
            found[path.relative_to(directory).as_posix()] = sha(path.read_bytes())
    return found


def extract(archive_data, scratch):
    baseline = {}
    with tarfile.open(fileobj=io.BytesIO(archive_data), mode='r:') as archive:
        members = archive.getmembers()
        for member in members:
            name = PurePosixPath(member.name)
            assert not name.is_absolute() and '..' not in name.parts
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
    return baseline


def main():
    assert git('rev-parse', 'HEAD').decode().strip() == HEAD
    assert sha(MANIFEST.read_bytes()) == MANIFEST_SHA
    entries = {}
    for line in MANIFEST.read_text().splitlines():
        digest, path = line.split(None, 1)
        entries[path] = digest
    assert len(entries) == 17
    blobs = {}
    origins = {}
    for path, expected in entries.items():
        origin = PREIMAGES.get(path, ROOT / path)
        blobs[path] = origin.read_bytes()
        assert sha(blobs[path]) == expected, (path, str(origin))
        origins[path] = dict(path=str(origin), sha256=expected,
                             kind='retained preimage' if path in PREIMAGES else 'verified working tree')
    runner = PREFIX + 'runner/internal/execution/simulation_root_integration_test.go'
    current_runner = (ROOT / runner).read_bytes()
    assert sha(current_runner) == '02544185057eff83b65b288b6062c374376b3a10137dc526405c222e052ec682'
    recovered = current_runner
    for suffix in ('PartialIOAndLifetime', 'DirectoryAndChdir', 'AccessAndBounds', 'ReplayRejectsWriteBeforeMutation'):
        line = ('\t\t"TestProcessFilesystemHandle' + suffix + '",\n').encode()
        assert recovered.count(line) == 1
        recovered = recovered.replace(line, b'')
    assert recovered == blobs[runner]
    selected = ['tools/gomad3', 'go.mod', 'go.sum', 'tools/gomad3sim',
                'tools/gomad3integration/qualification/tests.generator.json',
                'tools/gomad3integration/qualification/tests.json']
    archive_data = git('archive', '--format=tar', HEAD, *selected)
    scratch = Path(tempfile.mkdtemp(prefix='fn109-task17-checkpoint.'))
    baseline = extract(archive_data, scratch)
    for path in GENERATED:
        assert baseline[path] == entries[path], path
    assert not (scratch / PREFIX / 'internal/gomadtool/architecture').exists()
    env = os.environ.copy()
    for name in ('GOMADSEED', 'GOMAD3_CHILD_SEED'):
        env.pop(name, None)
    env.update(GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2',
               GOMAD3_STOCK_GO=str(GO),
               PATH=str(GO.parent) + os.pathsep + env.get('PATH', ''))
    module = scratch / 'tools/gomad3'
    observations = []

    def run(label, command, expect=True):
        completed = subprocess.run(command, cwd=module, env=env,
                                   stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=600)
        log = OUT / ('checkpoint-' + label + '.log')
        log.write_bytes(completed.stdout)
        output = completed.stdout.decode(errors='replace')
        observations.append(dict(label=label, command=command, cwd=str(module),
            exit_code=completed.returncode, log=str(log), log_sha256=sha(completed.stdout),
            selected_tests=re.findall(r'^Test\w+$', output, re.MULTILINE),
            environment={k: env[k] for k in ('GOWORK', 'GOTOOLCHAIN', 'GOMAXPROCS', 'PATH', 'GOMAD3_STOCK_GO')},
            unset=['GOMADSEED', 'GOMAD3_CHILD_SEED']))
        print(label + ': exit ' + str(completed.returncode), flush=True)
        if expect:
            assert completed.returncode == 0, output
        return completed.returncode, output

    run('go-version', [str(GO), 'version'])
    baseline_validation = run('baseline-validate', ['make', 'validate'], expect=False)
    assert inventory(scratch) == baseline
    for path, data in blobs.items():
        destination = scratch / path
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(data)
    candidate = inventory(scratch)
    changed = sorted(p for p in baseline.keys() | candidate.keys() if baseline.get(p) != candidate.get(p))
    assert len(changed) == 13, changed
    assert set(changed) == set(entries) - set(GENERATED)
    assert len([p for p in changed if p.startswith('tools/gomad3sim/')]) == 2
    test_filter = '^Test(' + ARCHITECTURE + '|NetworkHandlesOwnOneImplementation|SimulationGateSelectsProcessNetworkHandles|ProcessCommands.*)$'
    run('ownership-gate-architecture-list', [str(GO), 'test', '-count=1', '-tags', 'test_dep', '.', '-list', test_filter])
    run('ownership-gate-architecture', [str(GO), 'test', '-count=1', '-timeout=120s', '-tags', 'test_dep', '.', '-run', test_filter, '-v'])
    run('generation-version', [str(GO), 'test', '-count=1', '-timeout=120s', '-tags', 'test_dep', './internal/gomadtool/generation/...', './toolchain/version', '-v'])
    run('version-check', [str(GO), 'run', './cmd/gomadtool', 'version-generate', '-check'])
    run('protocol-check', [str(GO), 'run', './cmd/gomadtool', 'protocol-generate', '-check'])
    candidate_validation = run('candidate-validate', ['make', 'validate'], expect=False)
    run('owned-root-package-vet', [str(GO), 'vet', '-tags', 'test_dep', '.'])
    go_files = [str(scratch / p) for p in entries if p.endswith('.go')]
    fmt = subprocess.check_output([str(GO.parent / 'gofmt'), '-l', *go_files])
    (OUT / 'checkpoint-gofmt.log').write_bytes(fmt)
    assert fmt == b'', fmt
    assert inventory(scratch) == candidate
    assert git('rev-parse', 'HEAD').decode().strip() == HEAD
    for path, origin in origins.items():
        assert sha(Path(origin['path']).read_bytes()) == entries[path], path
    historical = json.loads((OUT / 'evidence.json').read_text())
    retained = {}
    for command in historical['commands']:
        name = command['name']
        if name in ('generate-validate', 'gate-correction-validate', 'developmental-overlay',
                    'developmental-repeat', 'developmental-race', 'root-developmental-link',
                    'runner-integration-compile', 'gate-selection-red', 'gate-selection-green'):
            log = OUT / command['log']
            retained[name] = dict(command=command['command'], log=str(log),
                                  log_sha256=sha(log.read_bytes()), historical_exit_code=command['exit_code'],
                                  scope=command['scope'])
    for key, binary in (('root-developmental-binary', Path('/tmp/gomad-task17.WFwzYI/root-developmental.test')),
                        ('runner-compile-only-binary', Path('/tmp/gomad-task17.WFwzYI/runner-integration.test'))):
        retained[key] = dict(path=str(binary), sha256=sha(binary.read_bytes()), executed=False)
    report = dict(base_head=HEAD, scratch=str(scratch), selected_archive_paths=selected,
                  archive_sha256=sha(archive_data), historical_manifest_sha256=MANIFEST_SHA,
                  archived_files=len(baseline), baseline_sha256=baseline, candidate_sha256=candidate,
                  historical_source_origins=origins, changed_source_paths=changed,
                  already_committed_generated_paths=GENERATED,
                  full_source_unchanged_after_checks=True,
                  ignored_generated_cache_prefix='tools/gomad3/.toolchain/',
                  observations=observations, retained_original_evidence=retained,
                  baseline_validation_exit=baseline_validation[0], candidate_validation_exit=candidate_validation[0],
                  historical_native_gaps=historical['native_gaps'],
                  native_acceptance=False, formal_ship_verdict=False,
                  root_simulation_evidence='Two exact retained source files and prior developmental link-only evidence; no root workload executed.',
                  routing=dict(judgment='checkpoint-judgment.json', reason='no_key',
                               implementer='gpt-6.1-sol', effort='high', family='same-family Codex', actual_model_metadata='unknown'),
                  shared_source_writes=False, git_index_writes=False, flow_mutations=False)
    destination = OUT / 'checkpoint-verification.json'
    destination.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(dict(report=str(destination), report_sha256=sha(destination.read_bytes()),
                         scratch=str(scratch), changed_source_paths=changed)), flush=True)


if __name__ == '__main__':
    main()
