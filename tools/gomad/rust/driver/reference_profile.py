#!/usr/bin/env python3
"""Invocation-only glue for the immutable Go model and standalone transport oracle."""
import argparse
import base64
import copy
import os
from pathlib import Path
import subprocess
import tarfile

import native_observations as custody

ORIGINAL = '1d695061b6382009d0659238c652a5a765b1d5f5'
GLUE = 'tools/gomad/cmd/gomad-reference/main.go'
ENGINE = 'cranelift-fuel-nan-canonical-nounwind-v2'
HERE = Path(__file__).resolve().parent


def put(path, raw):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(descriptor, 'wb') as output:
        output.write(raw)


def save(path, value):
    put(path, custody.canonical(value))


def derive(repo, output):
    repo, output = Path(repo).resolve(strict=True), Path(output)
    custody.require(output.is_absolute() and not output.exists(), 'fresh absolute derived root required')
    output.mkdir(mode=0o700)
    archive = output / 'original.tar'
    subprocess.run(['git', '-C', str(repo), 'archive', '--format=tar', '--output=' + str(archive), ORIGINAL], check=True)
    with archive.open('rb') as source_archive:
        original = subprocess.run(['git', 'get-tar-commit-id'], stdin=source_archive, stdout=subprocess.PIPE, check=True).stdout.decode().strip()
    custody.require(original == ORIGINAL, 'archive does not name original immutable commit')
    source = output / 'source'
    source.mkdir(mode=0o700)
    with tarfile.open(archive) as bundle:
        bundle.extractall(source, filter='data')
    files = tree(source)
    glue = source / GLUE
    custody.require(not glue.exists(), 'reference glue overwrites original source')
    glue.parent.mkdir(parents=True, mode=0o700)
    put(glue, (HERE / 'reference/main.go.txt').read_bytes())
    manifest = dict(schema='gomad.derived-go-reference/v1', original_commit=ORIGINAL,
                    archive=custody.snapshot(archive, 1 << 30), original_files=files,
                    glue={GLUE: custody.digest(glue.read_bytes())})
    save(output / 'source.json', manifest)
    verify(output)
    return manifest


def tree(root):
    files = {}
    for path in sorted(root.rglob('*')):
        name = str(path.relative_to(root))
        if path.is_symlink():
            files[name] = dict(kind='symlink', target=os.readlink(path))
        elif path.is_file():
            files[name] = dict(kind='file', **custody.snapshot(path, 512 << 20))
    return files


def verify(root):
    root = custody.absolute(root)
    manifest, _ = custody.document(root / 'source.json')
    custody.require(manifest['schema'] == 'gomad.derived-go-reference/v1'
                    and manifest['original_commit'] == ORIGINAL, 'foreign reference source')
    custody.require(custody.snapshot(root / 'original.tar', 1 << 30) == manifest['archive'], 'original archive changed')
    actual = tree(root / 'source')
    expected = dict(manifest['original_files'])
    custody.require(set(manifest['glue']) == {GLUE}
                    and manifest['glue'][GLUE] == custody.digest((HERE / 'reference/main.go.txt').read_bytes()),
                    'reference invocation glue changed')
    expected[GLUE] = dict(kind='file', bytes=(HERE / 'reference/main.go.txt').stat().st_size,
                          sha256=manifest['glue'][GLUE])
    custody.require(actual == expected, 'derived source changed outside declared invocation glue')
    return manifest


def byte_text(value):
    custody.require(isinstance(value, list) and all(type(byte) is int and 0 <= byte <= 255 for byte in value), 'invalid native byte string')
    return bytes(value).decode('utf-8')


