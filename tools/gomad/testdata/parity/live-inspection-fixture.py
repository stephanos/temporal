#!/usr/bin/env python3
"""Prepare exact retained inspection cells using unchanged Go fixture sources."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'compiler', 'overlay', 'source', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--runtime-identity', required=True)
    args = parser.parse_args()
    for name in ('binary', 'compiler', 'overlay', 'source', 'work'):
        setattr(args, name, getattr(args, name).resolve())
    args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
    commands = []
    env = {key: value for key, value in os.environ.items()
           if not key.startswith('GOMAD_') and key not in ('GOFLAGS', 'GOROOT', 'GOEXPERIMENT')}
    env.update(GOOS='wasip1', GOARCH='wasm', GOEXPERIMENT='nogreenteagc', GOWORK='off',
               GOENV='off', GOTOOLCHAIN='local', CGO_ENABLED='0', GOMAXPROCS='1')

    def run(name, command, status=0, timeout=600):
        result = subprocess.run(list(map(str, command)), env=env, cwd=args.source,
                                capture_output=True, timeout=timeout)
        commands.append(dict(name=name, argv=list(map(str, command)), status=result.returncode))
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        (args.work / 'commands.json').write_text(json.dumps(commands, indent=2) + '\n')
        assert result.returncode == status, (name, result.returncode, result.stderr)
        return result.stdout

    fixture = args.source / 'common/gomaddebug/testdata/fixture/main.go'
    originals = {'fixture': sha(fixture.read_bytes()), 'compiler': sha(args.compiler.read_bytes()),
                 'overlay': sha(args.overlay.read_bytes()), 'runtime': args.runtime_identity}
    # This is the exact second source file created by the original network root.
    network_source = Path(__file__).with_name('targets').joinpath('live-inspection-network.go.txt').read_bytes()
    originals['network'] = sha(network_source)
    preparation = args.work / 'preparation'
    preparation.mkdir(mode=0o700)
    modules = {}
    for name in ('inspection', 'network'):
        source = preparation / name
        source.mkdir(mode=0o700)
        shutil.copyfile(fixture, source / 'main.go')
        if name == 'network':
            (source / 'network.go').write_bytes(network_source)
        module = preparation / (name + '.wasm')
        run('compile-' + name, [args.compiler, 'build', '-p=1', '-trimpath', '-buildvcs=false',
                                '-tags=gomad,test_dep', '-overlay', args.overlay, '-o', module,
                                *sorted(source.glob('*.go'))])
        modules[name] = module
        shutil.rmtree(source)
    runtime = args.runtime_identity
    limits = dict(fuel=1 << 60, memory_bytes=256 << 20, wall_millis=60000,
                  output_bytes=1 << 20, evidence_bytes=(8 << 20) + (1 << 20) + 2 * (1 << 20) + (64 << 20))
    model = dict(entropy_key=[0] * 32, working_directory_sha256=sha(b'/workspace'),
                 arguments=dict(bytes=0, count=0, chunks=0), environment=dict(bytes=0, count=0, chunks=0),
                 mode=1, limit=1 << 20, diagnostics=False, epoch_nanos=946684800000000000,
                 read_step_nanos=0, now=1, limits=dict(calls=10000, transcript_bytes=8 << 20, pending_events=4096))
    namespace = dict(entries=[], writable_directories=[list(b'/workspace'), list(b'/tmp')],
                     preopens=[list(b'/tmp'), list(b'/workspace')], working_directory=list(b'/workspace'), stdin=[],
                     limits=dict(files=1000, descriptors=64, filesystem_bytes=8 << 20, output_bytes=1 << 20),
                     epoch_nanos=946684800000000000)
    network = dict(profile='gomad.simulation-network-stream/v3',
                   limits=dict(listeners=8, connections=8, bytes=8 << 20, connection_bytes=4 << 20,
                               stall_nanos=10000000000, transitions=8192), faults=[])
    artifacts = {}
    for name in ('standalone', 'network', 'restart'):
        module = modules['network' if name == 'network' else 'inspection']
        digest = sha(module.read_bytes())
        build = sha(canonical(dict(inputs=originals, module=digest)))
        identity = dict(TargetSHA256=list(bytes.fromhex(digest)), ToolchainBuildKey=build,
                        GOOS='wasip1', GOARCH='wasm', ImplementationSHA256=list(bytes.fromhex(sha(runtime.encode()))))
        config = dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',
                      profile='gomad.wasi-cooperative/v1', limits=copy.deepcopy(limits),
                      arguments=['main'], environment={'PWD': '/workspace'}, runtime_sha256=runtime,
                      model=copy.deepcopy(model), namespace=copy.deepcopy(namespace), storage=None)
        if name == 'standalone':
            # The fixture case seed is the execution seed; the model must not repeat it.
            request = dict(schema='gomad.replay-fixture-input/v1', cases=[dict(
                name='retained-inspection', module_path=str(module), module_sha256=digest,
                toolchain_build_key=build, seed=7, configuration=config, choice_identity=identity)])
            command = 'fixture-replay-input'
        else:
            handles = [('a', 1), ('b', 1)] + ([('a', 2)] if name == 'restart' else [])
            incarnations = []
            for node, incarnation in handles:
                item = copy.deepcopy(config)
                for key in ('engine', 'profile'):
                    item.pop(key)
                item.update(handle=dict(node=node, incarnation=incarnation),
                            module=dict(file='target/module.wasm', sha256=digest,
                                        bytes=module.stat().st_size, limit=512 << 20),
                            choice_identity=identity, network=network if name == 'network' else None)
                incarnations.append(item)
            def step(action, node, incarnation=1):
                return dict(action=action, handle=dict(node=node, incarnation=incarnation))
            steps = ([step('start', 'a'), step('wait', 'a'), step('start', 'b'), step('wait', 'b')]
                     if name == 'network' else [step('start', 'a'), step('start', 'b'), step('wait', 'a'),
                     step('wait', 'b'), step('restart', 'a', 2), step('wait', 'a', 2)])
            cluster = dict(engine=config['engine'], profile=config['profile'], limits=limits,
                           nodes=['a', 'b'], incarnations=incarnations, steps=steps, boundaries=[], crash_groups=[],
                           broker=dict(seed=7, pause_on_settle=False, limits=dict(max_requests=2 << 20,
                               max_events=2 << 20, max_queued_events=1 << 16, max_transitions=8 << 20,
                               max_payload_bytes=64 << 20, max_string_bytes=1024)),
                           network=dict(model=network, addresses={'a': '10.0.0.1', 'b': '10.0.0.2'},
                                        links=[dict(**{'from': a, 'to': b}, enabled=True)
                                               for a, b in [('a', 'b'), ('b', 'a')]],
                                        strict_topology=True, faults=[]) if name == 'network' else None)
            request = dict(schema='gomad.cluster-fixture-input/v1', toolchain_sha256=build,
                           configuration=cluster, modules={'target/module.wasm': str(module)})
            command = 'fixture-replay-cluster-input'
        path = args.work / (name + '-input.json')
        path.write_bytes(canonical(request))
        destination = args.work / name
        bundle = json.loads(run('record-' + name, [args.binary, command, '--root', destination, '--input', path],
                                timeout=300))
        assert len(bundle['records']) == 1
        artifacts[name] = str(destination / bundle['records'][0]['artifact'])
    # Retained executables must be sufficient after every private preparation input is gone.
    shutil.rmtree(preparation)
    (args.work / 'inputs.json').write_text(json.dumps(dict(schema='gomad.live-inspection-inputs/v1',
        origins=originals, artifacts=artifacts, detached_preparation=str(preparation)), indent=2) + '\n')
    assert sha(fixture.read_bytes()) == originals['fixture']


if __name__ == '__main__':
    main()
