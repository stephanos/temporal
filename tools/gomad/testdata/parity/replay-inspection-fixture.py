#!/usr/bin/env python3
"""Produce own-lineage inspection artifacts from the unchanged registered Go fixture."""
import argparse
import base64
import hashlib
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
status = 0 if args.success else 1
fixture_sha256 = sha(fixture_source.read_bytes())
installation = args.work / 'exact-installation.json'
run('go-install', [baseline, 'install', '--compiler', args.compiler,
                  '--helper', args.baseline / 'bin/gomad-wasmhost', '--profile', 'gomad.wasi-cooperative/v1',
                  '--runtime-root', args.source / 'tools/gomad/internal/model/runtimesource',
                  '--fuel', str(1 << 60), '--cache', args.work / 'preparation', '--output', installation])
run('go-explore', [baseline, '--installation', installation, 'explore', '--json', '--choices',
                  '--keep-successes=all',
                  '--seeds=7', '--parallel=1', '--on-failure=all', '--working-dir', args.source,
                  '--build-tag=gomad', '--build-tag=test_dep', '--execution-timeout=30s', '--overall-timeout=5m',
                  '--artifacts', args.work / 'go-artifacts', 'go-run', guest_package], status)
if args.success:
    fixture_source.unlink()
    fixture_source.parent.rmdir()
artifacts = [path.parent for path in (args.work / 'go-artifacts').rglob('manifest.json')
             if json.loads(path.read_bytes()).get('artifact_kind') == ('gomad.success/v1' if args.success else 'gomad.target-failure/v1')]
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
rust_artifact = args.work / 'rust' / bundle['records'][0]['artifact']
assert (rust_artifact / 'evidence/choice').read_bytes() == (artifact / 'choices.bin').read_bytes()
assert b'reproduced=true' in run('rust-replay', [args.binary, 'replay', *(['--' + args.role] if args.role else []), rust_artifact], status)
assert b'reproduced=true' in run('go-replay', [baseline, '--installation', installation, 'replay', artifact], status)
print(json.dumps({'artifact': str(rust_artifact), 'go_reference': str(args.work / 'registered-reference.json'),
                  'input': str(input_path), 'go_binary': str(baseline)}))