def original_request(native, module, helper, helper_sha256, toolchain, identity, platform):
    custody.require(set(native) <= {'kind', 'guest', 'network', 'compile_cache'}
                    and native['kind'] == 'standalone' and native.get('network') is None
                    and native.get('compile_cache') is None, 'reference requires standalone uncached input')
    guest = native['guest']
    custody.require(set(guest) == {'module', 'module_sha256', 'limits', 'model', 'namespace', 'storage'}
                    and guest['storage'] is None, 'unsupported reference guest shape/storage')
    limits, model, namespace = guest['limits'], guest['model'], guest['namespace']
    custody.require(set(limits) == {'fuel', 'memory_bytes', 'output_bytes', 'evidence_bytes', 'wall_millis'}, 'unsupported native execution limits')
    custody.require(set(model) == {'configuration', 'arguments', 'environment', 'replay_trace'}
                    and model['replay_trace'] is None, 'reference requires record-only native model')
    config = model['configuration']
    custody.require(set(config) <= {'entropy_key', 'working_directory_sha256', 'arguments', 'environment', 'seed', 'mode',
        'limit', 'diagnostics', 'epoch_nanos', 'read_step_nanos', 'now', 'limits', 'digest_only', 'split', 'inspection', 'replay'}
        and config['mode'] == 1 and config['now'] == 1 and config['read_step_nanos'] == 0 and not config.get('split')
        and config.get('inspection') is None and config.get('replay') is None, 'unsupported reference model controls')
    custody.require(namespace is not None and set(namespace) == {'entries', 'writable_directories', 'preopens',
        'working_directory', 'stdin', 'limits', 'epoch_nanos'}, 'unsupported reference namespace')
    writable = [byte_text(value) for value in namespace['writable_directories']]
    expected = {}
    for name in writable:
        custody.require(name.startswith('/') and '..' not in name.split('/') and name != '/', 'invalid reference writable path')
        parts = name.strip('/').split('/')
        for index in range(len(parts)):
            path = '/' + '/'.join(parts[:index + 1])
            expected[path] = dict(path=list(path.encode()), directory=True, readonly=False, data=[])
    custody.require(namespace['entries'] == [expected[name] for name in sorted(expected)]
                    and namespace['preopens'] == [list(name.encode()) for name in sorted(set(writable))],
                    'reference does not translate captured/read-only namespace inputs')
    cwd = byte_text(namespace['working_directory'])
    custody.require(custody.digest(cwd.encode()) == config['working_directory_sha256']
                    and namespace['epoch_nanos'] == config['epoch_nanos'], 'native namespace/model identity mismatch')
    for key in ('arguments', 'environment'):
        custody.require(config[key] == dict(bytes=0, count=0, chunks=0), 'native declaration must be derived from supplied strings')
    custody.require(set(config['limits']) == {'calls', 'transcript_bytes', 'pending_events'}, 'unsupported native model limits')
    custody.require(type(config['diagnostics']) is bool, 'native diagnostic control must be boolean')
    maximum_stream_bytes = (limits['output_bytes'] + config['limits']['transcript_bytes']
                            + config['limit'] * (2 if config['diagnostics'] else 1) + (64 << 20))
    custody.require(limits['evidence_bytes'] >= maximum_stream_bytes,
                    'native aggregate evidence bound may bind; no original Go counterpart')
    fs_limits = namespace['limits']
    custody.require(set(fs_limits) == {'files', 'descriptors', 'filesystem_bytes', 'output_bytes'}, 'unsupported native filesystem limits')
    custody.require(fs_limits['output_bytes'] == limits['output_bytes'], 'native output limits disagree')
    custody.require(isinstance(toolchain, str) and len(toolchain) == 64 and bytes.fromhex(toolchain).hex() == toolchain,
                    'original toolchain build key required')
    goos, goarch = platform
    custody.require((goos, goarch) in (('linux', 'aarch64'), ('linux', 'x86_64')), 'reference host platform unsupported')
    original = dict(Profile='gomad.wasi-cooperative/v1', Args=model['arguments'], Environment=model['environment'],
        WorkingDirectory=cwd, WritableDirectories=writable, Stdin=base64.b64encode(bytes(namespace['stdin'])).decode(),
        EntropyKey=config['entropy_key'], Clock=dict(EpochNanos=config['epoch_nanos'], ReadStepNanos=config['read_step_nanos']),
        Limits=dict(OutputBytes=limits['output_bytes'], TranscriptBytes=config['limits']['transcript_bytes'],
            Calls=config['limits']['calls'], PendingEvents=config['limits']['pending_events'],
            Descriptors=fs_limits['descriptors'], Files=fs_limits['files'], FilesystemBytes=fs_limits['filesystem_bytes']))
    custody.require(set(identity) == {'TargetSHA256', 'ToolchainBuildKey', 'GOOS', 'GOARCH', 'ImplementationSHA256'}
                    and identity['TargetSHA256'] == list(bytes.fromhex(guest['module_sha256']))
                    and identity['ToolchainBuildKey'] == toolchain and identity['GOOS'] == 'wasip1' and identity['GOARCH'] == 'wasm'
                    and len(identity['ImplementationSHA256']) == 32 and any(identity['ImplementationSHA256'])
                    and all(type(byte) is int and 0 <= byte <= 255 for byte in identity['ImplementationSHA256']),
                    'native choice identity mismatch')
    return dict(HelperPath=str(helper), HelperSHA256=helper_sha256, ModulePath=str(module), ModuleSHA256=guest['module_sha256'],
        Engine=dict(name='wasmtime', version='47.0.3', configuration=ENGINE, host_os=goos, host_arch=goarch),
        MemoryBytes=limits['memory_bytes'], Fuel=limits['fuel'], Config=original, Timeout=limits['wall_millis'] * 1_000_000,
        DigestTranscript=config.get('digest_only', False), Runtime=dict(Seed=config['seed'], Diagnostics=config['diagnostics'],
            Choice=dict(Mode=1, ExecutionIdentity=identity, Limit=config['limit'])))


