#!/usr/bin/env python3
"""Actual helper-originated one-choice storage interruption; no tape codec here."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

sys.dont_write_bytecode = True


def encode(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':')).encode()


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


parser = argparse.ArgumentParser()
parser.add_argument('--binary', type=Path, required=True)
parser.add_argument('--reference', type=Path, required=True)
parser.add_argument('--work', type=Path, required=True)
args = parser.parse_args()
args.work.mkdir(mode=0o700, parents=True, exist_ok=False)
reference_raw = args.reference.read_bytes()
reference = json.loads(reference_raw)
assert reference['schema'] == 'gomad.replay-storage-origin-reference/v1'
assert reference['scope'] == 'unchanged frozen helper transcript fixture; zero runtime decisions'
assert sha(Path(reference['module_path']).read_bytes()) == reference['module_sha256']
assert reference['terminal']['termination'] == 'storage_crash'
assert reference['storage_input']['descriptor'] == reference['request']['storage']
request = reference['request']
model = dict(request['local_model'])
seed = model.pop('seed')
assert seed == 7 and model['now'] == 1
module = Path(__file__).resolve().parent / 'targets/replay-storage-prefix.wat'
module_digest = sha(module.read_bytes())
implementation = sha(b'gomad.storage-prefix-explicit-test-runtime/v1')
configuration = dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',
    profile='gomad.wasi-cooperative/v1',
    limits=dict(fuel=request['fuel'], memory_bytes=request['memory_bytes'], wall_millis=1000,
                output_bytes=request['output_bytes'], evidence_bytes=1 << 20),
    arguments=[], environment={'PWD': '/data'}, runtime_sha256=implementation, model=model,
    storage=reference['storage_input'], namespace=dict(
        entries=[dict(path=list(b'/data'), directory=True, readonly=False, data=[])],
        writable_directories=[list(b'/data')], preopens=[list(b'/data')],
        working_directory=list(b'/data'), stdin=[], epoch_nanos=1000,
        limits=dict(files=10, descriptors=16, filesystem_bytes=1024, output_bytes=1024)))
inputs = dict(schema='gomad.replay-storage-prefix-input/v1', module_path=str(module),
    module_sha256=module_digest, seed=seed, configuration=configuration,
    choice_identity=dict(TargetSHA256=list(bytes.fromhex(module_digest)),
        ToolchainBuildKey=reference['fixture_sha256'], GOOS='wasip1', GOARCH='wasm',
        ImplementationSHA256=list(bytes.fromhex(implementation))))
input_path = args.work / 'input.json'
input_path.write_bytes(encode(inputs))
root = args.work / 'execution'
marker = args.work / 'engine-start'
command = [str(args.binary), 'fixture-replay-storage-prefix', '--root', str(root), '--input', str(input_path)]
result = subprocess.run(command, capture_output=True, timeout=60,
                        env=dict(os.environ, GOMAD_FIXTURE_GUEST_START_FILE=str(marker)))
receipt = dict(argv=command, status=result.returncode, stdout=result.stdout.decode(),
               stderr=result.stderr.decode(), binary_sha256=sha(args.binary.read_bytes()),
               driver_sha256=sha(Path(__file__).read_bytes()), reference_sha256=sha(reference_raw))
(args.work / 'command.json').write_bytes(encode(receipt))
assert result.returncode == 0, receipt
assert marker.is_file()
report_raw = (root / 'storage-prefix-result.json').read_bytes()
report = json.loads(report_raw)
assert report_raw == encode(report) and result.stdout == report_raw + b'\n'
assert report['schema'] == 'gomad.replay-storage-prefix-result/v1'
assert report['input_sha256'] == sha(input_path.read_bytes())
assert report['module_sha256'] == module_digest
assert report['full_projection']['decisions'] == 2
assert report['prefix_projection']['decisions'] == 1
rows = {row['name']: row for row in report['executions']}
assert len(rows) == 5
for row in rows.values():
    for stream in row['evidence'].values():
        raw = (root / stream['file']).read_bytes()
        assert len(raw) == stream['bytes'] and sha(raw) == stream['sha256']
assert rows['full-record']['outcome']['termination'] == 'exit'
assert rows['full-record']['outcome']['exit_code'] == 0
assert rows['full-record']['outcome']['local_model']['finished'] is True
assert rows['crash-record']['outcome']['termination'] == 'storage_crash'
remaining = rows['full-replay']['outcome']
assert remaining['termination'] == 'divergence', remaining
metadata = remaining['local_model']
divergence = remaining['divergence']
assert metadata['cursor'] == 1 and metadata['tape_records'] == 2 and not metadata['finished']
assert divergence['Ordinal'] == 1 and divergence['Reason'] == 7 and divergence['TapeRecords'] == 2
assert divergence['Expected'] == metadata['next_expected'] and metadata['next_expected'] is not None
assert divergence['Observed'] is None
for name in ('prefix-replay', 'prefix-repeat'):
    outcome = rows[name]['outcome']
    assert outcome['termination'] == 'storage_crash', outcome
    assert outcome['message'] == 'modeled storage crash: stop-sync'
    assert 'divergence' not in outcome
    metadata = outcome['local_model']
    assert metadata['cursor'] == metadata['tape_records'] == 1
    assert metadata['next_expected'] is None and metadata['finished'] is False
    assert metadata['tape_sha256'] == report['prefix_projection']['tape_sha256']
assert rows['prefix-replay']['evidence'] == {
    key: dict(value, file=rows['prefix-replay']['evidence'][key]['file'])
    for key, value in rows['prefix-repeat']['evidence'].items()}
print('storage prefix: real two-choice tape rejects unconsumed ordinal 1; authentic one-choice prefix interrupts cleanly')
