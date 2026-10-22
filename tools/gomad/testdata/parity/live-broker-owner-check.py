#!/usr/bin/env python3
"""Execute the original helper/namespace ownership cells against observed Go-owner bytes."""
import argparse
import base64
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import struct
import sys

sys.dont_write_bytecode = True


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def decode(value):
    return base64.b64decode(value or '', validate=True)


def configuration(request):
    config = request['Config']
    limits = config['Limits']
    choice = request['Runtime']['Choice']
    environment = dict(item.split('=', 1) for item in config['Environment'])
    return dict(engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1', profile=config['Profile'],
                limits=dict(memory_bytes=request['MemoryBytes'], fuel=request['Fuel'],
                            wall_millis=request['Timeout'] // 1000000, output_bytes=limits['OutputBytes'],
                            evidence_bytes=64 << 20),
                arguments=config['Args'], environment=environment,
                runtime_sha256=bytes(choice['ExecutionIdentity']['ImplementationSHA256']).hex(),
                model=dict(entropy_key=config['EntropyKey'], mode=1, limit=choice['Limit'], diagnostics=request['Runtime']['Diagnostics'],
                           epoch_nanos=config['Clock']['EpochNanos'], read_step_nanos=config['Clock']['ReadStepNanos'], now=int(config['Clock']['ReadStepNanos'] == 0),
                           working_directory_sha256=sha(config['WorkingDirectory'].encode()),
                           arguments=dict(bytes=0, count=0, chunks=0), environment=dict(bytes=0, count=0, chunks=0),
                           limits=dict(calls=limits['Calls'], transcript_bytes=limits['TranscriptBytes'], pending_events=limits['PendingEvents']),
                           digest_only=request['DigestTranscript']),
                namespace=dict(entries=[], writable_directories=[list(item.encode()) for item in config['WritableDirectories']],
                               preopens=[list(item.encode()) for item in sorted(config['WritableDirectories'])],
                               working_directory=list(config['WorkingDirectory'].encode()), stdin=list(decode(config['Stdin'])),
                               epoch_nanos=config['Clock']['EpochNanos'],
                               limits=dict(files=limits['Files'], descriptors=limits['Descriptors'],
                                           filesystem_bytes=limits['FilesystemBytes'], output_bytes=limits['OutputBytes'])), storage=None)