def raw_run(raw, output, owner):
    value = custody.decoded(raw)
    result = value['result']
    streams = {name: base64.b64decode(payload or '', validate=True) for name, payload in dict(
        stdout=result['Stdout'], stderr=result['Stderr'], wasi=result['Transcript'], choice=result['ChoiceTrace']['Bytes'],
        network=result['NetworkTranscript'], terminal=value['terminal']).items()}
    catalog = {}
    for name, data in streams.items():
        put(output / name, data)
        catalog[name] = dict(path=str(output / name), bytes=len(data), sha256=custody.digest(data))
    run = dict(schema='gomad.go-reference-run/v1', owner=owner, original_commit=ORIGINAL,
        raw_result=dict(path=str(output / 'raw-result.json'), sha256=custody.digest(raw)), streams=catalog,
        termination=result['Termination'], exit_code=result['ExitCode'], reaped=result['Reaped'],
        cancelled=result['Cancelled'], watchdog_timeout=result['WatchdogTimeout'], error=value['error'],
        transcript_sha256=bytes(result['TranscriptSHA256']).hex(), calls=result['Calls'], time_advances=result['TimeAdvances'],
        environment_identity=result['EnvironmentIdentity'], fuel_remaining=result['FuelRemaining'],
        authorization_to_terminal_nanos=result['AuthorizationToTerminalNanos'], select_poll=result['ChoiceTrace']['Summary']['SelectPoll'])
    save(output / 'run.json', run)
    return run


def bind_preparation(native, provenance):
    configuration = provenance['configuration']
    guest = native['guest']
    expected = copy.deepcopy(configuration['model'])
    custody.require('seed' not in expected, 'prepared model must derive native seed')
    expected['seed'] = guest['model']['configuration']['seed']
    environment = [name + '=' + value for name, value in sorted(configuration['environment'].items())]
    if configuration.get('derive_seed_environment', False):
        custody.require('GOMADSEED' not in configuration['environment'], 'prepared seed environment conflict')
        environment.append('GOMADSEED=' + str(expected['seed']))
    custody.require(guest['model'] == dict(configuration=expected, arguments=configuration['arguments'],
            environment=environment, replay_trace=None)
        and guest['limits'] == configuration['limits'] and guest['namespace'] == configuration['namespace']
        and guest['storage'] == configuration.get('storage') and native.get('network') == configuration.get('network'),
        'native request differs from exact prepared configuration')


def same_original_evidence(left, right):
    for name in ('stdout', 'stderr', 'choice', 'network', 'terminal'):
        custody.require(left['streams'][name]['sha256'] == right['streams'][name]['sha256'],
                        'original runtime evidence differs: ' + name)
    for field in ('transcript_sha256', 'termination', 'exit_code'):
        custody.require(left[field] == right[field], 'original runtime evidence differs: ' + field)


def sealed(path, files, maximum=512 << 20):
    identity = custody.snapshot(path, maximum)
    custody.require(files.get(str(path)) == identity['sha256'], 'unsealed reference input: ' + str(path))
    return identity


