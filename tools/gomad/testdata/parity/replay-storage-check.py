#!/usr/bin/env python3
"""Replay the unchanged helper-owned crash fixture and authenticate its retained evidence."""
import argparse
import hashlib
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys

sys.dont_write_bytecode = True


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def sha(value):
    return hashlib.sha256(value).hexdigest()


p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--reference', type=Path, required=True)
p.add_argument('--work', type=Path, required=True)
a = p.parse_args()
a.work.mkdir(mode=0o700, parents=True, exist_ok=False)
corpus = Path(__file__).resolve().parent
loader = SourceFileLoader('behavioral_parity', str(corpus.parent.parent / 'scripts/behavioral-parity'))
parity = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(parity)
receipts = []


def run(name, arguments, status):
    marker = a.work / (name + '-engine-start')
    result = subprocess.run([str(a.binary), *map(str, arguments)], capture_output=True,
                            env=dict(os.environ, GOMAD_FIXTURE_GUEST_START_FILE=str(marker)), timeout=60)
    receipts.append(dict(name=name, argv=[str(a.binary), *map(str, arguments)],
                         status=result.returncode, expected=status, stdout=result.stdout.decode(),
                         stderr=result.stderr.decode(), engine_entry=marker.exists()))
    (a.work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
    assert result.returncode == status, (name, result.returncode, result.stderr)
    return result


reference = json.loads(a.reference.read_bytes())
assert reference['schema'] == 'gomad.replay-storage-origin-reference/v1', reference['schema']
assert reference['scope'] == 'unchanged frozen helper transcript fixture; zero runtime decisions'
module = Path(reference['module_path'])
assert sha(module.read_bytes()) == reference['module_sha256']
assert reference['terminal']['termination'] == 'storage_crash'
assert reference['terminal']['message'] == 'modeled storage crash: stop-sync'
request = reference['request']
model = dict(request['local_model'])
seed = model.pop('seed')
assert seed == 7 and model['now'] == 1 and model['epoch_nanos'] == 1000
assert request['namespace']['files'] == 2 and request['namespace']['bytes'] == 151
configuration = dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',
    profile='gomad.wasi-stock-exploratory/v1',
    limits=dict(fuel=request['fuel'], memory_bytes=request['memory_bytes'], wall_millis=1000,
                output_bytes=request['output_bytes'], evidence_bytes=1 << 20),
    arguments=[], environment={'PWD': '/data'}, runtime_sha256=None, model=model,
    storage=reference['storage_input'], namespace=dict(
        entries=[dict(path=list(b'/data'), directory=True, readonly=False, data=[])],
        writable_directories=[list(b'/data')], preopens=[list(b'/data')],
        working_directory=list(b'/data'), stdin=[], epoch_nanos=1000,
        limits=dict(files=10, descriptors=16, filesystem_bytes=1024, output_bytes=1024)))
input_path = a.work / 'input.json'
input_path.write_bytes(canonical(dict(schema='gomad.replay-fixture-input/v1', cases=[dict(
    name='helper-originated-storage-crash', module_path=str(module),
    module_sha256=reference['module_sha256'], toolchain_build_key=reference['fixture_sha256'],
    seed=seed, choice_identity=None, configuration=configuration)])))
root = a.work / 'rust'
run('produce', ['fixture-replay-input', '--root', root, '--input', input_path], 0)
bundle = json.loads((root / 'foundation-result.json').read_bytes())
parity.authenticate(root, bundle)
artifact = root / bundle['records'][0]['artifact']
record = json.loads((artifact / 'record.json').read_bytes())
assert record['outcome']['class'] == 'runner_failure'
assert record['outcome']['termination'] == 'storage_crash'
assert record['outcome']['reason'] == reference['terminal']['message']
assert record['outcome']['exit_code'] is None
for stream, expected in reference['evidence'].items():
    raw = (artifact / 'evidence' / stream).read_bytes()
    assert len(raw) == expected['bytes'] and sha(raw) == expected['sha256'], stream
run('verify', ['replay', '--verify-only', artifact], 0)
observed = a.work / 'observed'
result = run('replay', ['replay', '--observed', observed, artifact], 1)
assert result.stdout == b'gomad: reproduced=true diagnostic=false result=target_failure choice-replay=none\n'
metadata = json.loads((observed / 'backend-evidence.bin').read_bytes())['local_model']
assert metadata['finished'] is False and metadata['cursor'] == metadata['tape_records'] == 0
assert metadata['next_expected'] is None and metadata['tape_sha256'] == ''
assert not (a.work / 'verify-engine-start').exists() and (a.work / 'replay-engine-start').is_file()
bad = a.work / 'unexpected-image'
shutil.copytree(artifact, bad)
manifest = json.loads((bad / 'manifest.json').read_bytes())
inputs = json.loads((bad / 'evidence/replay.json').read_bytes())
inputs['configuration']['storage']['image'] = [0]
raw = canonical(inputs)
(bad / 'evidence/replay.json').write_bytes(raw)
for entry in (manifest['files']['evidence/replay.json'], manifest['record']['streams']['evidence/replay.json']):
    entry.update(bytes=len(raw), sha256=sha(raw))
manifest['record']['identity']['configuration'] = sha(canonical(inputs['configuration']))
manifest['record']['execution_identity'] = sha(canonical(manifest['record']['identity']))
raw = canonical(manifest['record'])
(bad / 'record.json').write_bytes(raw)
manifest['record_sha256'] = sha(raw)
manifest['files']['record.json'].update(bytes=len(raw), sha256=sha(raw))
(bad / 'manifest.json').write_bytes(canonical(manifest))
result = run('unexpected-image', ['replay', bad], 2)
assert not result.stdout and not (a.work / 'unexpected-image-engine-start').exists()
print('storage crash: actual helper-originated evidence, interrupted metadata and no-start storage refusal passed')
