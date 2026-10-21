#!/usr/bin/env python3
"""Verify public minimizers refuse genuine replayable cluster failures."""
import argparse
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent


def load(name, path):
    loader = SourceFileLoader(name, str(path))
    module = module_from_spec(spec_from_loader(name, loader))
    loader.exec_module(module)
    return module


check = load('cluster_refusal_check', CORPUS / 'campaign-check.py')
reference = load('cluster_refusal_reference', CORPUS / 'campaign-reference-check.py')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'baseline', 'source', 'go-compiler', 'work', 'input', 'preparation-root'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--guest', type=Path)
    parser.add_argument('--go-installation', type=Path)
    args = parser.parse_args()
    for name, value in vars(args).items():
        if value is not None:
            setattr(args, name, value.resolve())
    guest = args.guest or args.input.parent / 'guest'
    go_installation = args.go_installation or args.input.parent / 'go-installation.json'
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    binary_hash = check.parity.digest(args.binary.read_bytes())
    environment = {key: value for key, value in os.environ.items()
                   if key not in ('GOROOT', 'GOFLAGS', 'GOMOD', 'GOPATH', 'GOEXPERIMENT')
                   and not key.startswith('GOMAD_')}
    environment.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', CGO_ENABLED='0',
                       GOPROXY='off', GOSUMDB='off')
    commands = []

    def run(name, command, status=0):
        command = list(map(str, command))
        result = subprocess.run(command, env=environment, capture_output=True, timeout=600)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        commands.append(dict(name=name, argv=command, status=result.returncode, expected=status,
                             stdout_sha256=check.parity.digest(result.stdout),
                             stderr_sha256=check.parity.digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(check.parity.canonical(commands) + b'\n')
        check.equal(result.returncode, status, name + ': ' + result.stderr.decode(errors='replace'))
        return result.stdout

    module = args.source / 'tools/gomad'
    scenario = json.loads((module / 'internal/backends/wasm/command/testdata/scenario/deadlock.json').read_bytes())
    scenario['spec']['nodes'] = [dict(id='a', boot='test', address='10.0.0.1')]
    scenario['spec']['links'] = []
    scenario['boots'] = [dict(id='test', args=['-test.run=^TestFailure$', '-test.v'])]
    scenario['actions'] = [dict(action=action, handle=dict(node='a', incarnation=1)) for action in ('start', 'wait')]
    scenario_path = args.work / 'scenario.json'
    scenario_path.write_bytes(check.parity.canonical(scenario))

    identity_root = args.work / 'identity'
    bundle = json.loads(run('rust-identity', [args.binary, 'fixture-replay-input', '--root', identity_root,
                                            '--input', args.input]))
    check.parity.authenticate(identity_root, bundle)
    identity_artifact = next(identity_root / row['artifact'] for row in bundle['records']
                            if check.parity.read(identity_root / row['artifact'] / 'manifest.json')['record']['profile']
                            == 'gomad.wasi-cooperative/v1')
    inputs = check.canonical_file(identity_artifact / 'evidence/replay.json')
    configuration = inputs['configuration']
    installation = args.work / 'rust-installation.json'
    installation.write_bytes(check.parity.canonical(dict(
        schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
        compiler_sha256=check.parity.digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
        runtime_root=str(module / 'internal/model/runtimesource'), cache_root=str(args.preparation_root),
        profile='gomad.wasi-cooperative/v1', fuel=configuration['limits']['fuel'],
        memory_bytes=configuration['limits']['memory_bytes'], helper_sha256=inputs['helper_sha256'],
        host_sha256=inputs['host_sha256'], runtime_implementation_sha256=inputs['runtime_implementation_sha256'],
        model=configuration['model'], namespace=configuration['namespace'], storage=configuration.get('storage'),
        network=None, build_tags=[])))
    rows = []
    for lineage, binary, selected in [('go', args.baseline / 'bin/gomad', go_installation),
                                      ('rust', args.binary, installation)]:
        destination = args.work / (lineage + '-parents')
        prefix = [binary, '--installation', selected]
        run(lineage + '-explore', [*prefix, 'explore', '--json', '--scenario', scenario_path,
            '--seeds=7', '--parallel=1', '--on-failure=all', '--execution-timeout=30s', '--overall-timeout=5m',
            '--artifacts', destination, '--working-dir', guest, 'go-test', '.'], 1)
        if lineage == 'go':
            artifacts = [path.parent for path in destination.rglob('manifest.json')
                         if json.loads(path.read_bytes()).get('artifact_kind') == 'gomad.target-failure/v1']
            check.equal(len(artifacts), 1, 'go.cluster_failure_count')
            manifest = reference.authenticate_artifact(artifacts[0])
            check.equal(manifest['replay_mode'], 'exact', 'go.cluster_replay_mode')
            check.equal(manifest['target']['backend']['evidence']['schema'],
                        'gomad.wasm-simulation-retained/v2', 'go.cluster_evidence_schema')
        else:
            artifacts = [path.parent for path in destination.rglob('manifest.json')
                         if json.loads(path.read_bytes())['record']['outcome']['class'] == 'target_failure']
            check.equal(len(artifacts), 1, 'rust.cluster_failure_count')
            check.authenticate_artifact(artifacts[0])
            check.equal((artifacts[0] / 'evidence/cluster-replay.json').is_file(), True, 'rust.cluster_inputs')
        artifact = artifacts[0]
        run(lineage + '-inspect', [binary, 'inspect', '--json', artifact])
        replay = run(lineage + '-replay', [*prefix, 'replay', artifact], 1)
        check.equal(b'reproduced=true' in replay, True, lineage + '.exact_reproduction')
        if lineage == 'rust':
            check.equal(b'choice-replay=exact' in replay, True, 'rust.exact_incarnation_choices')
        run(lineage + '-minimize', [*prefix, 'minimize', '--json', '--artifacts',
                                  args.work / (lineage + '-minimized'), artifact], 3)
        rows.append(dict(lineage=lineage, artifact=str(artifact), replay_status=1, minimize_status=3,
                         refusal=(args.work / (lineage + '-minimize.stderr')).read_text().strip()))
    check.equal(check.parity.digest(args.binary.read_bytes()), binary_hash, 'Rust binary unchanged')
    report = dict(schema='gomad.minimize-cluster-refusal/v1', results=rows,
                  driver_sha256=check.parity.digest(Path(__file__).read_bytes()),
                  rust_binary_sha256=binary_hash,
                  go_binary_sha256=check.parity.digest((args.baseline / 'bin/gomad').read_bytes()))
    (args.work / 'minimize-cluster-refusal-result.json').write_bytes(check.parity.canonical(report) + b'\n')
    print('public Go/Rust cluster failures authenticated, exactly replayed, and refused by minimize')


if __name__ == '__main__':
    main()