def build(derived, stock_go, output, files):
    import driver
    derived, stock_go = custody.absolute(derived), custody.absolute(stock_go)
    manifest = verify(derived)
    sealed(derived / 'source.json', files)
    sealed(stock_go, files)
    output = Path(output)
    custody.require(output.is_absolute() and not output.exists(), 'fresh absolute reference build root required')
    output.mkdir(mode=0o700)
    temporary = output / 'go-tmp'
    temporary.mkdir(mode=0o700)
    binary = output / 'gomad-reference'
    config = dict(repo=str(derived / 'source/tools/gomad'), environment=dict(GOMAXPROCS='1', CGO_ENABLED='0', GOFLAGS='', GOTMPDIR=str(temporary)))
    argv = [str(stock_go), 'build', '-p=1', '-mod=readonly', '-tags=test_dep', '-trimpath', '-buildvcs=false', '-o', str(binary), './cmd/gomad-reference']
    driver.command(config, output, 'build', argv, timeout=180)
    verify(derived)
    receipt = dict(schema='gomad.reference-build/v1', original_commit=ORIGINAL,
        source_manifest=custody.snapshot(derived / 'source.json', 16 << 20),
        stock_go=dict(path=str(stock_go), **custody.snapshot(stock_go, 512 << 20)),
        binary=dict(path=str(binary), **custody.snapshot(binary, 512 << 20)),
        command=dict(path=str(output / 'build.command.json'), **custody.snapshot(output / 'build.command.json', 16 << 20)),
        glue=manifest['glue'])
    save(output / 'build.json', receipt)
    return receipt


