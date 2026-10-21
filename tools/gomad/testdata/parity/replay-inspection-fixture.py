#!/usr/bin/env python3
"""Produce own-lineage inspection artifacts from the unchanged registered Go fixture."""
import argparse
import base64
import hashlib
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True
p = argparse.ArgumentParser()
for name in ('binary', 'baseline', 'source', 'compiler', 'work'):
    p.add_argument('--' + name, type=Path, required=True)
p.add_argument('--role', choices=['stress-jitter', 'determinism-fixture'])
p.add_argument('--success', action='store_true', help='separate successful registered guest for restart coverage')
p.add_argument('--network', choices=['ordinary', 'faults'], help='standalone network exact-replay fixture')
p.add_argument('--preparation-root', type=Path, help='declared stable toolchain preparation input for network goldens')
args = p.parse_args()
for name in ('binary', 'baseline', 'source', 'compiler', 'work'):
    setattr(args, name, getattr(args, name).resolve())
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
environment = {key: value for key, value in os.environ.items()
               if not key.startswith('GOMAD_') and key not in ('GOROOT', 'GOFLAGS', 'GOMOD', 'GOEXPERIMENT')}
environment.update(GOENV='off', GOTOOLCHAIN='local', GOWORK='off', CGO_ENABLED='0', GOPROXY='off', GOSUMDB='off')
receipts = []


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def run(name, command, status=0):
    result = subprocess.run(list(map(str, command)), env=environment, capture_output=True, timeout=600)
    (args.work / (name + '.stdout')).write_bytes(result.stdout)
    (args.work / (name + '.stderr')).write_bytes(result.stderr)
    receipts.append(dict(name=name, argv=list(map(str, command)), status=result.returncode,
                         stdout=result.stdout.decode(), stderr=result.stderr.decode()))
    (args.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
    assert result.returncode == status, (name, result.returncode, result.stderr)
    return result.stdout


baseline = args.baseline / 'bin/gomad'
guest_package = './common/gomaddebug/testdata/fixture'
fixture_source = args.source / guest_package / 'main.go'
assert not (args.success and args.network), 'success registry and network fixtures are separate'
assert not args.network or args.preparation_root, 'network fixtures require a stable declared preparation root'
if args.network:
    original = (Path(__file__).resolve().parent / 'targets/replay-network.go.txt' if args.network == 'ordinary'
                else args.source / 'tools/gomad/internal/backends/wasm/testdata/network_streams/main.go')
    guest_package = './common/gomaddebug/testdata/fn232network' + args.network
    generated = args.source / guest_package
    generated.mkdir(mode=0o700, exist_ok=True)
    fixture_source = generated / 'main.go'
    assert not fixture_source.exists() or fixture_source.read_bytes() == original.read_bytes()
    fixture_source.write_bytes(original.read_bytes())
    (args.work / 'network-guest.go').write_bytes(original.read_bytes())
if args.success:
    original = fixture_source.read_bytes()
    assert original.count(b'\tos.Exit(1)\n') == 1, 'registered success fixture transformation changed'
    transformed = original.replace(b'\tos.Exit(1)\n', b'\tos.Exit(0)\n')
    guest_package = './common/gomaddebug/testdata/fn232successfixture'
    generated = args.source / guest_package
    generated.mkdir(mode=0o700, exist_ok=True)
    fixture_source = generated / 'main.go'
    assert not fixture_source.exists() or fixture_source.read_bytes() == transformed
    fixture_source.write_bytes(transformed)
    (args.work / 'success-guest.go').write_bytes(transformed)
status = 0 if args.success or args.network else 1
fixture_sha256 = sha(fixture_source.read_bytes())
installation_options = []
if args.network:
    config = json.loads((args.source / 'tools/gomad/internal/backends/wasm/command/testdata/scenario/config.json').read_bytes())
    config['network_v2']['limits'].update(listeners=16, connections=128, bytes=8 << 20,
        connection_bytes=4 << 20, transitions=16384)
    if args.network == 'faults':
        config['network_v2']['faults'] = [
            dict(name='stream-delay', action='delay', delay_nanos=5000000),
            dict(name='dial-partition', action='partition', at_nanos=50000000),
            dict(name='dial-heal', action='heal', at_nanos=90000000),
            dict(name='connection-reset', action='reset', at_nanos=120000000, port=9000)]
    config_path = args.work / 'network-config.json'
    config_path.write_bytes(canonical(config))
    installation_options = ['--config', config_path]
installation = args.work / 'exact-installation.json'
run('go-install', [baseline, 'install', '--compiler', args.compiler,
                  '--helper', args.baseline / 'bin/gomad-wasmhost', '--profile', 'gomad.wasi-cooperative/v1',
                  '--runtime-root', args.source / 'tools/gomad/internal/model/runtimesource',
                  '--fuel', str(1 << 60), '--cache', args.preparation_root or args.work / 'preparation', '--output', installation,
                  *installation_options])
run('go-explore', [baseline, '--installation', installation, 'explore', '--json', '--choices',
                  '--keep-successes=all',
                  '--seeds=7', '--parallel=1', '--on-failure=all', '--working-dir', args.source,
                  '--build-tag=gomad', '--build-tag=test_dep', '--execution-timeout=30s', '--overall-timeout=5m',
                  '--artifacts', args.work / 'go-artifacts', 'go-run', guest_package], status)
if args.success or args.network:
    fixture_source.unlink()
    fixture_source.parent.rmdir()
artifacts = [path.parent for path in (args.work / 'go-artifacts').rglob('manifest.json')
             if json.loads(path.read_bytes()).get('artifact_kind') == ('gomad.success/v1' if status == 0 else 'gomad.target-failure/v1')]
assert len(artifacts) == 1, artifacts
artifact = artifacts[0]
manifest = json.loads((artifact / 'manifest.json').read_bytes())
for entry in manifest['files']:
    raw = (artifact / entry['path']).read_bytes()
    assert len(raw) == int(entry['size']) and sha(raw) == entry['sha256'].removeprefix('sha256:'), entry['path']
provenance = json.loads((artifact / manifest['target']['backend']['provenance']['file']).read_bytes())
config = provenance['Config']
limits = config['Limits']
limits['OutputBytes'] = int(manifest['limits']['output_bytes'])
limits['TranscriptBytes'] = int(manifest['limits']['io_transcript_bytes'])
arguments = manifest['target']['argv']
guest_environment = {item['name']: item['value'] for item in manifest['environment']}
guest_environment['PWD'] = config['WorkingDirectory']
choice_limit = int(manifest['choice_profile']['trace']['limit'])
implementation = manifest['choice_profile']['implementation_sha256'].removeprefix('sha256:')
module = (artifact / manifest['target']['file']).read_bytes()
module_path = args.work / 'registered.wasm'
module_path.write_bytes(module)
module_path.chmod(0o600)
configuration = {
    'engine': 'cranelift-fuel-nan-canonical-nounwind-epoch-v1', 'profile': config['Profile'],
    'limits': {'fuel': provenance['Fuel'], 'memory_bytes': provenance['MemoryBytes'],
               'wall_millis': int(manifest['limits']['execution_timeout_nanos']) // 1000000,
               'output_bytes': limits['OutputBytes'],
               'evidence_bytes': limits['TranscriptBytes'] + choice_limit + 2 * limits['OutputBytes'] + (64 << 20)},
    'arguments': arguments, 'environment': guest_environment, 'runtime_sha256': implementation,
    'model': {'entropy_key': config['EntropyKey'], 'mode': 1, 'limit': choice_limit, 'diagnostics': False,
              'epoch_nanos': config['Clock']['EpochNanos'], 'read_step_nanos': config['Clock']['ReadStepNanos'],
              'now': 1, 'working_directory_sha256': sha(config['WorkingDirectory'].encode()),
              'arguments': {'bytes': 0, 'count': 0, 'chunks': 0}, 'environment': {'bytes': 0, 'count': 0, 'chunks': 0},
              'limits': {'calls': limits['Calls'], 'transcript_bytes': limits['TranscriptBytes'], 'pending_events': limits['PendingEvents']}},
    'namespace': {'entries': [], 'writable_directories': [list(path.encode()) for path in config['WritableDirectories']],
                  'preopens': [list(path.encode()) for path in sorted(set(config['WritableDirectories']))],
                  'working_directory': list(config['WorkingDirectory'].encode()),
                  'stdin': list(base64.b64decode(config['Stdin'] or '', validate=True)),
                  'limits': {'files': limits['Files'], 'descriptors': limits['Descriptors'],
                             'filesystem_bytes': limits['FilesystemBytes'], 'output_bytes': limits['OutputBytes']},
                  'epoch_nanos': config['Clock']['EpochNanos']}}
if args.network:
    configuration['network'] = config['network_v2']
case = {'name': 'registered-seed7', 'module_path': str(module_path), 'module_sha256': sha(module),
        'toolchain_build_key': provenance['BuildKey'], 'seed': 7, 'configuration': configuration,
        'choice_identity': {'TargetSHA256': list(bytes.fromhex(sha(module))), 'ToolchainBuildKey': provenance['BuildKey'],
                            'GOOS': 'wasip1', 'GOARCH': 'wasm', 'ImplementationSHA256': list(bytes.fromhex(implementation))}}
input_path = args.work / 'registered-input.json'
input_path.write_bytes(canonical({'schema': 'gomad.replay-fixture-input/v1', 'cases': [case]}))
reference = {'schema': 'gomad.replay-inspection-reference/v1', 'go_binary_sha256': sha(baseline.read_bytes()),
             'source_fixture_sha256': fixture_sha256,
             'records': [{'mode': 'exact', 'case': 'match', 'seed': 7, 'artifact': str(artifact)}]}
(args.work / 'registered-reference.json').write_text(json.dumps(reference, indent=2) + '\n')
bundle = json.loads(run('rust-produce', [args.binary, 'fixture-replay-input', '--root', args.work / 'rust', '--input', input_path,
                                       *(['--role', args.role] if args.role else [])]))
assert len(bundle['records']) == 1
loader = SourceFileLoader('inspection_fixture_parity', str(Path(__file__).resolve().parents[2] / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)
parity.authenticate(args.work / 'rust', bundle)
rust_artifact = args.work / 'rust' / bundle['records'][0]['artifact']
assert (rust_artifact / 'evidence/choice').read_bytes() == (artifact / 'choices.bin').read_bytes()
assert b'reproduced=true' in run('rust-replay', [args.binary, 'replay', *(['--' + args.role] if args.role else []), rust_artifact], status)
assert b'reproduced=true' in run('go-replay', [baseline, '--installation', installation, 'replay', artifact], status)
if args.network:
    envelope = json.loads((artifact / manifest['target']['backend']['evidence']['file']).read_bytes())
    network = base64.b64decode(envelope['Network'], validate=True)
    rust_network = (rust_artifact / 'evidence/network').read_bytes()
    parity.check_equal(json.loads(rust_network), json.loads(network))
    assert len(rust_network) == len(network), 'network evidence byte accounting differs'
    assert (rust_artifact / 'evidence/transcript').read_bytes() == base64.b64decode(envelope['WASI'], validate=True)
    for name in ('stdout', 'stderr'):
        expected = (artifact / name).read_bytes()
        actual = rust_artifact / 'output' / name
        assert (actual.read_bytes() if actual.exists() else b'') == expected
    assert b'"kind":"write"' in network
    if args.network == 'faults':
        for fault in ('stream-delay', 'dial-partition', 'dial-heal', 'connection-reset'):
            assert fault.encode() in network
    repeated_root = args.work / 'rust-repeat'
    repeated = json.loads(run('rust-repeat', [args.binary, 'fixture-replay-input', '--root', repeated_root, '--input', input_path]))
    parity.authenticate(repeated_root, repeated)
    repeated_artifact = repeated_root / repeated['records'][0]['artifact']
    view = dict(schema='gomad.replay-network-golden/v1',
        record=(rust_artifact / 'record.json').read_text(), manifest=(rust_artifact / 'manifest.json').read_text(),
        ledger=(args.work / 'rust' / bundle['ledger_file']).read_text(), network=dict(bytes=len(rust_network), sha256=sha(rust_network)))
    repeated_network = (repeated_artifact / 'evidence/network').read_bytes()
    repeated_view = dict(schema='gomad.replay-network-golden/v1',
        record=(repeated_artifact / 'record.json').read_text(), manifest=(repeated_artifact / 'manifest.json').read_text(),
        ledger=(repeated_root / repeated['ledger_file']).read_text(), network=dict(bytes=len(repeated_network), sha256=sha(repeated_network)))
    parity.check_equal(view, repeated_view)
    parity.golden(Path(__file__).resolve().parent / 'goldens-replay' / ('network-' + args.network + '.json'),
                  view, os.environ.get('GOMAD_GOLDEN_REGENERATE') == '1')
print(json.dumps({'artifact': str(rust_artifact), 'go_reference': str(args.work / 'registered-reference.json'),
                  'input': str(input_path), 'go_binary': str(baseline)}))
