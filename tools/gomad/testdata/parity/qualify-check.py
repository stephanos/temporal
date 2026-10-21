#!/usr/bin/env python3
"""Probe actual WASM qualification grammar; missing Rust parity fails closed."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import runpy
import subprocess
import sys

sys.dont_write_bytecode = True
CORPUS = Path(__file__).resolve().parent
MODULE = CORPUS.parents[1]


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), allow_nan=False).encode()


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def load(path):
    return json.loads(path.read_bytes())


def equal(actual, expected, path):
    if actual != expected:
        raise ValueError(f'{path}: got {actual!r}, expected {expected!r}')


def prepare(args, run):
    for key in ('input', 'go_compiler', 'go_helper', 'go_source'):
        if getattr(args, key) is None:
            raise ValueError('--prepare-only requires --' + key.replace('_', '-'))
    identity = args.work / 'identity'
    result = run('prepare-identity', [args.binary, 'fixture-replay-input', '--root', identity, '--input', args.input], 0)
    bundle = json.loads(result.stdout)
    artifact = next(identity / row['artifact'] for row in bundle['records']
        if load(identity / row['artifact'] / 'record.json')['profile'] == 'gomad.wasi-cooperative/v1')
    runpy.run_path(str(CORPUS / 'campaign-check.py'))['authenticate_artifact'](artifact)
    replay = load(artifact / 'evidence/replay.json')
    namespace_image = runpy.run_path(str(CORPUS / 'replay-cluster-reference.py'))['namespace_image']
    for cell in ('fixture', 'sqlite', 'onebox'):
        manifest = args.work / ('go-' + cell + '-installation.json')
        command = [args.go_binary, 'install', '--compiler', args.go_compiler, '--helper', args.go_helper,
            '--runtime-root', args.go_source / 'tools/gomad/internal/model/runtimesource',
            '--cache', args.work / ('go-' + cell + '-preparation'), '--output', manifest, '--fuel', str(1 << 60)]
        if cell == 'onebox':
            command += ['--temporal-checkout', args.go_source, '--temporal-package', './tests/gomadfunctional']
        elif cell == 'sqlite':
            config = dict(Profile='gomad.wasi-cooperative/v1', Environment=['PWD=/workspace', 'TZ=UTC'],
                WorkingDirectory='/workspace', WritableDirectories=['/workspace', '/tmp', '/volume'],
                Clock=dict(EpochNanos=946684800000000000),
                Limits=dict(OutputBytes=8 << 20, TranscriptBytes=64 << 20, Calls=1000000,
                    Descriptors=128, Files=1024, FilesystemBytes=8 << 20, PendingEvents=4096),
                storage=dict(volumes=[dict(ID='sqlite', Path='/volume', CapacityBytes=4 << 20)],
                    limits=dict(PendingOperations=10000, Transitions=100000)))
            config_path = args.work / 'go-sqlite-config.json'
            config_path.write_bytes(encode(config))
            command += ['--config', config_path, '--profile', 'gomad.wasi-cooperative/v1']
        else:
            command += ['--profile', 'gomad.wasi-cooperative/v1']
        run('prepare-go-' + cell, command, 0)
        setattr(args, 'go_' + cell + '_installation', manifest)
        options = load(manifest)['options']
        config, bounds = options['Config'], options['Config']['Limits']
        model = copy.deepcopy(replay['configuration']['model'])
        model.pop('seed', None)
        model.update(mode=1, read_step_nanos=config['Clock']['ReadStepNanos'],
            epoch_nanos=config['Clock']['EpochNanos'], working_directory_sha256=digest(config['WorkingDirectory'].encode()),
            limits=dict(calls=bounds['Calls'], transcript_bytes=bounds['TranscriptBytes'], pending_events=bounds['PendingEvents']))
        storage = config.get('storage')
        if storage:
            storage = dict(descriptor=dict(model=storage, return_image=False,
                model_identity='sha256:' + digest(json.dumps(storage, separators=(',', ':')).encode())), image=None, context=None)
        installation = dict(schema='gomad.rust-wasm-installation/v1', compiler_path=str(args.go_compiler),
            compiler_sha256=digest(args.go_compiler.read_bytes()), go_version='go1.27.1',
            runtime_root=str(MODULE / 'internal/model/runtimesource'),
            cache_root=str((args.preparation_root or args.work) / ('rust-' + cell + '-preparation')),
            profile='gomad.wasi-cooperative/v1', fuel=options['Fuel'], memory_bytes=options['MemoryBytes'],
            helper_sha256=replay['helper_sha256'], host_sha256=replay['host_sha256'],
            runtime_implementation_sha256=replay['runtime_implementation_sha256'], model=model,
            namespace=namespace_image(config), storage=storage, network=config.get('network_v2'),
            build_tags=options.get('BuildTags') or [])
        manifest = args.work / ('rust-' + cell + '-installation.json')
        manifest.write_bytes(encode(installation))
        setattr(args, 'rust_' + cell + '_installation', manifest)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'go-binary', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    for name in ('input', 'go-compiler', 'go-helper', 'go-source', 'preparation-root'):
        parser.add_argument('--' + name, type=Path)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument('--prepare-only', action='store_true')
    mode.add_argument('--grammar-only', action='store_true')
    args = parser.parse_args()
    for key, value in vars(args).items():
        if isinstance(value, Path):
            setattr(args, key, value.resolve())
    args.work.mkdir(parents=True, exist_ok=False)
    environment = {key: value for key, value in os.environ.items()
                   if not key.startswith('GOMAD_') and key != 'GOROOT'}
    environment.update(GOENV='off', GOWORK='off', GOTOOLCHAIN='local', GOMAXPROCS='2')
    receipts = []

    def run(name, argv, status):
        result = subprocess.run(list(map(str, argv)), env=environment,
                                capture_output=True, timeout=120)
        (args.work / (name + '.stdout')).write_bytes(result.stdout)
        (args.work / (name + '.stderr')).write_bytes(result.stderr)
        receipts.append(dict(name=name, argv=list(map(str, argv)), status=result.returncode,
            stdout_sha256=digest(result.stdout), stderr_sha256=digest(result.stderr)))
        (args.work / 'commands.json').write_bytes(encode(receipts))
        equal(result.returncode, status, '$.' + name + '.status')
        return result

    if args.prepare_only:
        prepare(args, run)
        print('qualification preparation: six lineage-specific manifests retained; no qualification claim')
        return
    declaration = load(CORPUS / 'qualify-corpus.json')
    equal(declaration['schema'], 'gomad.qualify-corpus/v1', '$.corpus.schema')
    if args.grammar_only:
        for case in declaration['grammar']:
            argv = [str(args.work / 'missing-spec.json') if item == '$MISSING_SPEC' else item
                    for item in case['argv']]
            result = run('go-' + case['name'], [args.go_binary, *argv], case['status'])
            equal(case['stderr_contains'].encode() in result.stderr, True,
                  '$.go.' + case['name'] + '.stderr')
        for lineage, binary in (('go', args.go_binary), ('rust', args.binary)):
            for case in declaration['refusals']:
                result = run(lineage + '-' + case['name'], [binary, *case['argv']], case['status'])
                combined = result.stdout + result.stderr
                equal(any(marker in combined for marker in
                          (b'unknown command', b'unknown gomad command', b'unsupported command')),
                      True, '$.' + lineage + '.' + case['name'] + '.unknown_command')
    result = run('rust-qualify-spec',
                 [args.binary, 'qualify', '--spec', args.work / 'missing-spec.json'], 2)
    if b'unsupported command' in result.stdout + result.stderr:
        raise ValueError('$.rust.qualify.availability: actual public qualify --spec command is missing')
    equal(b'no such file or directory' in result.stderr, True, '$.rust.qualify.missing_spec')
    print('qualification grammar: Go and Rust probes agree; run qualify-lane.py for workload parity and goldens')


if __name__ == '__main__':
    main()