def require_diagnostic(records, expected, choice_limit):
    assert choice_limit >= 160 and len(records) % 96 == 0
    capacity = 64 + (choice_limit - 64) // 96 * 96
    assert 64 + len(records) <= capacity <= 64 << 20
    header = bytearray(64)
    header[:8] = b'GOMADDG\x01'
    struct.pack_into('>I', header, 8, 1)
    header[12] = 1
    struct.pack_into('>QQQ', header, 16, capacity, 64 + len(records), len(records) // 96)
    assert bytes(header) + records == expected, 'independently framed diagnostic evidence differs'


def compare(root, expected, choice_limit):
    result = json.loads((root / 'result.json').read_bytes())
    outcome = result['outcome']
    assert outcome['termination'] == expected['Termination'] == 'exit'
    assert outcome['exit_code'] == expected['ExitCode'] == 0
    streams = {'stdout': expected['Stdout'], 'stderr': expected['Stderr'], 'transcript': expected['Transcript'],
               'choice': expected['ChoiceTrace']['Bytes'], 'diagnostic': expected['DiagnosticTrace']['Bytes']}
    for name, encoded in streams.items():
        actual = (root / name).read_bytes() if name in result['streams'] else b''
        original = decode(encoded)
        if name == 'diagnostic':
            require_diagnostic(actual, original, choice_limit)
        else:
            assert actual == original, f'{root.name}/{name}: owner evidence differs'
    metadata = outcome['local_model']
    assert metadata['finished'] and metadata['records'] > 0
    assert metadata['calls'] == expected['Calls']
    assert metadata['advances'] == expected['TimeAdvances']
    assert metadata['transcript_sha256'] == bytes(expected['TranscriptSHA256']).hex()
    metrics = outcome['metrics']
    assert metrics['prepare_nanos'] > 0 and metrics['compile_nanos'] > 0
    assert metrics['instantiate_nanos'] > 0 and metrics['execute_nanos'] > 0
    assert metrics['resource_usage_available'] and metrics['peak_rss_bytes'] > 0
    assert result['input_admission_nanos'] > 0 and result['wall_elapsed_nanos'] > 0
    assert result['authorization_to_terminal_nanos'] > 0
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('binary', 'observation', 'work'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    work = args.work.resolve()
    work.mkdir(mode=0o700, parents=True, exist_ok=False)
    observations = args.observation.resolve()
    derivation = json.loads((observations / 'derivation.json').read_bytes())
    index = json.loads((observations / 'observation-index.json').read_bytes())
    assert index['schema'] == 'gomad.live-broker-observation-index/v1'
    assert index['derivation_sha256'] == sha((observations / 'derivation.json').read_bytes())
    assert index['go_test_sha256'] == sha((observations / 'go-test.jsonl').read_bytes())
    for name, descriptor in index['files'].items():
        assert Path(name).name == name and name not in ('.', '..')
        raw = (observations / 'observations' / name).read_bytes()
        assert len(raw) == descriptor['bytes'] and sha(raw) == descriptor['sha256']
    assert derivation['schema'] == 'gomad.live-broker-observation-derivation/v1'
    assert sha((observations / 'test-only-helper.go').read_bytes()) == derivation['helper_sha256']
    assert sha((observations / 'test-only.diff').read_bytes()) == derivation['diff_sha256']
    assert {37, 38}.issubset(derivation['roots']), 'both exact owner roots are required'
    observe_path = Path(__file__).with_name('live-broker-observe.py')
    spec = importlib.util.spec_from_file_location('broker_observe', observe_path)
    observe = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(observe)
    observe.validate_events((observations / 'go-test.jsonl').read_bytes(), [37, 38])
    rows = [json.loads((observations / 'observations' / name).read_bytes()) for name in index['files'] if name.startswith('observation-')]
    all_rows = rows
    rows = [row for row in rows if row['remote_model'] and row['test'].split('/')[0] in (observe.ROOTS[37], observe.ROOTS[38])]
    assert len(rows) == 6, f'exact owner matrix needs six Go-owner cells, got {len(rows)}'
    cells = {(row['test'].split('/')[0], row['request']['DigestTranscript'], row['request']['Config']['Clock']['ReadStepNanos']) for row in rows}
    expected = {(observe.ROOTS[37], digest, step) for digest in (False, True) for step in (0, 1000)}
    expected |= {(observe.ROOTS[38], digest, 1000) for digest in (False, True)}
    assert cells == expected, 'owner cells missing or duplicated'
    receipts = []
    for index, row in enumerate(sorted(rows, key=lambda row: row['ordinal'])):
        request, oracle = row['request'], row['result']
        local = [candidate for candidate in all_rows if not candidate['remote_model'] and candidate['test'] == row['test']
                 and candidate['request']['DigestTranscript'] == request['DigestTranscript']
                 and candidate['request']['Config']['Clock'] == request['Config']['Clock']
                 and candidate['request']['Runtime']['Choice']['Mode'] == 1]
        assert len(local) == 1, 'original helper-owner recording must be paired with its Go owner'
        assert local[0]['request']['Config'] == request['Config']
        assert local[0]['result']['EnvironmentIdentity'] == oracle['EnvironmentIdentity']
        assert local[0]['result']['GoFilesystemCalls'] == {}, 'original helper owner leaked filesystem callbacks'
        assert request['MemoryBytes'] == 256 << 20 and request['Fuel'] == 4000000000
        assert request['Timeout'] == 60000000000 and request['Runtime']['Seed'] == 7
        assert request['Runtime']['Choice']['Limit'] == 1 << 20
        module = Path(request['ModulePath'])
        assert module.stat().st_size <= 8 << 20 and sha(module.read_bytes()) == request['ModuleSHA256']
        assert oracle['Reaped'] and oracle['ChoiceTrace']['Bytes']
        calls = oracle['GoFilesystemCalls']
        if row['test'] == observe.ROOTS[38]:
            assert all(calls.get(name, 0) > 0 for name in ('path_open', 'fd_pwrite', 'fd_readdir', 'path_rename'))
            assert b'namespace owner parity' in decode(oracle['Stdout']) and decode(oracle['Stderr']) == b'namespace stderr\n'
        trace = work / f'{index}-oracle.choices'
        trace.write_bytes(decode(oracle['ChoiceTrace']['Bytes']))
        for mode in (1, 2, 3) if row['test'].startswith(observe.ROOTS[37]) else (1, 2):
            name = f'cell-{index}-mode-{mode}'
            payload = dict(schema='gomad.live-broker-input/v1', module_path=str(module), module_sha256=request['ModuleSHA256'],
                           seed=7, configuration=configuration(request))
            if mode != 1:
                payload['forcing'] = dict(mode=mode, trace_path=str(trace), identity=request['Runtime']['Choice']['ExecutionIdentity'])
            input_path = work / (name + '.json')
            input_path.write_text(json.dumps(payload, sort_keys=True, separators=(',', ':')))
            result = subprocess.run([str(args.binary.resolve()), 'fixture-live-broker', '--root', str(work / name), '--input', str(input_path)],
                                    capture_output=True, timeout=90)
            (work / (name + '.stdout')).write_bytes(result.stdout)
            (work / (name + '.stderr')).write_bytes(result.stderr)
            receipts.append(dict(test=row['test'], digest=request['DigestTranscript'], step=request['Config']['Clock']['ReadStepNanos'], mode=mode,
                                 command=result.args, exit=result.returncode, oracle_sha256=sha(json.dumps(row, sort_keys=True).encode())))
            (work / 'commands.json').write_text(json.dumps(receipts, indent=2) + '\n')
            assert result.returncode == 0, (name, result.returncode, result.stderr)
            compare(work / name, oracle, request['Runtime']['Choice']['Limit'])
    (work / 'result.json').write_text(json.dumps(dict(schema='gomad.live-broker-owner-check/v1', roots=[37, 38], executions=len(receipts),
         derivation_sha256=sha((observations / 'derivation.json').read_bytes()), binary_sha256=sha(args.binary.read_bytes())), indent=2) + '\n')
    print('helper/namespace original ownership cells: 16 linked executions matched')


if __name__ == '__main__':
    main()