def run(derived, native_request, request_sha256, provenance, provenance_sha256, files, build_receipt, binary, helper, helper_receipt, output):
    import driver
    import platform
    derived, binary, helper = map(custody.absolute, (derived, binary, helper))
    source_manifest = verify(derived)
    static = {str(path): sealed(path, files) for path in (derived / 'source.json', Path(build_receipt), binary, helper, Path(helper_receipt))}
    helper_build, _ = custody.document(helper_receipt)
    custody.require(helper_build['schema'] == 'gomad.reference-helper-build/v1'
                    and helper_build['cargo_artifact']['features'] == ['reference-helper']
                    and helper_build['command']['exit'] == 0 and helper_build['command']['observation'] == 'completed'
                    and helper_build['binary'] == dict(path=str(helper), **static[str(helper)]),
                    'reference helper lacks exact framed transport feature/build custody')
    receipt, _ = custody.document(build_receipt)
    custody.require(receipt['schema'] == 'gomad.reference-build/v1' and receipt['original_commit'] == ORIGINAL
                    and receipt['source_manifest'] == static[str(derived / 'source.json')]
                    and receipt['glue'] == source_manifest['glue']
                    and receipt['binary'] == dict(path=str(binary), **static[str(binary)]), 'reference binary/source build custody mismatch')
    command, command_identity = custody.document(receipt['command']['path'], canonical_required=False)
    custody.require(dict(path=receipt['command']['path'], **command_identity) == receipt['command']
                    and command['exit'] == 0 and command['observation'] == 'completed', 'reference build failed/unbound')
    native, native_identity = custody.document(native_request)
    provenance_value, provenance_identity = custody.document(provenance)
    custody.require(native_identity['sha256'] == request_sha256 and provenance_identity['sha256'] == provenance_sha256,
                    'changed actual native request/provenance')
    bind_preparation(native, provenance_value)
    module = custody.absolute(native['guest']['module'])
    module_identity = custody.snapshot(module, 512 << 20)
    custody.require(module_identity['sha256'] == native['guest']['module_sha256'] == provenance_value['module_sha256']
                    and provenance_value['schema'] == 'gomad.rust-wasm-provenance/v1'
                    and provenance_value['build_key'] == custody.digest(custody.canonical(provenance_value['inputs'])),
                    'native module/preparation provenance mismatch')
    output = Path(output)
    custody.require(output.is_absolute() and not output.exists(), 'fresh absolute reference result root required')
    output.mkdir(mode=0o700)
    put(output / 'native-request.json', Path(native_request).read_bytes())
    put(output / 'provenance.json', Path(provenance).read_bytes())
    # Pin the exact module in the private attempt; paths never change guest bytes.
    pinned_module = output / 'module.wasm'
    put(pinned_module, module.read_bytes())
    custody.require(custody.snapshot(pinned_module, 512 << 20) == module_identity, 'module changed while pinning')
    request = original_request(native, pinned_module, helper, static[str(helper)]['sha256'],
                               provenance_value['build_key'], provenance_value['choice_identity'], (platform.system().lower(), platform.machine()))
    results, observations = {}, {}
    for owner in ('go', 'helper'):
        directory = output / owner
        directory.mkdir(mode=0o700)
        invocation = directory / 'invocation.json'
        save(invocation, dict(request=request, remote_model=owner == 'go'))
        command_error = None
        try:
            driver.command(dict(repo=str(derived / 'source/tools/gomad')), directory, 'invoke',
                [str(binary), str(invocation)], timeout=request['Timeout'] / 1_000_000_000 + 10)
        except (driver.Refusal, subprocess.TimeoutExpired) as error:
            command_error = str(error)
        raw_path = directory / 'invoke.stdout'
        custody.snapshot(raw_path, 512 << 20)
        raw = raw_path.read_bytes()
        put(directory / 'raw-result.json', raw)
        try:
            observations[owner] = raw_run(raw, directory, owner)
        finally:
            save(directory / 'status.json', dict(command_error=command_error))
        custody.require(command_error is None, 'original reference run refused: ' + str(command_error))
        observed = observations[owner]
        custody.require(observed['termination'] == 'exit' and observed['exit_code'] == 0 and observed['reaped'] is True
                        and observed['cancelled'] is False and observed['watchdog_timeout'] is False and not observed['error']
                        and observed['authorization_to_terminal_nanos'] > 0, 'original reference did not complete successfully')
        results[owner] = dict(path=str(directory / 'run.json'), **custody.snapshot(directory / 'run.json', 16 << 20),
                             command=dict(path=str(directory / 'invoke.command.json'), **custody.snapshot(directory / 'invoke.command.json', 16 << 20)))
    same_original_evidence(observations['go'], observations['helper'])
    verify(derived)
    custody.require(custody.snapshot(native_request, 16 << 20) == native_identity
                    and custody.snapshot(provenance, 16 << 20) == provenance_identity
                    and custody.snapshot(module, 512 << 20) == module_identity, 'native inputs changed during reference run')
    for path, identity in static.items():
        custody.require(custody.snapshot(path, 512 << 20) == identity, 'reference static input changed during run')
    manifest = dict(schema='gomad.runtime-reference-pair/v1', original_commit=ORIGINAL,
        native_request=dict(path=str(native_request), **native_identity), provenance=dict(path=str(provenance), **provenance_identity),
        module=dict(path=str(module), pinned_path=str(pinned_module), **module_identity),
        static_inputs=static, source_manifest=str(derived / 'source.json'), build_receipt=str(build_receipt), helper_receipt=str(helper_receipt), runs=results)
    save(output / 'manifest.json', manifest)
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    derive_command = commands.add_parser('derive')
    derive_command.add_argument('--repo', required=True)
    derive_command.add_argument('--output', required=True)
    check = commands.add_parser('verify')
    check.add_argument('--derived', required=True)
    compile_command = commands.add_parser('build')
    for name in ('derived', 'stock-go', 'output', 'files'):
        compile_command.add_argument('--' + name, required=True)
    execute = commands.add_parser('run')
    for name in ('derived', 'native-request', 'request-sha256', 'provenance', 'provenance-sha256', 'files', 'build-receipt', 'reference-binary', 'helper', 'helper-receipt', 'output'):
        execute.add_argument('--' + name, required=True)
    args = parser.parse_args()
    if args.command == 'derive':
        derive(args.repo, args.output)
    elif args.command == 'verify':
        verify(args.derived)
    elif args.command == 'build':
        files, _ = custody.document(args.files)
        build(args.derived, args.stock_go, args.output, files)
    else:
        files, _ = custody.document(args.files)
        run(args.derived, args.native_request, args.request_sha256, args.provenance, args.provenance_sha256,
            files, args.build_receipt, args.reference_binary, args.helper, args.helper_receipt, args.output)


if __name__ == '__main__':
    main()
